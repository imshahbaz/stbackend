package client

import (
	"backend/model"
	"context"
	"fmt"
	"strings"

	"google.golang.org/genai"
)

var (
	flash25     = "gemini-2.5-flash"
	instruction = `You are a professional NSE technical analyst and quant trader.

TASK:
Using ONLY the provided daily OHLC data (1 month), predict TOMORROW'S PRICE RANGE.

STRICT RULES:
- Do NOT use news, fundamentals, sentiment, or assumptions.
- Do NOT guess.
- Base analysis strictly on price action and volatility.
- Assume tomorrow is a normal trading session (no gap unless justified by data).

ANALYSIS REQUIREMENTS:
1. Identify the current short-term trend (Bullish / Bearish / Sideways).
2. Detect key support and resistance levels using recent highs/lows.
3. Measure volatility using:
   - Average true range approximation
   - Recent candle ranges
4. Identify momentum:
   - Higher highs / higher lows OR lower highs / lower lows
5. Determine likely price behavior for NEXT trading day only.

PREDICTION OUTPUT:
- Provide a realistic TOMORROW LOW and TOMORROW HIGH range.
- The range must be achievable within normal NSE intraday volatility.
- Do NOT give extreme or unlikely levels.

OUTPUT FORMAT:
Return ONLY valid, minified JSON.
No explanation outside JSON.
No markdown.
No extra text.`

	responseSchema = &genai.Schema{
		Type: genai.TypeObject,
		Properties: map[string]*genai.Schema{
			"action": {
				Type: genai.TypeString,
				Enum: []string{"BUY", "SELL", "HOLD"},
			},
			"trend": {
				Type: genai.TypeString,
				Enum: []string{"BULLISH", "BEARISH", "SIDEWAYS"},
			},
			"tomorrow_low":  {Type: genai.TypeNumber},
			"tomorrow_high": {Type: genai.TypeNumber},
			"confidence":    {Type: genai.TypeInteger},
			"reasoning":     {Type: genai.TypeString},
		},
		Required: []string{
			"action",
			"trend",
			"tomorrow_low",
			"tomorrow_high",
			"confidence",
			"reasoning",
		},
	}
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
		Temperature:        ptrFloat32(0.0),
		TopP:               ptrFloat32(0.1),
		Seed:               ptrInt(21),
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: responseSchema,
		SystemInstruction:  genai.NewContentFromText(instruction, genai.RoleUser),
	}

	prompt := fmt.Sprintf(
		`Stock: %s
		Data format: Date|Open|High|Low|Close (Daily candles)
		
		OHLC DATA (Latest → Oldest):
		%s
		
		Predict tomorrow's intraday range.`,
		symbol,
		strings.Join(dataRows, "\n"),
	)

	// 3. Generate Prediction
	resp, err := c.client.Models.GenerateContent(context.Background(), flash25, genai.Text(prompt), config)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v", resp.Candidates[0].Content.Parts[0].Text), nil
}

func ptrFloat32(v float32) *float32 { return &v }
func ptrInt(v int32) *int32         { return &v }
