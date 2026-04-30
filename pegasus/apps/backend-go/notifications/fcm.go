package notifications

import (
	"context"
	"fmt"
	"log"
	"time"

	"backend-go/ws"

	"cloud.google.com/go/spanner"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
)

type FCMClient struct {
	client *messaging.Client
	// noOp is true when Firebase credentials are absent (graceful degradation)
	noOp bool
	// SpannerClient is optional — when set, stale FCM tokens are auto-purged
	SpannerClient *spanner.Client
}

// InitFCM boots the Firebase Admin SDK using your Google Cloud service account key
func InitFCM(credentialsFilePath string) (*FCMClient, error) {
	ctx := context.Background()
	opt := option.WithCredentialsFile(credentialsFilePath)

	app, err := firebase.NewApp(ctx, nil, opt)
	if err != nil {
		return nil, fmt.Errorf("FCM boot failure: %v", err)
	}

	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("FCM messaging client failure: %v", err)
	}

	log.Println("[COMMUNICATION SPINE] Firebase Cloud Messaging Online.")
	return &FCMClient{client: client}, nil
}

// NewNoOpFCMClient returns a degraded FCM client that always falls through to
// the Telegram fallback. Use this when Firebase credentials are not available.
func NewNoOpFCMClient() *FCMClient {
	log.Println("[COMMUNICATION SPINE] FCM running in no-op mode — Telegram fallback will handle all alerts.")
	return &FCMClient{noOp: true}
}

// WakeRetailer fires the physical push notification to the device
func (f *FCMClient) WakeRetailer(deviceToken string, orderTotal int64, shopName string) error {
	if f.noOp {
		return fmt.Errorf("FCM no-op: credentials not configured")
	}

	ctx := context.Background()

	message := &messaging.Message{
		Notification: &messaging.Notification{
			Title: "AI Restock Authorization",
			Body:  fmt.Sprintf("%s inventory low. Tap to authorize %d restock payload.", shopName, orderTotal),
		},
		Data: map[string]string{
			"type":   ws.EventAIPrediction,
			"action": "AUTHORIZE_DISPATCH",
		},
		Token: deviceToken,
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		if messaging.IsRegistrationTokenNotRegistered(err) {
			log.Printf("[FCM] Token unregistered — scheduling cleanup for token=%s...%s", deviceToken[:8], deviceToken[len(deviceToken)-4:])
			go f.purgeStaleToken(deviceToken)
		}
		return fmt.Errorf("failed to wake device: %v", err)
	}

	log.Printf("[COMMUNICATION SPINE] Push fired successfully. Message ID: %s", response)
	return nil
}

// WakeRetailerWithFallback attempts FCM first.
// If FCM fails for any reason (no token, Firebase down, unregistered device),
// it immediately falls back to Telegram within the same call.
//
// Parameters:
//   - deviceToken: FCM device token (may be empty — triggers immediate fallback)
//   - telegramChatID: Telegram chat_id for the retailer (read from Spanner)
//   - orderTotal: predicted order order amount
//   - shopName: retailer display name for the message body
//   - tg: pre-initialised TelegramClient
func (f *FCMClient) WakeRetailerWithFallback(
	deviceToken string,
	telegramChatID string,
	orderTotal int64,
	shopName string,
	tg *TelegramClient,
) {
	// --- Primary Channel: FCM Push ---
	if deviceToken != "" {
		err := f.WakeRetailer(deviceToken, orderTotal, shopName)
		if err == nil {
			log.Printf("[COMMUNICATION SPINE] FCM primary succeeded for %s", shopName)
			return
		}
		log.Printf("[COMMUNICATION SPINE] FCM primary failed (%v) — engaging Telegram fallback", err)
	} else {
		log.Printf("[COMMUNICATION SPINE] No FCM token for %s — engaging Telegram fallback immediately", shopName)
	}

	// --- Fallback Channel: Telegram Bot ---
	if telegramChatID == "" {
		log.Printf("[COMMUNICATION SPINE] No Telegram chat_id for %s — all notification channels exhausted", shopName)
		return
	}

	msg := FormatPredictionAlert(shopName, orderTotal)
	if err := tg.SendMessage(telegramChatID, msg); err != nil {
		log.Printf("[COMMUNICATION SPINE] Telegram fallback also failed for %s: %v", shopName, err)
	}
}

// SendDataMessage fires a silent FCM data-only payload to the device.
// This is used by the DRIVER_APPROACHING push protocol to wake the native app
// and trigger the QR popup without a visible notification banner.
func (f *FCMClient) SendDataMessage(deviceToken string, data map[string]string) error {
	if f.noOp {
		log.Printf("[FCM NO-OP] Data payload would be sent to token=%s data=%v", deviceToken, data)
		return fmt.Errorf("FCM no-op: credentials not configured")
	}

	ctx := context.Background()

	message := &messaging.Message{
		Data:  data,
		Token: deviceToken,
		Android: &messaging.AndroidConfig{
			Priority: "high",
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					ContentAvailable: true,
				},
			},
		},
	}

	response, err := f.client.Send(ctx, message)
	if err != nil {
		if messaging.IsRegistrationTokenNotRegistered(err) {
			log.Printf("[FCM] Data token unregistered — scheduling cleanup for token=%s...%s", deviceToken[:8], deviceToken[len(deviceToken)-4:])
			go f.purgeStaleToken(deviceToken)
		}
		return fmt.Errorf("FCM data payload failed: %w", err)
	}

	log.Printf("[FCM] Data payload delivered. Message ID: %s", response)
	return nil
}

// purgeStaleToken removes an unregistered FCM token from the DeviceTokens table.
func (f *FCMClient) purgeStaleToken(token string) {
	if f.SpannerClient == nil {
		log.Printf("[FCM] Cannot purge stale token — no Spanner client")
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := f.SpannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `DELETE FROM DeviceTokens WHERE Token = @token`,
			Params: map[string]interface{}{"token": token},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		log.Printf("[FCM] Failed to purge stale token: %v", err)
		return
	}
	log.Printf("[FCM] Purged stale device token from DeviceTokens")
}
