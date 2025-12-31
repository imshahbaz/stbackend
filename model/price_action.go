package model

const PACollectionName = "price_action"

type Info struct {
	Date string  `bson:"date" json:"date"`
	High float64 `bson:"high" json:"high"`
	Low  float64 `bson:"low" json:"low"`
}

type StockRecord struct {
	Symbol      string `bson:"_id" json:"symbol"`
	OrderBlocks []Info `bson:"order_blocks" json:"orderBlocks"`
	Fvg         []Info `bson:"fvg" json:"fvg"`
}

type ObRequest struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
}

type ObResponse struct {
	StockMarginDto
	Date string `json:"date"`
}

// --- Huma Structs ---

type TriggerAutomationResponse struct {
	Body Response
}

type GetPAInput struct {
	Symbol string `path:"symbol" required:"true"`
}

type ObInput struct {
	Body ObRequest
}
