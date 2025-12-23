package service

import (
	"backend/client"
	"backend/model"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// 1. Interface Definition
type ChartInkService interface {
	FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error)
	FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error)
}

// 2. Concrete Implementation Struct
type ChartInkServiceImpl struct {
	client        *client.ChartinkClient
	marginService MarginService // Properly injected MarginService
	// FIX: Replaced sync.Map with go-cache
	cache     *cache.Cache
	xsrfToken string
	cookie    string
	userAgent string
	mu        sync.RWMutex
}

// NewChartInkService acts as the Constructor
func NewChartInkService(c *client.ChartinkClient, ms MarginService) ChartInkService {
	return &ChartInkServiceImpl{
		client:        c,
		marginService: ms,
		// FIX: Set default expiration to 1 minute, cleanup every 2 minutes
		cache:     cache.New(1*time.Minute, 2*time.Minute),
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
}

func (s *ChartInkServiceImpl) FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	s.mu.RLock()
	hasTokens := s.xsrfToken != "" && s.cookie != ""
	s.mu.RUnlock()

	if !hasTokens {
		s.refreshTokens(ctx)
	}

	payload := map[string]string{"scan_clause": strategy.ScanClause}

	s.mu.RLock()
	res, err := s.client.FetchData(ctx, s.xsrfToken, s.cookie, s.userAgent, payload)
	s.mu.RUnlock()

	if err != nil {
		s.refreshTokens(ctx)
		s.mu.RLock()
		res, err = s.client.FetchData(ctx, s.xsrfToken, s.cookie, s.userAgent, payload)
		s.mu.RUnlock()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch data after retry: %w", err)
		}
	}

	var dto model.ChartInkResponseDto
	if err := json.Unmarshal([]byte(res), &dto); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	// FIX: Store in cache with the default 1-minute expiration
	s.cache.Set(strategy.Name, &dto, cache.DefaultExpiration)
	return &dto, nil
}

func (s *ChartInkServiceImpl) refreshTokens(ctx context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resp, _, err := s.client.GetHomepage(ctx)
	if err != nil {
		return
	}

	cookies := resp.Cookies()
	var cookieParts []string
	for _, c := range cookies {
		if c.Name == "XSRF-TOKEN" {
			decoded, _ := url.QueryUnescape(c.Value)
			s.xsrfToken = decoded
		}
		if c.Name == "ci_session" || c.Name == "laravel_session" || c.Name == "XSRF-TOKEN" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
	}
	s.cookie = strings.Join(cookieParts, "; ")
}

func (s *ChartInkServiceImpl) FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error) {
	// FIX: Use go-cache Get instead of sync.Map Load
	val, ok := s.cache.Get(strategy.Name)
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

	var result []model.StockMarginDto

	marginStore := s.marginService.GetStore()
	for _, stock := range response.Data {
		// FIX: Replaced mocked global lookup with the injected MarginService
		val, exists := marginStore.Get(stock.NSECode)
		if !exists {
			continue
		}

		margin, ok := val.(model.Margin)
		if !ok {
			continue
		}

		result = append(result, model.StockMarginDto{
			Name:   stock.Name,
			Symbol: stock.NSECode,
			Margin: margin.Margin,
			Close:  stock.Close,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})

	return result, nil
}
