package controller

import (
	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PriceActionController struct {
	priceActionService service.PriceActionService
	isProduction       bool
}

func NewPriceActionController(s service.PriceActionService, isProduction bool) *PriceActionController {
	return &PriceActionController{
		priceActionService: s,
		isProduction:       isProduction,
	}
}

func (ctrl *PriceActionController) RegisterRoutes(router *gin.RouterGroup) {
	priceActionGrp := router.Group("/price-action")
	obGroup := priceActionGrp.Group("/ob")
	{
		obGroup.POST("/check", ctrl.CheckOBMitigation)
		obGroup.GET("/mitigation", ctrl.GetOBMitigation)
		obGroup.POST("/automate", ctrl.AutomateOrderBlock)
	}

	protectedGrp := obGroup.Group("")
	protectedGrp.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
	{
		protectedGrp.GET("/:symbol", ctrl.GetObBySymbol)
		protectedGrp.PATCH("", ctrl.UpdateOrderBlock)
		protectedGrp.POST("", ctrl.SaveOrderBlock)
		protectedGrp.DELETE("", ctrl.DeleteOrderBlock)
	}
}

// SaveOrderBlock godoc
// @Summary      Save an Order Block
// @Description  Creates a new order block for a specific symbol and date.
// @Tags         PriceAction
// @Accept       json
// @Produce      json
// @Param        request body model.ObRequest true "Order Block Details"
// @Success      200 {object} model.Response
// @Failure      400 {object} model.Response
// @Failure      500 {object} model.Response
// @Router       /price-action/ob [post]
func (c *PriceActionController) SaveOrderBlock(ctx *gin.Context) {
	c.priceActionService.SaveOrderBlock(ctx)
}

// DeleteOrderBlock godoc
// @Summary      Delete an Order Block
// @Description  Removes a specific order block entry from a stock's record based on symbol and date.
// @Tags         PriceAction
// @Accept       json
// @Produce      json
// @Param        request body model.ObRequest true "Symbol and Date of block to delete"
// @Success      200 {object} model.Response
// @Failure      400 {object} model.Response
// @Failure      500 {object} model.Response
// @Router       /price-action/ob [delete]
func (c *PriceActionController) DeleteOrderBlock(ctx *gin.Context) {
	c.priceActionService.DeleteOrderBlock(ctx)
}

// CheckOBMitigation godoc
// @Summary      Check and Refresh Order Block Mitigations
// @Description  Fetches strategy symbols, checks them against NSE live data, identifies non-mitigated blocks, and updates the cache.
// @Tags         PriceAction
// @Produce      json
// @Success      200 {object} model.Response{data=[]model.ObResponse}
// @Failure      500 {object} model.Response
// @Router       /price-action/ob/check [POST]
func (c *PriceActionController) CheckOBMitigation(ctx *gin.Context) {
	c.priceActionService.CheckOBMitigation(ctx)
}

// CheckOBMitigation godoc
// @Summary      Check and Refresh Order Block Mitigations
// @Description  Fetches strategy symbols, checks them against NSE live data, identifies non-mitigated blocks, and updates the cache.
// @Tags         PriceAction
// @Produce      json
// @Success      200 {object} model.Response{data=[]model.ObResponse}
// @Failure      500 {object} model.Response
// @Router       /price-action/ob/mitigation [get]
func (c *PriceActionController) GetOBMitigation(ctx *gin.Context) {
	val, exists := cache.PriceActionCache.Get("ObCache")
	if exists {
		resp := val.([]model.ObResponse)
		ctx.JSON(http.StatusOK, model.Response{
			Success: true,
			Message: "Order block fetch success",
			Data:    resp,
		})
		return
	}
	c.priceActionService.CheckOBMitigation(ctx)
}

// GetObBySymbol godoc
// @Summary      Get Order Blocks by Symbol
// @Description  Retrieves the full list of order blocks for a specific stock symbol from the MongoDB cache.
// @Tags         PriceAction
// @Produce      json
// @Param        symbol  path      string  true  "Stock Symbol (e.g., RELIANCE)"
// @Success      200     {object}  model.Response{data=model.StockRecord} "Successfully retrieved order blocks"
// @Failure      400     {object}  model.Response "Invalid symbol provided"
// @Failure      404     {object}  model.Response "Stock symbol not found in cache"
// @Router       /price-action/ob/{symbol} [get]
func (c *PriceActionController) GetObBySymbol(ctx *gin.Context) {
	c.priceActionService.GetPABySymbol(ctx)
}

// UpdateOrderBlock godoc
// @Summary      Update an Order Block
// @Description  Updates an existing one for a specific symbol and date.
// @Tags         PriceAction
// @Accept       json
// @Produce      json
// @Param        request body model.ObRequest true "Order Block Details"
// @Success      200 {object} model.Response
// @Failure      400 {object} model.Response
// @Failure      500 {object} model.Response
// @Router       /price-action/ob [patch]
func (c *PriceActionController) UpdateOrderBlock(ctx *gin.Context) {
	c.priceActionService.UpdateOrderBlock(ctx)
}

// AutomateOrderBlock godoc
// @Summary      Automate Order Block Discovery
// @Description  Fetches stocks based on the "BULLISH CLOSE 200" strategy, retrieves historical data from NSE (utilizing time cache), and persists Order Blocks.
// @Tags         Price Action
// @Accept       json
// @Produce      json
// @Success      200  {object}  model.Response{message=string}
// @Failure      401  {object}  model.Response{error=string}
// @Failure      500  {object}  model.Response{error=string}
// @Router       /price-action/ob/automate [post]
func (c *PriceActionController) AutomateOrderBlock(ctx *gin.Context) {
	c.priceActionService.AutomateOrderBlock(ctx)
}
