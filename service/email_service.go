package service

import (
	"backend/client" // Adjust to your module path
	"backend/model"
	"context"
)

// 1. Interface Definition
type EmailService interface {
	SendEmail(ctx context.Context, request model.BrevoEmailRequest) error
}

// 2. Implementation Struct
type EmailServiceImpl struct {
	brevoClient *client.BrevoClient
	apiKey      string
}

// NewEmailService acts as the @RequiredArgsConstructor
func NewEmailService(bc *client.BrevoClient, apiKey string) EmailService {
	return &EmailServiceImpl{
		brevoClient: bc,
		apiKey:      apiKey,
	}
}

// SendEmail replaces the @Override method
func (s *EmailServiceImpl) SendEmail(ctx context.Context, request model.BrevoEmailRequest) error {
	// We use the BrevoClient we created in the previous step
	_, err := s.brevoClient.SendTransactionalEmail(ctx, s.apiKey, request)
	return err
}
