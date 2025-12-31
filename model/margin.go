package model

// --- MARGIN ---
// Margin represents the database entity for stock leverage
type Margin struct {
	Symbol string  `bson:"_id" json:"symbol"`
	Name   string  `bson:"name" json:"name"`
	Margin float32 `bson:"margin" json:"margin"`
}

// StockMarginDto combines stock price with margin requirements
type StockMarginDto struct {
	Name   string  `json:"name"`
	Symbol string  `json:"symbol"`
	Margin float32 `json:"margin"`
	Close  float32 `json:"close"`
}

// --- Huma Structs ---

type GetMarginInput struct {
	Symbol string `path:"symbol" doc:"Stock Symbol" example:"RELIANCE"`
}
