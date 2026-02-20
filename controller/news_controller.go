package controller

import (
	"backend/model"
	"backend/service"
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
)

type NewsController struct {
	newsSvc service.NewsService
}

func NewNewsController(newsSvc service.NewsService) *NewsController {
	return &NewsController{
		newsSvc: newsSvc,
	}
}

func (ctrl *NewsController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-stock-news",
		Method:      http.MethodGet,
		Path:        "/api/news/{symbol}",
		Summary:     "Get Stock News",
		Description: "Fetches stock news for a specific symbol",
		Tags:        []string{"News"},
	}, ctrl.fetchTvNews)

	huma.Register(api, huma.Operation{
		OperationID: "get-stock-ai-analysis",
		Method:      http.MethodGet,
		Path:        "/api/news/ai/{symbol}",
		Summary:     "Get Stock Ai Analysis",
		Description: "Fetches stock ai analysis for a specific symbol",
		Tags:        []string{"News"},
	}, ctrl.fetchGenAiAnalysis)

}

func (ctrl *NewsController) fetchTvNews(ctx context.Context, input *struct {
	Symbol string `path:"symbol" doc:"Stock Symbol" example:"RELIANCE"`
}) (*model.DefaultResponse, error) {
	return ctrl.newsSvc.FetchTVNews(input.Symbol)
}

func (ctrl *NewsController) fetchGenAiAnalysis(ctx context.Context, input *struct {
	Symbol string `path:"symbol" doc:"Stock Symbol" example:"RELIANCE"`
}) (*model.DefaultResponse, error) {
	return ctrl.newsSvc.FetchGenAiAnalysis(input.Symbol)
}
