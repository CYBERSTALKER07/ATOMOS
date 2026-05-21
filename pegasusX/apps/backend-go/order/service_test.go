package order

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type testTxnBuffer struct {
	events []outbox.Event
}

func (b *testTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

type testRepo struct {
	order          Order
	found          bool
	getErr         error
	createErr      error
	updateErr      error
	createCalls    int
	updateCalls    int
	created        Order
	captured       Order
	bufferedEvents int
}

func (r *testRepo) CreateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error {
	if r.createErr != nil {
		return r.createErr
	}
	r.createCalls++
	r.created = o
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
	}
	return nil
}

type testWarehouseResolver struct {
	warehouseID string
	err         error
	calls       int
}

func (r *testWarehouseResolver) ResolveNearestWarehouseID(_ context.Context, _ string, _, _ float64) (string, error) {
	r.calls++
	if r.err != nil {
		return "", r.err
	}
	return r.warehouseID, nil
}

func (r *testRepo) UpdateOrder(ctx context.Context, o Order, emit func(outbox.TxnBuffer) error) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updateCalls++
	r.captured = o
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
	}
	return nil
}

func (r *testRepo) GetOrder(_ context.Context, _ string) (Order, bool, error) {
	if r.getErr != nil {
		return Order{}, false, r.getErr
	}
	if !r.found {
		return Order{}, false, nil
	}
	return r.order, true, nil
}

func newTestService(repo Repository, now time.Time) *Service {
	return newTestServiceWithResolver(repo, nil, now)
}

