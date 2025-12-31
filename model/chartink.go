package model

// ChartInkResponseDto maps the wrapper from ChartInk API
type ChartInkResponseDto struct {
	// json:"data" tells the parser to map the JSON key "data" to this field
	Data []StockData `json:"data"`
}

// StockData represents a single row from a scan result
type StockData struct {
	NSECode string  `json:"nsecode"`
	Name    string  `json:"name"`
	Close   float32 `json:"close"`
}

// --- Huma Structs ---

type ChartInkInput struct {
	Strategy string `query:"strategy" doc:"Name of the strategy to run" example:"Bullish_Engulfing" required:"true"`
}
