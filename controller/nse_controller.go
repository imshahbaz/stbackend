package controller

import (
	"context"
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type NseController struct {
	nseService service.NseService
}

func NewNseController(ns service.NseService) *NseController {
	return &NseController{
		nseService: ns,
	}
}

func (ctrl *NseController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-stock-history",
		Method:      http.MethodGet,
		Path:        "/api/nse/history",
		Summary:     "Get Historical Stock Data",
		Description: "Fetches stock history for a specific symbol. Utilizes a 1-hour time cache.",
		Tags:        []string{"Stocks"},
	}, ctrl.GetStockHistory)

	huma.Register(api, huma.Operation{
		OperationID: "get-all-indices",
		Method:      http.MethodGet,
		Path:        "/api/nse/allindices",
		Summary:     "Get All NSE Indices",
		Tags:        []string{"Stocks"},
	}, ctrl.GetAllIndices)
}

func (ctrl *NseController) GetStockHistory(ctx context.Context, input *model.NseHistoryInput) (*model.DefaultResponse, error) {
	data, err := ctrl.nseService.FetchStockData(ctx, input.Symbol)
	if err != nil {
		return NewErrorResponse("Failed to get history"), nil
	}
	return NewResponse(data, "Fetch Success"), nil
}

func (ctrl *NseController) GetAllIndices(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	data, err := ctrl.nseService.FetchAllIndices()
	if err != nil {
		return NewErrorResponse("Failed to get all indices data"), nil
	}
	return NewResponse(data, "Fetch Success"), nil
}
