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

// RegisterRoutes sets up the route group for email operations.
func (ctrl *EmailController) RegisterRoutes(router *gin.RouterGroup) {
	emailGroup := router.Group("/email")
	{
		// Mapping to POST /api/email/send
		emailGroup.POST("/send", ctrl.sendEmail)
	}
}

// sendEmail handles email dispatching via Brevo.
// @Summary      Send an email
// @Description  Sends a transactional email using the Brevo API provider
// @Tags         Email
// @Accept       json
// @Produce      json
// @Param        request  body      model.BrevoEmailRequest  true  "Email content and recipients"
// @Success      200      {object}  map[string]string "Email sent successfully"
// @Failure      400      {object}  map[string]string "Invalid request body"
// @Failure      500      {object}  map[string]string "Failed to send email"
// @Router       /email/send [post]
func (ctrl *EmailController) sendEmail(c *gin.Context) {
	var request model.BrevoEmailRequest

	// 1. Bind JSON body
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request payload"})
		return
	}

	// 2. Call service layer with Request Context
	// This ensures that if the client disconnects, the email process can be cancelled
	if err := ctrl.emailService.SendEmail(c.Request.Context(), request); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to send email",
			"message": err.Error(),
		})
		return
	}

	// 3. Return a consistent JSON response
	c.JSON(http.StatusOK, gin.H{"message": "Email sent successfully"})
}
