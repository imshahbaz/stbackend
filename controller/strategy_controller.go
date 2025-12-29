package controller

import (
	"net/http"

	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type StrategyController struct {
	strategyService service.StrategyService
	isProduction    bool
}

func NewStrategyController(ss service.StrategyService, isProduction bool) *StrategyController {
	return &StrategyController{
		strategyService: ss,
		isProduction:    isProduction,
	}
}

// RegisterRoutes maps endpoints to the /strategy group with appropriate middleware.
func (ctrl *StrategyController) RegisterRoutes(router *gin.RouterGroup) {
	strategyGroup := router.Group("/strategy")
	{
		// Public route - typically used by the scanner dashboard
		strategyGroup.GET("", ctrl.getAllStrategies)

		// Protected routes - requires Admin role and JWT
		adminGroup := strategyGroup.Group("")
		adminGroup.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
		{
			adminGroup.POST("", ctrl.createStrategy)
			adminGroup.PUT("", ctrl.updateStrategy)
			adminGroup.DELETE("", ctrl.deleteStrategy)
			adminGroup.POST("/reload", ctrl.reloadAllStrategies)
			adminGroup.GET("/admin", ctrl.getAllStrategiesAdmin)
		}
	}
}

// getAllStrategies retrieves active trading strategies.
// @Summary      Get all strategies
// @Description  Returns a list of all configured active trading strategies
// @Tags         Strategy
// @Produce      json
// @Success      200  {array}  model.StrategyDto
// @Router       /strategy [get]
func (ctrl *StrategyController) getAllStrategies(c *gin.Context) {
	strategies := ctrl.strategyService.GetAllStrategies()
	if strategies == nil {
		c.JSON(http.StatusOK, []model.StrategyDto{})
		return
	}
	c.JSON(http.StatusOK, strategies)
}

// createStrategy adds a new strategy.
// @Summary      Create a strategy
// @Description  Saves a new trading strategy configuration to MongoDB
// @Tags         Strategy
// @Accept       json
// @Produce      json
// @Param        request  body      model.StrategyDto  true  "Strategy Details"
// @Success      201      {object}  model.Strategy
// @Failure      400      {object}  map[string]string
// @Router       /strategy [post]
func (ctrl *StrategyController) createStrategy(c *gin.Context) {
	var request model.StrategyDto
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid strategy format: " + err.Error()})
		return
	}

	res, err := ctrl.strategyService.CreateStrategy(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// updateStrategy modifies an existing strategy.
// @Summary      Update a strategy
// @Description  Updates an existing strategy configuration by name/ID
// @Tags         Strategy
// @Accept       json
// @Produce      json
// @Param        request  body      model.StrategyDto  true  "Updated Details"
// @Success      200      {object}  model.Strategy
// @Router       /strategy [put]
func (ctrl *StrategyController) updateStrategy(c *gin.Context) {
	var request model.StrategyDto
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	res, err := ctrl.strategyService.UpdateStrategy(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// deleteStrategy removes a strategy.
// @Summary      Delete a strategy
// @Description  Removes a strategy from the system using its ID/Name
// @Tags         Strategy
// @Param        id   query     string  true  "Strategy ID (Name)"
// @Success      204  "No Content"
// @Router       /strategy [delete]
func (ctrl *StrategyController) deleteStrategy(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Strategy ID is required"})
		return
	}

	if err := ctrl.strategyService.DeleteStrategy(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// reloadAllStrategies refreshes cache from DB.
// @Summary      Reload strategies
// @Description  Syncs the in-memory strategy cache with MongoDB
// @Tags         Strategy
// @Produce      json
// @Success      200  {object}  map[string]string
// @Router       /strategy/reload [post]
func (ctrl *StrategyController) reloadAllStrategies(c *gin.Context) {
	if err := ctrl.strategyService.ReloadAllStrategies(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Strategies reloaded successfully"})
}

// getAllStrategiesAdmin retrieves all strategies including internal details.
// @Summary      Get all strategies (Admin)
// @Description  Returns all trading strategies with full administrative details
// @Tags         Strategy
// @Produce      json
// @Success      200  {array}  model.StrategyDto
// @Router       /strategy/admin [get]
func (ctrl *StrategyController) getAllStrategiesAdmin(c *gin.Context) {
	strategies := ctrl.strategyService.GetAllStrategiesAdmin()
	if strategies == nil {
		c.JSON(http.StatusOK, []model.StrategyDto{})
		return
	}
	c.JSON(http.StatusOK, strategies)
}
