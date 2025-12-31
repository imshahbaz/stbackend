package client

import (
	"backend/model"
	"context"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type BrevoClient struct {
	client *resty.Client
}

func NewBrevoClient() *BrevoClient {
	client := resty.New().
		SetBaseURL("https://api.brevo.com/v3").
		SetTimeout(10*time.Second).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json")

	return &BrevoClient{
		client: client,
	}
}

func (c *BrevoClient) SendTransactionalEmail(ctx context.Context, apiKey string, emailReq model.BrevoEmailRequest) (string, error) {
	resp, err := c.client.R().
		SetContext(ctx).
		SetHeader("api-key", apiKey).
		SetBody(emailReq).
		Post("/smtp/email")

	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}

	if !resp.IsSuccess() {
		return resp.String(), fmt.Errorf("brevo api error (status %d): %s", resp.StatusCode(), resp.String())
	}

	return resp.String(), nil
}
