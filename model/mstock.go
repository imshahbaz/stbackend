package model

type MstockLoginInput struct {
	Username string `json:"username" form:"username" doc:"m.Stock Client Code"`
	Password string `json:"password" form:"password" doc:"m.Stock Password"`
	APIKey   string `json:"apiKey" form:"apiKey" doc:"m.Stock Type A API Key"`
}

type MstockVerifyOtpInput struct {
	Otp string `json:"otp" binding:"required,len=6" example:"123456"`
}

type MstockOrderInput struct {
	Symbol   string `json:"symbol" form:"symbol" doc:"Trading Symbol e.g. INFY"`
	Exchange string `json:"exchange" form:"exchange" doc:"NSE or BSE"`
	Side     string `json:"side" form:"side" doc:"BUY or SELL"`
	Type     string `json:"type" form:"type" doc:"LIMIT or MARKET"`
	Qty      string `json:"qty" form:"qty"`
	Product  string `json:"product" form:"product" doc:"MIS, DELIVERY, etc."`
	Validity string `json:"validity" form:"validity" doc:"DAY, etc."`
	Price    string `json:"price" form:"price"`
	Variety  string `json:"variety" form:"variety" default:"regular"`
}

type MstockRedisCache struct {
	AccessToken string `json:"accessToken"`
	Username    string `json:"username"`
	APIKey      string `json:"apiKey"`
}

type MstockOrderResponse struct {
	OrderId         string  `json:"order_id"`
	AveragePrice    float64 `json:"average_price"`
	Quantity        int     `json:"quantity"`
	FilledQuantity  int     `json:"filled_quantity"`
	Status          string  `json:"status"`
	PendingQuantity int     `json:"pending_quantity"`
}

type MinimalInstrument struct {
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	Token   string `json:"token"`
	ExchSeg string `json:"exch_seg"`
	Expiry  string `json:"expiry"`
	LotSize string `json:"lotsize"`
	Strike  string `json:"strike"`
}

type MstockOrderRequest struct {
	Name   string  `json:"name"`
	Strike string  `json:"strike"`
	Expiry string  `json:"expiry"`
	Lots   int     `json:"lots" validate:"required,gte=1,lte=100"`
	Profit float64 `json:"profit" validate:"gte=0"`
	Action string  `json:"action" validate:"required"`
}
