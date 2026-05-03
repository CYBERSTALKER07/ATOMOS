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
	"backend-go/cache"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
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

const (
	defaultNotificationInboxLimit  int64 = 50
	defaultNotificationInboxOffset int64 = 0
	notificationInboxCacheTTL            = 30 * time.Second
	notificationInboxCachePrefix         = "cache:notifications:inbox:"
)

func notificationInboxCacheKey(recipientID string) string {
	return notificationInboxCachePrefix + recipientID
}

func shouldUseNotificationInboxCache(unreadOnly bool, limit, offset int64) bool {
	return !unreadOnly && limit == defaultNotificationInboxLimit && offset == defaultNotificationInboxOffset
}

func readNotificationInboxCache(ctx context.Context, recipientID string) (NotificationInboxResponse, bool) {
	var cached NotificationInboxResponse
	rc := cache.GetClient()
	if rc == nil {
		return cached, false
	}

	blob, err := rc.Get(ctx, notificationInboxCacheKey(recipientID)).Bytes()
	if err != nil {
		return cached, false
	}

	if err := json.Unmarshal(blob, &cached); err != nil {
		return cached, false
	}

	return cached, true
}

func writeNotificationInboxCache(ctx context.Context, recipientID string, resp NotificationInboxResponse) {
	rc := cache.GetClient()
	if rc == nil {
		return
	}

	blob, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[NOTIFICATIONS] cache marshal failed for %s: %v", recipientID, err)
		return
	}

	if err := rc.Set(ctx, notificationInboxCacheKey(recipientID), blob, notificationInboxCacheTTL).Err(); err != nil {
		log.Printf("[NOTIFICATIONS] cache set failed for %s: %v", recipientID, err)
	}
}

// InvalidateNotificationInboxCache drops the cached default inbox response for
// recipientID after notification mutations (insert/read/delete).
func InvalidateNotificationInboxCache(ctx context.Context, recipientID string) {
	if recipientID == "" {
		return
	}
	cache.Invalidate(ctx, notificationInboxCacheKey(recipientID))
}

func newNotificationID() string {
	return uuid.NewString()
}

func notificationRecipientID(claims *auth.PegasusClaims) string {
	if claims == nil {
		return ""
	}
	switch claims.Role {
	case "SUPPLIER", "PAYLOADER":
		return claims.ResolveSupplierID()
	default:
		return claims.UserID
	}
}

func HandleNotificationInbox(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()
		recipientID := notificationRecipientID(claims)
		if recipientID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		unreadOnly := strings.EqualFold(r.URL.Query().Get("unread_only"), "true")

		// Pagination: limit (default 50, max 200) and offset (default 0)
		limit := defaultNotificationInboxLimit
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

		useInboxCache := shouldUseNotificationInboxCache(unreadOnly, limit, offset)
		if useInboxCache {
			if cached, ok := readNotificationInboxCache(ctx, recipientID); ok {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(cached)
				return
			}
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
				"recipientId": recipientID,
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
			Params: map[string]interface{}{"recipientId": recipientID},
		}
		countIter := client.Single().Query(ctx, countStmt)
		defer countIter.Stop()
		countRow, err := countIter.Next()
		if err == nil {
			countRow.Columns(&resp.UnreadCount)
		}

		if useInboxCache {
			writeNotificationInboxCache(ctx, recipientID, resp)
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

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
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
		recipientID := notificationRecipientID(claims)
		if recipientID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		now := time.Now()

		if req.MarkAll {
			// Mark all unread for this user
			stmt := spanner.Statement{
				SQL: `UPDATE Notifications SET ReadAt = @now
					WHERE RecipientId = @recipientId AND ReadAt IS NULL`,
				Params: map[string]interface{}{
					"recipientId": recipientID,
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
					stmt := spanner.Statement{
						SQL: `UPDATE Notifications SET ReadAt = @now
							WHERE NotificationId = @notificationId AND RecipientId = @recipientId`,
						Params: map[string]interface{}{
							"notificationId": nid,
							"recipientId":    recipientID,
							"now":            now,
						},
					}
					if _, err := txn.Update(ctx, stmt); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Printf("[NOTIFICATIONS] Mark read failed: %v", err)
				http.Error(w, `{"error":"update_failed"}`, http.StatusInternalServerError)
				return
			}
		}

		InvalidateNotificationInboxCache(ctx, recipientID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "OK"})
	}
}

// ── Send Notification Helper ────────────────────────────────────────────────
// Writes a notification record to Spanner. Used by other backend services.

func InsertNotification(ctx context.Context, client *spanner.Client, recipientID, recipientRole, notifType, title, body, payload, channel string) error {
	nid := newNotificationID()
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("Notifications",
				[]string{"NotificationId", "RecipientId", "RecipientRole", "Type", "Title", "Body", "Payload", "Channel", "CreatedAt"},
				[]interface{}{nid, recipientID, recipientRole, notifType, title, body, payload, channel, spanner.CommitTimestamp},
			),
		})
	})
	if err != nil {
		log.Printf("[NOTIFICATIONS] Insert failed for %s: %v", recipientID, err)
	} else {
		InvalidateNotificationInboxCache(ctx, recipientID)
	}
	return err
}

// InsertNotificationWithCorrelation writes a notification with CorrelationId
// and optional ExpiresAt for lifecycle-managed alerts (e.g. pre-order confirmations).
func InsertNotificationWithCorrelation(ctx context.Context, client *spanner.Client, recipientID, recipientRole, notifType, title, body, payload, channel, correlationID string, expiresAt *time.Time) error {
	nid := newNotificationID()
	cols := []string{"NotificationId", "RecipientId", "RecipientRole", "Type", "Title", "Body", "Payload", "Channel", "CorrelationId", "CreatedAt"}
	vals := []interface{}{nid, recipientID, recipientRole, notifType, title, body, payload, channel, correlationID, spanner.CommitTimestamp}
	if expiresAt != nil {
		cols = append(cols, "ExpiresAt")
		vals = append(vals, *expiresAt)
	}
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("Notifications", cols, vals),
		})
	})
	if err != nil {
		log.Printf("[NOTIFICATIONS] Correlated insert failed for %s (corr: %s): %v", recipientID, correlationID, err)
	} else {
		InvalidateNotificationInboxCache(ctx, recipientID)
	}
	return err
}

// DeleteByCorrelationId removes all unread notifications matching a CorrelationId.
// Used to clear stale "Confirm your order" alerts after the retailer confirms.
func DeleteByCorrelationId(ctx context.Context, client *spanner.Client, correlationID string) (int64, error) {
	var deleted int64
	affectedRecipients := make(map[string]struct{})
	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL: `SELECT NotificationId, RecipientId FROM Notifications
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
			var recipientID string
			if err := row.Columns(&nid, &recipientID); err != nil {
				return err
			}
			keys = append(keys, spanner.Key{nid})
			affectedRecipients[recipientID] = struct{}{}
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
		return deleted, err
	}

	if deleted > 0 {
		for recipientID := range affectedRecipients {
			InvalidateNotificationInboxCache(ctx, recipientID)
		}
	}
	return deleted, err
}
