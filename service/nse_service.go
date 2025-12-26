package service

import (
	localCache "backend/cache"
	"backend/model"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"time"

	"github.com/andybalholm/brotli"
	"github.com/patrickmn/go-cache"
)

var (
	nseUrl             = "https://www.nseindia.com"
	userAgent          = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
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

// WarmUp ensures we have a fresh session/cookies
func (s *NseServiceImpl) WarmUp() error {
	req, _ := http.NewRequest("GET", nseUrl, nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Referer", "https://www.google.com/")

	resp, err := s.client.Do(req)
	if err != nil {
		log.Printf("Error in warmup %s", err.Error())
		return fmt.Errorf("warmup request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("warmup failed with status: %d", resp.StatusCode)
	}
	return nil
}

func (s *NseServiceImpl) FetchStockData(symbol string) ([]model.NSEHistoricalData, error) {
	cacheKey := "history_" + symbol
	if cached, found := localCache.NseHistoryCache.Get(cacheKey); found {
		return cached.([]model.NSEHistoricalData), nil
	}

	bodyBytes, err := s.doRequestWithRetry(fmt.Sprintf(historicalEndpoint, symbol,
		time.Now().AddDate(0, -1, 0).Format("02-01-2006"),
		time.Now().Format("02-01-2006")),
		fmt.Sprintf("https://www.nseindia.com/get-quote/equity/%s", symbol))

	if err != nil {
		return nil, err
	}

	// Try format A: {"data": [...]}
	var wrapper struct {
		Data []model.NSEHistoricalData `json:"data"`
	}
	if err := json.Unmarshal(bodyBytes, &wrapper); err == nil && len(wrapper.Data) > 0 {
		localCache.NseHistoryCache.Set(cacheKey, wrapper.Data, cache.DefaultExpiration)
		return wrapper.Data, nil
	}

	// Try format B: [...] (Direct Array)
	var direct []model.NSEHistoricalData
	if err := json.Unmarshal(bodyBytes, &direct); err == nil {
		if len(direct) > 0 {
			localCache.NseHistoryCache.Set(cacheKey, direct, cache.DefaultExpiration)
		}
		return direct, nil
	}

	return nil, fmt.Errorf("failed to parse NSE JSON. Preview: %s", string(bodyBytes[:min(50, len(bodyBytes))]))
}

func (s *NseServiceImpl) FetchHeatMap() ([]model.SectorData, error) {
	cacheKey := "heatmap_sectoral"
	if cached, found := localCache.HeatMapCache.Get(cacheKey); found {
		return cached.([]model.SectorData), nil
	}

	bodyBytes, err := s.doRequestWithRetry(heatMapEndpoint, "https://www.nseindia.com/market-data/live-market-indices/heatmap")
	if err != nil {
		return nil, err
	}

	var data []model.SectorData
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		log.Printf("Error parsing sector data %s", err.Error())
		return nil, fmt.Errorf("heatmap decode error: %w", err)
	}

	localCache.HeatMapCache.Set(cacheKey, data, cache.DefaultExpiration)
	return data, nil
}

// Internal helper to handle WarmUp, Execution, and Decompression
func (s *NseServiceImpl) doRequestWithRetry(endpoint, referer string) ([]byte, error) {
	if err := s.WarmUp(); err != nil {
		return nil, err
	}

	req, _ := http.NewRequest("GET", nseUrl+endpoint, nil)
	s.setHeaders(req, referer)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error calling nse api %s err: {%d}", endpoint, resp.StatusCode)
		return nil, fmt.Errorf("NSE returned status: %d", resp.StatusCode)
	}

	return s.decompressBody(resp)
}

func (s *NseServiceImpl) decompressBody(resp *http.Response) ([]byte, error) {
	var reader io.ReadCloser
	switch resp.Header.Get("Content-Encoding") {
	case "br":
		reader = io.NopCloser(brotli.NewReader(resp.Body))
	case "gzip":
		gz, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gz.Close()
		reader = gz
	default:
		reader = resp.Body
	}
	return io.ReadAll(reader)
}

func (s *NseServiceImpl) setHeaders(req *http.Request, referer string) {
	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         referer,
		"User-Agent":      userAgent,
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
}
