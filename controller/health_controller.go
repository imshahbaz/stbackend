package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// RegisterRoutes sets up the health check endpoint.
// In our main router, this is typically attached to the root or /api group.
func (ctrl *HealthController) RegisterRoutes(router *gin.RouterGroup) {
	// These resolve to [base_path]/health
	router.GET("/health", ctrl.healthCheck)
	router.HEAD("/health", ctrl.healthCheck)
}

// healthCheck returns the current status of the server.
// @Summary      System Health Check
// @Description  Confirm that the server is up and running. Used by Load Balancers and Uptime Monitors.
// @Tags         System
// @Produce      json
// @Success      200  {object}  map[string]string "Status OK"
// @Router       /health [get]
// @Router       /health [head]
func (ctrl *HealthController) healthCheck(c *gin.Context) {
	// For GET requests, returning a small JSON body is better for debugging.
	// For HEAD requests, Gin will automatically handle the headers only.
	if c.Request.Method == http.MethodGet {
		c.JSON(http.StatusOK, gin.H{"status": "UP"})
		return
	}

	c.Status(http.StatusOK)
}
