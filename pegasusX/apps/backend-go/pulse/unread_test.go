package pulse

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
)

type unreadFailRepo struct{}

func (unreadFailRepo) Create(context.Context, notifications.Notification) error { return nil }
func (unreadFailRepo) ListByRecipient(context.Context, string, int, int) ([]notifications.Notification, error) {
	return []notifications.Notification{}, nil
}
func (unreadFailRepo) MarkRead(context.Context, []string) error  { return nil }
func (unreadFailRepo) MarkAllRead(context.Context, string) error { return nil }
func (unreadFailRepo) UnreadCount(context.Context, string) (int64, error) {
	return 0, errors.New("spanner_unavailable")
}
func (unreadFailRepo) GetPreference(context.Context, string, string, string) (*notifications.NotificationPreference, error) {
	return nil, nil
}
func (unreadFailRepo) UpsertPreference(context.Context, notifications.NotificationPreference) error {
	return nil
}

func TestListForRecipient_UnreadError(t *testing.T) {
	svc := NewService(Config{Notifications: notifications.NewService(unreadFailRepo{}, nil, nil)})
	_, err := svc.ListForRecipient(context.Background(), "ret-1", "RETAILER", "ret-1", 10)
	if err == nil || !strings.Contains(err.Error(), "pulse unread") {
		t.Fatalf("err=%v", err)
	}
}
