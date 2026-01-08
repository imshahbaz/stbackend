package controller

import (
	"context"
	"net/http"

	"backend/model"
	"backend/service"

	"github.com/danielgtaylor/huma/v2"
)

type EmailController struct {
	emailService service.EmailService
}

func NewEmailController(es service.EmailService) *EmailController {
	return &EmailController{
		emailService: es,
	}
}

func (ctrl *EmailController) RegisterRoutes(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "send-email",
		Method:      http.MethodPost,
		Path:        "/api/email/send",
		Summary:     "Send an email",
		Description: "Sends a transactional email using the Brevo API provider",
		Tags:        []string{"Email"},
	}, ctrl.sendEmail)
}

func (ctrl *EmailController) sendEmail(ctx context.Context, input *model.SendEmailRequest) (*model.DefaultResponse, error) {
	if err := ctrl.emailService.SendEmail(ctx, input.Body); err != nil {
		return NewErrorResponse("Failed to send email: " + err.Error()), nil
	}

	return NewResponse(nil, "Email sent successfully"), nil
}
