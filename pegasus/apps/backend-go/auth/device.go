package auth

import (
	"context"
	"log"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// DeviceFingerprint represents a tracked device session.
type DeviceFingerprint struct {
	FingerprintId string
	UserId        string
	Role          string
	DeviceId      string
	Platform      string
	AppVersion    string
	Active        bool
}

// RecordDeviceLogin registers a device fingerprint on login and force-logouts
// any other active device for the same user+role.
// Returns a list of device IDs that were deactivated (for WebSocket FORCE_LOGOUT push).
func RecordDeviceLogin(ctx context.Context, client *spanner.Client, userID, role, deviceID, platform, appVersion string) ([]string, error) {
	if deviceID == "" {
		return nil, nil // No device ID header — skip fingerprinting
	}

	var deactivatedDevices []string

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Find all active fingerprints for this user+role on OTHER devices
		stmt := spanner.Statement{
			SQL: `SELECT FingerprintId, DeviceId FROM DeviceFingerprints
			      WHERE UserId = @userId AND Role = @role AND Active = true AND DeviceId != @deviceId`,
			Params: map[string]interface{}{
				"userId":   userID,
				"role":     role,
				"deviceId": deviceID,
			},
		}
		iter := txn.Query(ctx, stmt)

		var deactivateIds []string
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				iter.Stop()
				return err
			}
			var fpId, devId string
			if err := row.Columns(&fpId, &devId); err != nil {
				continue
			}
			deactivateIds = append(deactivateIds, fpId)
			deactivatedDevices = append(deactivatedDevices, devId)
		}
		iter.Stop()

		// 2. Deactivate old devices
		for _, fpId := range deactivateIds {
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("DeviceFingerprints",
					[]string{"FingerprintId", "Active", "LastSeenAt"},
					[]interface{}{fpId, false, spanner.CommitTimestamp}),
			})
		}

		// 3. Upsert current device fingerprint
		// Check if this exact device already has an active fingerprint
		existingStmt := spanner.Statement{
			SQL: `SELECT FingerprintId FROM DeviceFingerprints
			      WHERE UserId = @userId AND Role = @role AND DeviceId = @deviceId AND Active = true
			      LIMIT 1`,
			Params: map[string]interface{}{
				"userId":   userID,
				"role":     role,
				"deviceId": deviceID,
			},
		}
		existingIter := txn.Query(ctx, existingStmt)
		existRow, existErr := existingIter.Next()
		existingIter.Stop()

		if existErr == nil && existRow != nil {
			// Update LastSeenAt on existing fingerprint
			var fpId string
			_ = existRow.Columns(&fpId)
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("DeviceFingerprints",
					[]string{"FingerprintId", "LastSeenAt", "AppVersion"},
					[]interface{}{fpId, spanner.CommitTimestamp, appVersion}),
			})
		} else {
			// Insert new fingerprint
			newID := generateUUID()
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("DeviceFingerprints",
					[]string{"FingerprintId", "UserId", "Role", "DeviceId", "Platform", "AppVersion", "LastSeenAt", "Active"},
					[]interface{}{newID, userID, role, deviceID, platform, appVersion, spanner.CommitTimestamp, true}),
			})
		}

		return nil
	})

	if err != nil {
		log.Printf("[DeviceAuth] Failed to record device login for user %s: %v", userID, err)
		return nil, err
	}

	if len(deactivatedDevices) > 0 {
		log.Printf("[DeviceAuth] User %s (%s) logged in on %s — deactivated %d other device(s)",
			userID, role, deviceID, len(deactivatedDevices))
	}

	return deactivatedDevices, nil
}

// generateUUID creates a UUID v4.
func generateUUID() string {
	return uuid.New().String()
}
