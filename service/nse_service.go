package service

import (
	localCache "backend/cache"
	"backend/middleware"
	"backend/model"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
)

var (
	nseUrl         = "https://www.nseindia.com"
	userAgent      = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
	historicalPath = "/api/NextApi/apiClient/GetQuoteApi"
	heatMapPath    = "/api/heatmap-index"
)

type NseService interface {
	FetchStockData(symbol string) ([]model.NSEHistoricalData, error)
	FetchHeatMap() ([]model.SectorData, error)
}

type NseServiceImpl struct {
	client *resty.Client
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

func (s *NseServiceImpl) WarmUp() error {
	resp, err := s.client.R().
		SetHeader("Referer", "https://www.google.com/").
		SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8").
		SetHeader("Accept-Language", "en-US,en;q=0.9").
		Get("/")

	if err != nil || !resp.IsSuccess() {
		return fmt.Errorf("warmup failed: %v (status: %d)", err, resp.StatusCode())
	}
	return nil
}

func (s *NseServiceImpl) FetchStockData(symbol string) ([]model.NSEHistoricalData, error) {
	cacheKey := "history_" + symbol
	if cached, found := localCache.NseHistoryCache.Get(cacheKey); found {
		return cached.([]model.NSEHistoricalData), nil
	}

	if err := s.WarmUp(); err != nil {
		return nil, err
	}

	resp, err := s.setHeaders(s.client.R(), fmt.Sprintf("%s/get-quote/equity/%s", nseUrl, symbol)).
		SetQueryParams(map[string]string{
			"functionName": "getHistoricalTradeData",
			"symbol":       symbol,
			"series":       "EQ",
			"fromDate":     time.Now().AddDate(0, -1, 0).Format("02-01-2006"),
			"toDate":       time.Now().Format("02-01-2006"),
		}).
		Get(historicalPath)

	if err != nil || !resp.IsSuccess() {
		return nil, fmt.Errorf("NSE API error: %v", err)
	}

	var data []model.NSEHistoricalData
	if err := json.Unmarshal(resp.Body(), &data); err != nil {
		log.Printf("Error parsing historical data %s", err.Error())
		return nil, fmt.Errorf("historical data decode error: %w", err)
	}

	localCache.NseHistoryCache.Set(cacheKey, data, cache.DefaultExpiration)
	return data, nil
}

func (s *NseServiceImpl) FetchHeatMap() ([]model.SectorData, error) {
	cacheKey := "heatmap_sectoral"
	if cached, found := localCache.HeatMapCache.Get(cacheKey); found {
		return cached.([]model.SectorData), nil
	}

	if err := s.WarmUp(); err != nil {
		return nil, err
	}

	resp, err := s.setHeaders(s.client.R(), nseUrl+"/market-data/live-market-indices/heatmap").
		SetQueryParam("type", "Sectoral Indices").
		Get(heatMapPath)

	if err != nil || !resp.IsSuccess() {
		log.Println("Error calling sector data api %", err.Error())
		return nil, fmt.Errorf("heatmap error: %v", err)
	}

	var data []model.SectorData
	if err := json.Unmarshal(resp.Body(), &data); err != nil {
		log.Printf("Error parsing sector data %s", err.Error())
		return nil, fmt.Errorf("heatmap decode error: %w", err)
	}

	localCache.HeatMapCache.Set(cacheKey, data, cache.DefaultExpiration)
	return data, nil
}

func (s *NseServiceImpl) setHeaders(req *resty.Request, referer string) *resty.Request {
	headers := map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         referer,
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return req
}
