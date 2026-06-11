package order

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/packages/handoff"
)

func (s *Service) applyHandoffLifecycle(current *Order, previousStatus Status, previousDriverID string) {
	if s == nil || current == nil || s.handoff == nil {
		return
	}
	s.handoff.ApplyTransition(
		&current.QRToken,
		string(previousStatus),
		string(current.Status),
		previousDriverID,
		strings.TrimSpace(current.DriverID),
	)
}

func (s *Service) validateDeliveryToken(orderRecord Order, token string) error {
	if s.handoff == nil {
		s.handoff = handoff.FromEnv()
	}
	return s.handoff.Validate(orderRecord.OrderID, orderRecord.QRToken, token)
}

func (s *Service) publicDeliveryToken(orderRecord Order) string {
	if s.handoff == nil {
		s.handoff = handoff.FromEnv()
	}
	return s.handoff.PublicToken(orderRecord.OrderID, orderRecord.QRToken, string(orderRecord.Status))
}

// ResolveDeliveryToken returns the active public token for retailer telemetry fan-out.
func (s *Service) ResolveDeliveryToken(ctx context.Context, orderID string) (string, error) {
	orderRecord, found, err := s.repo.GetOrder(ctx, strings.TrimSpace(orderID))
	if err != nil {
		return "", fmt.Errorf("resolve delivery token %s: %w", orderID, err)
	}
	if !found {
		return "", ErrOrderNotFound
	}
	token := s.publicDeliveryToken(orderRecord)
	if strings.TrimSpace(token) == "" {
		return "", errors.New("delivery token not active")
	}
	return token, nil
}
