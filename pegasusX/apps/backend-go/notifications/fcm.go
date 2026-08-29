package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/messaging"
	"google.golang.org/api/option"
	"google.golang.org/grpc/codes"
)

// FCMClient sends Firebase Cloud Messaging payloads. Runs no-op when credentials are absent.
type FCMClient struct {
	client        *messaging.Client
	noOp          bool
	spannerClient *spanner.Client
	log           *slog.Logger
}

// InitFCM boots Firebase Admin Messaging.
// Prefer a service-account JSON path when present and non-empty; otherwise use
// Application Default Credentials (Workload Identity on GKE) with projectID.
func InitFCM(credentialsFilePath, projectID string, spannerClient *spanner.Client, log *slog.Logger) (*FCMClient, error) {
	if log == nil {
		log = slog.Default()
	}
	ctx := context.Background()
	app, err := newFirebaseApp(ctx, credentialsFilePath, projectID)
	if err != nil {
		return nil, fmt.Errorf("fcm boot: %w", err)
	}
	client, err := app.Messaging(ctx)
	if err != nil {
		return nil, fmt.Errorf("fcm messaging client: %w", err)
	}
	log.Info("FCM client online", "project_id", strings.TrimSpace(projectID), "mode", firebaseCredMode(credentialsFilePath))
	return &FCMClient{client: client, spannerClient: spannerClient, log: log}, nil
}

func newFirebaseApp(ctx context.Context, credentialsFilePath, projectID string) (*firebase.App, error) {
	path := strings.TrimSpace(credentialsFilePath)
	pid := strings.TrimSpace(projectID)
	if path != "" && !isStubCredentialsFile(path) {
		if pid != "" {
			return firebase.NewApp(ctx, &firebase.Config{ProjectID: pid}, option.WithCredentialsFile(path))
		}
		return firebase.NewApp(ctx, nil, option.WithCredentialsFile(path))
	}
	// ADC / Workload Identity path (org policies often block SA user keys).
	if pid == "" {
		return nil, fmt.Errorf("project ID is required when credentials file is empty (set FIREBASE_PROJECT_ID)")
	}
	return firebase.NewApp(ctx, &firebase.Config{ProjectID: pid})
}

func isStubCredentialsFile(path string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		return true
	}
	s := strings.TrimSpace(string(b))
	return s == "" || s == "{}" || s == "null"
}

func firebaseCredMode(credentialsFilePath string) string {
	path := strings.TrimSpace(credentialsFilePath)
	if path != "" && !isStubCredentialsFile(path) {
		return "credentials_file"
	}
	return "adc"
}

// NewNoOpFCMClient returns a degraded client that skips push delivery.
// G1.D: always log as push_degraded so prod alerting can catch silent no-op.
func NewNoOpFCMClient(log *slog.Logger) *FCMClient {
	if log == nil {
		log = slog.Default()
	}
	log.Error("FCM running in no-op mode",
		"push_degraded", true,
		"alert", "fcm_noop",
		"message", "push notifications will not be delivered until Firebase credentials/project are configured",
	)
	return &FCMClient{noOp: true, log: log}
}

// IsNoOp reports whether this client skips all push delivery (G1.D honesty).
func (f *FCMClient) IsNoOp() bool {
	return f == nil || f.noOp || f.client == nil
}

// SendDataMessage delivers a silent data payload to one device token.
func (f *FCMClient) SendDataMessage(ctx context.Context, deviceToken string, data map[string]string) error {
	if f.IsNoOp() || deviceToken == "" {
		if f != nil && f.log != nil && f.noOp {
			f.log.WarnContext(ctx, "fcm send skipped: push_degraded no-op",
				"push_degraded", true, "alert", "fcm_noop_send")
		}
		return fmt.Errorf("fcm no-op or empty token")
	}
	var notification *messaging.Notification
	if title := strings.TrimSpace(data["title"]); title != "" {
		notification = &messaging.Notification{
			Title: title,
			Body:  strings.TrimSpace(data["body"]),
		}
	}
	message := &messaging.Message{
		Notification: notification,
		Data:         data,
		Token:        deviceToken,
		Android:      &messaging.AndroidConfig{Priority: "high"},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{ContentAvailable: true},
			},
		},
	}
	_, err := f.client.Send(ctx, message)
	if err != nil {
		if messaging.IsRegistrationTokenNotRegistered(err) {
			sessionID := strings.TrimSpace(data["session_id"])
			go f.purgeStaleTokenWithSession(deviceToken, sessionID)
		}
		return fmt.Errorf("fcm send: %w", err)
	}
	return nil
}

// PurgeStaleToken removes a stale token conditionally based on sessionID.
// If sessionID is empty, it removes the token unconditionally using a blind write.
// If sessionID is non-empty, it reads the current SessionId in Spanner and only deletes
// the token if the stored SessionId matches sessionID (protecting newer session registrations).
func (f *FCMClient) PurgeStaleToken(ctx context.Context, token string, sessionID string) error {
	if f == nil || f.spannerClient == nil || token == "" {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	tctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if sessionID == "" {
		// Blind delete for legacy / non-session-scoped purges
		_, err := f.spannerClient.Apply(tctx, []*spanner.Mutation{
			spanner.Delete("DeviceTokens", spanner.Key{token}),
		})
		if err != nil && f.log != nil {
			f.log.Warn("fcm stale token purge failed", "err", err, "token", token)
		}
		return err
	}

	_, err := f.spannerClient.ReadWriteTransaction(tctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "DeviceTokens", spanner.Key{token}, []string{"SessionId"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return nil
			}
			return err
		}
		var currentSession spanner.NullString
		if err := row.Columns(&currentSession); err != nil {
			return err
		}
		if currentSession.Valid && currentSession.StringVal != "" && currentSession.StringVal != sessionID {
			// Newer session has registered this token; skip purge.
			return nil
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("DeviceTokens", spanner.Key{token}),
		})
	})
	if err != nil && f.log != nil {
		f.log.Warn("fcm stale token purge with session failed", "err", err, "token", token, "session_id", sessionID)
	}
	return err
}

func (f *FCMClient) purgeStaleToken(token string) {
	_ = f.PurgeStaleToken(context.Background(), token, "")
}

func (f *FCMClient) purgeStaleTokenWithSession(token string, sessionID string) {
	_ = f.PurgeStaleToken(context.Background(), token, sessionID)
}
