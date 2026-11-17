package notifications

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// InboxHandlers serves GET/POST /v1/user/notifications for any role using RecipientIDFromClaims.
type InboxHandlers struct {
	Service *Service
	Log     *slog.Logger
}

// HandleList serves GET /v1/user/notifications.
func (h *InboxHandlers) HandleList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeInboxJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeInboxJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	recipientID := RecipientIDFromClaims(claims)
	if recipientID == "" {
		writeInboxJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	limit, offset := ParseInboxPagination(r)
	if h == nil || h.Service == nil {
		writeInboxJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	notifs, err := h.Service.ListForRecipient(r.Context(), recipientID, limit, offset)
	if err != nil {
		if h.Log != nil {
			h.Log.ErrorContext(r.Context(), "list notifications failed", "recipient_id", recipientID, "err", err)
		}
		writeInboxJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_list_failed"})
		return
	}
	wire := ToInboxWireList(notifs)
	unread, unreadErr := h.Service.UnreadCount(r.Context(), recipientID)
	if unreadErr != nil {
		if h.Log != nil {
			h.Log.ErrorContext(r.Context(), "unread count failed", "recipient_id", recipientID, "err", unreadErr)
		}
		writeInboxJSON(w, http.StatusInternalServerError, map[string]string{"error": "inbox_unread_failed"})
		return
	}
	writeInboxJSON(w, http.StatusOK, map[string]any{
		"notifications": wire,
		"unread_count":  unread,
		"total":         len(wire),
		"limit":         limit,
		"offset":        offset,
		"has_more":      len(wire) == limit,
	})
}

// HandleMarkRead serves POST /v1/user/notifications/read.
func (h *InboxHandlers) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeInboxJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeInboxJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	recipientID := RecipientIDFromClaims(claims)
	if recipientID == "" {
		writeInboxJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	var req MarkReadRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeInboxJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	if h == nil || h.Service == nil {
		writeInboxJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "inbox_unavailable"})
		return
	}
	if err := ApplyMarkRead(r.Context(), h.Service, recipientID, req); err != nil {
		if h.Log != nil {
			h.Log.ErrorContext(r.Context(), "mark notifications read failed", "recipient_id", recipientID, "err", err)
		}
		writeInboxJSON(w, http.StatusInternalServerError, map[string]string{"error": "mark_read_failed"})
		return
	}
	writeInboxJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeInboxJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
