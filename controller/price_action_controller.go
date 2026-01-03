package controller

import (
	"context"
	"net/http"
	"time"

	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type PriceActionController struct {
	paService    service.PriceActionService
	isProduction bool
}

func NewPriceActionController(s service.PriceActionService, isProd bool) *PriceActionController {
	return &PriceActionController{paService: s, isProduction: isProd}
}

func (ctrl *PriceActionController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "trigger-automation",
		Method:      http.MethodPost,
		Path:        "/api/price-action/automate",
		Summary:     "Trigger PA Automation",
		Tags:        []string{"PriceAction"},
	}, ctrl.TriggerAutomation)

	huma.Register(api, huma.Operation{
		OperationID: "get-pa-by-symbol",
		Method:      http.MethodGet,
		Path:        "/api/price-action/{symbol}",
		Summary:     "Get PA by Symbol",
		Tags:        []string{"PriceAction"},
	}, ctrl.GetPABySymbol)

	huma.Register(api, huma.Operation{
		OperationID: "check-ob-mitigation",
		Method:      http.MethodPost,
		Path:        "/api/price-action/ob/check",
		Summary:     "Force Refresh OB Mitigations",
		Tags:        []string{"PriceAction"},
	}, ctrl.CheckOBMitigation)

	huma.Register(api, huma.Operation{
		OperationID: "get-ob-mitigation",
		Method:      http.MethodGet,
		Path:        "/api/price-action/ob/mitigation",
		Summary:     "Get Cached OB Mitigations",
		Tags:        []string{"PriceAction"},
	}, ctrl.GetOBMitigation)

	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)
	adminMw := middleware.HumaAdminOnly(api)

	huma.Register(api, huma.Operation{
		OperationID: "save-ob",
		Method:      http.MethodPost,
		Path:        "/api/price-action/ob",
		Summary:     "Save OB",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.SaveOrderBlock)

	huma.Register(api, huma.Operation{
		OperationID: "update-ob",
		Method:      http.MethodPatch,
		Path:        "/api/price-action/ob",
		Summary:     "Update OB",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.UpdateOrderBlock)

	huma.Register(api, huma.Operation{
		OperationID: "delete-ob",
		Method:      http.MethodDelete,
		Path:        "/api/price-action/ob",
		Summary:     "Delete OB",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.DeleteOrderBlock)

	huma.Register(api, huma.Operation{
		OperationID: "check-fvg-mitigation",
		Method:      http.MethodPost,
		Path:        "/api/price-action/fvg/check",
		Summary:     "Force Refresh FVG Mitigations",
		Tags:        []string{"PriceAction"},
	}, ctrl.CheckFvgMitigation)

	huma.Register(api, huma.Operation{
		OperationID: "get-fvg-mitigation",
		Method:      http.MethodGet,
		Path:        "/api/price-action/fvg/mitigation",
		Summary:     "Get Cached FVG Mitigations",
		Tags:        []string{"PriceAction"},
	}, ctrl.GetFvgMitigation)

	huma.Register(api, huma.Operation{
		OperationID: "cleanup",
		Method:      http.MethodPost,
		Path:        "/api/price-action/cleanup",
		Summary:     "Clean up filled FVGs & OBs",
		Tags:        []string{"PriceAction"},
	}, ctrl.PACleanUp)

	huma.Register(api, huma.Operation{
		OperationID: "save-fvg",
		Method:      http.MethodPost,
		Path:        "/api/price-action/fvg",
		Summary:     "Save FVG",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.SaveFvg)

	huma.Register(api, huma.Operation{
		OperationID: "update-fvg",
		Method:      http.MethodPatch,
		Path:        "/api/price-action/fvg",
		Summary:     "Update FVG",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.UpdateFvg)

	huma.Register(api, huma.Operation{
		OperationID: "delete-fvg",
		Method:      http.MethodDelete,
		Path:        "/api/price-action/fvg",
		Summary:     "Delete FVG",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"PriceAction (Admin)"},
	}, ctrl.DeleteFvg)

	huma.Register(api, huma.Operation{
		OperationID: "load-pa-from-csv",
		Method:      http.MethodPost,
		Path:        "/api/price-action/load-from-csv",
		Summary:     "Load fvg & ob from CSV",
		Tags:        []string{"PriceAction"},
	}, ctrl.loadFromCsv)

}

func (ctrl *PriceActionController) TriggerAutomation(ctx context.Context, input *struct{}) (*model.TriggerAutomationResponse, error) {
	bgCtx := context.Background()
	go func() {
		_ = ctrl.paService.AutomateOrderBlock(bgCtx, 0)
		_ = ctrl.paService.AutomateFvg(bgCtx, 0)
	}()
	return &model.TriggerAutomationResponse{Body: model.Response{Success: true, Message: "Scanning started"}}, nil
}

func (ctrl *PriceActionController) GetPABySymbol(ctx context.Context, input *model.GetPAInput) (*model.DefaultResponse, error) {
	data, err := ctrl.paService.GetPABySymbol(ctx, input.Symbol)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(data, "PA data fetched successfully"), nil
}

func (ctrl *PriceActionController) CheckOBMitigation(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	data, err := ctrl.paService.CheckOBMitigation(ctx)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(data, "OB mitigation check complete"), nil
}

func (ctrl *PriceActionController) GetOBMitigation(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	result := make([]model.ObResponse, 0)
	if ok, err := cache.GetPriceActionResponseCache("ObCache", &result); ok && err == nil {
		return NewResponse(result, "Cached OB mitigation data fetched"), nil
	}

	data, err := ctrl.paService.CheckOBMitigation(ctx)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(data, "OB mitigation data fetched"), nil
}

func (ctrl *PriceActionController) SaveOrderBlock(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	err := ctrl.paService.SaveOrderBlock(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Order Block saved successfully"), nil
}

func (ctrl *PriceActionController) UpdateOrderBlock(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	err := ctrl.paService.UpdateOrderBlock(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Order Block updated successfully"), nil
}

func (ctrl *PriceActionController) DeleteOrderBlock(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	req := input.Body
	err := ctrl.paService.DeleteOrderBlock(ctx, req.Symbol, req.Date)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Order Block deleted successfully"), nil
}

func (ctrl *PriceActionController) CheckFvgMitigation(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	data, err := ctrl.paService.CheckFvgMitigation(ctx)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(data, "FVG mitigation check complete"), nil
}

func (ctrl *PriceActionController) GetFvgMitigation(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	result := make([]model.ObResponse, 0)
	if ok, err := cache.GetPriceActionResponseCache("FvgCache", &result); ok && err == nil {
		return NewResponse(result, "Cached FVG mitigation data fetched"), nil
	}

	data, err := ctrl.paService.CheckFvgMitigation(ctx)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(data, "FVG mitigation data fetched"), nil
}

func (ctrl *PriceActionController) PACleanUp(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	err := ctrl.paService.PACleanUp(ctx)
	if err != nil {
		return NewErrorResponse("Failed to cleanup: " + err.Error()), nil
	}
	return NewResponse(nil, "Price Action cleanup task executed successfully"), nil
}

func (ctrl *PriceActionController) SaveFvg(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	err := ctrl.paService.SaveFvg(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Saved successfully"), nil
}

func (ctrl *PriceActionController) UpdateFvg(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	err := ctrl.paService.UpdateFvg(ctx, input.Body)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Updated successfully"), nil
}

func (ctrl *PriceActionController) DeleteFvg(ctx context.Context, input *model.ObInput) (*model.DefaultResponse, error) {
	req := input.Body
	err := ctrl.paService.DeleteFvg(ctx, req.Symbol, req.Date)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}
	return NewResponse(nil, "Deleted successfully"), nil
}

func (ctrl *PriceActionController) loadFromCsv(ctx context.Context, input *model.UploadMarginInput) (*model.DefaultResponse, error) {
	formData := input.RawBody.Data()
	ctrl.paService.AddOlderFvgAndOb(ctx, formData.File.Filename, formData.File.File, time.Now().AddDate(0, -1, 0).Format("02-01-2006"))
	return NewResponse(nil, "Margins loaded successfully from CSV"), nil
}
