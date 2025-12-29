package controller

import (
	"context"
	"net/http"

	"backend/cache"
	"backend/middleware"
	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type PriceActionController struct {
	paService    service.PriceActionService
	isProduction bool
}

func NewPriceActionController(s service.PriceActionService, isProd bool) *PriceActionController {
	return &PriceActionController{paService: s, isProduction: isProd}
}

func (ctrl *PriceActionController) RegisterRoutes(router *gin.RouterGroup) {
	pa := router.Group("/price-action")
	{
		pa.POST("/automate", ctrl.TriggerAutomation)
		pa.GET("/:symbol", ctrl.GetPABySymbol)

		// Order Block Group
		ob := pa.Group("/ob")
		{
			ob.POST("/check", ctrl.CheckOBMitigation)
			ob.GET("/mitigation", ctrl.GetOBMitigation)
			ob.POST("/old/:stopDate", ctrl.AddOlderObController)
			admin := ob.Group("")
			admin.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
			{
				admin.POST("", ctrl.SaveOrderBlock)
				admin.PATCH("", ctrl.UpdateOrderBlock)
				admin.DELETE("", ctrl.DeleteOrderBlock)
			}
		}

		// FVG Group
		fvg := pa.Group("/fvg")
		{
			fvg.POST("/check", ctrl.CheckFvgMitigation)
			fvg.GET("/mitigation", ctrl.GetFvgMitigation)
			fvg.POST("/old/:stopDate", ctrl.AddOlderFvgController)
			admin := fvg.Group("")
			admin.Use(middleware.AuthMiddleware(ctrl.isProduction), middleware.AdminOnly())
			{
				admin.POST("", ctrl.SaveFvg)
				admin.PATCH("", ctrl.UpdateFvg)
				admin.DELETE("", ctrl.DeleteFvg)
			}
		}
	}
}

// --- General Handlers ---

// TriggerAutomation godoc
// @Summary      Trigger PA Automation
// @Tags         PriceAction
// @Success      202      {object}  model.Response
// @Router       /price-action/automate [post]
func (ctrl *PriceActionController) TriggerAutomation(c *gin.Context) {
	bgCtx := context.Background()
	go func() {
		_ = ctrl.paService.AutomateOrderBlock(bgCtx)
		_ = ctrl.paService.AutomateFvg(bgCtx)
	}()
	c.JSON(http.StatusAccepted, model.Response{Success: true, Message: "Scanning started"})
}

// GetPABySymbol godoc
// @Summary      Get PA by Symbol
// @Tags         PriceAction
// @Param        symbol   path      string  true  "Symbol"
// @Success      200      {object}  model.Response
// @Router       /price-action/{symbol} [get]
func (ctrl *PriceActionController) GetPABySymbol(c *gin.Context) {
	data, err := ctrl.paService.GetPABySymbol(c.Request.Context(), c.Param("symbol"))
	ctrl.respond(c, data, err)
}

// --- OB Handlers ---

// SaveOrderBlock godoc
// @Summary      Save OB
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "OB Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/ob [post]
func (ctrl *PriceActionController) SaveOrderBlock(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Success: false, Error: "Invalid body"})
		return
	}
	err := ctrl.paService.SaveOrderBlock(c.Request.Context(), req)
	ctrl.respond(c, nil, err)
}

// UpdateOrderBlock godoc
// @Summary      Update OB
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "Update Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/ob [patch]
func (ctrl *PriceActionController) UpdateOrderBlock(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Success: false, Error: "Invalid body"})
		return
	}
	err := ctrl.paService.UpdateOrderBlock(c.Request.Context(), req)
	ctrl.respond(c, nil, err)
}

// DeleteOrderBlock godoc
// @Summary      Delete OB
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "Delete Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/ob [delete]
func (ctrl *PriceActionController) DeleteOrderBlock(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
	err := ctrl.paService.DeleteOrderBlock(c.Request.Context(), req.Symbol, req.Date)
	ctrl.respond(c, nil, err)
}

// GetOBMitigation godoc
// @Summary      Get Cached OB Mitigations
// @Tags         PriceAction
// @Router       /price-action/ob/mitigation [get]
func (ctrl *PriceActionController) GetOBMitigation(c *gin.Context) {
	if val, exists := cache.PriceActionCache.Get("ObCache"); exists {
		c.JSON(http.StatusOK, model.Response{Success: true, Data: val})
		return
	}
	data, err := ctrl.paService.CheckOBMitigation(c.Request.Context())
	ctrl.respond(c, data, err)
}

