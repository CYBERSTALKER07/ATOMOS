package notifications

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Notification Inbox ──────────────────────────────────────────────────────
// GET /v1/user/notifications — Returns notification history for the authenticated user.
//   Query params: ?unread_only=true&limit=50

type NotificationItem struct {
	NotificationID string  `json:"notification_id"`
	Type           string  `json:"type"`
	Title          string  `json:"title"`
	Body           string  `json:"body"`
	Payload        string  `json:"payload,omitempty"`
	Channel        string  `json:"channel"`
	ReadAt         *string `json:"read_at"`
	CreatedAt      string  `json:"created_at"`
}

type NotificationInboxResponse struct {
	Notifications []NotificationItem `json:"notifications"`
	UnreadCount   int64              `json:"unread_count"`
	Total         int                `json:"total"`
	Limit         int64              `json:"limit"`
	Offset        int64              `json:"offset"`
}

func HandleNotificationInbox(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		unreadOnly := strings.EqualFold(r.URL.Query().Get("unread_only"), "true")

		// Pagination: limit (default 50, max 200) and offset (default 0)
		limit := int64(50)
		offset := int64(0)
		if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 64); err == nil && l > 0 {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
		if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 64); err == nil && o >= 0 {
			offset = o
		}

		// Build query
		sql := `SELECT NotificationId, Type, Title, Body, Payload, Channel, ReadAt, CreatedAt
			FROM Notifications
			WHERE RecipientId = @recipientId`
		if unreadOnly {
			sql += ` AND ReadAt IS NULL`
		}
		sql += ` ORDER BY CreatedAt DESC LIMIT @limit OFFSET @offset`

		stmt := spanner.Statement{
			SQL: sql,
			Params: map[string]interface{}{
				"recipientId": claims.UserID,
				"limit":       limit,
				"offset":      offset,
			},
		}

		resp := NotificationInboxResponse{Notifications: []NotificationItem{}}
		iter := client.Single().Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
				return
			}

			var item NotificationItem
			var payload spanner.NullString
			var readAt spanner.NullTime
			var createdAt time.Time
			if err := row.Columns(&item.NotificationID, &item.Type, &item.Title, &item.Body, &payload, &item.Channel, &readAt, &createdAt); err != nil {
				log.Printf("[NOTIFICATIONS] Decode error: %v", err)
				continue
			}
			if payload.Valid {
				item.Payload = payload.StringVal
			}
			if readAt.Valid {
				s := readAt.Time.Format(time.RFC3339)
				item.ReadAt = &s
			}
			item.CreatedAt = createdAt.Format(time.RFC3339)
			resp.Notifications = append(resp.Notifications, item)
		}
		resp.Total = len(resp.Notifications)
		resp.Limit = limit
		resp.Offset = offset

		// Unread count
		countStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Notifications
				WHERE RecipientId = @recipientId AND ReadAt IS NULL`,
			Params: map[string]interface{}{"recipientId": claims.UserID},
		}
		countIter := client.Single().Query(ctx, countStmt)
		defer countIter.Stop()
		countRow, err := countIter.Next()
		if err == nil {
			countRow.Columns(&resp.UnreadCount)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// ── Mark Notification Read ──────────────────────────────────────────────────
// POST /v1/user/notifications/read — Mark one or many notifications as read.

func HandleMarkNotificationRead(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			NotificationIDs []string `json:"notification_ids"`
			MarkAll         bool     `json:"mark_all"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		now := time.Now()

		if req.MarkAll {
			// Mark all unread for this user
			stmt := spanner.Statement{
				SQL: `UPDATE Notifications SET ReadAt = @now
					WHERE RecipientId = @recipientId AND ReadAt IS NULL`,
				Params: map[string]interface{}{
					"recipientId": claims.UserID,
					"now":         now,
				},
			}
			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				_, err := txn.Update(ctx, stmt)
				return err
			})
			if err != nil {
				log.Printf("[NOTIFICATIONS] Mark all read failed: %v", err)
				http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
				return
			}
		} else if len(req.NotificationIDs) > 0 {
			// Mark specific notifications
			_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
				for _, nid := range req.NotificationIDs {
					txn.BufferWrite([]*spanner.Mutation{
						spanner.Update("Notifications",
							[]string{"NotificationId", "ReadAt"},
							[]interface{}{nid, now},
						),
					})
				}
				return nil
			})
			if err != nil {
				log.Printf("[NOTIFICATIONS] Mark read failed: %v", err)
				http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
				return
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	}
}

// ── Send Notification Helper ────────────────────────────────────────────────
// Writes a notification record to Spanner. Used by other backend services.

func InsertNotification(ctx context.Context, client *spanner.Client, recipientID, recipientRole, notifType, title, body, payload, channel string) error {
	nid := "NOTIF-" + time.Now().Format("20060102150405") + "-" + recipientID[:8]
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Notifications",
			[]string{"NotificationId", "RecipientId", "RecipientRole", "Type", "Title", "Body", "Payload", "Channel", "CreatedAt"},
			[]interface{}{nid, recipientID, recipientRole, notifType, title, body, payload, channel, time.Now()},
		),
	})
	if err != nil {
		log.Printf("[NOTIFICATIONS] Insert failed for %s: %v", recipientID, err)
	}
	return err
}

// InsertNotificationWithCorrelation writes a notification with CorrelationId
// and optional ExpiresAt for lifecycle-managed alerts (e.g. pre-order confirmations).
func InsertNotificationWithCorrelation(ctx context.Context, client *spanner.Client, recipientID, recipientRole, notifType, title, body, payload, channel, correlationID string, expiresAt *time.Time) error {
	nid := "NOTIF-" + time.Now().Format("20060102150405") + "-" + recipientID[:8]
	cols := []string{"NotificationId", "RecipientId", "RecipientRole", "Type", "Title", "Body", "Payload", "Channel", "CorrelationId", "CreatedAt"}
	vals := []interface{}{nid, recipientID, recipientRole, notifType, title, body, payload, channel, correlationID, time.Now()}
	if expiresAt != nil {
		cols = append(cols, "ExpiresAt")
		vals = append(vals, *expiresAt)
	}
	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.Insert("Notifications", cols, vals),
	})
	if err != nil {
		log.Printf("[NOTIFICATIONS] Correlated insert failed for %s (corr: %s): %v", recipientID, correlationID, err)
	}
	return err
}

// DeleteByCorrelationId removes all unread notifications matching a CorrelationId.
// Used to clear stale "Confirm your order" alerts after the retailer confirms.
func DeleteByCorrelationId(ctx context.Context, client *spanner.Client, correlationID string) (int64, error) {
	var deleted int64
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT NotificationId FROM Notifications
			      WHERE CorrelationId = @cid AND ReadAt IS NULL`,
			Params: map[string]interface{}{"cid": correlationID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		var keys []spanner.Key
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var nid string
			if err := row.Columns(&nid); err != nil {
				return err
			}
			keys = append(keys, spanner.Key{nid})
			deleted++
		}
		if len(keys) == 0 {
			return nil
		}
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Delete("Notifications", spanner.KeySetFromKeys(keys...)),
		})
		return nil
	})
	if err != nil {
		log.Printf("[NOTIFICATIONS] DeleteByCorrelationId failed (corr: %s): %v", correlationID, err)
	}
	return deleted, err
}
