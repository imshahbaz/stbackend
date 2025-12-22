package controller

import (
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
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

// RegisterRoutes sets up the route group (Equivalent to @RequestMapping("/api/chartink"))
func (ctrl *ChartInkController) RegisterRoutes(router *gin.RouterGroup) {
	chartinkGroup := router.Group("/chartink")
	{
		chartinkGroup.GET("/fetch", ctrl.fetchData)
		chartinkGroup.GET("/fetchWithMargin", ctrl.fetchWithMargin)
	}
}

// fetchData replaces GET /fetch
func (ctrl *ChartInkController) fetchData(c *gin.Context) {
	strategyName := c.Query("strategy")

	// Equivalent to StrategyService.strategyMap.get(strategy)
	strategyDto, exists := ctrl.findStrategy(strategyName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	data, err := ctrl.chartInkService.FetchData(strategyDto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// fetchWithMargin replaces GET /fetchWithMargin
func (ctrl *ChartInkController) fetchWithMargin(c *gin.Context) {
	strategyName := c.Query("strategy")

	strategyDto, exists := ctrl.findStrategy(strategyName)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Strategy not found"})
		return
	}

	data, err := ctrl.chartInkService.FetchWithMargin(strategyDto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, data)
}

// Helper to bridge the logic from your StrategyService map
func (ctrl *ChartInkController) findStrategy(name string) (model.StrategyDto, bool) {
	strategies := ctrl.strategyService.GetAllStrategies()
	for _, s := range strategies {
		if s.Name == name {
			return s, true
		}
	}
	return model.StrategyDto{}, false
}
