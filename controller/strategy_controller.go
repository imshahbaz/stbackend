package controller

import (
	"context"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type StrategyController struct {
	strategyService service.StrategyService
	isProduction    bool
}

func NewStrategyController(ss service.StrategyService, isProduction bool) *StrategyController {
	return &StrategyController{
		strategyService: ss,
		isProduction:    isProduction,
	}
}

func (ctrl *StrategyController) RegisterRoutes(api huma.API) {
	// Public route
	huma.Register(api, huma.Operation{
		OperationID: "get-strategies",
		Method:      http.MethodGet,
		Path:        "/api/strategy",
		Summary:     "Get all strategies",
		Description: "Returns a list of all configured active trading strategies",
		Tags:        []string{"Strategy"},
	}, ctrl.getAllStrategies)

	// Protected routes
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)
	adminMw := middleware.HumaAdminOnly(api)

	huma.Register(api, huma.Operation{
		OperationID: "create-strategy",
		Method:      http.MethodPost,
		Path:        "/api/strategy",
		Summary:     "Create a strategy",
		Description: "Saves a new trading strategy configuration to MongoDB",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy"},
	}, ctrl.createStrategy)

	huma.Register(api, huma.Operation{
		OperationID: "update-strategy",
		Method:      http.MethodPut,
		Path:        "/api/strategy",
		Summary:     "Update a strategy",
		Description: "Updates an existing strategy configuration by name/ID",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy"},
	}, ctrl.updateStrategy)

	huma.Register(api, huma.Operation{
		OperationID: "delete-strategy",
		Method:      http.MethodDelete,
		Path:        "/api/strategy",
		Summary:     "Delete a strategy",
		Description: "Removes a strategy from the system using its ID/Name",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy"},
	}, ctrl.deleteStrategy)

	huma.Register(api, huma.Operation{
		OperationID: "reload-strategies",
		Method:      http.MethodPost,
		Path:        "/api/strategy/reload",
		Summary:     "Reload strategies",
		Description: "Syncs the in-memory strategy cache with MongoDB",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy"},
	}, ctrl.reloadAllStrategies)

	huma.Register(api, huma.Operation{
		OperationID: "get-strategies-admin",
		Method:      http.MethodGet,
		Path:        "/api/strategy/admin",
		Summary:     "Get all strategies (Admin)",
		Description: "Returns all trading strategies with full administrative details",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy"},
	}, ctrl.getAllStrategiesAdmin)
}

func (ctrl *StrategyController) getAllStrategies(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	strategies := ctrl.strategyService.GetAllStrategies()
	if strategies == nil {
		strategies = []model.StrategyDto{}
	}
	return NewResponse(strategies, "Strategies fetched successfully"), nil
}

func (ctrl *StrategyController) createStrategy(ctx context.Context, input *model.CreateStrategyRequest) (*model.DefaultResponse, error) {
	res, err := ctrl.strategyService.CreateStrategy(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(res, "Strategy created successfully"), nil
}

func (ctrl *StrategyController) updateStrategy(ctx context.Context, input *model.CreateStrategyRequest) (*model.DefaultResponse, error) {
	res, err := ctrl.strategyService.UpdateStrategy(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(res, "Strategy updated successfully"), nil
}

func (ctrl *StrategyController) deleteStrategy(ctx context.Context, input *model.DeleteStrategyInput) (*model.DefaultResponse, error) {
	if input.ID == "" {
		return NewErrorResponse("Strategy ID is required"), nil
	}

	if err := ctrl.strategyService.DeleteStrategy(ctx, input.ID); err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Strategy deleted successfully"), nil
}

func (ctrl *StrategyController) reloadAllStrategies(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	if err := ctrl.strategyService.ReloadAllStrategies(ctx); err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Strategies reloaded successfully"), nil
}

func (ctrl *StrategyController) getAllStrategiesAdmin(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	strategies := ctrl.strategyService.GetAllStrategiesAdmin()
	if strategies == nil {
		strategies = []model.StrategyDto{}
	}
	return NewResponse(strategies, "Admin strategies fetched successfully"), nil
}
