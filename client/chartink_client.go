package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ChartinkClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewChartinkClient() *ChartinkClient {
	return &ChartinkClient{
		BaseURL: "https://chartink.com",
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetHomepage mimics ResponseEntity<String> getHomepage()
// Useful for grabbing initial cookies/tokens
func (c *ChartinkClient) GetHomepage(ctx context.Context) (*http.Response, string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL, nil)
	if err != nil {
		return nil, "", err
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	return resp, string(body), nil
}

// FetchData mimics the @PostMapping /screener/process
func (c *ChartinkClient) FetchData(
	ctx context.Context,
	xsrfToken string,
	cookie string,
	userAgent string,
	payload map[string]string,
) (string, error) {
	apiURL := fmt.Sprintf("%s/screener/process", c.BaseURL)

	// Chartink usually expects form-encoded data for this endpoint,
	// but your Java code used @RequestBody Map.
	// If it's JSON, use json.Marshal. If it's Form, use url.Values.
	data := url.Values{}
	for k, v := range payload {
		data.Set(k, v)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}

	// Set Headers (Equivalent to @RequestHeader)
	req.Header.Set("x-xsrf-token", xsrfToken)
	req.Header.Set("cookie", cookie)
	req.Header.Set("user-agent", userAgent)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return string(body), fmt.Errorf("chartink error: status %d", resp.StatusCode)
	}

	return string(body), nil
}
