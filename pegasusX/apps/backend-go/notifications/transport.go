package notifications

import (
	"context"
	"log/slog"
)

// Transport defines a notification delivery channel.
type Transport interface {
	Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error
}

// LogTransport is a fallback that logs notifications without delivering them.
// Used when WS/FCM/APNs transports are not yet wired.
type LogTransport struct {
	Log *slog.Logger
}

// Deliver logs the notification delivery attempt.
func (t *LogTransport) Deliver(ctx context.Context, recipientID, recipientRole string, notif FormattedNotification) error {
	t.Log.InfoContext(ctx, "notification delivery (log transport)",
		"recipient_id", recipientID,
		"recipient_role", recipientRole,
		"title", notif.Title,
		"priority", notif.Priority,
	)
	return nil
}
