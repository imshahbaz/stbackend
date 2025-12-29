package controller

import (
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
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

// RegisterRoutes sets up protected admin-only configuration endpoints.
func (ctrl *ConfigController) RegisterRoutes(router *gin.RouterGroup) {
	configGroup := router.Group("/config")
	// Enforce both Authentication and Admin RBAC
	configGroup.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
	{
		configGroup.POST("/reload", ctrl.reloadMongoEnvConfig)
		configGroup.GET("/active", ctrl.getActiveMongoEnvConfig)
		configGroup.PATCH("/update", ctrl.updateMongoEnvConfig)
	}
}

// reloadMongoEnvConfig godoc
// @Summary      Reload System Configuration
// @Description  Triggers a fresh fetch from MongoDB to update the in-memory cache.
// @Tags         Config
// @Produce      json
// @Success      200  {object}  model.Response
// @Failure      500  {object}  model.Response
// @Router       /config/reload [post]
func (ctrl *ConfigController) reloadMongoEnvConfig(ctx *gin.Context) {
	ctrl.cfgSvc.LoadMongoEnvConfig(ctx)
}

// updateMongoEnvConfig godoc
// @Summary      Update System Configuration
// @Description  Updates MongoDB and hot-swaps active memory config.
// @Tags         Config
// @Accept       json
// @Produce      json
// @Param        request  body      model.MongoEnvConfig  true  "Update Config Fields"
// @Success      200      {object}  model.Response
// @Failure      400      {object}  model.Response
// @Failure      500      {object}  model.Response
// @Router       /config/update [patch]
func (ctrl *ConfigController) updateMongoEnvConfig(ctx *gin.Context) {
	var request model.MongoEnvConfig
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid Request Body",
		})
		return
	}
	ctrl.cfgSvc.UpdateMongoEnvConfig(ctx, request)
}

// getActiveMongoEnvConfig godoc
// @Summary      Get Active Configuration
// @Description  Returns current system settings (Leverage, API Keys, etc.) from memory.
// @Tags         Config
// @Produce      json
// @Success      200  {object}  model.MongoEnvConfig
// @Failure      500  {object}  model.Response
// @Router       /config/active [get]
func (ctrl *ConfigController) getActiveMongoEnvConfig(ctx *gin.Context) {
	ctrl.cfgSvc.GetActiveMongoEnvConfig(ctx)
}
