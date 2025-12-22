package client

import (
	"backend/model"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type BrevoClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewBrevoClient() *BrevoClient {
	return &BrevoClient{
		BaseURL: "https://api.brevo.com/v3",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second, // Essential for production
		},
	}
}

// SendTransactionalEmail mimics the Feign Client PostMapping
func (c *BrevoClient) SendTransactionalEmail(ctx context.Context, apiKey string, emailReq model.BrevoEmailRequest) (string, error) {
	url := fmt.Sprintf("%s/smtp/email", c.BaseURL)

	// 1. Marshal the request body to JSON
	jsonData, err := json.Marshal(emailReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal email request: %w", err)
	}

	// 2. Create the Request with Context (supports cancellations/timeouts)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// 3. Set Headers (Equivalent to @RequestHeader and consumes/produces)
	req.Header.Set("api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 4. Execute Request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// 5. Read and check the response
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return string(body), fmt.Errorf("brevo api error (status %d): %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
