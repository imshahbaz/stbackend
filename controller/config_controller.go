package controller

import (
	"context"
	"net/http"

	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type ConfigController struct {
	cfgSvc       service.ConfigService
	isProduction bool
}

func NewConfigController(cfgSvc service.ConfigService, isProduction bool) *ConfigController {
	return &ConfigController{
		cfgSvc:       cfgSvc,
		isProduction: isProduction,
	}
}

func (ctrl *ConfigController) RegisterRoutes(api huma.API) {
	authMw := middleware.HumaAuthMiddleware(api, ctrl.isProduction)
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
}

func (ctrl *ConfigController) reloadMongoEnvConfig(ctx context.Context, input *struct{}) (*model.DefaultResponse, error) {
	if err := ctrl.cfgSvc.LoadMongoEnvConfig(ctx); err != nil {
		return NewErrorResponse("Error Loading Mongo Configs: " + err.Error()), nil
	}
	return NewResponse(nil, "Mongo Configs Loaded Successfully"), nil
}

func (ctrl *ConfigController) getActiveMongoEnvConfig(ctx context.Context, input *struct{}) (*model.ConfigResponse, error) {
	cfg := ctrl.cfgSvc.GetActiveMongoEnvConfig()
	return &model.ConfigResponse{Body: cfg}, nil
}

func (ctrl *ConfigController) updateMongoEnvConfig(ctx context.Context, input *model.UpdateConfigInput) (*model.DefaultResponse, error) {
	req := input.Body

	if err := ctrl.cfgSvc.UpdateMongoEnvConfig(ctx, req); err != nil {
		return NewErrorResponse("Error Updating Mongo Configs: " + err.Error()), nil
	}
	cache.EnableRedisCache = req.RedisCache
	return NewResponse(nil, "Mongo Configs Updated Successfully"), nil
}
