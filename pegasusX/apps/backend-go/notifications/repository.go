package notifications

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Notification mirrors the Notifications Spanner row.
type Notification struct {
	NotificationID string    `json:"notification_id"`
	RecipientID    string    `json:"recipient_id"`
	RecipientRole  string    `json:"recipient_role"`
	EventType      string    `json:"event_type"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	DeepLink       string    `json:"deep_link"`
	IsRead         bool      `json:"is_read"`
	CreatedAt      time.Time `json:"created_at"`
}

// Repository defines the data access contract for notifications.
type Repository interface {
	Create(ctx context.Context, n Notification) error
	ListByRecipient(ctx context.Context, recipientID string, limit int) ([]Notification, error)
	MarkRead(ctx context.Context, notificationIDs []string) error
	UnreadCount(ctx context.Context, recipientID string) (int64, error)
}

// SpannerRepository implements Repository backed by Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed notifications repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

var notifColumns = []string{
	"NotificationId", "RecipientId", "RecipientRole", "EventType",
	"Title", "Body", "DeepLink", "IsRead", "CreatedAt",
}

func scanNotification(row *spanner.Row) (Notification, error) {
	var n Notification
	var body, deepLink spanner.NullString
	if err := row.Columns(&n.NotificationID, &n.RecipientID, &n.RecipientRole, &n.EventType,
		&n.Title, &body, &deepLink, &n.IsRead, &n.CreatedAt); err != nil {
		return Notification{}, fmt.Errorf("scan notification: %w", err)
	}
	n.Body = body.StringVal
	n.DeepLink = deepLink.StringVal
	return n, nil
}

// Create inserts a new notification row.
func (r *SpannerRepository) Create(ctx context.Context, n Notification) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertMap("Notifications", map[string]any{
			"NotificationId": n.NotificationID,
			"RecipientId":    n.RecipientID,
			"RecipientRole":  n.RecipientRole,
			"EventType":      n.EventType,
			"Title":          n.Title,
			"Body":           spanner.NullString{StringVal: n.Body, Valid: n.Body != ""},
			"DeepLink":       spanner.NullString{StringVal: n.DeepLink, Valid: n.DeepLink != ""},
			"IsRead":         false,
			"CreatedAt":      spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	if err != nil {
		return fmt.Errorf("create notification %s: %w", n.NotificationID, err)
	}
	return nil
}

// ListByRecipient returns the most recent notifications for a recipient.
func (r *SpannerRepository) ListByRecipient(ctx context.Context, recipientID string, limit int) ([]Notification, error) {
	if limit <= 0 {
		limit = 50
	}
	stmt := spanner.Statement{
		SQL:    "SELECT NotificationId, RecipientId, RecipientRole, EventType, Title, Body, DeepLink, IsRead, CreatedAt FROM Notifications@{FORCE_INDEX=Idx_Notifications_ByRecipientCreated} WHERE RecipientId = @rid ORDER BY CreatedAt DESC LIMIT @lim",
		Params: map[string]any{"rid": recipientID, "lim": int64(limit)},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(5 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()

	var notifs []Notification
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list notifications for %s: %w", recipientID, err)
		}
		n, scanErr := scanNotification(row)
		if scanErr != nil {
			return nil, scanErr
		}
		notifs = append(notifs, n)
	}
	return notifs, nil
}

// MarkRead sets IsRead=true on the given notification IDs.
func (r *SpannerRepository) MarkRead(ctx context.Context, notificationIDs []string) error {
	if len(notificationIDs) == 0 {
		return nil
	}
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(notificationIDs))
		for _, nid := range notificationIDs {
			m := spanner.UpdateMap("Notifications", map[string]any{
				"NotificationId": nid,
				"IsRead":         true,
			})
			mutations = append(mutations, m)
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("mark notifications read: %w", err)
	}
	return nil
}

// UnreadCount returns the count of unread notifications for a recipient.
func (r *SpannerRepository) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	stmt := spanner.Statement{
		SQL:    "SELECT COUNT(*) FROM Notifications@{FORCE_INDEX=Idx_Notifications_ByRecipientUnread} WHERE RecipientId = @rid AND IsRead = FALSE",
		Params: map[string]any{"rid": recipientID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(5 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0, fmt.Errorf("unread count for %s: %w", recipientID, err)
	}
	var count int64
	if err := row.Columns(&count); err != nil {
		return 0, fmt.Errorf("scan unread count: %w", err)
	}
	return count, nil
}
