package model

// NSEHistoricalData represents stock history
type NSEHistoricalData struct {
	Symbol    string  `json:"chSymbol"`
	Open      float64 `json:"chOpeningPrice"`
	High      float64 `json:"chTradeHighPrice"`
	Low       float64 `json:"chTradeLowPrice"`
	Close     float64 `json:"chClosingPrice"`
	Timestamp string  `json:"mtimestamp"`
}
