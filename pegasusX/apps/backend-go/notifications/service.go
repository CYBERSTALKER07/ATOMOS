package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
)

// Service implements notification inbox operations.
type Service struct {
	repo  Repository
	cache *cache.Cache
	log   *slog.Logger
}

// NewService creates a notifications service.
func NewService(repo Repository, c *cache.Cache, log *slog.Logger) *Service {
	return &Service{repo: repo, cache: c, log: log}
}

// CreateNotification persists a new notification for a recipient.
func (s *Service) CreateNotification(ctx context.Context, recipientID, recipientRole, eventType, title, body, deepLink string) error {
	return s.CreateNotificationWithMetadata(ctx, recipientID, recipientRole, eventType, title, body, deepLink, nil)
}

// CreateNotificationWithMetadata persists a notification with optional handoff metadata.
func (s *Service) CreateNotificationWithMetadata(ctx context.Context, recipientID, recipientRole, eventType, title, body, deepLink string, handoff *HandoffCardMetadata) error {
	n := Notification{
		NotificationID:  uuid.NewString(),
		RecipientID:     recipientID,
		RecipientRole:   recipientRole,
		EventType:       eventType,
		Title:           title,
		Body:            body,
		DeepLink:        deepLink,
		HandoffMetadata: handoff,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "notifications:"+recipientID)
	}
	return nil
}

// ListForRecipient returns recent notifications with limit and offset pagination.
func (s *Service) ListForRecipient(ctx context.Context, recipientID string, limit, offset int) ([]Notification, error) {
	notifs, err := s.repo.ListByRecipient(ctx, recipientID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	if notifs == nil {
		notifs = []Notification{}
	}
	return notifs, nil
}

// MarkRead marks notification IDs as read and invalidates cache.
func (s *Service) MarkRead(ctx context.Context, recipientID string, notificationIDs []string) error {
	if err := s.repo.MarkRead(ctx, notificationIDs); err != nil {
		return fmt.Errorf("mark read: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "notifications:"+recipientID)
	}
	return nil
}

// MarkAllRead marks every unread notification for a recipient as read.
func (s *Service) MarkAllRead(ctx context.Context, recipientID string) error {
	if err := s.repo.MarkAllRead(ctx, recipientID); err != nil {
		return fmt.Errorf("mark all read: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "notifications:"+recipientID)
	}
	return nil
}

// UnreadCount returns the unread notification count for a recipient.
func (s *Service) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return s.repo.UnreadCount(ctx, recipientID)
}

// IsQuietHour checks if the given time falls within the quiet hours [quietFrom, quietTo).
func IsQuietHour(now time.Time, quietFrom, quietTo string) bool {
	if quietFrom == "" || quietTo == "" {
		return false
	}

	layout := "15:04:05"
	if len(quietFrom) == 5 {
		layout = "15:04"
	}

	from, err := time.Parse(layout, quietFrom)
	if err != nil {
		return false
	}

	layoutTo := "15:04:05"
	if len(quietTo) == 5 {
		layoutTo = "15:04"
	}

	to, err := time.Parse(layoutTo, quietTo)
	if err != nil {
		return false
	}

	nowSecs := now.Hour()*3600 + now.Minute()*60 + now.Second()
	fromSecs := from.Hour()*3600 + from.Minute()*60 + from.Second()
	toSecs := to.Hour()*3600 + to.Minute()*60 + to.Second()

	if fromSecs > toSecs {
		// Crosses midnight
		return nowSecs >= fromSecs || nowSecs < toSecs
	}
	// Same day
	return nowSecs >= fromSecs && nowSecs < toSecs
}

// ShouldSendNotification evaluates if a notification should be sent based on preferences and current time.
func (s *Service) ShouldSendNotification(ctx context.Context, principalID, eventType, channel string, now time.Time) (bool, error) {
	pref, err := s.repo.GetPreference(ctx, principalID, eventType, channel)
	if err != nil {
		return false, fmt.Errorf("get preference: %w", err)
	}

	// Default to sending if no preference is configured
	if pref == nil {
		return true, nil
	}

	if !pref.Enabled {
		return false, nil
	}

	if IsQuietHour(now, pref.QuietFrom, pref.QuietTo) {
		return false, nil
	}

	return true, nil
}
