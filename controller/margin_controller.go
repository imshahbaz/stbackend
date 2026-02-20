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

	huma.Register(api, huma.Operation{
		OperationID: "load-margin-from-csv",
		Method:      http.MethodPost,
		Path:        "/api/margin/load-from-csv",
		Summary:     "Load margins from CSV",
		Tags:        []string{"Margin"},
	}, ctrl.loadFromCsv)

	huma.Register(api, huma.Operation{
		OperationID: "sync-mtf-from-json",
		Method:      http.MethodPost,
		Path:        "/api/margin/json",
		Summary:     "Sync selected MTF data",
		Tags:        []string{"Margin"},
	}, ctrl.syncMTF)

	huma.Register(api, huma.Operation{
		OperationID: "sync-margin-token",
		Method:      http.MethodPost,
		Path:        "/api/margin/sync-token",
		Summary:     "Sync Margin Token",
		Description: "Syncs instrument tokens from Angel One for matching margins in local cache",
		Tags:        []string{"Margin"},
	}, ctrl.syncMarginToken)

	huma.Register(api, huma.Operation{
		OperationID: "get-all-options",
		Method:      http.MethodGet,
		Path:        "/api/margin/options",
		Summary:     "Get all options",
		Description: "Returns available expiries and strikes for major indices",
		Tags:        []string{"Margin"},
	}, ctrl.getAllOptions)

	huma.Register(api, huma.Operation{
		OperationID: "reload-options",
		Method:      http.MethodPost,
		Path:        "/api/margin/options/reload",
		Summary:     "Reload options",
		Description: "Forces a reload of all options from MongoDB into the memory cache",
		Tags:        []string{"Margin"},
	}, ctrl.reloadAllOptions)
}

func (ctrl *MarginController) getAllMargins(ctx context.Context, input *struct{}) (*model.TypedResponse[[]model.Margin], error) {
	margins := ctrl.marginService.GetAllMargins()
	if margins == nil {
		margins = []model.Margin{}
	}
	return NewTypedResponse(margins, "Success"), nil
}

func (ctrl *MarginController) getMargin(ctx context.Context, input *model.GetMarginInput) (*model.TypedResponse[*model.Margin], error) {
	margin, exists := ctrl.marginService.GetMargin(input.Symbol)
	if !exists {
		return NewTypedError[*model.Margin]("Margin not found for symbol: " + input.Symbol), nil
	}
	return NewTypedResponse(margin, "Success"), nil
}

func (ctrl *MarginController) reloadAllMargins(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	if err := ctrl.marginService.ReloadAllMargins(ctx); err != nil {
		return NewTypedError[any]("Failed to reload margins: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Margins reloaded successfully"), nil
}

func (ctrl *MarginController) loadFromCsv(ctx context.Context, input *model.UploadMarginInput) (*model.TypedResponse[any], error) {
	formData := input.RawBody.Data()
	err := ctrl.marginService.LoadFromCsv(ctx, formData.File.Filename, formData.File.File)
	if err != nil {
		return nil, huma.Error500InternalServerError("Failed to load margins from CSV: " + err.Error())
	}
	return NewTypedResponse[any](nil, "Margins loaded successfully from CSV"), nil
}

func (ctrl *MarginController) syncMTF(ctx context.Context, input *model.MTFInput) (*model.TypedResponse[any], error) {
	formData := input.RawBody.Data().File.File
	if err := ctrl.marginService.SyncMTF(ctx, formData); err != nil {
		return nil, huma.Error500InternalServerError("Failed to sync MTF data: " + err.Error())
	}
	return NewTypedResponse[any](nil, "MTF data synced successfully"), nil
}

func (ctrl *MarginController) syncMarginToken(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	if err := ctrl.marginService.SyncMarginToken(ctx); err != nil {
		return nil, huma.Error500InternalServerError("Failed to sync margin tokens: " + err.Error())
	}
	return NewTypedResponse[any](nil, "Margin tokens synced successfully"), nil
}

func (ctrl *MarginController) getAllOptions(ctx context.Context, input *struct{}) (*model.TypedResponse[[]model.OptionOutput], error) {
	options := ctrl.marginService.GetAllOptions()
	var data []model.OptionOutput
	if options != nil {
		data = *options
	}
	return NewTypedResponse(data, "Success"), nil
}

func (ctrl *MarginController) reloadAllOptions(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	if err := ctrl.marginService.ReloadAllOptions(ctx); err != nil {
		return NewTypedError[any]("Failed to reload options: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Options reloaded successfully"), nil
}
