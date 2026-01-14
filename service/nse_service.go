package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/cookiejar"
	"strconv"
	"sync"
	"time"

	"backend/cache"
	"backend/client"
	"backend/database"
	"backend/middleware"
	"backend/model"
	"backend/util"

	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"golang.org/x/sync/singleflight"
)

const (
	nseUrl                = "https://www.nseindia.com"
	userAgent             = "Mozilla/5.0 (iPhone; CPU iPhone OS 18_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.5 Mobile/15E148 Safari/604.1"
	historicalPath        = "/api/NextApi/apiClient/GetQuoteApi"
	heatMapPath           = "/api/heatmap-index"
	allIndicesPath        = "/api/allindices"
	nseDateFormat         = "02-01-2006"
	deliveryPercentageUrl = "/api/historicalOR/generateSecurityWiseHistoricalData"
)

var sfGroup singleflight.Group

type NseService interface {
	FetchStockData(ctx context.Context, symbol string) ([]model.NSEHistoricalData, error)
	FetchAllIndices() ([]model.AllIndicesResponse, error)
	ClearStockDataCache(symbol string)
	FetchDeliveryData(ctx context.Context, symbol string) (float32, error)
}

type NseServiceImpl struct {
	client      *resty.Client
	sfGroup     singleflight.Group
	lastWarmup  time.Time
	warmupLock  sync.RWMutex
	yahooClient *client.YahooClient
}

func NewNseService(yahooClient *client.YahooClient) NseService {
	client := resty.New().
		SetBaseURL(nseUrl).
		SetTimeout(30*time.Second).
		SetHeader("User-Agent", userAgent).
		SetRetryCount(2).
		SetRetryWaitTime(1 * time.Second)

	client.OnAfterResponse(middleware.DecompressMiddleware)

	return &NseServiceImpl{client: client, yahooClient: yahooClient}
}

func (s *NseServiceImpl) WarmUp() error {
	s.warmupLock.RLock()
	isFresh := time.Since(s.lastWarmup) < 2*time.Minute
	s.warmupLock.RUnlock()

	if isFresh {
		return nil
	}

	_, err, _ := s.sfGroup.Do("nse-session-refresh", func() (any, error) {
		log.Info().Msg("Refreshing NSE session...")

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
			return nil, fmt.Errorf("NSE warmup failed: %v status: %d", err, resp.StatusCode())
		}

		s.warmupLock.Lock()
		s.lastWarmup = time.Now()
		s.warmupLock.Unlock()

		return nil, nil
	})
	return err
}

func (s *NseServiceImpl) FetchStockData(ctx context.Context, symbol string) ([]model.NSEHistoricalData, error) {
	val, err, _ := sfGroup.Do(symbol, func() (any, error) {
		if yahooResp, err := s.yahooClient.GetHistoricalData(ctx, symbol, model.Range1mo); err == nil {
			return yahooResp, nil
		}

		var data []model.NSEHistoricalData
		cacheKey := s.getHistoryCacheKey(symbol)
		if ok, _ := database.RedisHelper.GetAsStruct(cacheKey, &data); ok {
			return data, nil
		}

		err := s.executeNseRequest(
			fmt.Sprintf("%s/get-quote/equity/%s", nseUrl, symbol),
			historicalPath,
			map[string]string{
				"functionName": "getHistoricalTradeData",
				"symbol":       symbol,
				"series":       "EQ",
				"fromDate":     time.Now().AddDate(0, -1, 0).Format(nseDateFormat),
				"toDate":       time.Now().Format(nseDateFormat),
			},
			&data,
		)

		if err == nil {
			cache.GoSet(cacheKey, data, util.NseCacheExpiryTime())
		}

		time.AfterFunc(10*time.Second, func() {
			sfGroup.Forget(symbol)
		})

		return data, err
	})

	if err != nil {
		return nil, err
	}

	return val.([]model.NSEHistoricalData), nil
}

func (s *NseServiceImpl) FetchAllIndices() ([]model.AllIndicesResponse, error) {
	cacheKey := "heatmap_all_indices"
	var data []model.AllIndicesResponse

	if ok, _ := database.RedisHelper.GetAsStruct(cacheKey, &data); ok {
		return data, nil
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

	data = s.convertIndices(result.Data)
	cache.GoSet(cacheKey, data, time.Hour)
	return data, nil
}

func (s *NseServiceImpl) executeNseRequest(referer, path string, params map[string]string, target any) error {
	if err := s.WarmUp(); err != nil {
		return err
	}

	req := s.client.R().SetHeaders(s.getStandardHeaders(referer))

	if params != nil {
		req.SetQueryParams(params)
	}

	resp, err := req.Get(path)
	if err != nil || !resp.IsSuccess() {
		return fmt.Errorf("NSE request failed for path %s: %v", path, err)
	}

	if err := json.Unmarshal(resp.Body(), target); err != nil {
		return fmt.Errorf("failed to decode NSE response for path %s: %w", path, err)
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
	cache.GoDelete(s.getHistoryCacheKey(symbol))
}

func (s *NseServiceImpl) getHistoryCacheKey(symbol string) string {
	return "nse_history_" + symbol
}

func (s *NseServiceImpl) getStandardHeaders(referer string) map[string]string {
	return map[string]string{
		"Accept":          "*/*",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         referer,
		"sec-fetch-dest":  "empty",
		"sec-fetch-mode":  "cors",
		"sec-fetch-site":  "same-origin",
	}
}

func (s *NseServiceImpl) FetchDeliveryData(ctx context.Context, symbol string) (float32, error) {
	var resp model.NseDeliveryData
	err := s.executeNseRequest("https://www.nseindia.com/report-detail/eq_security", deliveryPercentageUrl,
		map[string]string{
			"from":   time.Now().AddDate(0, 0, -7).Format(nseDateFormat),
			"to":     time.Now().Format(nseDateFormat),
			"symbol": symbol,
			"type":   "priceVolumeDeliverable",
			"series": "ALL",
		}, &resp)

	if err != nil {
		log.Err(err).Msg("Error calling nse delivery api")
		return 0, nil
	}

	data := resp.Data
	return data[len(data)-1].DeliveryPercent, nil
}
