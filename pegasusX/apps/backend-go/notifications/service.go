package notifications

import (
	"context"
	"fmt"
	"log/slog"

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
	n := Notification{
		NotificationID: uuid.NewString(),
		RecipientID:    recipientID,
		RecipientRole:  recipientRole,
		EventType:      eventType,
		Title:          title,
		Body:           body,
		DeepLink:       deepLink,
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "notifications:"+recipientID)
	}
	return nil
}

// ListForRecipient returns recent notifications with a configurable limit.
func (s *Service) ListForRecipient(ctx context.Context, recipientID string, limit int) ([]Notification, error) {
	notifs, err := s.repo.ListByRecipient(ctx, recipientID, limit)
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

// UnreadCount returns the unread notification count for a recipient.
func (s *Service) UnreadCount(ctx context.Context, recipientID string) (int64, error) {
	return s.repo.UnreadCount(ctx, recipientID)
}
