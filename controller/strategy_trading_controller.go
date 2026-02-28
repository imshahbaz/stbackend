package controller

import (
	"context"
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type StrategyTradingController struct {
	strategyTradingService service.StrategyTradingService
	isProduction           bool
}

func NewStrategyTradingController(sts service.StrategyTradingService) *StrategyTradingController {
	return &StrategyTradingController{
		strategyTradingService: sts,
	}
}

func (ctrl *StrategyTradingController) RegisterRoutes(api huma.API) {

	huma.Register(api, huma.Operation{
		OperationID: "continuous-trading",
		Method:      http.MethodPost,
		Path:        "/api/strategy-trading/continuous",
		Summary:     "Start continuous trading",
		Description: "Triggers the automated continuous trading execution",
		Tags:        []string{"Strategy Trading"},
	}, ctrl.continuousTrade)
}

func (ctrl *StrategyTradingController) continuousTrade(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	err := ctrl.strategyTradingService.ContinuousTrade(ctx)
	if err != nil {
		return NewTypedError[any](err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Continuous trading triggered"), nil
}
