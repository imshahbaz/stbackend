package model

type AngelOneLTPInput struct {
	TradingSymbol string `query:"tradingSymbol" doc:"The trading symbol of the instrument" required:"true"`
	SymbolToken   string `query:"symbolToken" doc:"The symbol token of the instrument" required:"true"`
}

type AngelOneMultipleLTPInput struct {
	Body struct {
		Tokens []string `json:"tokens" doc:"List of symbol tokens" required:"true"`
	}
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