// CheckOBMitigation godoc
// @Summary      Force Refresh OB Mitigations
// @Description  Triggers a fresh scan of all stocks against saved Order Blocks to find active mitigations.
// @Tags         PriceAction
// @Produce      json
// @Success      200      {object}  model.Response
// @Failure      500      {object}  model.Response
// @Router       /price-action/ob/check [post]
func (ctrl *PriceActionController) CheckOBMitigation(c *gin.Context) {
	data, err := ctrl.paService.CheckOBMitigation(c.Request.Context())
	ctrl.respond(c, data, err)
}

// --- FVG Handlers ---

// SaveFvg godoc
// @Summary      Save FVG
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "FVG Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/fvg [post]
func (ctrl *PriceActionController) SaveFvg(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Success: false, Error: "Invalid body"})
		return
	}
	err := ctrl.paService.SaveFvg(c.Request.Context(), req)
	ctrl.respond(c, nil, err)
}

// UpdateFvg godoc
// @Summary      Update FVG
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "Update Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/fvg [patch]
func (ctrl *PriceActionController) UpdateFvg(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.Response{Success: false, Error: "Invalid body"})
		return
	}
	err := ctrl.paService.UpdateFvg(c.Request.Context(), req)
	ctrl.respond(c, nil, err)
}

// DeleteFvg godoc
// @Summary      Delete FVG
// @Tags         PriceAction (Admin)
// @Param        request  body      model.ObRequest  true  "Delete Details"
// @Success      200      {object}  model.Response
// @Router       /price-action/fvg [delete]
func (ctrl *PriceActionController) DeleteFvg(c *gin.Context) {
	var req model.ObRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return
	}
	err := ctrl.paService.DeleteFvg(c.Request.Context(), req.Symbol, req.Date)
	ctrl.respond(c, nil, err)
}

// GetFvgMitigation godoc
// @Summary      Get Cached FVG Mitigations
// @Tags         PriceAction
// @Router       /price-action/fvg/mitigation [get]
func (ctrl *PriceActionController) GetFvgMitigation(c *gin.Context) {
	if val, exists := cache.PriceActionCache.Get("FvgCache"); exists {
		c.JSON(http.StatusOK, model.Response{Success: true, Data: val})
		return
	}
	data, err := ctrl.paService.CheckFvgMitigation(c.Request.Context())
	ctrl.respond(c, data, err)
}

// CheckFvgMitigation godoc
// @Summary      Force Refresh FVG Mitigations
// @Description  Triggers a fresh scan of all stocks against saved Fair Value Gaps (FVG) to identify active mitigations. This bypasses the cache and updates it.
// @Tags         PriceAction
// @Produce      json
// @Success      200      {object}  model.Response
// @Failure      500      {object}  model.Response
// @Router       /price-action/fvg/check [post]
func (ctrl *PriceActionController) CheckFvgMitigation(c *gin.Context) {
	data, err := ctrl.paService.CheckFvgMitigation(c.Request.Context())
	ctrl.respond(c, data, err)
}

// --- Helper ---

func (ctrl *PriceActionController) respond(c *gin.Context, data any, err error) {
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{Success: false, Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, model.Response{Success: true, Data: data})
}

// AddOlderObController
// @Summary      Process Historical Order Blocks
// @Description  Upload a CSV to find and save Order Blocks from a specific stop date backwards.
// @Tags         PriceAction
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData  file    true  "CSV File"
// @Param        stopDate  path      string  true  "Stop Date (YYYY-MM-DD)"
// @Success      200       {object}  map[string]string
// @Router       /price-action/ob/old/{stopDate} [post]
func (pc *PriceActionController) AddOlderObController(c *gin.Context) {
	stopDate := c.Param("stopDate")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Using your service logic
	pc.paService.AddOlderOb(c.Request.Context(), fileHeader.Filename, file, stopDate)

	c.JSON(http.StatusOK, gin.H{"message": "Order Block processing completed"})
}

// AddOlderFvgController
// @Summary      Process Historical Fair Value Gaps (FVG)
// @Description  Upload a CSV to find and save FVGs from a specific stop date backwards.
// @Tags         PriceAction
// @Accept       multipart/form-data
// @Produce      json
// @Param        file      formData  file    true  "CSV File"
// @Param        stopDate  path      string  true  "Stop Date (YYYY-MM-DD)"
// @Success      200       {object}  map[string]string
// @Router       /price-action/fvg/old/{stopDate} [post]
func (pc *PriceActionController) AddOlderFvgController(c *gin.Context) {
	stopDate := c.Param("stopDate")
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "CSV file required"})
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open file"})
		return
	}
	defer file.Close()

	// Using your service logic
	pc.paService.AddOlderFvg(c.Request.Context(), fileHeader.Filename, file, stopDate)

	c.JSON(http.StatusOK, gin.H{"message": "FVG processing completed"})
}
