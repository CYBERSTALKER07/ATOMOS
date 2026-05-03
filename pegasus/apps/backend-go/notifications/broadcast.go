package notifications

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"backend-go/auth"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── System Broadcast ────────────────────────────────────────────────────────
// POST /v1/admin/broadcast — Sends a notification to all users of a given role.
// Writes Notifications rows + fires FCM data payloads to all registered devices.
//
// Request body:
//   { "title": "...", "body": "...", "role": "RETAILER|DRIVER|ALL", "data": {...} }

type BroadcastRequest struct {
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Role  string            `json:"role"` // RETAILER, DRIVER, or ALL
	Data  map[string]string `json:"data"` // Optional FCM data payload
}

// BroadcastService holds dependencies for system broadcast.
type BroadcastService struct {
	Spanner *spanner.Client
	FCM     *FCMClient
}

// HandleBroadcast processes a system-wide notification push.
func (bs *BroadcastService) HandleBroadcast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// SOVEREIGN ACTION: System-wide broadcast requires GLOBAL_ADMIN
	claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
	if !ok || claims.UserID == "" {
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}
	if err := auth.RequireGlobalAdmin(w, claims); err != nil {
		return
	}

	var req BroadcastRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid_json"}`, http.StatusBadRequest)
		return
	}
	if req.Title == "" || req.Body == "" {
		http.Error(w, `{"error":"title and body required"}`, http.StatusBadRequest)
		return
	}
	if req.Role == "" {
		req.Role = "ALL"
	}

	// Resolve target device tokens
	tokens, err := bs.resolveDeviceTokens(r.Context(), req.Role)
	if err != nil {
		log.Printf("[BROADCAST] Failed to resolve tokens: %v", err)
		http.Error(w, `{"error":"failed to resolve recipients"}`, http.StatusInternalServerError)
		return
	}

	// Write Notification records for each recipient
	recipients := bs.resolveRecipientIDs(r.Context(), req.Role)
	var mutations []*spanner.Mutation
	for _, recipientID := range recipients {
		notifID := newNotificationID()
		mutations = append(mutations, spanner.InsertOrUpdate("Notifications",
			[]string{"NotificationId", "RecipientId", "Type", "Title", "Body", "Channel", "CreatedAt"},
			[]interface{}{notifID, recipientID, "BROADCAST", req.Title, req.Body, "FCM", spanner.CommitTimestamp},
		))
	}
	if len(mutations) > 0 {
		// Batch write in chunks of 500 (Spanner limit)
		for i := 0; i < len(mutations); i += 500 {
			end := i + 500
			if end > len(mutations) {
				end = len(mutations)
			}
			_, err := bs.Spanner.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				return txn.BufferWrite(mutations[i:end])
			})
			if err != nil {
				log.Printf("[BROADCAST] Failed to write notification records: %v", err)
			}
		}
	}

	// Fire FCM data payloads concurrently with bounded parallelism
	var sentAtomic, failedAtomic atomic.Int64
	const maxConcurrentFCM = 20
	sem := make(chan struct{}, maxConcurrentFCM)
	var wg sync.WaitGroup

	data := map[string]string{
		"type":  ws.EventSystemBroadcast,
		"title": req.Title,
		"body":  req.Body,
	}
	for k, v := range req.Data {
		data[k] = v
	}

	for _, token := range tokens {
		wg.Add(1)
		sem <- struct{}{}
		go func(t string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := bs.FCM.SendDataMessage(t, data); err != nil {
				failedAtomic.Add(1)
			} else {
				sentAtomic.Add(1)
			}
		}(token)
	}
	wg.Wait()

	sent := int(sentAtomic.Load())
	failed := int(failedAtomic.Load())

	log.Printf("[BROADCAST] Completed: role=%s recipients=%d sent=%d failed=%d", req.Role, len(recipients), sent, failed)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "broadcast_complete",
		"recipients": len(recipients),
		"fcm_sent":   sent,
		"fcm_failed": failed,
	})
}

// resolveDeviceTokens queries DeviceTokens for all devices matching the role filter.
func (bs *BroadcastService) resolveDeviceTokens(ctx context.Context, role string) ([]string, error) {
	var sql string
	params := map[string]interface{}{}

	if role == "ALL" {
		sql = `SELECT Token FROM DeviceTokens`
	} else {
		sql = `SELECT Token FROM DeviceTokens WHERE Role = @role`
		params["role"] = role
	}

	iter := bs.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var tokens []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var token string
		if row.Columns(&token) == nil {
			tokens = append(tokens, token)
		}
	}
	return tokens, nil
}

// resolveRecipientIDs returns user IDs for notification record creation.
func (bs *BroadcastService) resolveRecipientIDs(ctx context.Context, role string) []string {
	var sql string
	params := map[string]interface{}{}

	switch role {
	case "RETAILER":
		sql = `SELECT RetailerId FROM Retailers`
	case "DRIVER":
		sql = `SELECT DriverId FROM Drivers`
	default: // ALL — union both
		sql = `SELECT RetailerId AS UserId FROM Retailers UNION ALL SELECT DriverId AS UserId FROM Drivers`
	}

	iter := bs.Spanner.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var ids []string
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			break
		}
		var id string
		if row.Columns(&id) == nil {
			ids = append(ids, id)
		}
	}
	return ids
}
