package model

type OBInfo struct {
	Date string  `bson:"date" json:"date"`
	High float64 `bson:"high" json:"high"`
	Low  float64 `bson:"low" json:"low"`
}

type StockRecord struct {
	Symbol      string   `bson:"_id" json:"symbol"`
	OrderBlocks []OBInfo `bson:"order_blocks" json:"orderBlocks"`
}
