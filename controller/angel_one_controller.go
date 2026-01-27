package controller

import (
	"backend/model"
	"backend/service"
	"context"
	"net/http"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
)

type AngelOneController struct {
	angelOneSvc    service.AngelOneService
	isProduction   bool
	angelOneWebSvc service.AngelOneWebSocket
}

func NewAngelOneController(angelOneSvc service.AngelOneService, isProduction bool, angelOneWebSvc service.AngelOneWebSocket) *AngelOneController {
	return &AngelOneController{angelOneSvc: angelOneSvc, isProduction: isProduction, angelOneWebSvc: angelOneWebSvc}
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

	huma.Register(api, huma.Operation{
		OperationID: "angel-one-ws-connect",
		Method:      http.MethodPost,
		Path:        "/api/angelone/ws/connect",
		Summary:     "Connect Angel One Smart Stream",
		Description: "Establishes a background WebSocket connection to Angel One for real-time data.",
		Tags:        []string{"Angel One"},
	}, ctrl.wsConnect)

	huma.Register(api, huma.Operation{
		OperationID: "angel-one-ws-subscribe",
		Method:      http.MethodPost,
		Path:        "/api/angelone/ws/subscribe",
		Summary:     "Subscribe to tokens",
		Description: "Adds tokens to the active background WebSocket subscription list.",
		Tags:        []string{"Angel One"},
	}, ctrl.wsSubscribe)

	huma.Register(api, huma.Operation{
		OperationID: "angel-one-ws-disconnect",
		Method:      http.MethodPost,
		Path:        "/api/angelone/ws/disconnect",
		Summary:     "Disconnect Angel One WebSocket",
		Description: "Gracefully closes the background Angel One WebSocket connection.",
		Tags:        []string{"Angel One"},
	}, ctrl.wsDisconnect)
}

func (ctrl *AngelOneController) refreshSession(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	_, _, err := ctrl.angelOneSvc.RefreshBrokerSession()
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

func (ctrl *AngelOneController) wsConnect(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	err := ctrl.angelOneWebSvc.StartWebsocket()
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to initialize WebSocket")
	}

	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "WebSocket connected successfully"}}, nil
}

func (ctrl *AngelOneController) wsSubscribe(ctx context.Context, input *struct {
	Body struct {
		Tokens []string `json:"tokens" doc:"List of symbol tokens to subscribe to" required:"true"`
	}
}) (*model.ResponseWrapper, error) {

	for _, token := range input.Body.Tokens {
		_, err := ctrl.angelOneWebSvc.Subscribe(token)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to subscribe to token: " + err.Error())
		}
	}

	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "Subscription requests sent for " + strconv.Itoa(len(input.Body.Tokens)) + " tokens"}}, nil
}

func (ctrl *AngelOneController) wsDisconnect(ctx context.Context, input *struct{}) (*model.ResponseWrapper, error) {
	ctrl.angelOneWebSvc.Disconnect()
	ctrl.angelOneWebSvc.StopUpdateChannel()
	return &model.ResponseWrapper{Body: model.Response{Success: true, Message: "WebSocket disconnected"}}, nil
}
