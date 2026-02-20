package controller

import (
	"context"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"strings"

	"github.com/danielgtaylor/huma/v2"
)

type StrategyOrderController struct {
	service      service.StrategyOrderService
	isProduction service.IsProduction
}

func NewStrategyOrderController(s service.StrategyOrderService, isProduction service.IsProduction) *StrategyOrderController {
	return &StrategyOrderController{
		service:      s,
		isProduction: isProduction,
	}
}

func (ctrl *StrategyOrderController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, bool(ctrl.isProduction))
	adminMw := middleware.HumaAdminOnly(api)

	// Admin routes
	huma.Register(api, huma.Operation{
		OperationID: "get-strategy-orders",
		Method:      http.MethodGet,
		Path:        "/api/strategy-order",
		Summary:     "Get all strategy orders (Admin)",
		Description: "Returns a list of all strategy orders, optionally filtered by strategy name. Admin Only.",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.getAll)

	huma.Register(api, huma.Operation{
		OperationID: "get-strategy-order",
		Method:      http.MethodGet,
		Path:        "/api/strategy-order/{id}",
		Summary:     "Get strategy order by ID",
		Description: "Returns a single strategy order by its ID",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.getOne)

	huma.Register(api, huma.Operation{
		OperationID: "update-strategy-order",
		Method:      http.MethodPut,
		Path:        "/api/strategy-order/{id}",
		Summary:     "Update strategy order",
		Description: "Updates an existing strategy order",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.update)

	huma.Register(api, huma.Operation{
		OperationID: "delete-strategy-order",
		Method:      http.MethodDelete,
		Path:        "/api/strategy-order/{id}",
		Summary:     "Delete strategy order",
		Description: "Deletes a strategy order by ID",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.delete)

	// User routes
	huma.Register(api, huma.Operation{
		OperationID: "create-strategy-order",
		Method:      http.MethodPost,
		Path:        "/api/strategy-order",
		Summary:     "Create strategy order",
		Description: "Creates a new strategy order for the authenticated user",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.create)

	huma.Register(api, huma.Operation{
		OperationID: "get-my-strategy-orders",
		Method:      http.MethodGet,
		Path:        "/api/strategy-order/my",
		Summary:     "Get my strategy orders",
		Description: "Returns a list of all strategy orders for the currently authenticated user",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Strategy Order"},
	}, ctrl.getMyOrders)
}

func (ctrl *StrategyOrderController) getAll(ctx context.Context, input *model.GetAllStrategyOrdersInput) (*model.TypedResponse[[]model.StrategyOrderDto], error) {
	res, err := ctrl.service.GetAll(ctx, input.StrategyName)
	if err != nil {
		return NewTypedError[[]model.StrategyOrderDto](err.Error()), nil
	}
	return NewTypedResponse(res, "Strategy orders fetched successfully"), nil
}

func (ctrl *StrategyOrderController) getOne(ctx context.Context, input *model.GetStrategyOrderInput) (*model.TypedResponse[model.StrategyOrderDto], error) {
	res, err := ctrl.service.Get(ctx, input.ID)
	if err != nil {
		return NewTypedError[model.StrategyOrderDto](err.Error()), nil
	}
	if res.ID == "" {
		return NewTypedError[model.StrategyOrderDto]("Strategy order not found"), nil
	}
	return NewTypedResponse(res, "Strategy order fetched successfully"), nil
}

func (ctrl *StrategyOrderController) create(ctx context.Context, input *model.RequestBody[model.StrategyOrderDto]) (*model.TypedResponse[model.StrategyOrderDto], error) {
	user := ctx.Value("user").(model.UserDto)
	input.Body.UserID = user.UserID

	res, err := ctrl.service.Create(ctx, input.Body)
	if err != nil {
		if strings.Contains(err.Error(), "E11000") {
			return nil, huma.Error409Conflict("Strategy order already exists for this user on this date")
		}
		return nil, huma.Error500InternalServerError(err.Error())
	}
	return NewTypedResponse(res, "Strategy order created successfully"), nil
}

func (ctrl *StrategyOrderController) update(ctx context.Context, input *model.IDInput[model.StrategyOrderDto]) (*model.TypedResponse[model.StrategyOrderDto], error) {
	user := ctx.Value("user").(model.UserDto)
	input.Body.UserID = user.UserID
	input.Body.ID = input.ID
	res, err := ctrl.service.Update(ctx, input.Body)
	if err != nil {
		if strings.Contains(err.Error(), "E11000") {
			return nil, huma.Error400BadRequest("Strategy order already exists for this user on this date")
		}
		return NewTypedError[model.StrategyOrderDto](err.Error()), nil
	}
	return NewTypedResponse(res, "Strategy order updated successfully"), nil
}

func (ctrl *StrategyOrderController) delete(ctx context.Context, input *model.GetStrategyOrderInput) (*model.TypedResponse[any], error) {
	err := ctrl.service.Delete(ctx, input.ID)
	if err != nil {
		return NewTypedError[any](err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Strategy order deleted successfully"), nil
}

func (ctrl *StrategyOrderController) getMyOrders(ctx context.Context, input *struct{}) (*model.TypedResponse[[]model.StrategyOrderDto], error) {
	user := ctx.Value("user").(model.UserDto)
	res, err := ctrl.service.GetByUserID(ctx, user.UserID)
	if err != nil {
		return NewTypedError[[]model.StrategyOrderDto](err.Error()), nil
	}
	return NewTypedResponse(res, "User strategy orders fetched successfully"), nil
}
