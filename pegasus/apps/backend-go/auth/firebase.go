package auth

import (
	"context"
	"fmt"
	"log"
	"os"

	firebase "firebase.google.com/go/v4"
	firebaseAuth "firebase.google.com/go/v4/auth"
	"google.golang.org/api/option"
)

// FirebaseAuthClient wraps the Firebase Admin Auth client.
// When nil, the system operates in legacy-only JWT mode.
var FirebaseAuthClient *firebaseAuth.Client

// InitFirebaseAuth initializes the Firebase Auth client from a Firebase App.
// When FIREBASE_AUTH_EMULATOR_HOST is set, the Go Admin SDK automatically
// connects to the local emulator — no credentials file is needed.
func InitFirebaseAuth(ctx context.Context) (*firebaseAuth.Client, error) {
	var app *firebase.App
	var err error

	credPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	emulatorHost := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")

	if emulatorHost != "" {
		// Emulator mode — no credentials needed.
		// The SDK reads FIREBASE_AUTH_EMULATOR_HOST automatically.
		projectID := os.Getenv("GCLOUD_PROJECT")
		if projectID == "" {
			projectID = "demo-pegasus"
		}
		conf := &firebase.Config{ProjectID: projectID}
		app, err = firebase.NewApp(ctx, conf)
		if err != nil {
			return nil, fmt.Errorf("firebase app init (emulator): %w", err)
		}
	} else if credPath != "" {
		// Production mode — use service account credentials.
		opt := option.WithCredentialsFile(credPath)
		app, err = firebase.NewApp(ctx, nil, opt)
		if err != nil {
			return nil, fmt.Errorf("firebase app init (credentials): %w", err)
		}
	} else {
		log.Println("[FIREBASE AUTH] No emulator or credentials configured — Firebase Auth disabled")
		return nil, nil
	}

	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("firebase auth client: %w", err)
	}

	FirebaseAuthClient = client
	return client, nil
}

// CreateFirebaseUser creates a new user in Firebase Auth and sets custom claims.
// Returns the Firebase UID. Gracefully returns empty string if client is nil.
func CreateFirebaseUser(ctx context.Context, email, password, displayName, phone, role string, extraClaims map[string]interface{}) (string, error) {
	if FirebaseAuthClient == nil {
		return "", nil
	}

	params := (&firebaseAuth.UserToCreate{}).DisplayName(displayName)
	if email != "" {
		params = params.Email(email).EmailVerified(false)
	}
	if password != "" {
		params = params.Password(password)
	}
	if phone != "" {
		params = params.PhoneNumber(phone)
	}

	user, err := FirebaseAuthClient.CreateUser(ctx, params)
	if err != nil {
		return "", fmt.Errorf("create firebase user: %w", err)
	}

	// Set custom claims (role + any extra IDs)
	claims := map[string]interface{}{"role": role}
	for k, v := range extraClaims {
		claims[k] = v
	}
	if err := FirebaseAuthClient.SetCustomUserClaims(ctx, user.UID, claims); err != nil {
		log.Printf("[FIREBASE AUTH] Failed to set claims for %s: %v", user.UID, err)
	}

	return user.UID, nil
}

// MintCustomToken generates a Firebase Custom Token for the given UID with custom claims.
// The client SDK exchanges this for a Firebase ID token via signInWithCustomToken().
// Returns empty string if client is nil (graceful degradation).
func MintCustomToken(ctx context.Context, uid string, claims map[string]interface{}) (string, error) {
	if FirebaseAuthClient == nil {
		return "", nil
	}

	token, err := FirebaseAuthClient.CustomTokenWithClaims(ctx, uid, claims)
	if err != nil {
		return "", fmt.Errorf("mint custom token for %s: %w", uid, err)
	}
	return token, nil
}

// VerifyFirebaseToken verifies a Firebase ID token and extracts PegasusClaims.
// Returns nil claims if the token is not a valid Firebase token (allows fallback to legacy JWT).
func VerifyFirebaseToken(ctx context.Context, tokenStr string) (*PegasusClaims, error) {
	if FirebaseAuthClient == nil {
		return nil, fmt.Errorf("firebase auth not initialized")
	}

	decoded, err := FirebaseAuthClient.VerifyIDToken(ctx, tokenStr)
	if err != nil {
		return nil, err
	}

	// Extract role from custom claims
	role, _ := decoded.Claims["role"].(string)
	if role == "" {
		role = "UNKNOWN"
	}

	// Extract scope claims from Firebase custom token
	warehouseID, _ := decoded.Claims["warehouse_id"].(string)
	supplierRole, _ := decoded.Claims["supplier_role"].(string)
	factoryID, _ := decoded.Claims["factory_id"].(string)
	factoryRole, _ := decoded.Claims["factory_role"].(string)

	// The UID in Firebase maps to our UserID
	claims := &PegasusClaims{
		UserID:       decoded.UID,
		Role:         role,
		WarehouseID:  warehouseID,
		SupplierRole: supplierRole,
		FactoryID:    factoryID,
		FactoryRole:  factoryRole,
	}

	return claims, nil
}

// LookupFirebaseUID retrieves the Firebase UID for a user if it exists in the
// emulator/production. Returns empty string if not found or client is nil.
func LookupFirebaseUID(ctx context.Context, email string) string {
	if FirebaseAuthClient == nil || email == "" {
		return ""
	}
	user, err := FirebaseAuthClient.GetUserByEmail(ctx, email)
	if err != nil {
		return ""
	}
	return user.UID
}
