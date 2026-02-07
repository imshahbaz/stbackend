package service

import (
	localCache "backend/cache"
	"backend/client"
	"backend/model"
	"backend/util"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

const (
	tokenKey          = "XSRF-TOKEN"
	chartInkUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
)

type ChartInkService interface {
	FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error)
	FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error)
	FetchBacktestData(strategy model.StrategyDto) ([]model.ChartinkBacktestSignal, error)
	FetchBacktestWithMargin(strategy model.StrategyDto) ([]model.ChartinkBacktestSignalWithMargin, error)
	FetchBacktestTodayWithMargin(strategy model.StrategyDto) ([]model.ChartinkBacktestSignalWithMargin, error)
}

type ChartInkServiceImpl struct {
	client        *client.ChartinkClient
	marginService MarginService
	xsrfToken     string
	userAgent     string
	mu            sync.RWMutex
	nseSvc        NseService
}

func NewChartInkService(c *client.ChartinkClient, ms MarginService, nseSvc NseService) ChartInkService {
	return &ChartInkServiceImpl{
		client:        c,
		marginService: ms,
		userAgent:     chartInkUserAgent,
		nseSvc:        nseSvc,
	}
}

func (s *ChartInkServiceImpl) FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	resp, err := s.executeWithRetry(ctx, strategy.ScanClause)
	if err != nil {
		return nil, err
	}

	var dto model.ChartInkResponseDto
	if err := json.Unmarshal(resp.Body(), &dto); err != nil {
		return nil, fmt.Errorf("failed to parse chartink json: %w", err)
	}

	return &dto, nil
}

func (s *ChartInkServiceImpl) FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error) {
	result := make([]model.StockMarginDto, 0)
	if ok, err := localCache.GetChartInkResponseCache(strategy.Name, &result); ok && err == nil {
		return result, nil
	}

	response, err := s.FetchData(strategy)
	if err != nil {
		return nil, err
	}

	for _, stock := range response.Data {
		if m, exists := s.marginService.GetMargin(stock.NSECode); exists {
			result = append(result, model.StockMarginDto{
				Name:   stock.Name,
				Symbol: stock.NSECode,
				Margin: m.Margin,
				Close:  stock.Close,
			})
		}
	}

	if strategy.Name == "BULLISH MARUBOZU" {
		delMap, err := s.nseSvc.GetDeliveryDataMap(context.Background())
		if err != nil {
			return nil, err
		}
		filteredData := make([]model.StockMarginDto, 0, len(result))
		for i := range result {
			if delMap[result[i].Symbol] > 50 {
				result[i].DeliveryPercent = float32(delMap[result[i].Symbol])
				filteredData = append(filteredData, result[i])
			}
		}
		result = filteredData
	}

	s.sortResultByMargin(result)
	localCache.SetChartInkResponseCache(strategy.Name, result)
	return result, nil
}

func (s *ChartInkServiceImpl) executeWithRetry(ctx context.Context, scanClause string) (*resty.Response, error) {
	payload := map[string]string{"scan_clause": scanClause}

	token := s.getStoredToken()
	if token == "" {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		token = s.getStoredToken()
	}

	resp, err := s.client.FetchData(ctx, token, s.userAgent, payload)

	if err != nil || (resp != nil && resp.StatusCode() == 419) {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		resp, err = s.client.FetchData(ctx, s.getStoredToken(), s.userAgent, payload)
	}

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("chartink api error: %d status: %s", resp.StatusCode(), resp.Status())
	}

	return resp, nil
}

func (s *ChartInkServiceImpl) getStoredToken() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.xsrfToken
}

func (s *ChartInkServiceImpl) refreshTokens(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp, err := s.client.GetHomepage(ctx)
	if err != nil {
		return err
	}

	for _, c := range resp.Cookies() {
		if c.Name == tokenKey {
			decoded, _ := url.QueryUnescape(c.Value)
			s.xsrfToken = decoded
			return nil
		}
	}
	return fmt.Errorf("%s not found in cookies", tokenKey)
}

