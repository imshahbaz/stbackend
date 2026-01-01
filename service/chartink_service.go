package service

import (
	localCache "backend/cache"
	"backend/client"
	"backend/model"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type ChartInkService interface {
	FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error)
	FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error)
}

type ChartInkServiceImpl struct {
	client        *client.ChartinkClient
	marginService MarginService
	xsrfToken     string
	userAgent     string
	mu            sync.RWMutex
}

func NewChartInkService(c *client.ChartinkClient, ms MarginService) ChartInkService {
	return &ChartInkServiceImpl{
		client:        c,
		marginService: ms,
		userAgent:     "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})

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
		return nil, fmt.Errorf("chartink api error: %d", resp.StatusCode())
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
		if c.Name == "XSRF-TOKEN" {
			decoded, _ := url.QueryUnescape(c.Value)
			s.xsrfToken = decoded
			return nil
		}
	}
	return fmt.Errorf("XSRF-TOKEN not found in cookies")
}
