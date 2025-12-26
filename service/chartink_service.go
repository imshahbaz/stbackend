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

	"github.com/patrickmn/go-cache"
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

// FetchData handles the API call and CSRF retry logic
func (s *ChartInkServiceImpl) FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	s.mu.RLock()
	token := s.xsrfToken
	s.mu.RUnlock()

	if token == "" {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		token = s.xsrfToken
	}

	payload := map[string]string{"scan_clause": strategy.ScanClause}
	resp, err := s.client.FetchData(ctx, token, s.userAgent, payload)

	// Retry on 419 (CSRF Mismatch)
	if err != nil || (resp != nil && resp.StatusCode() == 419) {
		if err := s.refreshTokens(ctx); err != nil {
			return nil, err
		}
		resp, err = s.client.FetchData(ctx, s.xsrfToken, s.userAgent, payload)
		if err != nil {
			return nil, err
		}
	}

	if !resp.IsSuccess() {
		return nil, fmt.Errorf("chartink api error: %d", resp.StatusCode())
	}

	var dto model.ChartInkResponseDto
	if err := json.Unmarshal(resp.Body(), &dto); err != nil {
		return nil, fmt.Errorf("failed to parse chartink json: %w", err)
	}

	localCache.ChartInkResponseCache.Set(strategy.Name, &dto, cache.DefaultExpiration)
	return &dto, nil
}

// FetchWithMargin joins ChartInk results with local Margin data
func (s *ChartInkServiceImpl) FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error) {
	// 1. Check Cache first
	val, ok := localCache.ChartInkResponseCache.Get(strategy.Name)
	var response *model.ChartInkResponseDto

	if !ok {
		var err error
		response, err = s.FetchData(strategy)
		if err != nil {
			return nil, err
		}
	} else {
		response = val.(*model.ChartInkResponseDto)
	}

	// 2. Map Stock results to Margin data
	var result []model.StockMarginDto
	marginStore := localCache.MarginCache

	for _, stock := range response.Data {
		marginVal, exists := marginStore.Get(stock.NSECode)
		if !exists {
			continue
		}

		m, ok := marginVal.(model.Margin)
		if !ok {
			continue
		}

		result = append(result, model.StockMarginDto{
			Name:   stock.Name,
			Symbol: stock.NSECode,
			Margin: m.Margin,
			Close:  stock.Close,
		})
	}

	// 3. Sort by Margin (Highest first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})

	return result, nil
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
	return fmt.Errorf("could not find XSRF-TOKEN")
}
