package order

import (
	"context"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/packages/handoff"
)

func TestUpdateStatusMintsDeliveryTokenOnLoaded(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusPending)}
	svc := NewService(ServiceConfig{
		Repo:    repo,
		Handoff: handoff.New(handoff.Config{LegacyOrderIDFallback: true, Mint: func() string { return "minted-token" }}),
		Now:     func() time.Time { return now },
	})

	_, err := svc.UpdateStatus(context.Background(), auth.Claims{Role: auth.RoleAdmin, Subject: "admin-1"}, "ord-1", UpdateStatusRequest{Status: "LOADED"})
	if err != nil {
		t.Fatalf("update status: %v", err)
	}
	if repo.captured.QRToken != "minted-token" {
		t.Fatalf("stored token=%q want minted-token", repo.captured.QRToken)
	}
}
