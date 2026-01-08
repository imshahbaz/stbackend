package model

type ChartInkResponseDto struct {
	Data []StockData `json:"data"`
}

type StockData struct {
	NSECode string  `json:"nsecode"`
	Name    string  `json:"name"`
	Close   float32 `json:"close"`
}


type ChartInkInput struct {
	Strategy string `query:"strategy" doc:"Name of the strategy to run" example:"Bullish_Engulfing" required:"true"`
}
