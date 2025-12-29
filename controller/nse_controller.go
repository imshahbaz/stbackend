package controller

import (
	"net/http"

	"backend/model"
	"backend/service"

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

// RegisterRoutes sets up the route group for NSE data retrieval.
func (ctrl *NseController) RegisterRoutes(router *gin.RouterGroup) {
	nseGroup := router.Group("/nse")
	{
		nseGroup.GET("/history", ctrl.GetStockHistory)
		nseGroup.GET("/heatmap", ctrl.GetHeatMap)
		nseGroup.GET("/allindices", ctrl.GetAllIndices)
	}
}

// GetStockHistory handles historical data requests.
// @Summary      Get Historical Stock Data
// @Description  Fetches stock history for a specific symbol. Utilizes a 1-hour time cache.
// @Tags         Stocks
// @Accept       json
// @Produce      json
// @Param        symbol  query     string  true  "Stock Symbol (e.g. RELIANCE)"
// @Success      200     {object}  model.Response{data=[]model.NSEHistoricalData}
// @Failure      400     {object}  model.Response
// @Failure      500     {object}  model.Response
// @Router       /nse/history [get]
func (ctrl *NseController) GetStockHistory(c *gin.Context) {
	symbol := c.Query("symbol")
	if symbol == "" {
		c.JSON(http.StatusBadRequest, model.Response{
			Success: false,
			Message: "Symbol parameter is required",
		})
		return
	}

	data, err := ctrl.nseService.FetchStockData(symbol)
	if err != nil {
		ctrl.handleError(c, "Failed to get history", err)
		return
	}

	ctrl.handleSuccess(c, "Fetch Success", data)
}

// GetHeatMap fetches sectoral performance.
// @Summary      Get NSE Sectoral Heatmap
// @Description  Fetches latest sectoral data. Uses warmup time-cache strategy to avoid NSE rate limits.
// @Tags         Stocks
// @Produce      json
// @Success      200     {object}  model.Response{data=[]model.SectorData}
// @Failure      500     {object}  model.Response
// @Router       /nse/heatmap [get]
func (ctrl *NseController) GetHeatMap(c *gin.Context) {
	data, err := ctrl.nseService.FetchHeatMap()
	if err != nil {
		ctrl.handleError(c, "Failed to get heat map", err)
		return
	}

	ctrl.handleSuccess(c, "Fetch Success", data)
}

// GetAllIndices fetches all indices performance data.
// @Summary      Get All NSE Indices
// @Description  Fetches latest performance data for all indices using the warmup strategy.
// @Tags         Stocks
// @Produce      json
// @Success      200     {object}  model.Response{data=[]model.AllIndicesResponse}
// @Failure      500     {object}  model.Response
// @Router       /nse/allindices [get]
func (ctrl *NseController) GetAllIndices(c *gin.Context) {
	data, err := ctrl.nseService.FetchAllIndices()
	if err != nil {
		ctrl.handleError(c, "Failed to get all indices data", err)
		return
	}

	ctrl.handleSuccess(c, "Fetch Success", data)
}

// --- Internal Response Helpers ---

func (ctrl *NseController) handleSuccess(c *gin.Context, message string, data any) {
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func (ctrl *NseController) handleError(c *gin.Context, message string, err error) {
	c.JSON(http.StatusInternalServerError, model.Response{
		Success: false,
		Message: message,
		Error:   err.Error(),
	})
}
