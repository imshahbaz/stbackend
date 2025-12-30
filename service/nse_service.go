package service

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http/cookiejar"
	"strconv"
	"sync"
	"time"

	localCache "backend/cache"
	"backend/middleware"
	"backend/model"

	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"golang.org/x/sync/singleflight"
)

const (
	nseUrl         = "https://www.nseindia.com"
	userAgent      = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
	historicalPath = "/api/NextApi/apiClient/GetQuoteApi"
	heatMapPath    = "/api/heatmap-index"
	allIndicesPath = "/api/allindices"
)

type NseService interface {
	FetchStockData(symbol string) ([]model.NSEHistoricalData, error)
	FetchHeatMap() ([]model.SectorData, error)
	FetchAllIndices() ([]model.AllIndicesResponse, error)
	ClearStockDataCache(symbol string)
}

type NseServiceImpl struct {
	client     *resty.Client
	sfGroup    singleflight.Group
	lastWarmup time.Time
	warmupLock sync.RWMutex
}

func NewNseService() NseService {
	client := resty.New().
		SetBaseURL(nseUrl).
		SetTimeout(30*time.Second).
		SetHeader("User-Agent", userAgent).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)

	client.OnAfterResponse(middleware.DecompressMiddleware)

	return &NseServiceImpl{client: client}
}

// WarmUp ensures we have a valid session cookie from NSE.
func (s *NseServiceImpl) WarmUp() error {
	s.warmupLock.RLock()
	isFresh := time.Since(s.lastWarmup) < 2*time.Minute
	s.warmupLock.RUnlock()

	if isFresh {
		return nil
	}

	_, err, _ := s.sfGroup.Do("nse-session-refresh", func() (any, error) {
		log.Println("Refreshing NSE session...")

		newJar, _ := cookiejar.New(nil)
		s.client.SetCookieJar(newJar)

		resp, err := s.client.R().
			SetHeaders(map[string]string{
				"Referer":         "https://www.google.com/",
				"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
				"Accept-Language": "en-US,en;q=0.9",
			}).
			Get("/")

		if err != nil || !resp.IsSuccess() {
			return nil, fmt.Errorf("warmup failed: %v", err)
		}

		s.warmupLock.Lock()
		s.lastWarmup = time.Now()
		s.warmupLock.Unlock()

		return nil, nil
	})
	return err
}

func (s *NseServiceImpl) FetchStockData(symbol string) ([]model.NSEHistoricalData, error) {
	cacheKey := "history_" + symbol
	if val, found := localCache.NseHistoryCache.Get(cacheKey); found {
		return val.([]model.NSEHistoricalData), nil
	}

	var data []model.NSEHistoricalData
	err := s.executeNseRequest(
		fmt.Sprintf("%s/get-quote/equity/%s", nseUrl, symbol),
		historicalPath,
		map[string]string{
			"functionName": "getHistoricalTradeData",
			"symbol":       symbol,
			"series":       "EQ",
			"fromDate":     time.Now().AddDate(0, -1, 0).Format("02-01-2006"),
			"toDate":       time.Now().Format("02-01-2006"),
		},
		&data,
	)

	if err == nil {
		localCache.NseHistoryCache.Set(cacheKey, data, cache.DefaultExpiration)
	}
	return data, err
}

func (s *NseServiceImpl) FetchHeatMap() ([]model.SectorData, error) {
	cacheKey := "heatmap_sectoral"
	if val, found := localCache.HeatMapCache.Get(cacheKey); found {
		return val.([]model.SectorData), nil
	}

	var data []model.SectorData
	err := s.executeNseRequest(
		nseUrl+"/market-data/live-market-indices/heatmap",
		heatMapPath,
		map[string]string{"type": "Sectoral Indices"},
		&data,
	)

	if err == nil {
		localCache.HeatMapCache.Set(cacheKey, data, cache.DefaultExpiration)
	}
	return data, err
}

func (s *NseServiceImpl) FetchAllIndices() ([]model.AllIndicesResponse, error) {
	cacheKey := "heatmap_all_indices"
	if val, found := localCache.HeatMapCache.Get(cacheKey); found {
		return val.([]model.AllIndicesResponse), nil
	}

	var result model.NseResponseWrapper[model.NseIndexData]
	err := s.executeNseRequest(
		nseUrl+"/market-data/live-market-indices",
		allIndicesPath,
		nil,
		&result,
	)

	if err != nil {
		return nil, err
	}

	data := s.convertIndices(result.Data)
	localCache.HeatMapCache.Set(cacheKey, data, cache.DefaultExpiration)
	return data, nil
}

// --- Private Helpers ---

func (s *NseServiceImpl) executeNseRequest(referer, path string, params map[string]string, target any) error {
	if err := s.WarmUp(); err != nil {
		return err
	}

	req := s.client.R().SetHeaders(map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         referer,
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
	})

	if params != nil {
		req.SetQueryParams(params)
	}

	resp, err := req.Get(path)
	if err != nil || !resp.IsSuccess() {
		return fmt.Errorf("NSE request failed: %v", err)
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		return fmt.Errorf("decode error: %w", err)
	}

	return nil
}

func (s *NseServiceImpl) convertIndices(input []model.NseIndexData) []model.AllIndicesResponse {
	output := make([]model.AllIndicesResponse, 0)
	for _, val := range input {
		if val.Key == "SECTORAL INDICES" && val.OneWeekAgoVal != 0 {
			change := ((val.Last - val.OneWeekAgoVal) / val.OneWeekAgoVal) * 100
			output = append(output, model.AllIndicesResponse{
				NseIndexData: val,
				PerChange1w:  s.formatToTwo(change),
			})
		}
	}
	return output
}

func (s *NseServiceImpl) formatToTwo(n float64) float64 {
	val, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", n), 64)
	return val
}

func (s *NseServiceImpl) ClearStockDataCache(symbol string) {
	cacheKey := "history_" + symbol
	localCache.NseHistoryCache.Delete(cacheKey)
}
