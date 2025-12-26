package service

import (
	localCache "backend/cache"
	"backend/model"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/andybalholm/brotli" // Required for 'br' encoding
	"github.com/patrickmn/go-cache"
)

var (
	nseUrl             = "https://www.nseindia.com"
	userAgent          = "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Mobile Safari/537.36"
	historicalEndpoint = "/api/NextApi/apiClient/GetQuoteApi?functionName=getHistoricalTradeData&symbol=%s&series=EQ&fromDate=%s&toDate=%s"
	heatMapEndpoint    = "/api/heatmap-index?type=Sectoral%20Indices"
)

type NseService interface {
	FetchStockData(symbol string) ([]model.NSEHistoricalData, error)
	FetchHeatMap() ([]model.SectorData, error)
}

type NseServiceImpl struct {
	client http.Client
}

func NewNseService() NseService {
	jar, _ := cookiejar.New(nil)
	return &NseServiceImpl{
		client: http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

func (s *NseServiceImpl) FetchStockData(symbol string) ([]model.NSEHistoricalData, error) {

	cacheKey := "history_" + symbol
	if cachedData, found := localCache.NseHistoryCache.Get(cacheKey); found {
		return cachedData.([]model.NSEHistoricalData), nil
	}

	if err := s.WarmUp(); err != nil {
		return nil, err
	}

	// 3. Execute API Call
	data, err := s.executeRequest(symbol)
	if err != nil {
		return nil, err
	}

	// 4. Save to Cache for 5 minutes before returning
	if len(data) > 0 {
		localCache.NseHistoryCache.Set(cacheKey, data, cache.DefaultExpiration)
	}

	return data, nil
}

func (s *NseServiceImpl) WarmUp() error {
	req, _ := http.NewRequest("GET", nseUrl, nil)
	req.Header.Set("User-Agent", userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return nil
}

func (s *NseServiceImpl) executeRequest(symbol string) ([]model.NSEHistoricalData, error) {

	now := time.Now()

	// 2. Calculate one month ago (From Date)
	// .AddDate(years, months, days)
	oneMonthAgo := now.AddDate(0, -1, 0)

	// 3. Format according to NSE requirement: DD-MM-YYYY
	toDate := now.Format("02-01-2006")
	fromDate := oneMonthAgo.Format("02-01-2006")

	url := nseUrl + fmt.Sprintf(historicalEndpoint, symbol, fromDate, toDate)

	req, _ := http.NewRequest("GET", url, nil)

	// 2. Set EXACT headers from your browser request
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br") // Browser defaults
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", fmt.Sprintf("https://www.nseindia.com/get-quote/equity/%s", symbol))
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("nse returned status: %d", resp.StatusCode)
	}

	// 3. MULTI-ENCODING DECOMPRESSION
	var reader io.ReadCloser
	encoding := resp.Header.Get("Content-Encoding")

	switch encoding {
	case "br":
		reader = io.NopCloser(brotli.NewReader(resp.Body))
	case "gzip":
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	default:
		reader = resp.Body
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	// 4. Handle Wrapper Structure
	var wrapper struct {
		Data []model.NSEHistoricalData `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapper); err != nil {
		var direct []model.NSEHistoricalData
		if err2 := json.Unmarshal(bodyBytes, &direct); err2 == nil {
			return direct, nil
		}
		return nil, fmt.Errorf("json unmarshal failed: %v", err)
	}

	return wrapper.Data, nil
}

func (s *NseServiceImpl) FetchHeatMap() ([]model.SectorData, error) {
	if err := s.WarmUp(); err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", nseUrl+heatMapEndpoint, nil)

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.nseindia.com/market-data/live-market-indices/heatmap")
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("NSE returned status: %d", resp.StatusCode)
	}

	var reader io.ReadCloser
	encoding := resp.Header.Get("Content-Encoding")

	switch encoding {
	case "br":
		reader = io.NopCloser(brotli.NewReader(resp.Body))
	case "gzip":
		gzReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzReader.Close()
		reader = gzReader
	default:
		reader = resp.Body
	}

	bodyBytes, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}

	var data []model.SectorData

	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return nil, fmt.Errorf("json decode error: %v", err)
	}

	return data, nil

}
