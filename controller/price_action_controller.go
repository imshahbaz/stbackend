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
	pa := router.Group("/price-action")
	{
		pa.POST("/automate", ctrl.AutomateOrderBlock)
		pa.GET("/:symbol", ctrl.GetPABySymbol)
	}

	// OB Routes
	ob := pa.Group("/ob")
	{
		ob.POST("/check", ctrl.CheckOBMitigation)
		ob.GET("/mitigation", ctrl.GetOBMitigation)

		admin := ob.Group("")
		admin.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
		{
			admin.POST("", ctrl.SaveOrderBlock)
			admin.PATCH("", ctrl.UpdateOrderBlock)
			admin.DELETE("", ctrl.DeleteOrderBlock)
		}
	}

	// FVG Routes
	fvg := pa.Group("/fvg")
	{
		fvg.POST("/check", ctrl.CheckFvgMitigation)
		fvg.GET("/mitigation", ctrl.GetFvgMitigation)

		admin := fvg.Group("")
		admin.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
		{
			admin.POST("", ctrl.SaveFvg)
			admin.PATCH("", ctrl.UpdateFvg)
			admin.DELETE("", ctrl.DeleteFvg)
		}
	}
}

// --- Helper Methods ---

func (ctrl *PriceActionController) handleCachedMitigation(ctx *gin.Context, cacheKey string, fallback func(*gin.Context)) {
	if val, exists := cache.PriceActionCache.Get(cacheKey); exists {
		ctx.JSON(http.StatusOK, model.Response{
			Success: true,
			Message: "Fetch success",
			Data:    val,
		})
		return
	}
	fallback(ctx)
}

// --- Handlers ---

// GetPABySymbol godoc
// @Summary      Get Price Action by Symbol
// @Description  Retrieves the full list of price action data for a specific stock symbol.
// @Tags         PriceAction
// @Produce      json
// @Param        symbol  path      string  true  "Stock Symbol"
// @Success      200     {object}  model.Response
// @Router       /price-action/{symbol} [get]
func (ctrl *PriceActionController) GetPABySymbol(ctx *gin.Context) {
	ctrl.priceActionService.GetPABySymbol(ctx)
}

// AutomateOrderBlock godoc
// @Summary      Automate Order Block Discovery
// @Description  Triggers scanners to automatically find and save Order Blocks.
// @Tags         PriceAction
// @Produce      json
// @Success      200      {object}  model.Response
// @Router       /price-action/automate [post]
func (ctrl *PriceActionController) AutomateOrderBlock(ctx *gin.Context) {
	ctrl.priceActionService.AutomateOrderBlock(ctx)
}

// --- OB Handlers ---

// SaveOrderBlock godoc
// @Summary      Save an Order Block
// @Tags         PriceAction
// @Accept       json
// @Param        request body model.ObRequest true "Order Block Details"
// @Success      200 {object} model.Response
// @Router       /price-action/ob [post]
// @Security     BearerAuth
func (ctrl *PriceActionController) SaveOrderBlock(ctx *gin.Context) {
	ctrl.priceActionService.SaveOrderBlock(ctx)
}

// UpdateOrderBlock godoc
// @Summary      Update an Order Block
// @Tags         PriceAction
// @Accept       json
// @Param        request body model.ObRequest true "Update Details"
// @Success      200 {object} model.Response
// @Router       /price-action/ob [patch]
// @Security     BearerAuth
func (ctrl *PriceActionController) UpdateOrderBlock(ctx *gin.Context) {
	ctrl.priceActionService.UpdateOrderBlock(ctx)
}

// DeleteOrderBlock godoc
// @Summary      Delete an Order Block
// @Tags         PriceAction
// @Param        request body model.ObRequest true "Delete Details"
// @Success      200 {object} model.Response
// @Router       /price-action/ob [delete]
// @Security     BearerAuth
func (ctrl *PriceActionController) DeleteOrderBlock(ctx *gin.Context) {
	ctrl.priceActionService.DeleteOrderBlock(ctx)
}

// CheckOBMitigation godoc
// @Summary      Check OB Mitigations
// @Tags         PriceAction
// @Success      200 {object} model.Response
// @Router       /price-action/ob/check [post]
func (ctrl *PriceActionController) CheckOBMitigation(ctx *gin.Context) {
	ctrl.priceActionService.CheckOBMitigation(ctx)
}

// GetOBMitigation godoc
// @Summary      Get Cached OB Mitigations
// @Tags         PriceAction
// @Success      200 {object} model.Response
// @Router       /price-action/ob/mitigation [get]
func (ctrl *PriceActionController) GetOBMitigation(ctx *gin.Context) {
	ctrl.handleCachedMitigation(ctx, "ObCache", ctrl.priceActionService.CheckOBMitigation)
}

// --- FVG Handlers ---

// SaveFvg godoc
// @Summary      Save Fair Value Gap
// @Tags         FVG
// @Accept       json
// @Param        request body model.ObRequest true "FVG Details"
// @Success      200 {object} model.Response
// @Router       /price-action/fvg [post]
func (ctrl *PriceActionController) SaveFvg(ctx *gin.Context) {
	ctrl.priceActionService.SaveFvg(ctx)
}

// UpdateFvg godoc
// @Summary      Update Fair Value Gap
// @Tags         FVG
// @Accept       json
// @Param        request body model.ObRequest true "Update Details"
// @Success      200 {object} model.Response
// @Router       /price-action/fvg [patch]
func (ctrl *PriceActionController) UpdateFvg(ctx *gin.Context) {
	ctrl.priceActionService.UpdateFvg(ctx)
}

// DeleteFvg godoc
// @Summary      Delete Fair Value Gap
// @Tags         FVG
// @Param        request body model.ObRequest true "Delete Details"
// @Success      200 {object} model.Response
// @Router       /price-action/fvg [delete]
func (ctrl *PriceActionController) DeleteFvg(ctx *gin.Context) {
	ctrl.priceActionService.DeleteFvg(ctx)
}

// CheckFvgMitigation godoc
// @Summary      Check Fvg Mitigations
// @Tags         FVG
// @Success      200 {object} model.Response
// @Router       /price-action/fvg/check [post]
func (ctrl *PriceActionController) CheckFvgMitigation(ctx *gin.Context) {
	ctrl.priceActionService.CheckFvgMitigation(ctx)
}

// GetFvgMitigation godoc
// @Summary      Get Cached Fvg Mitigations
// @Tags         FVG
// @Success      200 {object} model.Response
// @Router       /price-action/fvg/mitigation [get]
func (ctrl *PriceActionController) GetFvgMitigation(ctx *gin.Context) {
	ctrl.handleCachedMitigation(ctx, "FvgCache", ctrl.priceActionService.CheckFvgMitigation)
}
