package client

import (
	"backend/model"
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

var (
	flash25 = "gemini-2.5-flash"
)

type GenAiClient struct {
	client *genai.Client
}

func NewGenAiClient(config model.GoogleAuthCredentials) *GenAiClient {
	client, _ := genai.NewClient(context.Background(), &genai.ClientConfig{
		APIKey:  config.GeminiApiKey,
		Backend: genai.BackendGeminiAPI,
	})
	return &GenAiClient{
		client: client,
	}
}

func (c *GenAiClient) GetGenAiStockAnalysis(symbol string, data []model.NSEHistoricalData) (string, error) {

	var dataRows []string
	for _, d := range data {
		dataRows = append(dataRows, fmt.Sprintf("%s|%.2f|%.2f|%.2f|%.2f",
			d.Timestamp, d.Open, d.High, d.Low, d.Close))
	}

	config := &genai.GenerateContentConfig{
		Temperature:      ptrFloat32(0.0),
		TopP:             ptrFloat32(0.1),
		Seed:             ptrInt(21),
		ResponseMIMEType: "application/json",
		ResponseJsonSchema: &genai.Schema{
			Type: genai.TypeObject,
			Properties: map[string]*genai.Schema{
				"action":     {Type: genai.TypeString, Enum: []string{"BUY", "SELL", "HOLD"}},
				"confidence": {Type: genai.TypeInteger},
				"trend":      {Type: genai.TypeString},
				"reasoning":  {Type: genai.TypeString},
			},
			Required: []string{"action", "confidence", "reasoning", "trend"},
		},
		SystemInstruction: genai.NewContentFromText(
			"You are a Senior NSE Technical Analyst. Perform a logical trend analysis "+
				"based strictly on the provided OHLC data. Identify support/resistance "+
				"and price momentum. Output must be valid, minified JSON only.",
			genai.RoleUser),
	}

	prompt := fmt.Sprintf("Analyze 30-day trend for %s. Respond with strategy:\n%s",
		symbol, strings.Join(dataRows, "\n"))

	// 3. Generate Prediction
	resp, err := c.client.Models.GenerateContent(context.Background(), flash25, genai.Text(prompt), config)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0].Text), nil
}

func ptrFloat32(v float32) *float32 { return &v }
func ptrInt(v int32) *int32         { return &v }
