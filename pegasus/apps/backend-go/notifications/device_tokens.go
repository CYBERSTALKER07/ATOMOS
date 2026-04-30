package notifications

import (
	"context"
	"fmt"
	"log"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
)

// DeviceTokenService handles FCM/APNs token CRUD against the DeviceTokens table.
type DeviceTokenService struct {
	Spanner *spanner.Client
}

// RegisterTokenRequest is the expected JSON payload for device registration.
type RegisterTokenRequest struct {
	Token    string `json:"token"`
	Platform string `json:"platform"` // ANDROID | IOS | WEB
}

// RegisterToken upserts a device token for the given user/role/platform.
// Uses the unique index (UserId, Platform) to prevent duplicates.
func (s *DeviceTokenService) RegisterToken(ctx context.Context, userID, role string, req RegisterTokenRequest) error {
	if req.Token == "" {
		return fmt.Errorf("token is required")
	}
	if req.Platform == "" {
		req.Platform = "ANDROID" // default
	}

	// Check if a token already exists for this user+platform
	stmt := spanner.Statement{
		SQL:    `SELECT TokenId FROM DeviceTokens WHERE UserId = @uid AND Platform = @platform LIMIT 1`,
		Params: map[string]interface{}{"uid": userID, "platform": req.Platform},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	var existingTokenID string
	row, err := iter.Next()
	if err == nil {
		_ = row.Columns(&existingTokenID)
	}
	iter.Stop()

	if existingTokenID != "" {
		// Update existing token
		_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("DeviceTokens",
					[]string{"TokenId", "Token", "CreatedAt"},
					[]interface{}{existingTokenID, req.Token, spanner.CommitTimestamp},
				),
			})
		})
		if err != nil {
			return fmt.Errorf("failed to update device token: %w", err)
		}
		log.Printf("[DEVICE_TOKENS] Updated token for user=%s platform=%s", userID, req.Platform)
		return nil
	}

	// Insert new token
	tokenID := uuid.New().String()
	_, err = s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("DeviceTokens",
				[]string{"TokenId", "UserId", "Role", "Platform", "Token", "CreatedAt"},
				[]interface{}{tokenID, userID, role, req.Platform, req.Token, spanner.CommitTimestamp},
			),
		})
	})
	if err != nil {
		return fmt.Errorf("failed to register device token: %w", err)
	}
	log.Printf("[DEVICE_TOKENS] Registered token for user=%s role=%s platform=%s", userID, role, req.Platform)
	return nil
}

// UnregisterToken removes a device token for the given user and platform.
func (s *DeviceTokenService) UnregisterToken(ctx context.Context, userID, platform string) error {
	if platform == "" {
		platform = "ANDROID"
	}

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `DELETE FROM DeviceTokens WHERE UserId = @uid AND Platform = @platform`,
			Params: map[string]interface{}{"uid": userID, "platform": platform},
		}
		_, err := txn.Update(ctx, stmt)
		return err
	})
	if err != nil {
		return fmt.Errorf("failed to unregister device token: %w", err)
	}
	log.Printf("[DEVICE_TOKENS] Unregistered token for user=%s platform=%s", userID, platform)
	return nil
}
