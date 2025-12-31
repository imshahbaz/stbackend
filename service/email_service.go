package service

import (
	"backend/client"
	"backend/config"
	"backend/model"
	"context"
)

type EmailService interface {
	SendEmail(ctx context.Context, request model.BrevoEmailRequest) error
}

type EmailServiceImpl struct {
	brevoClient *client.BrevoClient
	cfg         *config.ConfigManager
}

func NewEmailService(bc *client.BrevoClient, cfg *config.ConfigManager) EmailService {
	return &EmailServiceImpl{
		brevoClient: bc,
		cfg:         cfg,
	}
}

func (s *EmailServiceImpl) SendEmail(ctx context.Context, request model.BrevoEmailRequest) error {
	_, err := s.brevoClient.SendTransactionalEmail(ctx, s.cfg.GetConfig().BrevoApiKey, request)
	return err
}
