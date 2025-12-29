package controller

import (
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type MarginController struct {
	marginService service.MarginService
}

func NewMarginController(ms service.MarginService) *MarginController {
	return &MarginController{
		marginService: ms,
	}
}

// RegisterRoutes sets up the route group for margin management.
func (ctrl *MarginController) RegisterRoutes(router *gin.RouterGroup) {
	marginGroup := router.Group("/margin")
	{
		marginGroup.GET("/all", ctrl.getAllMargins)
		marginGroup.GET("/symbol/:symbol", ctrl.getMargin)
		marginGroup.POST("/reload", ctrl.reloadAllMargins) // Changed to POST for action
		marginGroup.POST("/load-from-csv", ctrl.loadFromCsv)
	}
}

// getAllMargins retrieves all stock margins.
// @Summary      Get all margins
// @Description  Returns a list of all stock margins from the local memory cache
// @Tags         Margin
// @Produce      json
// @Success      200  {array}  model.Margin
// @Router       /margin/all [get]
func (ctrl *MarginController) getAllMargins(c *gin.Context) {
	margins := ctrl.marginService.GetAllMargins()
	// Return empty array instead of nil if no margins exist
	if margins == nil {
		c.JSON(http.StatusOK, []model.Margin{})
		return
	}
	c.JSON(http.StatusOK, margins)
}

// getMargin retrieves a single margin by symbol.
// @Summary      Get margin by symbol
// @Description  Fetches the margin details for a specific stock symbol
// @Tags         Margin
// @Produce      json
// @Param        symbol  path      string  true  "Stock Symbol"  example(RELIANCE)
// @Success      200     {object}  model.Margin
// @Failure      404     {object}  map[string]string
// @Router       /margin/symbol/{symbol} [get]
func (ctrl *MarginController) getMargin(c *gin.Context) {
	symbol := c.Param("symbol")
	margin, exists := ctrl.marginService.GetMargin(symbol)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Margin not found for symbol: " + symbol})
		return
	}
	c.JSON(http.StatusOK, margin)
}

// reloadAllMargins refreshes the margin cache from DB.
// @Summary      Reload margins
// @Description  Forces a reload of all margins from MongoDB into the memory cache
// @Tags         Margin
// @Produce      json
// @Success      200     {object}  map[string]string
// @Failure      500     {object}  map[string]string
// @Router       /margin/reload [post]
func (ctrl *MarginController) reloadAllMargins(c *gin.Context) {
	if err := ctrl.marginService.ReloadAllMargins(c.Request.Context()); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reload margins: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Margins reloaded successfully"})
}

// loadFromCsv handles CSV file upload for margins.
// @Summary      Upload Margin CSV
// @Description  Uploads a CSV file to bulk load or update margin data
// @Tags         Margin
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "Margin CSV file"
// @Success      200   {object}  map[string]string
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /margin/load-from-csv [post]
func (ctrl *MarginController) loadFromCsv(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file is required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not open uploaded file"})
		return
	}
	defer file.Close()

	if err = ctrl.marginService.LoadFromCsv(c.Request.Context(), fileHeader.Filename, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "CSV data processed successfully"})
}
