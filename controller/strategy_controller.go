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
}

func NewStrategyController(ss service.StrategyService) *StrategyController {
	return &StrategyController{
		strategyService: ss,
	}
}

// RegisterRoutes maps endpoints to the /api/strategy group
func (ctrl *StrategyController) RegisterRoutes(router *gin.RouterGroup) {
	strategyGroup := router.Group("/strategy")
	{
		strategyGroup.GET("", ctrl.getAllStrategies)
	}

	protectedGroup := strategyGroup.Group("")
	protectedGroup.Use(middleware.AuthMiddleware(), middleware.AdminOnly())
	{
		protectedGroup.POST("", ctrl.createStrategy)
		protectedGroup.PUT("", ctrl.updateStrategy)
		protectedGroup.DELETE("", ctrl.deleteStrategy)
		protectedGroup.POST("/reload", ctrl.reloadAllStrategies)
		protectedGroup.GET("/admin", ctrl.getAllStrategiesAdmin)
	}

}

// getAllStrategies retrieves all trading strategies
// @Summary      Get all strategies
// @Description  Returns a list of all configured trading strategies
// @Tags         Strategy
// @Produce      json
// @Success      200  {array}  model.StrategyDto
// @Router       /strategy [get]
func (ctrl *StrategyController) getAllStrategies(c *gin.Context) {
	strategies := ctrl.strategyService.GetAllStrategies()
	c.JSON(http.StatusOK, strategies)
}

// createStrategy adds a new strategy
// @Summary      Create a strategy
// @Description  Saves a new trading strategy configuration to the database
// @Tags         Strategy
// @Accept       json
// @Produce      json
// @Param        request  body      model.StrategyDto  true  "Strategy Details"
// @Success      201      {object}  model.Strategy
// @Failure      400      {object}  map[string]string
// @Router       /strategy [post]
func (ctrl *StrategyController) createStrategy(c *gin.Context) {
	var request model.StrategyDto
	// ShouldBindJSON validates against `binding:"required"` tags in the struct
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := ctrl.strategyService.CreateStrategy(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, res)
}

// updateStrategy modifies an existing strategy
// @Summary      Update a strategy
// @Description  Updates the configuration of an existing strategy by name
// @Tags         Strategy
// @Accept       json
// @Produce      json
// @Param        request  body      model.StrategyDto  true  "Updated Strategy Details"
// @Success      200      {object}  model.Strategy
// @Failure      400      {object}  map[string]string
// @Router       /strategy [put]
func (ctrl *StrategyController) updateStrategy(c *gin.Context) {
	var request model.StrategyDto
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := ctrl.strategyService.UpdateStrategy(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// deleteStrategy removes a strategy
// @Summary      Delete a strategy
// @Description  Removes a strategy from the system using its ID (Name)
// @Tags         Strategy
// @Param        id   query     string  true  "Strategy ID (Name)"
// @Success      204  "No Content"
// @Failure      400  {object}  map[string]string
// @Failure      500  {object}  map[string]string
// @Router       /strategy [delete]
func (ctrl *StrategyController) deleteStrategy(c *gin.Context) {
	id := c.Query("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	err := ctrl.strategyService.DeleteStrategy(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}

// reloadAllStrategies refreshes cache from DB
// @Summary      Reload strategies
// @Description  Syncs the in-memory strategy cache with the MongoDB database
// @Tags         Strategy
// @Success      200
// @Router       /strategy/reload [post]
func (ctrl *StrategyController) reloadAllStrategies(c *gin.Context) {
	err := ctrl.strategyService.ReloadAllStrategies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// getAllStrategies retrieves all trading strategies
// @Summary      Get all strategies
// @Description  Returns a list of all configured trading strategies
// @Tags         Strategy
// @Produce      json
// @Success      200  {array}  model.StrategyDto
// @Router       /strategy/admin [get]
func (ctrl *StrategyController) getAllStrategiesAdmin(c *gin.Context) {
	strategies := ctrl.strategyService.GetAllStrategiesAdmin()
	c.JSON(http.StatusOK, strategies)
}
