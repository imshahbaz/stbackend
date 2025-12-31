package controller

import (
	"context"
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type MarginController struct {
	marginService service.MarginService
}

func NewMarginController(ms service.MarginService) *MarginController {
	return &MarginController{
		marginService: ms,
	}
}

func (ctrl *MarginController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-all-margins",
		Method:      http.MethodGet,
		Path:        "/api/margin/all",
		Summary:     "Get all margins",
		Description: "Returns a list of all stock margins from the local memory cache",
		Tags:        []string{"Margin"},
	}, ctrl.getAllMargins)

	huma.Register(api, huma.Operation{
		OperationID: "get-margin",
		Method:      http.MethodGet,
		Path:        "/api/margin/symbol/{symbol}",
		Summary:     "Get margin by symbol",
		Description: "Fetches the margin details for a specific stock symbol",
		Tags:        []string{"Margin"},
	}, ctrl.getMargin)

	huma.Register(api, huma.Operation{
		OperationID: "reload-margins",
		Method:      http.MethodPost,
		Path:        "/api/margin/reload",
		Summary:     "Reload margins",
		Description: "Forces a reload of all margins from MongoDB into the memory cache",
		Tags:        []string{"Margin"},
	}, ctrl.reloadAllMargins)
}

func (ctrl *MarginController) getAllMargins(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	margins := ctrl.marginService.GetAllMargins()
	if margins == nil {
		margins = []model.Margin{}
	}
	return NewResponse(margins, "Success"), nil
}

func (ctrl *MarginController) getMargin(ctx context.Context, input *model.GetMarginInput) (*model.DefaultResponse, error) {
	margin, exists := ctrl.marginService.GetMargin(input.Symbol)
	if !exists {
		return NewErrorResponse("Margin not found for symbol: " + input.Symbol), nil
	}
	return NewResponse(margin, "Success"), nil
}

func (ctrl *MarginController) reloadAllMargins(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	if err := ctrl.marginService.ReloadAllMargins(ctx); err != nil {
		return NewErrorResponse("Failed to reload margins: " + err.Error()), nil
	}
	return NewResponse(nil, "Margins reloaded successfully"), nil
}
