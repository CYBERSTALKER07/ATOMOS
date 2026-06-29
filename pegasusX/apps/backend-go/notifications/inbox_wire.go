package notifications

import (
	"context"
	"time"
)

// InboxItemWire is the canonical inbox JSON shape for mobile and web clients.
// Mobile decodes `id`; desktop may use `notification_id` — both are populated.
type InboxItemWire struct {
	ID              string               `json:"id"`
	NotificationID  string               `json:"notification_id"`
	Type            string               `json:"type"`
	Title           string               `json:"title"`
	Body            string               `json:"body"`
	Payload         string               `json:"payload,omitempty"`
	Channel         string               `json:"channel"`
	ReadAt          string               `json:"read_at,omitempty"`
	CreatedAt       string               `json:"created_at"`
	HandoffMetadata *HandoffCardMetadata `json:"handoff_metadata,omitempty"`
}

// ToInboxWire maps a Spanner notification row to the client inbox DTO.
func ToInboxWire(n Notification) InboxItemWire {
	readAt := ""
	if n.IsRead {
		readAt = n.CreatedAt.UTC().Format(time.RFC3339Nano)
	}
	payload := n.DeepLink
	if payload == "" {
		payload = n.Body
	}
	return InboxItemWire{
		ID:              n.NotificationID,
		NotificationID:  n.NotificationID,
		Type:            n.EventType,
		Title:           n.Title,
		Body:            n.Body,
		Payload:         payload,
		Channel:         "PUSH",
		ReadAt:          readAt,
		CreatedAt:       n.CreatedAt.UTC().Format(time.RFC3339Nano),
		HandoffMetadata: n.HandoffMetadata,
	}
}

// ToInboxWireList maps a slice of notifications to wire DTOs.
func ToInboxWireList(notifs []Notification) []InboxItemWire {
	if len(notifs) == 0 {
		return []InboxItemWire{}
	}
	out := make([]InboxItemWire, len(notifs))
	for i := range notifs {
		out[i] = ToInboxWire(notifs[i])
	}
	return out
}

// MarkReadRequest is the POST /v1/user/notifications/read body.
type MarkReadRequest struct {
	NotificationIDs []string `json:"notification_ids"`
	MarkAll         *bool    `json:"mark_all"`
}

// InboxMarkReader marks notification inbox rows read.
type InboxMarkReader interface {
	MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error
	MarkAllRead(ctx context.Context, recipientID string) error
}

// ApplyMarkRead applies mark-by-id or mark-all for a recipient inbox.
func ApplyMarkRead(ctx context.Context, svc InboxMarkReader, recipientID string, req MarkReadRequest) error {
	if svc == nil || recipientID == "" {
		return nil
	}
	if req.MarkAll != nil && *req.MarkAll {
		return svc.MarkAllRead(ctx, recipientID)
	}
	if len(req.NotificationIDs) > 0 {
		return svc.MarkRead(ctx, recipientID, req.NotificationIDs)
	}
	return nil
}

// ToInboxWireFromAnyList converts adapter []any rows to inbox wire DTOs.
func ToInboxWireFromAnyList(items []any) []InboxItemWire {
	if len(items) == 0 {
		return []InboxItemWire{}
	}
	out := make([]InboxItemWire, 0, len(items))
	for _, item := range items {
		switch row := item.(type) {
		case Notification:
			out = append(out, ToInboxWire(row))
		case InboxItemWire:
			out = append(out, row)
		}
	}
	return out
}
