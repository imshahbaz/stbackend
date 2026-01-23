package controller

import (
	"context"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type OrderController struct {
	orderService service.OrderService
	isProduction bool
}

func NewOrderController(os service.OrderService, isProduction bool) *OrderController {
	return &OrderController{
		orderService: os,
		isProduction: isProduction,
	}
}

func (ctrl *OrderController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)

	huma.Register(api, huma.Operation{
		OperationID: "get-order-by-id",
		Method:      http.MethodGet,
		Path:        "/api/order/{id}",
		Summary:     "Get order by ID",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.Get)

	huma.Register(api, huma.Operation{
		OperationID: "get-orders-by-date",
		Method:      http.MethodGet,
		Path:        "/api/order/date",
		Summary:     "Get orders by date",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.GetByDate)

	huma.Register(api, huma.Operation{
		OperationID: "get-orders-by-user-id",
		Method:      http.MethodGet,
		Path:        "/api/order/user/{userId}",
		Summary:     "Get orders by user ID",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.GetByUserId)

	huma.Register(api, huma.Operation{
		OperationID: "create-order",
		Method:      http.MethodPost,
		Path:        "/api/order",
		Summary:     "Create new order",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.Create)

	huma.Register(api, huma.Operation{
		OperationID: "update-order",
		Method:      http.MethodPut,
		Path:        "/api/order/{id}",
		Summary:     "Update existing order",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.Update)

	huma.Register(api, huma.Operation{
		OperationID: "delete-order",
		Method:      http.MethodDelete,
		Path:        "/api/order/{id}",
		Summary:     "Delete order",
		Middlewares: huma.Middlewares{authMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Order"},
	}, ctrl.Delete)

	huma.Register(api, huma.Operation{
		OperationID: "initiate-mtf-orders",
		Method:      http.MethodPost,
		Path:        "/api/order/initiate-mtf",
		Summary:     "Initiate today's MTF orders",
		Description: "Fetches all orders for today and initiates trades on Zerodha",
		Tags:        []string{"Order"},
	}, ctrl.InitiateMtfOrders)
}

func (ctrl *OrderController) Get(ctx context.Context, input *model.GetOrderInput) (*model.DefaultResponse, error) {
	order, err := ctrl.orderService.Get(ctx, input.ID)
	if err != nil {
		return NewErrorResponse("Failed to fetch order: " + err.Error()), nil
	}
	if order == nil {
		return NewErrorResponse("Order not found"), nil
	}
	return NewResponse(order, "Order fetched successfully"), nil
}

func (ctrl *OrderController) GetByDate(ctx context.Context, input *model.GetOrdersByDateInput) (*model.DefaultResponse, error) {
	orders, err := ctrl.orderService.GetAllByDate(ctx, input.Date)
	if err != nil {
		return NewErrorResponse("Failed to fetch orders by date: " + err.Error()), nil
	}
	return NewResponse(orders, "Orders fetched successfully"), nil
}

func (ctrl *OrderController) GetByUserId(ctx context.Context, input *model.GetOrdersByUserIdInput) (*model.DefaultResponse, error) {
	orders, err := ctrl.orderService.GetAllByUserId(ctx, input.UserID)
	if err != nil {
		return NewErrorResponse("Failed to fetch orders by user ID: " + err.Error()), nil
	}
	return NewResponse(orders, "Orders fetched successfully"), nil
}

func (ctrl *OrderController) Create(ctx context.Context, input *model.CreateOrderInput) (*model.DefaultResponse, error) {
	err := ctrl.orderService.Create(ctx, input.Body)
	if err != nil {
		return NewErrorResponse("Failed to create order: " + err.Error()), nil
	}
	return NewResponse(nil, "Order created successfully"), nil
}

func (ctrl *OrderController) Update(ctx context.Context, input *model.UpdateOrderInput) (*model.DefaultResponse, error) {
	err := ctrl.orderService.Update(ctx, input.ID, input.Body)
	if err != nil {
		return NewErrorResponse("Failed to update order: " + err.Error()), nil
	}
	return NewResponse(nil, "Order updated successfully"), nil
}

func (ctrl *OrderController) Delete(ctx context.Context, input *model.GetOrderInput) (*model.DefaultResponse, error) {
	err := ctrl.orderService.Delete(ctx, input.ID)
	if err != nil {
		return NewErrorResponse("Failed to delete order: " + err.Error()), nil
	}
	return NewResponse(nil, "Order deleted successfully"), nil
}

func (ctrl *OrderController) InitiateMtfOrders(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	ctrl.orderService.InitiateMtfOrders(ctx)
	return NewResponse(nil, "MTF orders initiation triggered"), nil
}
