package controller

import (
	"net/http"

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
		strategyGroup.POST("", ctrl.createStrategy)
		strategyGroup.PUT("", ctrl.updateStrategy)
		strategyGroup.DELETE("/:id", ctrl.deleteStrategy) // Path variable
		strategyGroup.POST("/reload", ctrl.reloadAllStrategies)
	}
}

func (ctrl *StrategyController) getAllStrategies(c *gin.Context) {
	strategies := ctrl.strategyService.GetAllStrategies()
	c.JSON(http.StatusOK, strategies)
}

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

func (ctrl *StrategyController) deleteStrategy(c *gin.Context) {
	id := c.Param("id")
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

func (ctrl *StrategyController) reloadAllStrategies(c *gin.Context) {
	err := ctrl.strategyService.ReloadAllStrategies(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}
