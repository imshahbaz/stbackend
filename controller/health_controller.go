package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// RegisterRoutes sets up the health check endpoint under the /api group
// FIX: Changed *gin.Engine to *gin.RouterGroup
func (ctrl *HealthController) RegisterRoutes(router *gin.RouterGroup) {
	// These will now resolve to /api/health
	router.GET("/health", ctrl.healthCheck)
	router.HEAD("/health", ctrl.healthCheck)
}

// healthCheck returns the current status of the server
// @Summary      System Health Check
// @Description  Confirm that the server is up and running. Returns a 200 status code with no body.
// @Tags         System
// @Produce      plain
// @Success      200  {string}  string "OK"
// @Router       /health [get]
// @Router       /health [head]
func (ctrl *HealthController) healthCheck(c *gin.Context) {
	c.Status(http.StatusOK)
}