func (s *ChartInkServiceImpl) sortResultByMargin(result []model.StockMarginDto) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})
}

func (s *ChartInkServiceImpl) sortMarginByMargin(result []model.Margin) {
	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})
}

func (s *ChartInkServiceImpl) FetchBacktestData(strategy model.StrategyDto) ([]model.ChartinkBacktestSignal, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := s.executeBacktestWithRetry(ctx, strategy.ScanClause)
	if err != nil {
		return nil, err
	}

	var backtestResp model.ChartinkBacktestResponse
	if err := json.Unmarshal(resp.Body(), &backtestResp); err != nil {
		return nil, fmt.Errorf("failed to parse chartink backtest json: %w", err)
	}

	signals := make([]model.ChartinkBacktestSignal, 0)
	if len(backtestResp.MetaData) > 0 {
		meta := backtestResp.MetaData[0]
		for i, tradeTime := range meta.TradeTimes {
			ts := tradeTime
			if ts > 10000000000 {
				ts = ts / 1000
			}
			marketTime := time.Unix(ts, 0).In(util.IstLocation).Format("2006-01-02 15:04:05")

			stocks := make([]string, 0)
			if i < len(backtestResp.AggregatedStockList) {
				stockData := backtestResp.AggregatedStockList[i]
				for j := 0; j < len(stockData); j += 3 {
					stocks = append(stocks, stockData[j])
				}
			}

			signals = append(signals, model.ChartinkBacktestSignal{
				MarketTime: marketTime,
				Stocks:     stocks,
			})
		}
	}

	return signals, nil
}

func (s *ChartInkServiceImpl) FetchBacktestWithMargin(strategy model.StrategyDto) ([]model.ChartinkBacktestSignalWithMargin, error) {
	signals, err := s.FetchBacktestData(strategy)
	if err != nil {
		return nil, err
	}

	result := make([]model.ChartinkBacktestSignalWithMargin, 0)
	for _, signal := range signals {
		enrichedStocks := make([]model.Margin, 0)
		for _, symbol := range signal.Stocks {
			if m, exists := s.marginService.GetMargin(symbol); exists {
				enrichedStocks = append(enrichedStocks, *m)
			}
		}

		if len(enrichedStocks) > 0 {
			s.sortMarginByMargin(enrichedStocks)
			result = append(result, model.ChartinkBacktestSignalWithMargin{
				MarketTime: signal.MarketTime,
				Stocks:     enrichedStocks,
			})
		}
	}

	return result, nil
}

func (s *ChartInkServiceImpl) FetchBacktestTodayWithMargin(strategy model.StrategyDto) ([]model.ChartinkBacktestSignalWithMargin, error) {
	signals, err := s.FetchBacktestWithMargin(strategy)
	if err != nil {
		return nil, err
	}

	today := time.Now().In(util.IstLocation).Format("2006-01-02")
	result := make([]model.ChartinkBacktestSignalWithMargin, 0)
	for _, signal := range signals {
		if len(signal.MarketTime) >= 10 && signal.MarketTime[:10] == today {
			result = append(result, signal)
		}
	}

	return result, nil
}

func (s *ChartInkServiceImpl) executeBacktestWithRetry(ctx context.Context, scanClause string) (*resty.Response, error) {
	payload := map[string]string{"scan_clause": scanClause, "max_rows": "65"}

	token := s.getStoredToken()
	if token == "" {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		token = s.getStoredToken()
	}

	resp, err := s.client.FetchBackTestData(ctx, token, s.userAgent, payload)

	if err != nil || (resp != nil && resp.StatusCode() == 419) {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		resp, err = s.client.FetchBackTestData(ctx, s.getStoredToken(), s.userAgent, payload)
	}

	if err != nil {
		return nil, err
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("chartink backtest api error: %d status: %s", resp.StatusCode(), resp.Status())
	}

	return resp, nil
}
