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
