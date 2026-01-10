package model

type TVNewsResponse struct {
	Items []struct {
		Title     string `json:"title"`
		Published int64  `json:"published"`
	} `json:"items"`
}
