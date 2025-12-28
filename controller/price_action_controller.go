package controller

import (
	"backend/cache"
	"backend/model"
	"backend/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type PriceActionController struct {
	priceActionService service.PriceActionService
}

func NewPriceActionController(s service.PriceActionService) *PriceActionController {
	return &PriceActionController{
		priceActionService: s,
	}
}

func (ctrl *PriceActionController) RegisterRoutes(router *gin.RouterGroup) {
	priceActionGrp := router.Group("/price-action")
	obGroup := priceActionGrp.Group("/ob")
	{
		obGroup.POST("", ctrl.SaveOrderBlock)
		obGroup.DELETE("", ctrl.DeleteOrderBlock)
		obGroup.POST("/check", ctrl.CheckOBMitigation)
		obGroup.GET("/mitigation", ctrl.GetOBMitigation)
	}
}

// SaveOrderBlock godoc
// @Summary      Save or Update an Order Block
// @Description  Creates a new order block or updates an existing one for a specific symbol and date.
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
