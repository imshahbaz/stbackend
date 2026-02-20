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

	huma.Register(api, huma.Operation{
		OperationID: "fetch-chartink-backtest",
		Method:      http.MethodGet,
		Path:        "/api/chartink/backtest",
		Summary:     "Fetch ChartInk Backtest data",
		Description: "Fetches historical backtest signals for the given strategy",
		Tags:        []string{"ChartInk"},
	}, ctrl.fetchBacktestData)

	huma.Register(api, huma.Operation{
		OperationID: "fetch-chartink-backtest-with-margin",
		Method:      http.MethodGet,
		Path:        "/api/chartink/backtestWithMargin",
		Summary:     "Fetch ChartInk Backtest data with Margin info",
		Description: "Fetches historical backtest signals and maps each stock with margin data",
		Tags:        []string{"ChartInk"},
	}, ctrl.fetchBacktestWithMargin)

	huma.Register(api, huma.Operation{
		OperationID: "fetch-chartink-backtest-today-with-margin",
		Method:      http.MethodGet,
		Path:        "/api/chartink/backtestTodayWithMargin",
		Summary:     "Fetch ChartInk Backtest data for today with Margin info",
		Description: "Fetches backtest signals for the current day and maps each stock with margin data",
		Tags:        []string{"ChartInk"},
	}, ctrl.fetchBacktestTodayWithMargin)
}

func (ctrl *ChartInkController) fetchData(ctx context.Context, input *model.ChartInkInput) (*model.TypedResponse[*model.ChartInkResponseDto], error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewTypedError[*model.ChartInkResponseDto]("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchData(strategyDto)
	if err != nil {
		return NewTypedError[*model.ChartInkResponseDto](err.Error()), nil
	}

	return NewTypedResponse(data, "ChartInk data fetched"), nil
}

func (ctrl *ChartInkController) fetchWithMargin(ctx context.Context, input *model.ChartInkInput) (*model.TypedResponse[[]model.StockMarginDto], error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewTypedError[[]model.StockMarginDto]("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchWithMargin(strategyDto)
	if err != nil {
		return NewTypedError[[]model.StockMarginDto](err.Error()), nil
	}

	return NewTypedResponse(data, "ChartInk data with margin details fetched"), nil
}

func (ctrl *ChartInkController) fetchBacktestData(ctx context.Context, input *model.ChartInkInput) (*model.TypedResponse[[]model.ChartinkBacktestSignal], error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewTypedError[[]model.ChartinkBacktestSignal]("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchBacktestData(strategyDto)
	if err != nil {
		return NewTypedError[[]model.ChartinkBacktestSignal](err.Error()), nil
	}

	return NewTypedResponse(data, "ChartInk backtest data fetched"), nil
}

func (ctrl *ChartInkController) fetchBacktestWithMargin(ctx context.Context, input *model.ChartInkInput) (*model.TypedResponse[[]model.ChartinkBacktestSignalWithMargin], error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewTypedError[[]model.ChartinkBacktestSignalWithMargin]("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchBacktestWithMargin(strategyDto)
	if err != nil {
		return NewTypedError[[]model.ChartinkBacktestSignalWithMargin](err.Error()), nil
	}

	return NewTypedResponse(data, "ChartInk backtest data with margin details fetched"), nil
}

func (ctrl *ChartInkController) fetchBacktestTodayWithMargin(ctx context.Context, input *model.ChartInkInput) (*model.TypedResponse[[]model.ChartinkBacktestSignalWithMargin], error) {
	strategyDto, exists := ctrl.findStrategy(input.Strategy)
	if !exists {
		return NewTypedError[[]model.ChartinkBacktestSignalWithMargin]("Strategy not found"), nil
	}

	data, err := ctrl.chartInkService.FetchBacktestTodayWithMargin(strategyDto)
	if err != nil {
		return NewTypedError[[]model.ChartinkBacktestSignalWithMargin](err.Error()), nil
	}

	return NewTypedResponse(data, "ChartInk today's backtest data with margin details fetched"), nil
}

func (ctrl *ChartInkController) findStrategy(name string) (model.StrategyDto, bool) {
	strategies := ctrl.strategyService.GetAllStrategiesAdmin()
	for _, s := range strategies {
		if s.Name == name {
			return s, true
		}
	}
	return model.StrategyDto{}, false
}
