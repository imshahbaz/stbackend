package model

type TVNewsResponse struct {
	Items []struct {
		Title     string `json:"title"`
		Published int64  `json:"published"`
	} `json:"items"`
}

type AIAnalysis struct {
	Action     string `json:"action"`
	Confidence int    `json:"confidence"`
	Reasoning  string `json:"reasoning"`
	Trend      string `json:"trend"`
}
