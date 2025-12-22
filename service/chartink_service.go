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
)

// 1. Interface Definition
type ChartInkService interface {
	FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error)
	FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error)
}

// 2. Concrete Implementation Struct
type ChartInkServiceImpl struct {
	client    *client.ChartinkClient
	cache     sync.Map // Thread-safe cache for ChartInkResponseDto
	xsrfToken string   // Shared session token
	cookie    string   // Shared session cookie
	userAgent string
	mu        sync.RWMutex // Mutex to protect tokens during refresh
}

// NewChartInkService acts as the Constructor/Initializer
func NewChartInkService(c *client.ChartinkClient) ChartInkService {
	return &ChartInkServiceImpl{
		client:    c,
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
	}
}

// FetchData handles logic for fetching screener data with retry support
func (s *ChartInkServiceImpl) FetchData(strategy model.StrategyDto) (*model.ChartInkResponseDto, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Check if tokens exist
	s.mu.RLock()
	hasTokens := s.xsrfToken != "" && s.cookie != ""
	s.mu.RUnlock()

	if !hasTokens {
		s.refreshTokens(ctx)
	}

	payload := map[string]string{"scan_clause": strategy.ScanClause}

	// Attempt first call
	s.mu.RLock()
	res, err := s.client.FetchData(ctx, s.xsrfToken, s.cookie, s.userAgent, payload)
	s.mu.RUnlock()

	// If failed, refresh tokens and retry once (Equivalent to Java catch block)
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

	// Store in cache
	s.cache.Store(strategy.Name, &dto)
	return &dto, nil
}

// refreshTokens extracts XSRF-TOKEN and ci_session from the homepage
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
		// Extract XSRF-TOKEN for the header
		if c.Name == "XSRF-TOKEN" {
			decoded, _ := url.QueryUnescape(c.Value)
			s.xsrfToken = decoded
		}
		// Build the full cookie string
		if c.Name == "ci_session" || c.Name == "laravel_session" || c.Name == "XSRF-TOKEN" {
			cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", c.Name, c.Value))
		}
	}
	s.cookie = strings.Join(cookieParts, "; ")
}

// FetchWithMargin matches Chartink results with pre-loaded Margin data
func (s *ChartInkServiceImpl) FetchWithMargin(strategy model.StrategyDto) ([]model.StockMarginDto, error) {
	// Try loading from cache first
	val, ok := s.cache.Load(strategy.Name)
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

	// Iterate and merge with margin data
	for _, stock := range response.Data {
		// Mocked global lookup (You'll implement MarginService next)
		margin, exists := GetMarginFromGlobalMap(stock.NSECode)
		if !exists {
			continue
		}

		result = append(result, model.StockMarginDto{
			Name:   stock.Name,
			Symbol: stock.NSECode,
			Margin: margin.Margin,
			Close:  stock.Close,
		})
	}

	// Sort by Margin Descending (Equivalent to list.sort in Java)
	sort.Slice(result, func(i, j int) bool {
		return result[i].Margin > result[j].Margin
	})

	return result, nil
}

// Helper for demonstration (to be replaced by real MarginService)
func GetMarginFromGlobalMap(symbol string) (model.Margin, bool) {
	// Logic to access your pre-loaded CSV margins
	return model.Margin{}, false
}
