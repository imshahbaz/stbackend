package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HealthController struct{}

func NewHealthController() *HealthController {
	return &HealthController{}
}

// RegisterRoutes sets up the health check endpoint
func (ctrl *HealthController) RegisterRoutes(router *gin.Engine) {
	// Gin allows mapping multiple methods to the same handler
	router.GET("/health", ctrl.healthCheck)
	router.HEAD("/health", ctrl.healthCheck)
}

func (ctrl *HealthController) healthCheck(c *gin.Context) {
	// Equivalent to ResponseEntity.ok().build()
	// Returns 200 OK with no body
	c.Status(http.StatusOK)
}
