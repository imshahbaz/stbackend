package model

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
