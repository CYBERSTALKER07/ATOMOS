package claims

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestEvaluateClaimEligibility_OpenWindow(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	o := OrderSnapshot{
		OrderID:    "ord-1",
		RetailerID: "ret-1",
		Status:     OrderStatusCompleted,
		UpdatedAt:  now.Add(-6 * time.Hour),
	}
	elig := EvaluateClaimEligibility(o, now, 48*time.Hour)
	if !elig.Eligible {
		t.Fatalf("expected eligible, got %+v", elig)
	}
	if elig.EndsAt == nil {
		t.Fatal("ends_at nil")
	}
	if elig.WindowHours != 48 {
		t.Fatalf("window_hours=%d", elig.WindowHours)
	}
	if elig.HoursRemaining <= 0 || elig.HoursRemaining > 48 {
		t.Fatalf("hours_remaining=%v", elig.HoursRemaining)
	}
	if elig.Reason != "" {
		t.Fatalf("reason=%q", elig.Reason)
	}
	if len(elig.PhotoRequiredTypes) != 4 {
		t.Fatalf("photo types=%v", elig.PhotoRequiredTypes)
	}
}

func TestEvaluateClaimEligibility_Expired(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	o := OrderSnapshot{
		OrderID:   "ord-1",
		Status:    OrderStatusCompleted,
		UpdatedAt: now.Add(-72 * time.Hour),
	}
	elig := EvaluateClaimEligibility(o, now, 48*time.Hour)
	if elig.Eligible {
		t.Fatalf("expected closed: %+v", elig)
	}
	if elig.Reason != "claim_window_expired" {
		t.Fatalf("reason=%q", elig.Reason)
	}
	if elig.EndsAt == nil {
		t.Fatal("ends_at should still be set")
	}
	if elig.HoursRemaining != 0 {
		t.Fatalf("hours_remaining=%v", elig.HoursRemaining)
	}
}

func TestEvaluateClaimEligibility_NotCompleted(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	o := OrderSnapshot{
		OrderID:   "ord-1",
		Status:    "IN_TRANSIT",
		UpdatedAt: now,
	}
	elig := EvaluateClaimEligibility(o, now, 48*time.Hour)
	if elig.Eligible {
		t.Fatal("expected ineligible")
	}
	if elig.Reason != "order_not_completed" {
		t.Fatalf("reason=%q", elig.Reason)
	}
	if elig.EndsAt != nil {
		t.Fatalf("ends_at=%v want nil", *elig.EndsAt)
	}
}

func TestGetClaimEligibility_ForbiddenOtherRetailer(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo: NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: OrderSnapshot{
			OrderID: "ord-1", RetailerID: "ret-1", Status: OrderStatusCompleted, UpdatedAt: now,
		}},
		Now:    func() time.Time { return now },
		Window: 48 * time.Hour,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	_, err := svc.GetClaimEligibility(context.Background(), auth.Claims{
		Subject: "ret-other", Role: auth.RoleRetailer,
	}, "ord-1")
	if err != ErrForbidden {
		t.Fatalf("got %v want forbidden", err)
	}
}

func TestGetClaimEligibility_OwnerOK(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	svc := NewService(Config{
		Repo: NewMemoryRepository(),
		Orders: fakeOrders{ok: true, o: OrderSnapshot{
			OrderID: "ord-1", RetailerID: "ret-1", Status: OrderStatusCompleted, UpdatedAt: now.Add(-1 * time.Hour),
		}},
		Now:    func() time.Time { return now },
		Window: 48 * time.Hour,
		Log:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	elig, err := svc.GetClaimEligibility(context.Background(), auth.Claims{
		Subject: "ret-1", Role: auth.RoleRetailer,
	}, "ord-1")
	if err != nil {
		t.Fatal(err)
	}
	if !elig.Eligible {
		t.Fatalf("expected eligible: %+v", elig)
	}
}
