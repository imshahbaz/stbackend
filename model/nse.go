package model

type NseResponseWrapper[T any] struct {
	Data []T `json:"data"`
}

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

type NseIndexData struct {
	Key           string  `json:"key"`
	Index         string  `json:"index"`
	IndexSymbol   string  `json:"indexSymbol"`
	Last          float64 `json:"last"`
	PercentChange float64 `json:"percentChange"`
	PerChange365d float64 `json:"perChange365d"`
	PerChange30d  float64 `json:"perChange30d"`
	OneWeekAgoVal float64 `json:"oneWeekAgoVal"`
}

type AllIndicesResponse struct {
	NseIndexData
	PerChange1w float64 `json:"perChange1w"`
}
