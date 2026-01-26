package controller

import (
	"backend/model"
	"backend/service"
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type AngelOneController struct {
	angelOneSvc  service.AngelOneService
	isProduction bool
}

func NewAngelOneController(angelOneSvc service.AngelOneService, isProduction bool) *AngelOneController {
	return &AngelOneController{angelOneSvc: angelOneSvc, isProduction: isProduction}
}

func (ctrl *AngelOneController) RegisterRoutes(api huma.API) {

	huma.Register(api, huma.Operation{
		OperationID: "refresh-broker-session",
		Method:      http.MethodPost,
		Path:        "/api/angelone/refresh-session",
		Summary:     "Refresh Angel One Broker Session",
		Description: "Generates a new session for Angel One and stores the access token in Redis.",
		Tags:        []string{"Angel One"},
	}, ctrl.refreshSession)

	huma.Register(api, huma.Operation{
		OperationID: "get-angelone-ltp",
		Method:      http.MethodGet,
		Path:        "/api/angelone/ltp",
		Summary:     "Get LTP for a symbol",
		Description: "Fetches the Last Traded Price (LTP) for a given symbol from Angel One.",
		Tags:        []string{"Angel One"},
	}, ctrl.getLTP)

	huma.Register(api, huma.Operation{
		OperationID: "get-angelone-multiple-ltp",
		Method:      http.MethodPost,
		Path:        "/api/angelone/ltp/bulk",
		Summary:     "Get LTP for multiple symbols",
		Description: "Fetches the Last Traded Price (LTP) for multiple symbol tokens from Angel One.",
		Tags:        []string{"Angel One"},
	}, ctrl.getMultipleLTP)

	huma.Register(api, huma.Operation{
		OperationID: "get-angelone-historical",
		Method:      http.MethodGet,
		Path:        "/api/angelone/historical",
		Summary:     "Get historical candle data",
		Description: "Fetches historical candle data for a given symbol from Angel One.",
		Tags:        []string{"Angel One"},
	}, ctrl.getHistoricalData)
}

func (ctrl *AngelOneController) refreshSession(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	err := ctrl.angelOneSvc.RefreshBrokerSession()
	if err != nil {
		return nil, huma.Error500InternalServerError("Error refreshing broker session: " + err.Error())
	}
	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Broker session refreshed successfully"}}, nil
}

func (ctrl *AngelOneController) getLTP(ctx context.Context, input *model.AngelOneLTPInput) (*model.ResponseWrapper, error) {
	ltp, err := ctrl.angelOneSvc.GetLTP(input.TradingSymbol, input.SymbolToken)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error fetching LTP: " + err.Error())
	}
	return &model.ResponseWrapper{Body: model.Response{Success: true, Data: ltp}}, nil
}

func (ctrl *AngelOneController) getMultipleLTP(ctx context.Context, input *model.AngelOneMultipleLTPInput) (*model.ResponseWrapper, error) {
	ltpMap, err := ctrl.angelOneSvc.GetMultipleLTP(input.Body.Tokens)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error fetching bulk LTP: " + err.Error())
	}
	return &model.ResponseWrapper{Body: model.Response{Success: true, Data: ltpMap}}, nil
}

func (ctrl *AngelOneController) getHistoricalData(ctx context.Context, input *model.AngelOneHistoricalInput) (*model.ResponseWrapper, error) {
	candles, err := ctrl.angelOneSvc.GetHistoricalData(input.SymbolToken, input.Interval, input.FromDate, input.ToDate)
	if err != nil {
		return nil, huma.Error500InternalServerError("Error fetching historical data: " + err.Error())
	}
	return &model.ResponseWrapper{Body: model.Response{Success: true, Data: candles}}, nil
}
