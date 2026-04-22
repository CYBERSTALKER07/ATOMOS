// Package crypto implements the Lab Industries Hash Manifest Protocol.
//
// Security model:
//   - The raw delivery token NEVER leaves the backend.
//   - The driver device receives only SHA-256 hashes.
//   - At drop time, the device hashes the retailer's QR payload locally
//     and compares against the cached manifest—no network required.
package crypto

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// RouteManifest is the signed payload downloaded by the driver at depot.
// Hashes is keyed by OrderId; values are SHA-256(DeliveryToken).
type RouteManifest struct {
	DriverID  string            `json:"driver_id"`
	Date      string            `json:"date"`
	ExpiresAt int64             `json:"expires_at"` // Unix epoch — manifest is valid for one calendar day
	Hashes    map[string]string `json:"hashes"`     // OrderID → SHA256(DeliveryToken)
}

// GenerateSHA256 returns the lowercase hex-encoded SHA-256 digest of token.
// This is the sole hashing primitive used across the entire protocol.
func GenerateSHA256(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GetDriverManifestHandler returns an http.HandlerFunc that:
//  1. Reads the caller's DriverID from the JWT (injected by auth middleware as
//     the "X-Driver-Id" header after role verification).
//  2. Queries Spanner for today's orders assigned to that driver.
//  3. Hashes each order's DeliveryToken and returns the manifest JSON.
//
// Route: GET /v1/driver/manifest
// Auth:  DRIVER role required (enforced upstream by RequireRole middleware)
func GetDriverManifestHandler(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Injected by auth.RequireRole after JWT validation
		driverID := r.Header.Get("X-Driver-Id")
		if driverID == "" {
			// Fallback for local dev / emulator without full auth pipeline
			driverID = r.URL.Query().Get("driver_id")
		}
		if driverID == "" {
			http.Error(w, `{"error":"driver_id required"}`, http.StatusBadRequest)
			return
		}

		ctx := r.Context()
		today := time.Now().UTC().Format("2006-01-02")

		// ── Spanner read: orders assigned to this driver that need a token ──
		hashes := make(map[string]string)

		stmt := spanner.Statement{
			SQL: `SELECT OrderId, DeliveryToken
			      FROM Orders
			      WHERE DriverId = @driverID
			        AND State IN ('PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED')
			      LIMIT 200`,
			Params: map[string]interface{}{"driverID": driverID},
		}

		iter := spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()

		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("[MANIFEST] Spanner read error: %v", err)
				http.Error(w, `{"error":"internal spanner error"}`, http.StatusInternalServerError)
				return
			}

			var orderID, deliveryToken string
			if err := row.Columns(&orderID, &deliveryToken); err != nil {
				log.Printf("[MANIFEST] Row decode error: %v", err)
				continue
			}

			// SECURITY: hash before writing to the manifest—raw token never leaves.
			if deliveryToken != "" {
				hashes[orderID] = GenerateSHA256(deliveryToken)
			}
		}

		// Manifest expires at midnight UTC of the current day
		midnight := time.Now().UTC().Truncate(24 * time.Hour).Add(24 * time.Hour)

		manifest := RouteManifest{
			DriverID:  driverID,
			Date:      today,
			ExpiresAt: midnight.Unix(),
			Hashes:    hashes,
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store") // Never cache—fresh manifest each depot sync
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			log.Printf("[MANIFEST] JSON encode error: %v", err)
		}
	}
}
