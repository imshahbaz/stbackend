package controller

import (
	"net/http"

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

// RegisterRoutes sets up the route group (Equivalent to @RequestMapping("/api/margin"))
func (ctrl *MarginController) RegisterRoutes(router *gin.RouterGroup) {
	marginGroup := router.Group("/margin")
	{
		marginGroup.GET("/all", ctrl.getAllMargins)
		marginGroup.GET("/:symbol", ctrl.getMargin) // Path variable syntax
		marginGroup.GET("/reload", ctrl.reloadAllMargins)
		marginGroup.POST("/load-from-csv", ctrl.loadFromCsv)
	}
}

// getAllMargins retrieves all stock margins
// @Summary      Get all margins
// @Description  Returns a list of all stock margins stored in the database
// @Tags         Margin
// @Produce      json
// @Success      200  {array}  model.Margin
// @Router       /margin/all [get]
func (ctrl *MarginController) getAllMargins(c *gin.Context) {
	margins := ctrl.marginService.GetAllMargins()
	c.JSON(http.StatusOK, margins)
}

// getMargin retrieves a single margin by symbol
// @Summary      Get margin by symbol
// @Description  Fetches the margin details for a specific stock symbol
// @Tags         Margin
// @Produce      json
// @Param        symbol  path      string  true  "Stock Symbol"  example(RELIANCE)
// @Success      200     {object}  model.Margin
// @Failure      404     {object}  map[string]string
// @Router       /margin/{symbol} [get]
func (ctrl *MarginController) getMargin(c *gin.Context) {
	// Equivalent to @PathVariable String symbol
	symbol := c.Param("symbol")

	margin, exists := ctrl.marginService.GetMargin(symbol)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Margin not found for symbol: " + symbol})
		return
	}
	c.JSON(http.StatusOK, margin)
}

// reloadAllMargins refreshes the margin cache from DB
// @Summary      Reload margins
// @Description  Forces a reload of all margins from the persistent database into the memory cache
// @Tags         Margin
// @Success      200
// @Failure      500     {object}  map[string]string
// @Router       /margin/reload [get]
func (ctrl *MarginController) reloadAllMargins(c *gin.Context) {
	err := ctrl.marginService.ReloadAllMargins(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusOK)
}

// loadFromCsv handles CSV file upload for margins
// @Summary      Upload Margin CSV
// @Description  Uploads a CSV file to bulk load or update margin data
// @Tags         Margin
// @Accept       multipart/form-data
// @Param        file  formData  file  true  "Margin CSV file"
// @Success      200
// @Failure      400   {object}  map[string]string
// @Failure      500   {object}  map[string]string
// @Router       /margin/load-from-csv [post]
func (ctrl *MarginController) loadFromCsv(c *gin.Context) {
	// 1. Get file from multipart form (Equivalent to @RequestParam MultipartFile)
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File is required"})
		return
	}

	// 2. Open the file stream
	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// 3. Call service with the reader
	err = ctrl.marginService.LoadFromCsv(c.Request.Context(), fileHeader.Filename, file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.Status(http.StatusOK)
}
