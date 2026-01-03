package client

import (
	"backend/database"
	"backend/model"
	"backend/util"
	"context"
	"fmt"
	"log"
	"slices"
	"strconv"
	"time"

	"github.com/go-resty/resty/v2"
)

type YahooClient struct {
	client *resty.Client
}

func NewYahooClient() *YahooClient {
	client := resty.New().
		SetBaseURL("https://query1.finance.yahoo.com/v8/finance/chart").
		SetTimeout(10 * time.Second).
		SetHeaders(map[string]string{
			"Content-Type": "application/json",
			"Accept":       "application/json",
			"User-Agent":   "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		})

	return &YahooClient{
		client: client,
	}
}

func (y *YahooClient) GetHistoricalData(ctx context.Context, symbol string, timeRange model.YahooTimeRange) ([]model.NSEHistoricalData, error) {
	cacheKey := "yahoo_history_" + symbol + "_" + string(timeRange)
	var data []model.NSEHistoricalData

	if ok, _ := database.RedisHelper.GetAsStruct(cacheKey, &data); ok {
		return data, nil
	}

	var chartResponse model.YahooChartResponse
	resp, err := y.client.R().
		SetQueryParams(map[string]string{
			"range":    string(timeRange),
			"interval": string(model.Range1d),
		}).SetResult(&chartResponse).
		Get("/" + symbol + ".NS")

	if err != nil || !resp.IsSuccess() || chartResponse.Chart.Error != nil {
		log.Println("Error calling yahoo api")
		return nil, fmt.Errorf("Yahoo request failed: %v", err)
	}

	list := make([]model.NSEHistoricalData, 0, 10)

	timeframes := chartResponse.Chart.Result[0].Timestamp
	quote := chartResponse.Chart.Result[0].Indicators.Quote[0]
	open := quote.Open
	high := quote.High
	low := quote.Low
	close := quote.Close
	volume := quote.Volume
	for i := range timeframes {
		if volume[i] > 0 && open[i] != 0 {
			list = append(list, model.NSEHistoricalData{
				Symbol:    symbol,
				Open:      formatToTwo(open[i]),
				High:      formatToTwo(high[i]),
				Low:       formatToTwo(low[i]),
				Close:     formatToTwo(close[i]),
				Timestamp: time.Unix(timeframes[i], 0).In(util.IstLocation).Format(util.InputLayout),
			})
		}
	}

	if len(list) > 0 {
		slices.Reverse(list)
		database.RedisHelper.Set(cacheKey, list, util.NseCacheExpiryTime())
	}

	return list, nil
}

func formatToTwo(n float64) float64 {
	val, _ := strconv.ParseFloat(fmt.Sprintf("%.2f", n), 64)
	return val
}
