package model

type TVNewsItem struct {
	Title     string `json:"title"`
	Published int64  `json:"published"`
}

type TVNewsResponse struct {
	Items []TVNewsItem `json:"items"`
}

type AIAnalysis struct {
	Action       string  `json:"action"`
	Confidence   int     `json:"confidence"`
	Reasoning    string  `json:"reasoning"`
	Trend        string  `json:"trend"`
	TomorrowHigh float32 `json:"tomorrow_high"`
	TomorrowLow  float32 `json:"tomorrow_low"`
}
