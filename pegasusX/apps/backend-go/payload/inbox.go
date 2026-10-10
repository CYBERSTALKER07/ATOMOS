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

// HandleUserNotifications is not mounted. Live GET is notifications.InboxHandlers.HandleList
// (main.go last registration). Kept fail-closed so a remount cannot return empty [] as success.
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
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	notifs, listErr := s.notifSvc.ListForRecipient(r.Context(), recipientID, limit, offset)
	if listErr != nil {
		s.log.ErrorContext(r.Context(), "payload list notifications failed", "err", listErr, "recipient_id", recipientID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_list_failed"})
		return
	}
	wire := notifications.ToInboxWireList(notifs)
	unread, unreadErr := s.notifSvc.UnreadCount(r.Context(), recipientID)
	if unreadErr != nil {
		s.log.ErrorContext(r.Context(), "payload unread count failed", "err", unreadErr, "recipient_id", recipientID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_unread_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"notifications": wire,
		"unread_count":  unread,
		"total":         len(wire),
		"limit":         limit,
		"offset":        offset,
		"has_more":      len(wire) == limit,
	})
}

// HandleMarkNotificationsRead is not mounted. Live POST is notifications.InboxHandlers.HandleMarkRead.
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
	if s.notifSvc == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	if markErr := notifications.ApplyMarkRead(r.Context(), s.notifSvc, recipientID, req); markErr != nil {
		s.log.ErrorContext(r.Context(), "payload mark notifications read failed", "err", markErr, "recipient_id", recipientID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "mark_read_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
