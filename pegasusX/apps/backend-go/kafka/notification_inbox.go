package kafka

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

// persistInbox writes a durable notification row for inbox-capable roles.
func (d *NotificationDispatcher) persistInbox(ctx context.Context, recipientID, role string, payload []byte) {
	if d == nil || d.deps.Inbox == nil || recipientID == "" || role == "" {
		return
	}
	var envelope struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(payload, &envelope); err != nil || envelope.Type == "" {
		return
	}
	if !notifications.ShouldPersistInboxEvent(envelope.Type) {
		return
	}
	formatted := notifications.FormatFromEvent(envelope.Type, payload)
	handoff := notifications.BuildHandoffMetadata(envelope.Type, payload)
	if err := d.deps.Inbox.CreateNotificationWithMetadata(
		ctx,
		recipientID,
		role,
		envelope.Type,
		formatted.Title,
		formatted.Body,
		formatted.DeepLink,
		handoff,
	); err != nil {
		slog.ErrorContext(ctx, "notification inbox persist failed",
			"recipient_id", recipientID,
			"role", role,
			"event_type", envelope.Type,
			"err", err,
		)
	}
}
