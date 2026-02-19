package util

import (
	"context"

	"github.com/rs/zerolog/log"
)

type NotificationRequest struct {
	UserID int64             `json:"userId"`
	Title  string            `json:"title"`
	Body   string            `json:"body"`
	Data   map[string]string `json:"data"`
}

var NotifChan = make(chan NotificationRequest, 100)

type UserFinder func(ctx context.Context, userId int64) (string, error)
type Notifier func(ctx context.Context, token, title, body string, data map[string]string) error

func StartNotificationWorker(userFinder UserFinder, notifier Notifier) {
	log.Info().Msg("Starting FCM Notification Worker")
	go func() {
		for req := range NotifChan {
			token, err := userFinder(context.Background(), req.UserID)
			if err != nil {
				continue
			}

			if token == "" {
				continue
			}

			if err := notifier(context.Background(), token, req.Title, req.Body, req.Data); err != nil {
				continue
			}
		}
	}()
}
