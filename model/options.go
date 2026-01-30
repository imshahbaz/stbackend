package model

type OptionChain struct {
	Symbol        string `json:"symbol" bson:"_id"`
	Expiry        string `json:"expiry" bson:"expiry"`
	Strike        string `json:"strike" bson:"strike"`
	LotSize       string `json:"lotSize" bson:"lotSize"`
	ExchangeType  string `json:"exchangeType" bson:"exchangeType"`
	MstockSymbol  string `json:"mstockSymbol" bson:"mstockSymbol"`
	AngelOneToken string `json:"angelOneToken" bson:"angelOneToken"`
}

type OptionOutput struct {
	Name   string   `json:"name"`
	Expiry []string `json:"expiry"`
	Strike []string `json:"strike"`
}
