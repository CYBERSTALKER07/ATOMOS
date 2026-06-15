package auth

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

var (
	firebaseAdminMu     sync.Mutex
	firebaseAdminClient *firebaseAuth.Client
	firebaseAdminInit   bool
)

func initFirebaseAdminClient(ctx context.Context) (*firebaseAuth.Client, error) {
	firebaseAdminMu.Lock()
	defer firebaseAdminMu.Unlock()
	if firebaseAdminInit {
		return firebaseAdminClient, nil
	}
	firebaseAdminInit = true

	emulatorHost := strings.TrimSpace(os.Getenv("FIREBASE_AUTH_EMULATOR_HOST"))
	credPath := strings.TrimSpace(os.Getenv("FIREBASE_CREDENTIALS_PATH"))

	var app *firebase.App
	var err error
	switch {
	case emulatorHost != "":
		projectID := strings.TrimSpace(os.Getenv("GCLOUD_PROJECT"))
		if projectID == "" {
			projectID = strings.TrimSpace(os.Getenv("FIREBASE_PROJECT_ID"))
		}
		if projectID == "" {
			projectID = "demo-pegasus"
		}
		app, err = firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID})
	case credPath != "":
		app, err = firebase.NewApp(ctx, nil, option.WithCredentialsFile(credPath))
	default:
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("firebase admin app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase admin auth: %w", err)
	}
	firebaseAdminClient = client
	slog.Info("firebase admin auth client initialized")
	return firebaseAdminClient, nil
}

// MintCustomToken generates a Firebase custom token when Admin Auth is configured.
// Returns empty string when Firebase is unavailable (graceful degradation).
func MintCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return "", nil
	}
	client, err := initFirebaseAdminClient(ctx)
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", nil
	}
	token, err := client.CustomTokenWithClaims(ctx, uid, claims)
	if err != nil {
		return "", fmt.Errorf("mint custom token: %w", err)
	}
	return token, nil
}
