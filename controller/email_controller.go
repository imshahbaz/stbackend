package controller

import (
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/gin-gonic/gin"
)

type EmailController struct {
	emailService service.EmailService
}

func NewEmailController(es service.EmailService) *EmailController {
	return &EmailController{
		emailService: es,
	}
}

// RegisterRoutes sets up the route group (Equivalent to @RequestMapping("/api/email"))
func (ctrl *EmailController) RegisterRoutes(router *gin.RouterGroup) {
	emailGroup := router.Group("/email")
	{
		// Mapping to POST /api/email/send
		emailGroup.POST("/send", ctrl.sendEmail)
	}
}

func (ctrl *EmailController) sendEmail(c *gin.Context) {
	var request model.BrevoEmailRequest

	// 1. Bind JSON body to struct (Equivalent to @RequestBody)
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 2. Call service layer
	// We pass c.Request.Context() to handle cancellation/timeouts
	err := ctrl.emailService.SendEmail(c.Request.Context(), request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to send email"})
		return
	}

	// 3. Return 200 OK (void in Java returns 200 by default)
	c.Status(http.StatusOK)
}
