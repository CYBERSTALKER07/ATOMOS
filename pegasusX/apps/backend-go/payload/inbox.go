package payload

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

type notificationItemWire struct {
	NotificationID string `json:"notification_id"`
	Type           string `json:"type"`
	Title          string `json:"title"`
	Body           string `json:"body"`
	Payload        string `json:"payload,omitempty"`
	Channel        string `json:"channel"`
	ReadAt         string `json:"read_at,omitempty"`
	CreatedAt      string `json:"created_at"`
}

func notificationToWire(n notifications.Notification) notificationItemWire {
	readAt := ""
	if n.IsRead {
		readAt = n.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	return notificationItemWire{
		NotificationID: n.NotificationID,
		Type:           n.EventType,
		Title:          n.Title,
		Body:           n.Body,
		Channel:        "PUSH",
		ReadAt:         readAt,
		CreatedAt:      n.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func payloaderIDFromRequest(r *http.Request) string {
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Subject == "" {
		return ""
	}
	return claims.Subject
}

// HandleUserNotifications serves GET /v1/user/notifications for PAYLOAD role clients.
func (s *Service) HandleUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	recipientID := payloaderIDFromRequest(r)
	if recipientID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	limit, offset := notifications.ParseInboxPagination(r)
	if s.notifSvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []notificationItemWire{}, "unread_count": 0, "total": 0, "limit": limit, "offset": offset})
		return
	}
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), recipientID, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "payload list notifications failed", "err", listErr, "recipient_id", recipientID)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []notificationItemWire{}, "unread_count": 0, "total": 0, "limit": limit, "offset": offset})
		return
	}
	wire := make([]notificationItemWire, len(notifs))
	for i := range notifs {
		wire[i] = notificationToWire(notifs[i])
	}
	unread, _ := s.notifSvc.UnreadCount(r.Context(), recipientID)
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": wire,
		"unread_count":  unread,
		"total":         len(wire),
		"limit":         limit,
		"offset":        offset,
		"has_more":      len(wire) == limit,
	})
}

// HandleMarkNotificationsRead serves POST /v1/user/notifications/read.
func (s *Service) HandleMarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	recipientID := payloaderIDFromRequest(r)
	if recipientID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req struct {
		NotificationIDs []string `json:"notification_ids"`
		MarkAll         *bool    `json:"mark_all"`
	}
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if s.notifSvc != nil && len(req.NotificationIDs) > 0 {
		if markErr := s.notifSvc.MarkRead(r.Context(), recipientID, req.NotificationIDs); markErr != nil {
			s.log.ErrorContext(r.Context(), "payload mark notifications read failed", "err", markErr, "recipient_id", recipientID)
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleDeviceTokenNoop accepts FCM registration from payload clients when push is not wired.
func (s *Service) HandleDeviceTokenNoop(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost, http.MethodDelete:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}
