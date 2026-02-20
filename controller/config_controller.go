package controller

import (
	"context"
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type ConfigController struct {
	cfgSvc       service.ConfigService
	isProduction service.IsProduction
}

func NewConfigController(cfgSvc service.ConfigService, isProduction service.IsProduction) *ConfigController {
	return &ConfigController{
		cfgSvc:       cfgSvc,
		isProduction: isProduction,
	}
}

func (ctrl *ConfigController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, bool(ctrl.isProduction))
	adminMw := middleware.HumaAdminOnly(api)

	huma.Register(api, huma.Operation{
		OperationID: "reload-config",
		Method:      http.MethodPost,
		Path:        "/api/config/reload",
		Summary:     "Reload System Configuration",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Config"},
	}, ctrl.reloadMongoEnvConfig)

	huma.Register(api, huma.Operation{
		OperationID: "get-active-config",
		Method:      http.MethodGet,
		Path:        "/api/config/active",
		Summary:     "Get Active Configuration",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Config"},
	}, ctrl.getActiveMongoEnvConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-config",
		Method:      http.MethodPatch,
		Path:        "/api/config/update",
		Summary:     "Update System Configuration",
		Middlewares: huma.Middlewares{authMw, adminMw},
		Security:    []map[string][]string{{"bearer": {}}},
		Tags:        []string{"Config"},
	}, ctrl.updateMongoEnvConfig)

	//client
	huma.Register(api, huma.Operation{
		OperationID: "get-active-client-config",
		Method:      http.MethodGet,
		Path:        "/api/config/client/active",
		Summary:     "Get Active Client Configuration",
		Tags:        []string{"Config"},
	}, ctrl.getActiveMongoClientConfig)

	huma.Register(api, huma.Operation{
		OperationID: "reload-client-config",
		Method:      http.MethodPost,
		Path:        "/api/config/client/reload",
		Summary:     "Reload Client Configuration",
		Tags:        []string{"Config"},
	}, ctrl.reloadMongoClientConfig)

	huma.Register(api, huma.Operation{
		OperationID: "update-client-config",
		Method:      http.MethodPatch,
		Path:        "/api/config/client/update",
		Summary:     "Update Client Configuration",
		Tags:        []string{"Config"},
	}, ctrl.updateMongoClientConfig)

}

//backend

func (ctrl *ConfigController) reloadMongoEnvConfig(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	if err := ctrl.cfgSvc.LoadMongoEnvConfig(ctx); err != nil {
		return NewTypedError[any]("Error Loading Mongo Configs: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Mongo Configs Loaded Successfully"), nil
}

func (ctrl *ConfigController) getActiveMongoEnvConfig(ctx context.Context, input *struct{}) (*model.TypedResponse[model.MongoEnvConfig], error) {
	cfg := ctrl.cfgSvc.GetActiveMongoEnvConfig()
	return NewTypedResponse(cfg, "Success"), nil
}

func (ctrl *ConfigController) updateMongoEnvConfig(ctx context.Context, input *model.RequestBody[model.MongoEnvConfig]) (*model.TypedResponse[any], error) {
	req := input.Body

	if err := ctrl.cfgSvc.UpdateMongoEnvConfig(ctx, req); err != nil {
		return NewTypedError[any]("Error Updating Mongo Configs: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Mongo Configs Updated Successfully"), nil
}

//client

func (ctrl *ConfigController) getActiveMongoClientConfig(ctx context.Context, input *struct{}) (*model.TypedResponse[model.ClientConfigs], error) {
	cfg := ctrl.cfgSvc.GetActiveMongoClientConfig()
	return NewTypedResponse(cfg, "Client config fetch success"), nil
}

func (ctrl *ConfigController) reloadMongoClientConfig(ctx context.Context, input *struct{}) (*model.TypedResponse[any], error) {
	if err := ctrl.cfgSvc.LoadClientConfig(ctx); err != nil {
		return NewTypedError[any]("Error Loading Mongo Client Configs: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Mongo Client Configs Loaded Successfully"), nil
}

func (ctrl *ConfigController) updateMongoClientConfig(ctx context.Context, input *model.RequestBody[model.ClientConfigs]) (*model.TypedResponse[any], error) {
	req := input.Body

	if err := ctrl.cfgSvc.UpdateMongoClientConfig(ctx, req); err != nil {
		return NewTypedError[any]("Error Updating Mongo Configs: " + err.Error()), nil
	}
	return NewTypedResponse[any](nil, "Mongo Configs Updated Successfully"), nil
}
