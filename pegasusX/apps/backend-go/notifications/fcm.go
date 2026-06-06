package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/spanner"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

// FCMClient sends Firebase Cloud Messaging payloads. Runs no-op when credentials are absent.
type FCMClient struct {
	client        *messaging.Client
	noOp          bool
	spannerClient *spanner.Client
	log           *slog.Logger
}

// InitFCM boots Firebase Admin using a service account JSON path.
func InitFCM(credentialsFilePath string, spannerClient *spanner.Client, log *slog.Logger) (*FCMClient, error) {
	if log == nil {
		log = slog.Default()
	}
	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialsFilePath)
	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("fcm boot: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm messaging client: %w", err)
	}
	log.Info("FCM client online")
	return &FCMClient{client: client, spannerClient: spannerClient, log: log}, nil
}

// NewNoOpFCMClient returns a degraded client that skips push delivery.
func NewNoOpFCMClient(log *slog.Logger) *FCMClient {
	if log == nil {
		log = slog.Default()
	}
	log.Info("FCM running in no-op mode")
	return &FCMClient{noOp: true, log: log}
}

// SendDataMessage delivers a silent data payload to one device token.
func (f *FCMClient) SendDataMessage(ctx context.Context, deviceToken string, data map[string]string) error {
	if f.noOp || deviceToken == "" {
		return fmt.Errorf("fcm no-op or empty token")
	}
	message := &messaging.Message{
		Data:  data,
		Token: deviceToken,
		Android: &messaging.AndroidConfig{Priority: "high"},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{ContentAvailable: true},
			},
		},
	}
	_, err := f.client.Send(ctx, message)
	if err != nil {
		if messaging.IsRegistrationTokenNotRegistered(err) {
			go f.purgeStaleToken(deviceToken)
		}
		return fmt.Errorf("fcm send: %w", err)
	}
	return nil
}

func (f *FCMClient) purgeStaleToken(token string) {
	if f.spannerClient == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := f.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.Delete("DeviceTokens", spanner.Key{token})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		f.log.Warn("fcm stale token purge failed", "err", err)
	}
}