func newTestServiceWithResolver(repo Repository, resolver WarehouseResolver, now time.Time) *Service {
	return NewService(ServiceConfig{
		Repo:       repo,
		Warehouse:  resolver,
		SupplierID: "sup-1",
		Currency:   "UZS",
		Now: func() time.Time {
			return now
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
}

func TestValidateStatusTransitionMatrix(t *testing.T) {
	cases := []struct {
		name    string
		current Status
		next    Status
		wantErr bool
	}{
		{name: "pending to loaded", current: StatusPending, next: StatusLoaded},
		{name: "pending to cancelled", current: StatusPending, next: StatusCancelled},
		{name: "loaded to in transit", current: StatusLoaded, next: StatusInTransit},
		{name: "loaded to cancelled", current: StatusLoaded, next: StatusCancelled},
		{name: "in transit to arrived", current: StatusInTransit, next: StatusArrived},
		{name: "arrived to completed", current: StatusArrived, next: StatusCompleted},
		{name: "no-op allowed", current: StatusCompleted, next: StatusCompleted},
		{name: "pending to completed denied", current: StatusPending, next: StatusCompleted, wantErr: true},
		{name: "loaded to arrived denied", current: StatusLoaded, next: StatusArrived, wantErr: true},
		{name: "in transit to completed denied", current: StatusInTransit, next: StatusCompleted, wantErr: true},
		{name: "completed to cancelled denied", current: StatusCompleted, next: StatusCancelled, wantErr: true},
		{name: "cancelled to pending denied", current: StatusCancelled, next: StatusPending, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateStatusTransition(tc.current, tc.next)
			if tc.wantErr {
				if !errors.Is(err, ErrInvalidStatusTransition) {
					t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}

func TestServiceCreateAssignsNearestWarehouse(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	resolver := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := newTestServiceWithResolver(repo, resolver, now)

	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 2, UnitPrice: 500}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if resolver.calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", resolver.calls)
	}
	if repo.createCalls != 1 {
		t.Fatalf("create calls = %d, want 1", repo.createCalls)
	}
	if repo.created.WarehouseID != "wh-1" {
		t.Fatalf("persisted warehouse_id = %q, want %q", repo.created.WarehouseID, "wh-1")
	}
	if resp.WarehouseID != "wh-1" {
		t.Fatalf("response warehouse_id = %q, want %q", resp.WarehouseID, "wh-1")
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("buffered events = %d, want 1", repo.bufferedEvents)
	}
}

func TestServiceCreateFailsWhenWarehouseResolutionFails(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	resolver := &testWarehouseResolver{err: errors.New("resolver_down")}
	svc := newTestServiceWithResolver(repo, resolver, now)

	_, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 1, UnitPrice: 250}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if err == nil {
		t.Fatal("expected error when resolver fails")
	}
	if repo.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", repo.createCalls)
	}
}

func TestServiceUpdateStatusOrderNotFound(t *testing.T) {
	repo := &testRepo{found: false}
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.UpdateStatus(context.Background(), auth.Claims{Role: auth.RoleAdmin, Subject: "admin-1"}, "ord-1", UpdateStatusRequest{Status: "LOADED"})
	if !errors.Is(err, ErrOrderNotFound) {
		t.Fatalf("expected ErrOrderNotFound, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no update calls, got %d", repo.updateCalls)
	}
}

func TestServiceUpdateStatusRetailerScopeEnforced(t *testing.T) {
	baseOrder := Order{
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		Status:     StatusPending,
		Version:    3,
		UpdatedAt:  time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
	}

	t.Run("retailer cannot move to non-cancelled state", func(t *testing.T) {
		repo := &testRepo{found: true, order: baseOrder}
		svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

		_, err := svc.UpdateStatus(context.Background(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1"}, "ord-1", UpdateStatusRequest{Status: "LOADED"})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
		if repo.updateCalls != 0 {
			t.Fatalf("expected no update calls, got %d", repo.updateCalls)
		}
	})

	t.Run("retailer cannot cancel another retailer order", func(t *testing.T) {
		repo := &testRepo{found: true, order: baseOrder}
		svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

		_, err := svc.UpdateStatus(context.Background(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-2"}, "ord-1", UpdateStatusRequest{Status: "CANCELLED"})
		if !errors.Is(err, ErrOrderForbidden) {
			t.Fatalf("expected ErrOrderForbidden, got %v", err)
		}
		if repo.updateCalls != 0 {
			t.Fatalf("expected no update calls, got %d", repo.updateCalls)
		}
	})
}

func TestServiceUpdateStatusInvalidTransition(t *testing.T) {
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusInTransit,
			Version:    9,
			UpdatedAt:  time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		},
	}
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.UpdateStatus(context.Background(), auth.Claims{Role: auth.RoleAdmin, Subject: "admin-1"}, "ord-1", UpdateStatusRequest{Status: "COMPLETED"})
	if !errors.Is(err, ErrInvalidStatusTransition) {
		t.Fatalf("expected ErrInvalidStatusTransition, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("expected no update calls, got %d", repo.updateCalls)
	}
}

func TestServiceUpdateStatusAdminSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		found: true,
		order: Order{
			OrderID:    "ord-1",
			SupplierID: "sup-1",
			RetailerID: "ret-1",
			Status:     StatusPending,
			Version:    7,
			UpdatedAt:  time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		},
	}
	svc := newTestService(repo, now)

	resp, err := svc.UpdateStatus(context.Background(), auth.Claims{
		Role:       auth.RoleAdmin,
		Subject:    "admin-1",
		SupplierID: "sup-1",
	}, "ord-1", UpdateStatusRequest{Status: "LOADED", Reason: "validated"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PreviousStatus != StatusPending {
		t.Fatalf("expected previous status %s, got %s", StatusPending, resp.PreviousStatus)
	}
	if resp.Status != StatusLoaded {
		t.Fatalf("expected status %s, got %s", StatusLoaded, resp.Status)
	}
	if resp.Version != 8 {
		t.Fatalf("expected version 8, got %d", resp.Version)
	}
	if resp.UpdatedAt != now.Format(time.RFC3339Nano) {
		t.Fatalf("unexpected updated_at: %s", resp.UpdatedAt)
	}
	if resp.EventType != events.EventOrderStatusChanged {
		t.Fatalf("expected event type %s, got %s", events.EventOrderStatusChanged, resp.EventType)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("expected one update call, got %d", repo.updateCalls)
	}
	if repo.captured.Status != StatusLoaded {
		t.Fatalf("expected persisted status %s, got %s", StatusLoaded, repo.captured.Status)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("expected one buffered outbox event, got %d", repo.bufferedEvents)
	}
}
