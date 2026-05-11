package firebase

import (
	"context"
	"fmt"
	"strconv"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/option"
)

const firebaseScope = "https://www.googleapis.com/auth/firebase.messaging"

type Sender struct {
	client *messaging.Client
}

func NewSender(ctx context.Context, credentialsJSON string) (*Sender, error) {
	var opts []option.ClientOption
	if credentialsJSON != "" {
		creds, err := google.CredentialsFromJSON(ctx, []byte(credentialsJSON), firebaseScope)
		if err != nil {
			return nil, fmt.Errorf("parse firebase credentials: %w", err)
		}
		opts = append(opts, option.WithCredentials(creds))
	}
	// без опций firebase.NewApp использует GOOGLE_APPLICATION_CREDENTIALS из env

	app, err := firebase.NewApp(ctx, nil, opts...)
	if err != nil {
		return nil, fmt.Errorf("firebase.NewApp: %w", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase.Messaging: %w", err)
	}

	return &Sender{client: client}, nil
}

// Send отправляет push с уведомлением на все токены.
// Возвращает список невалидных токенов, которые нужно удалить из БД.
func (s *Sender) Send(ctx context.Context, tokens []string, title, body string, data map[string]string, badge int) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Notification: &messaging.Notification{
			Title: title,
			Body:  body,
		},
		Data: data,
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-push-type": "alert",
				"apns-priority":  "10",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{Badge: &badge},
			},
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
	}

	br, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("SendEachForMulticast: %w", err)
	}

	return collectInvalidTokens(tokens, br), nil
}

// SendBadge отправляет data-only silent push с обновлённым badge.
// На iOS обновляет число на иконке без показа баннера.
func (s *Sender) SendBadge(ctx context.Context, tokens []string, badge int) ([]string, error) {
	if len(tokens) == 0 {
		return nil, nil
	}

	msg := &messaging.MulticastMessage{
		Tokens: tokens,
		Data:   map[string]string{"badge": strconv.Itoa(badge)},
		APNS: &messaging.APNSConfig{
			Headers: map[string]string{
				"apns-push-type": "background",
				"apns-priority":  "5",
			},
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Badge:            &badge,
					ContentAvailable: true,
				},
			},
		},
		Android: &messaging.AndroidConfig{
			Data: map[string]string{"badge": strconv.Itoa(badge)},
		},
	}

	br, err := s.client.SendEachForMulticast(ctx, msg)
	if err != nil {
		return nil, fmt.Errorf("SendEachForMulticast badge: %w", err)
	}

	return collectInvalidTokens(tokens, br), nil
}

func collectInvalidTokens(tokens []string, br *messaging.BatchResponse) []string {
	var invalid []string
	for i, r := range br.Responses {
		if !r.Success && messaging.IsUnregistered(r.Error) {
			invalid = append(invalid, tokens[i])
		}
	}
	return invalid
}
