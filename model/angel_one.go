package model

type AngelOneLTPInput struct {
	TradingSymbol string `query:"tradingSymbol" doc:"The trading symbol of the instrument" required:"true"`
	SymbolToken   string `query:"symbolToken" doc:"The symbol token of the instrument" required:"true"`
}

type AngelOneMultipleLTPDto struct {
	Tokens []string `json:"tokens" doc:"List of symbol tokens" required:"true"`
}

type AngelOneWsSubscribeDto struct {
	Tokens       []string     `json:"tokens" doc:"List of symbol tokens to subscribe to" required:"true"`
	ExchangeType ExchangeType `json:"exchangeType" doc:"Exchange type" required:"true"`
}

type AngelOneHistoricalInput struct {
	SymbolToken string `query:"symbolToken" doc:"The symbol token of the instrument" required:"true"`
	Interval    string `query:"interval" doc:"Interval (ONE_MINUTE, FIVE_MINUTE, etc)" required:"true"`
	FromDate    string `query:"fromDate" doc:"From date (YYYY-MM-DD HH:mm)" required:"true"`
	ToDate      string `query:"toDate" doc:"To date (YYYY-MM-DD HH:mm)" required:"true"`
}

type QuoteData struct {
	Exchange      string  `json:"exchange"`
	TradingSymbol string  `json:"tradingSymbol"`
	SymbolToken   string  `json:"symbolToken"`
	Ltp           float64 `json:"ltp"`
	Open          float64 `json:"open"`
	High          float64 `json:"high"`
	Low           float64 `json:"low"`
	Close         float64 `json:"close"`
}

type QuoteResponse struct {
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	Errorcode string `json:"errorcode"`
	Data      struct {
		Fetched []QuoteData `json:"fetched"`
	} `json:"data"`
}

type AngelOneLoginResponse struct {
	Status    bool   `json:"status"`
	Message   string `json:"message"`
	Errorcode string `json:"errorcode"`
	Data      struct {
		JwtToken     string `json:"jwtToken"`
		RefreshToken string `json:"refreshToken"`
		FeedToken    string `json:"feedToken"`
	} `json:"data"`
}

type CandleDataResponse struct {
	Status    bool    `json:"status"`
	Message   string  `json:"message"`
	Errorcode string  `json:"errorcode"`
	Data      [][]any `json:"data"`
}

type AngelOneCandle struct {
	Time   string  `json:"time"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

type ExchangeType int

const (
	NSE ExchangeType = 1
	NFO ExchangeType = 2
	BFO ExchangeType = 4
)
