package controller

import (
	"context"
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type ChartInkController struct {
	chartInkService service.ChartInkService
	strategyService service.StrategyService
}

func NewChartInkController(ci service.ChartInkService, ss service.StrategyService) *ChartInkController {
	return &ChartInkController{
		chartInkService: ci,
		strategyService: ss,
	}
}

func (ctrl *ChartInkController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "fetch-chartink-data",
		Method:      http.MethodGet,
		Path:        "/api/chartink/fetch",
		Summary:     "Fetch raw ChartInk data",
		Description: "Triggers a scan on ChartInk for the given strategy and returns raw stock data",
		Tags:        []string{"ChartInk"},
	}, ctrl.fetchData)

	huma.Register(api, huma.Operation{
		OperationID: "fetch-chartink-with-margin",
		Method:      http.MethodGet,
		Path:        "/api/chartink/fetchWithMargin",
		Summary:     "Fetch ChartInk data with Margin info",
		Description: "Triggers a scan and maps results with current margin and leverage data",
		Tags:        []string{"ChartInk"},
	}, ctrl.fetchWithMargin)
}

func (ctrl *ChartInkController) fetchData(ctx context.Context, input *model.ChartInkInput) (*model.DefaultResponse, error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewErrorResponse("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchData(strategyDto)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}

	return NewResponse(data, "ChartInk data fetched"), nil
}

func (ctrl *ChartInkController) fetchWithMargin(ctx context.Context, input *model.ChartInkInput) (*model.DefaultResponse, error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewErrorResponse("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchWithMargin(strategyDto)
	if err != nil {
		return NewErrorResponse(err.Error()), nil
	}

	return NewResponse(data, "ChartInk data with margin details fetched"), nil
}

func (ctrl *ChartInkController) findStrategy(name string) (model.StrategyDto, bool) {
	strategies := ctrl.strategyService.GetAllStrategies()
	for _, s := range strategies {
		if s.Name == name {
			return s, true
		}
	}
	return model.StrategyDto{}, false
}
