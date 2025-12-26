package service

import (
	"backend/client"
	"backend/config"
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
	cfg         *config.ConfigManager
}

// NewEmailService acts as the @RequiredArgsConstructor
func NewEmailService(bc *client.BrevoClient, cfg *config.ConfigManager) EmailService {
	return &EmailServiceImpl{
		brevoClient: bc,
		cfg:         cfg,
	}
}

// SendEmail replaces the @Override method
func (s *EmailServiceImpl) SendEmail(ctx context.Context, request model.BrevoEmailRequest) error {
	// We use the BrevoClient we created in the previous step
	_, err := s.brevoClient.SendTransactionalEmail(ctx, s.cfg.GetConfig().BrevoApiKey, request)
	return err
}
