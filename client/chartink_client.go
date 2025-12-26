package client

import (
	"context"
	"time"

	"github.com/go-resty/resty/v2"
)

type ChartinkClient struct {
	RestyClient *resty.Client
}

func NewChartinkClient() *ChartinkClient {
	c := resty.New().
		SetBaseURL("https://chartink.com").
		SetTimeout(30*time.Second).
		SetHeader("Accept", "application/json")

	return &ChartinkClient{RestyClient: c}
}

func (c *ChartinkClient) GetHomepage(ctx context.Context) (*resty.Response, error) {
	return c.RestyClient.R().SetContext(ctx).Get("/")
}

func (c *ChartinkClient) FetchData(ctx context.Context, token, ua string, payload map[string]string) (*resty.Response, error) {
	return c.RestyClient.R().
		SetContext(ctx).
		SetHeader("X-XSRF-TOKEN", token).
		SetHeader("User-Agent", ua).
		SetHeader("Referer", "https://chartink.com/").
		SetFormData(payload).
		Post("/screener/process")
}
