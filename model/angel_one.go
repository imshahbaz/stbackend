package model

type AngelOneLTPInput struct {
	TradingSymbol string `query:"tradingSymbol" doc:"The trading symbol of the instrument" required:"true"`
	SymbolToken   string `query:"symbolToken" doc:"The symbol token of the instrument" required:"true"`
}
