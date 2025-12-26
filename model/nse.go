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

type SectorData struct {
	Index         string  `json:"index"`
	IndexLongName string  `json:"indexLongName"`
	Current       float64 `json:"current"`
	Open          float64 `json:"open"`
	Close         float64 `json:"close"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	PChange       float64 `json:"pChange"`
	TimeStamp     string  `json:"timeStamp"`
}
