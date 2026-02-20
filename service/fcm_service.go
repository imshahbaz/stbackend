package service

import (
	"backend/config"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FcmService interface {
	SendNotification(ctx context.Context, token, title, body string, data map[string]string) error
}

type FcmServiceImpl struct {
	client *messaging.Client
}

func NewFcmService(cfgManager *config.ConfigManager) (FcmService, error) {
	cfg := cfgManager.GetConfig()
	if len(cfg.FcmConfig.ServiceAccount) == 0 {
		return nil, fmt.Errorf("FCM service account is empty")
	}

	serviceAccountBytes, err := json.Marshal(cfg.FcmConfig.ServiceAccount)
	if err != nil {
		return nil, fmt.Errorf("error marshaling service account: %v", err)
	}

	// Updated to the more specific option.WithAuthCredentialsJSON to address security risk and deprecation.
	opt := option.WithAuthCredentialsJSON(option.ServiceAccount, serviceAccountBytes)
	app, err := firebase.NewApp(context.Background(), nil, opt)
	if err != nil {
		return nil, fmt.Errorf("error initializing firebase app: %v", err)
	}

	client, err := app.Messaging(context.Background())
	if err != nil {
		return nil, fmt.Errorf("error getting messaging client: %v", err)
	}

	return &FcmServiceImpl{client: client}, nil
}

func (s *FcmServiceImpl) SendNotification(ctx context.Context, token, title, body string, data map[string]string) error {
	data["title"] = title
	data["body"] = body
	data["tag"] = strconv.FormatInt(time.Now().Unix(), 10)
	message := &messaging.Message{
		Data:  data,
		Token: token,
	}

	_, err := s.client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("error sending FCM message: %v", err)
	}

	return nil
}
