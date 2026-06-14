package payload

import (
	"encoding/json"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

func inboxRecipientID(r *http.Request) string {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		return ""
	}
	return notifications.RecipientIDFromClaims(claims)
}

// HandleUserNotifications serves GET /v1/user/notifications for PAYLOAD role clients.
func (s *Service) HandleUserNotifications(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	recipientID := inboxRecipientID(r)
	if recipientID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	limit, offset := notifications.ParseInboxPagination(r)
	if s.notifSvc == nil {
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []notifications.InboxItemWire{}, "unread_count": 0, "total": 0, "limit": limit, "offset": offset})
		return
	}
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), recipientID, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "payload list notifications failed", "err", listErr, "recipient_id", recipientID)
		writeJSON(w, http.StatusOK, map[string]any{"notifications": []notifications.InboxItemWire{}, "unread_count": 0, "total": 0, "limit": limit, "offset": offset})
		return
	}
	wire := notifications.ToInboxWireList(notifs)
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
	recipientID := inboxRecipientID(r)
	if recipientID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req notifications.MarkReadRequest
	if decErr := json.NewDecoder(r.Body).Decode(&req); decErr != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if markErr := notifications.ApplyMarkRead(r.Context(), s.notifSvc, recipientID, req); markErr != nil {
		s.log.ErrorContext(r.Context(), "payload mark notifications read failed", "err", markErr, "recipient_id", recipientID)
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
