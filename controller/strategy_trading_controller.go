package controller

import (
	"context"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type StrategyTradingController struct {
	strategyTradingService service.StrategyTradingService
	isProduction           bool
}

func NewStrategyTradingController(sts service.StrategyTradingService, isProduction bool) *StrategyTradingController {
	return &StrategyTradingController{
		strategyTradingService: sts,
		isProduction:           isProduction,
	}
}

type StrategyTradingInput struct {
	StrategyName string `json:"strategyName" validate:"required"`
}

func (ctrl *StrategyTradingController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)
	adminMw := middleware.HumaAdminOnly(api)

	huma.Register(api, huma.Operation{
		OperationID: "execute-strategy",
		Method:      http.MethodPost,
		Path:        "/api/strategy-trading/execute",
		Summary:     "Execute a trading strategy",
		Description: "Triggers the automated trading execution for a specific strategy",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Trading"},
	}, ctrl.executeStrategy)
}

func (ctrl *StrategyTradingController) executeStrategy(ctx context.Context, input *struct{ Body StrategyTradingInput }) (*model.DefaultResponse, error) {
	err := ctrl.strategyTradingService.ExecuteStrategy(input.Body.StrategyName)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Strategy execution triggered"), nil
}
