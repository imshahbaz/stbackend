package controller

import (
	"backend/middleware"
	"backend/model"
	"backend/service"
	"net/http"

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

func (ctrl *ConfigController) RegisterRoutes(router *gin.RouterGroup) {
	protectedGrp := router.Group("/config")
	protectedGrp.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
	{
		protectedGrp.POST("", ctrl.reloadMongoEnvConfig)
		protectedGrp.GET("", ctrl.getActiveMongoEnvConfig)
		protectedGrp.PATCH("", ctrl.updateMongoEnvConfig)
	}
}

// reloadMongoEnvConfig godoc
// @Summary      Reload System Configuration
// @Description  Triggers a fresh fetch from MongoDB to update the in-memory cache across all services.
// @Tags         Config
// @Produce      json
// @Success      200  {object}  model.Response  "Successfully reloaded"
// @Failure      500  {object}  model.Response  "Internal Server Error"
// @Router       /config [post]
func (s *ConfigController) reloadMongoEnvConfig(ctx *gin.Context) {
	s.cfgSvc.LoadMongoEnvConfig(ctx)
}

// updateMongoEnvConfig godoc
// @Summary      Update System Configuration
// @Description  Updates the configuration document in MongoDB and hot-swaps the active memory pointer.
// @Tags         Config
// @Accept       json
// @Produce      json
// @Param        request  body      model.MongoEnvConfig  true  "Update Config Fields"
// @Success      200      {object}  model.Response        "Successfully updated"
// @Failure      400      {object}  model.Response        "Invalid Request Body"
// @Failure      500      {object}  model.Response        "Internal Server Error"
// @Router       /config [patch]
func (s *ConfigController) updateMongoEnvConfig(ctx *gin.Context) {
	var request model.MongoEnvConfig
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Error:   "Invalid Request",
		})
		return
	}
	s.cfgSvc.UpdateMongoEnvConfig(ctx, request)
}

// getActiveMongoEnvConfig godoc
// @Summary      Get Active Configuration
// @Description  Returns the current system settings (Leverage, API Keys, etc.) from the real-time cache.
// @Tags         Config
// @Produce      json
// @Success      200  {object}  model.Response{data=model.MongoEnvConfig}  "Current active config"
// @Failure      500  {object}  model.Response                             "Internal Server Error"
// @Router       /config [get]
func (s *ConfigController) getActiveMongoEnvConfig(ctx *gin.Context) {
	s.cfgSvc.GetActiveMongoEnvConfig(ctx)
}
