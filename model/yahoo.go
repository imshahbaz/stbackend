package model

type YahooTimeRange string

// Simple constants with direct string values
const (
	Range1d  YahooTimeRange = "1d"
	Range5d  YahooTimeRange = "5d"
	Range1mo YahooTimeRange = "1mo"
	Range3mo YahooTimeRange = "3mo"
	Range6mo YahooTimeRange = "6mo"
	Range1y  YahooTimeRange = "1y"
	Range2y  YahooTimeRange = "2y"
	Range5y  YahooTimeRange = "5y"
	Range10y YahooTimeRange = "10y"
	RangeYtd YahooTimeRange = "ytd"
	RangeMax YahooTimeRange = "max"
)

// YahooChartResponse is the top-level container
type YahooChartResponse struct {
	Chart ChartData `json:"chart"`
}

type ChartData struct {
	Result []Result `json:"result"`
	Error  any      `json:"error"`
}

type Result struct {
	Timestamp  []int64    `json:"timestamp"`
	Indicators Indicators `json:"indicators"`
}

type Indicators struct {
	Quote []Quote `json:"quote"`
}

type Quote struct {
	Low    []float64 `json:"low"`
	High   []float64 `json:"high"`
	Open   []float64 `json:"open"`
	Volume []int64   `json:"volume"`
	Close  []float64 `json:"close"`
}
