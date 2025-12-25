package controller

import (
	"backend/model"
	"backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type NseController struct {
	nseService service.NseService
}

func NewNseController(ns service.NseService) *NseController {
	return &NseController{
		nseService: ns,
	}
}

func (ctrl *NseController) RegisterRoutes(router *gin.RouterGroup) {
	nseGroup := router.Group("/nse")
	{
		nseGroup.GET("/history", ctrl.GetStockHistory)
	}
}

// GetStockHistory handles historical data requests
// @Summary Get Historical Stock Data
// @Description Fetches stock history for a specific symbol (e.g., BEL). Data is cached for 1 hour.
// @Tags Stocks
// @Accept json
// @Produce json
// @Param symbol query string true "Stock Symbol (e.g. BEL)"
// @Success 200 {object} model.Response{data=[]model.NSEHistoricalData} "Fetch Success"
// @Failure 400 {object} model.Response "Invalid Request"
// @Failure 401 {object} model.Response "Unauthorized"
// @Failure 500 {object} model.Response "Internal Server Error"
// @Router /nse/history [get]
func (ctrl *NseController) GetStockHistory(c *gin.Context) {
	symbol := c.Query("symbol")

	data, err := ctrl.nseService.FetchStockData(symbol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Message: "Failed to get history",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: "Fetch Success",
		Data:    data,
	})
}
