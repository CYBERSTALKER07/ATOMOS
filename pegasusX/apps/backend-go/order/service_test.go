package order

import (
	"context"
	"cloud.google.com/go/spanner"
	"errors"
	"io"
	"log/slog"
	"strings"
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
	order               Order
	found               bool
	getErr              error
	createErr           error
	updateErr           error
	createCalls         int
	updateCalls         int
	created             Order
	captured            Order
	bufferedEvents      int
	lastEvents          []outbox.Event
	lastProofs          []DeliveryProofArtifact
	retailerWindowOpen  string
	retailerWindowClose string
	conditionReports    []ConditionReport
	createConditionErr  error
	listConditionErr    error
	// fiscalAttempts mirrors OrderFiscalReceipts for worker idempotency tests.
	fiscalAttempts map[string]FiscalReceiptRow
}

func (r *testRepo) CreateOrder(ctx context.Context, o *Order, emit func(outbox.TxnBuffer) error) error {
	if r.createErr != nil {
		return r.createErr
	}
	if o == nil {
		return errors.New("nil order")
	}
	r.createCalls++
	if r.retailerWindowOpen != "" || r.retailerWindowClose != "" {
		if err := SnapshotReceivingWindowsOnOrder(o, r.retailerWindowOpen, r.retailerWindowClose); err != nil {
			return err
		}
	}
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
		r.lastEvents = append(r.lastEvents, buf.events...)
	}
	r.created = *o
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

func (r *testRepo) UpdateOrder(ctx context.Context, o Order, proofs []DeliveryProofArtifact, emit func(outbox.TxnBuffer) error) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updateCalls++
	r.captured = o
	r.order = o
	r.lastProofs = append([]DeliveryProofArtifact(nil), proofs...)
	if r.fiscalAttempts == nil {
		r.fiscalAttempts = make(map[string]FiscalReceiptRow)
	}
	for _, fr := range o.PendingFiscalReceipts {
		key := fr.OrderID + ":" + fr.AttemptID
		r.fiscalAttempts[key] = fr
	}
	if o.FiscalReceiptUpdate != nil {
		u := *o.FiscalReceiptUpdate
		r.fiscalAttempts[u.OrderID+":"+u.AttemptID] = u
	}
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
		r.lastEvents = append(r.lastEvents, buf.events...)
	}
	return nil
}

func (r *testRepo) ClearBackorder(ctx context.Context, orderID string, emit func(outbox.TxnBuffer) error) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updateCalls++
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
		r.lastEvents = append(r.lastEvents, buf.events...)
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

func (r *testRepo) GetFiscalByReceiptID(_ context.Context, receiptID string) (FiscalReceiptRow, bool, error) {
	receiptID = strings.TrimSpace(receiptID)
	if receiptID == "" {
		return FiscalReceiptRow{}, false, nil
	}
	if r.fiscalAttempts != nil {
		for _, fr := range r.fiscalAttempts {
			if fr.FiscalReceiptID == receiptID {
				return fr, true, nil
			}
		}
	}
	if r.captured.FiscalReceiptUpdate != nil && r.captured.FiscalReceiptUpdate.FiscalReceiptID == receiptID {
		return *r.captured.FiscalReceiptUpdate, true, nil
	}
	return FiscalReceiptRow{}, false, nil
}

func (r *testRepo) GetFiscalAttempt(_ context.Context, orderID, attemptID string) (FiscalReceiptRow, bool, error) {
	if r.fiscalAttempts != nil {
		if fr, ok := r.fiscalAttempts[orderID+":"+attemptID]; ok {
			return fr, true, nil
		}
	}
	if r.captured.FiscalReceiptUpdate != nil {
		u := *r.captured.FiscalReceiptUpdate
		if u.OrderID == orderID && u.AttemptID == attemptID {
			return u, true, nil
		}
	}
	for _, fr := range r.captured.PendingFiscalReceipts {
		if fr.OrderID == orderID && fr.AttemptID == attemptID {
			return fr, true, nil
		}
	}
	return FiscalReceiptRow{}, false, nil
}

func (r *testRepo) CountFiscalAttemptsByStatus(_ context.Context, orderID, status string) (int64, error) {
	var n int64
	if r.fiscalAttempts != nil {
		for _, fr := range r.fiscalAttempts {
			if fr.OrderID == orderID && fr.Status == status {
				n++
			}
		}
		return n, nil
	}
	if r.captured.FiscalReceiptUpdate != nil && r.captured.FiscalReceiptUpdate.Status == status {
		n++
	}
	for _, fr := range r.captured.PendingFiscalReceipts {
		if fr.Status == status {
			n++
		}
	}
	return n, nil
}

func (r *testRepo) CreateConditionReport(_ context.Context, report ConditionReport, emit func(outbox.TxnBuffer) error) error {
	if r.createConditionErr != nil {
		return r.createConditionErr
	}
	r.conditionReports = append(r.conditionReports, report)
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.bufferedEvents += len(buf.events)
		r.lastEvents = append(r.lastEvents, buf.events...)
	}
	return nil
}

func (r *testRepo) ListConditionReports(_ context.Context, _ string) ([]ConditionReport, error) {
	if r.listConditionErr != nil {
		return nil, r.listConditionErr
	}
	return append([]ConditionReport(nil), r.conditionReports...), nil
}

func (r *testRepo) FindSiblingDriversForOrder(ctx context.Context, orderID string) ([]string, error) {
	return nil, nil
}

func (r *testRepo) ListRetailerOrders(_ context.Context, retailerID string, limit int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListWarehouseOrdersByDeliveryWindow(_ context.Context, warehouseID string, from, to time.Time, limit int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListOrdersByStatus(ctx context.Context, supplierID string, status string, limit int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListBackorderedOrders(ctx context.Context, limit int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListDueAutoConfirmOrders(_ context.Context, before time.Time, limit int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListWarehousePreorders(_ context.Context, _ string, _, _ int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
}

func (r *testRepo) ListOrdersForStockCommitment(_ context.Context, _ string, _ int) ([]Order, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	return []Order{r.order}, nil
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
		{name: "arrived to awaiting payment", current: StatusArrived, next: StatusAwaitingPayment},
		{name: "arrived to pending cash collection", current: StatusArrived, next: StatusPendingCashCollection},
		{name: "arrived to completed denied (fiscal hard-gate)", current: StatusArrived, next: StatusCompleted, wantErr: true},
		{name: "awaiting payment to fiscalizing", current: StatusAwaitingPayment, next: StatusFiscalizing},
		{name: "pending cash collection to fiscalizing", current: StatusPendingCashCollection, next: StatusFiscalizing},
		{name: "awaiting payment to completed denied", current: StatusAwaitingPayment, next: StatusCompleted, wantErr: true},
		{name: "pending cash collection to completed denied", current: StatusPendingCashCollection, next: StatusCompleted, wantErr: true},
		{name: "fiscalizing to completed", current: StatusFiscalizing, next: StatusCompleted},
		{name: "fiscalizing to fiscal failed", current: StatusFiscalizing, next: StatusFiscalFailed},
		{name: "fiscal failed to fiscalizing", current: StatusFiscalFailed, next: StatusFiscalizing},
		{name: "no-op allowed", current: StatusCompleted, next: StatusCompleted},
		{name: "pending to completed denied", current: StatusPending, next: StatusCompleted, wantErr: true},
		{name: "loaded to arrived denied", current: StatusLoaded, next: StatusArrived, wantErr: true},
		{name: "in transit to completed denied", current: StatusInTransit, next: StatusCompleted, wantErr: true},
		{name: "awaiting payment to arrived denied", current: StatusAwaitingPayment, next: StatusArrived, wantErr: true},
		{name: "completed to cancelled denied", current: StatusCompleted, next: StatusCancelled, wantErr: true},
		{name: "cancelled to pending denied", current: StatusCancelled, next: StatusPending, wantErr: true},
		{name: "scheduled to pending", current: StatusScheduled, next: StatusPending},
		{name: "scheduled to auto accepted", current: StatusScheduled, next: StatusAutoAccepted},
		{name: "auto accepted to pending", current: StatusAutoAccepted, next: StatusPending},
		{name: "scheduled to loaded denied", current: StatusScheduled, next: StatusLoaded, wantErr: true},
		{name: "auto accepted to in transit denied", current: StatusAutoAccepted, next: StatusInTransit, wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateStatusTransition(tc.current, tc.next)
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

func TestServiceCreateSnapshotsReceivingWindows(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{
		retailerWindowOpen:  "9:00",
		retailerWindowClose: "18:00",
	}
	resolver := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := newTestServiceWithResolver(repo, resolver, now)

	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 1, UnitPrice: 500}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if repo.created.ReceivingWindowOpen != "09:00" {
		t.Fatalf("persisted open = %q, want 09:00", repo.created.ReceivingWindowOpen)
	}
	if repo.created.ReceivingWindowClose != "18:00" {
		t.Fatalf("persisted close = %q, want 18:00", repo.created.ReceivingWindowClose)
	}
	if resp.ReceivingWindowOpen != "09:00" || resp.ReceivingWindowClose != "18:00" {
		t.Fatalf("response windows = %q/%q, want 09:00/18:00", resp.ReceivingWindowOpen, resp.ReceivingWindowClose)
	}
	if len(repo.lastEvents) != 1 {
		t.Fatalf("events = %d, want 1", len(repo.lastEvents))
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
	if !errors.Is(err, ErrServiceabilityUnavailable) {
		t.Fatalf("expected ErrServiceabilityUnavailable, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", repo.createCalls)
	}
}

func TestServiceCreateFailsClosedOnZoneMiss(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	resolver := &testWarehouseResolver{warehouseID: ""}
	svc := newTestServiceWithResolver(repo, resolver, now)

	_, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 1, UnitPrice: 250}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if err == nil {
		t.Fatal("expected zone miss error")
	}
	if !errors.Is(err, ErrZoneMiss) {
		t.Fatalf("expected ErrZoneMiss, got %v", err)
	}
	if repo.createCalls != 0 {
		t.Fatalf("create calls = %d, want 0", repo.createCalls)
	}
}

func TestServiceCreateFailsWhenWarehouseResolverUnavailable(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	svc := newTestServiceWithResolver(repo, nil, now)

	_, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		LineItems: []LineItem{{SKU: "sku-1", Quantity: 1, UnitPrice: 250}},
		H3Cell:    "872830828ffffff",
		Lat:       41.3,
		Lng:       69.2,
	})
	if err == nil {
		t.Fatal("expected serviceability unavailable error")
	}
	if !errors.Is(err, ErrServiceabilityUnavailable) {
		t.Fatalf("expected ErrServiceabilityUnavailable, got %v", err)
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

func TestServiceAssignOrderFirstAssignmentEmitsAssigned(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: Order{
		OrderID:     "ord-1",
		SupplierID:  "sup-1",
		RetailerID:  "ret-1",
		WarehouseID: "wh-1",
		Status:      StatusPending,
		Version:     4,
		UpdatedAt:   time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
	}}
	svc := newTestService(repo, now)

	resp, err := svc.AssignOrder(context.Background(), auth.Claims{Role: auth.RoleAdmin, Subject: "admin-1", SupplierID: "sup-1"}, "ord-1", AssignOrderRequest{
		DriverID:   "drv-1",
		VehicleID:  "veh-1",
		RouteID:    "route-1",
		ManifestID: "manifest-1",
	})
	if err != nil {
		t.Fatalf("unexpected assign error: %v", err)
	}
	if resp.EventType != events.EventOrderAssigned {
		t.Fatalf("event_type=%s want %s", resp.EventType, events.EventOrderAssigned)
	}
	if resp.DriverID != "drv-1" || resp.RouteID != "route-1" || resp.Version != 5 {
		t.Fatalf("assignment response = %+v", resp)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", repo.updateCalls)
	}
	if repo.captured.DriverID != "drv-1" || repo.captured.RouteID != "route-1" || repo.captured.ManifestID != "manifest-1" {
		t.Fatalf("captured assignment = %+v", repo.captured)
	}
	if repo.bufferedEvents != 1 || len(repo.lastEvents) != 1 {
		t.Fatalf("buffered events = %d len=%d, want 1", repo.bufferedEvents, len(repo.lastEvents))
	}
	if repo.lastEvents[0].AggregateID != "ord-1" || repo.lastEvents[0].TopicName != events.TopicMain {
		t.Fatalf("unexpected outbox event = %+v", repo.lastEvents[0])
	}
}

func TestServiceAssignOrderReassignmentEmitsReassigned(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: Order{
		OrderID:    "ord-1",
		SupplierID: "sup-1",
		RetailerID: "ret-1",
		DriverID:   "drv-old",
		RouteID:    "route-old",
		Status:     StatusPending,
		Version:    7,
		UpdatedAt:  time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
	}}
	svc := newTestService(repo, now)

	resp, err := svc.AssignOrder(context.Background(), auth.Claims{Role: auth.RoleWarehouseAdmin, Subject: "wh-admin", SupplierID: "sup-1"}, "ord-1", AssignOrderRequest{
		DriverID: "drv-new",
		RouteID:  "route-new",
	})
	if err != nil {
		t.Fatalf("unexpected reassign error: %v", err)
	}
	if resp.EventType != events.EventOrderReassigned {
		t.Fatalf("event_type=%s want %s", resp.EventType, events.EventOrderReassigned)
	}
	if repo.captured.DriverID != "drv-new" || repo.captured.RouteID != "route-new" {
		t.Fatalf("captured assignment = %+v", repo.captured)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("buffered events = %d, want 1", repo.bufferedEvents)
	}
}

func TestServiceAssignOrderRejectsUnauthorizedRoles(t *testing.T) {
	repo := &testRepo{found: true, order: Order{OrderID: "ord-1", SupplierID: "sup-1", RetailerID: "ret-1", Status: StatusPending, Version: 1}}
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.AssignOrder(context.Background(), auth.Claims{Role: auth.RoleRetailer, Subject: "ret-1", SupplierID: "sup-1"}, "ord-1", AssignOrderRequest{DriverID: "drv-1", RouteID: "route-1"})
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("expected ErrOrderForbidden, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
}

func TestServiceAssignOrderRejectsSupplierMismatch(t *testing.T) {
	repo := &testRepo{found: true, order: Order{OrderID: "ord-1", SupplierID: "sup-2", RetailerID: "ret-1", Status: StatusPending, Version: 1}}
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.AssignOrder(context.Background(), auth.Claims{Role: auth.RoleAdmin, Subject: "admin-1", SupplierID: "sup-1"}, "ord-1", AssignOrderRequest{DriverID: "drv-1", RouteID: "route-1"})
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("expected ErrOrderForbidden, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
}

func TestServiceDriverTransitionRequiresAssignedDriver(t *testing.T) {
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusInTransit)}
	repo.order.DriverID = "drv-2"
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.MarkArrived(context.Background(), driverClaims(), "ord-1")
	if !errors.Is(err, ErrOrderForbidden) {
		t.Fatalf("expected ErrOrderForbidden, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
}

func TestServiceDriverTransitionRejectsUnassignedOrder(t *testing.T) {
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusInTransit)}
	repo.order.DriverID = ""
	svc := newTestService(repo, time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC))

	_, err := svc.MarkArrived(context.Background(), driverClaims(), "ord-1")
	if !errors.Is(err, ErrAssignmentRequired) {
		t.Fatalf("expected ErrAssignmentRequired, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
}

func TestServiceMarkArrivedDriverSuccess(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusInTransit)}
	svc := newTestService(repo, now)

	resp, err := svc.MarkArrived(context.Background(), driverClaims(), "ord-1")
	if err != nil {
		t.Fatalf("unexpected mark arrived error: %v", err)
	}
	if resp.Status != StatusArrived {
		t.Fatalf("status = %s, want %s", resp.Status, StatusArrived)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", repo.updateCalls)
	}
	if repo.captured.Status != StatusArrived {
		t.Fatalf("captured status = %s, want %s", repo.captured.Status, StatusArrived)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("buffered events = %d, want 1", repo.bufferedEvents)
	}
}

func TestServiceConfirmOffloadEmitsSettlementAndPaymentEvents(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusArrived)}
	svc := newTestService(repo, now)

	resp, err := svc.ConfirmOffload(context.Background(), driverClaims(), ConfirmOffloadRequest{OrderID: "ord-1"})
	if err != nil {
		t.Fatalf("unexpected confirm offload error: %v", err)
	}
	if resp.State != StatusAwaitingPayment {
		t.Fatalf("state = %s, want %s", resp.State, StatusAwaitingPayment)
	}
	if resp.Amount != 1500 || resp.Currency != "UZS" {
		t.Fatalf("amount/currency = %d/%s, want 1500/UZS", resp.Amount, resp.Currency)
	}
	if repo.updateCalls != 1 {
		t.Fatalf("update calls = %d, want 1", repo.updateCalls)
	}
	if repo.captured.Status != StatusAwaitingPayment {
		t.Fatalf("captured status = %s, want %s", repo.captured.Status, StatusAwaitingPayment)
	}
	if repo.bufferedEvents != 3 {
		t.Fatalf("buffered events = %d, want 3", repo.bufferedEvents)
	}
}

func TestServiceConfirmOffloadIdempotentNoop(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, now)

	resp, err := svc.ConfirmOffload(context.Background(), driverClaims(), ConfirmOffloadRequest{OrderID: "ord-1"})
	if err != nil {
		t.Fatalf("unexpected confirm offload noop error: %v", err)
	}
	if resp.State != StatusAwaitingPayment {
		t.Fatalf("state = %s, want %s", resp.State, StatusAwaitingPayment)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
	if repo.bufferedEvents != 0 {
		t.Fatalf("buffered events = %d, want 0", repo.bufferedEvents)
	}
}

func TestServiceSubmitDeliveryPersistsProofArtifact(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	order := deliveryTestOrder(StatusArrived)
	order.QRToken = "qr-token"
	repo := &testRepo{found: true, order: order}
	svc := newTestService(repo, now)

	resp, err := svc.SubmitDelivery(context.Background(), driverClaims(), DeliverySubmitRequest{
		OrderID:      "ord-1",
		QRToken:      "qr-token",
		ScannedToken: "scan-token",
		Latitude:     41.311,
		Longitude:    69.279,
	})
	if err != nil {
		t.Fatalf("unexpected submit delivery error: %v", err)
	}
	if !resp.Success || resp.NewState != StatusAwaitingPayment {
		t.Fatalf("response=%+v want success with AWAITING_PAYMENT state (ADR-009 handoff opens payment)", resp)
	}
	if len(repo.lastProofs) != 1 {
		t.Fatalf("len(lastProofs)=%d want 1", len(repo.lastProofs))
	}
	proof := repo.lastProofs[0]
	if proof.ProofType != DeliveryProofTypeQRHandoff {
		t.Fatalf("proof type = %s, want %s", proof.ProofType, DeliveryProofTypeQRHandoff)
	}
	if proof.QRTokenHash != hashDeliveryProofToken("qr-token") {
		t.Fatalf("qr token hash mismatch: %s", proof.QRTokenHash)
	}
	if proof.ScannedTokenHash != hashDeliveryProofToken("scan-token") {
		t.Fatalf("scanned token hash mismatch: %s", proof.ScannedTokenHash)
	}
	if proof.Latitude == nil || proof.Longitude == nil {
		t.Fatalf("expected proof coordinates, got %+v", proof)
	}
	if proof.DistanceM == nil {
		t.Fatalf("expected proof distance, got nil")
	}
}

func TestServiceCollectCashGeofenceRejectsFarDriver(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, now)

	_, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:   "ord-1",
		Latitude:  40.0,
		Longitude: 68.0,
	})
	if !errors.Is(err, ErrGeofenceViolation) {
		t.Fatalf("expected ErrGeofenceViolation, got %v", err)
	}
	if repo.updateCalls != 0 {
		t.Fatalf("update calls = %d, want 0", repo.updateCalls)
	}
	if repo.bufferedEvents != 0 {
		t.Fatalf("buffered events = %d, want 0", repo.bufferedEvents)
	}
}

func TestServiceCollectCashEntersFiscalizingWithPaymentEvents(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, now)

	resp, err := svc.CollectCash(context.Background(), driverClaims(), CollectCashRequest{
		OrderID:   "ord-1",
		Latitude:  41.311,
		Longitude: 69.279,
	})
	if err != nil {
		t.Fatalf("unexpected collect cash error: %v", err)
	}
	if resp.State != StatusFiscalizing {
		t.Fatalf("state = %s, want %s", resp.State, StatusFiscalizing)
	}
	if resp.AttemptID == "" {
		t.Fatal("expected attempt_id on fiscalizing response")
	}
	if resp.Amount != 1500 || resp.Currency != "UZS" {
		t.Fatalf("amount/currency = %d/%s, want 1500/UZS", resp.Amount, resp.Currency)
	}
	if resp.DistanceM > deliveryGeofenceMeters {
		t.Fatalf("distance = %.2f, want within %.2f", resp.DistanceM, deliveryGeofenceMeters)
	}
	if repo.captured.Status != StatusFiscalizing {
		t.Fatalf("captured status = %s, want %s", repo.captured.Status, StatusFiscalizing)
	}
	if len(repo.captured.PendingFiscalReceipts) != 1 {
		t.Fatalf("pending fiscal receipts = %d, want 1", len(repo.captured.PendingFiscalReceipts))
	}
	// ORDER_STATUS_CHANGED + PAYMENT_CLEARED + FISCAL_RECEIPT_REQUESTED
	if repo.bufferedEvents != 3 {
		t.Fatalf("buffered events = %d, want 3", repo.bufferedEvents)
	}
	if len(repo.lastProofs) != 1 {
		t.Fatalf("len(lastProofs)=%d want 1", len(repo.lastProofs))
	}
	proof := repo.lastProofs[0]
	if proof.ProofType != DeliveryProofTypeCashCollectionGeo {
		t.Fatalf("proof type = %s, want %s", proof.ProofType, DeliveryProofTypeCashCollectionGeo)
	}
	if proof.DistanceM == nil || *proof.DistanceM > deliveryGeofenceMeters {
		t.Fatalf("proof distance = %+v want within %.2f", proof.DistanceM, deliveryGeofenceMeters)
	}

	// Worker success → COMPLETED + ORDER_FINALIZED
	attemptID := resp.AttemptID
	if err := svc.ApplyFiscalWorkerResult(context.Background(), "ord-1", attemptID); err != nil {
		t.Fatalf("fiscal worker: %v", err)
	}
	if repo.captured.Status != StatusCompleted {
		t.Fatalf("after fiscal status = %s, want COMPLETED", repo.captured.Status)
	}
	if repo.captured.FiscalStatus != FiscalStatusSuccess {
		t.Fatalf("fiscal status = %s, want SUCCESS", repo.captured.FiscalStatus)
	}
}

func TestServiceCompleteOrderFinalizesWithoutCoordinatesFails(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{found: true, order: deliveryTestOrder(StatusAwaitingPayment)}
	svc := newTestService(repo, now)

	_, err := svc.CompleteOrder(context.Background(), driverClaims(), CompleteOrderRequest{OrderID: "ord-1"})
	if err == nil {
		t.Fatal("expected complete order error due to missing coordinates")
	}
	if !strings.Contains(err.Error(), "latitude and longitude required") {
		t.Fatalf("expected missing coordinates error, got: %v", err)
	}
}

func TestServiceCreateManualPreorderSetsConfirmedMetadata(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	repo := &testRepo{}
	resolver := &testWarehouseResolver{warehouseID: "wh-1"}
	svc := newTestServiceWithResolver(repo, resolver, now)

	requested := now.AddDate(0, 0, 10).Format(time.RFC3339Nano)
	resp, err := svc.Create(context.Background(), "ret-1", CreateRequest{
		DeliveryMode:          DeliveryModeScheduled,
		RequestedDeliveryDate: requested,
		H3Cell:                "872830828ffffff",
		Lat:                   41.311,
		Lng:                   69.279,
		LineItems:             []LineItem{{SKU: "sku-1", Name: "Water", Quantity: 2, UnitPrice: 500}},
	})
	if err != nil {
		t.Fatalf("unexpected create preorder error: %v", err)
	}
	if resp.Source != OrderSourceManualPreorder {
		t.Fatalf("source=%s want %s", resp.Source, OrderSourceManualPreorder)
	}
	if resp.ConfirmationStatus != ConfirmationStatusConfirmed {
		t.Fatalf("confirmation_status=%s want %s", resp.ConfirmationStatus, ConfirmationStatusConfirmed)
	}
	if repo.created.Source != OrderSourceManualPreorder || repo.created.ConfirmationStatus != ConfirmationStatusConfirmed {
		t.Fatalf("created order metadata = %+v", repo.created)
	}
	if repo.created.RequestedDeliveryDate == nil || repo.created.RequestedDeliveryDate.Format(time.RFC3339Nano) != requested {
		t.Fatalf("requested delivery = %v want %s", repo.created.RequestedDeliveryDate, requested)
	}
}

func TestServiceConfirmAIOrderMarksConfirmed(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	autoConfirmAt := now.Add(2 * time.Hour)
	repo := &testRepo{found: true, order: Order{
		OrderID:            "ord-ai-1",
		SupplierID:         "sup-1",
		RetailerID:         "ret-1",
		WarehouseID:        "wh-1",
		Status:             StatusPending,
		Source:             OrderSourceAIPreorder,
		ConfirmationStatus: ConfirmationStatusPending,
		AutoConfirmAt:      &autoConfirmAt,
		LineItems:          []LineItem{{SKU: "sku-1", Name: "Water", Quantity: 1, UnitPrice: 500}},
		TotalMinor:         500,
		Currency:           "UZS",
		Version:            3,
		CreatedAt:          now.Add(-time.Hour),
		UpdatedAt:          now.Add(-time.Hour),
	}}
	svc := newTestService(repo, now)

	resp, err := svc.ConfirmAIOrder(context.Background(), "ret-1", ConfirmAIOrderRequest{OrderID: "ord-ai-1"})
	if err != nil {
		t.Fatalf("unexpected confirm ai order error: %v", err)
	}
	if resp.ConfirmationStatus != ConfirmationStatusConfirmed {
		t.Fatalf("confirmation_status=%s want %s", resp.ConfirmationStatus, ConfirmationStatusConfirmed)
	}
	if repo.captured.ConfirmationStatus != ConfirmationStatusConfirmed {
		t.Fatalf("captured confirmation_status=%s want %s", repo.captured.ConfirmationStatus, ConfirmationStatusConfirmed)
	}
	if repo.captured.AutoConfirmAt != nil {
		t.Fatalf("auto_confirm_at should be cleared, got %v", repo.captured.AutoConfirmAt)
	}
	if repo.captured.DecisionBy != "ret-1" {
		t.Fatalf("decision_by=%s want ret-1", repo.captured.DecisionBy)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("buffered events = %d, want 1", repo.bufferedEvents)
	}
}

func driverClaims() auth.Claims {
	return auth.Claims{Role: auth.RoleDriver, Subject: "drv-1", SupplierID: "sup-1"}
}

func deliveryTestOrder(status Status) Order {
	return Order{
		OrderID:     "ord-1",
		SupplierID:  "sup-1",
		RetailerID:  "ret-1",
		WarehouseID: "wh-1",
		DriverID:    "drv-1",
		RouteID:     "route-1",
		Status:      status,
		LineItems:   []LineItem{{SKU: "sku-1", Name: "Water", Quantity: 3, UnitPrice: 500}},
		TotalMinor:  1500,
		Currency:    "UZS",
		H3Cell:      "872830828ffffff",
		Lat:         41.311,
		Lng:         69.279,
		Version:     7,
		CreatedAt:   time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2026, 1, 1, 11, 0, 0, 0, time.UTC),
	}
}

func (r *testRepo) ListManifestOrders(_ context.Context, manifestID string) ([]Order, error) {
	return nil, nil
}

func (r *testRepo) FindPendingBuyerAcceptance(_ context.Context, _ int) ([]*Order, error) {
	return nil, nil
}

func (r *testRepo) UpdateOrderWithTxn(_ context.Context, o Order, _ []DeliveryProofArtifact, _ func(context.Context, *spanner.ReadWriteTransaction) error, emit func(outbox.TxnBuffer) error) error {
	r.updateCalls++
	r.captured = o
	r.order = o // Update the mocked return value so subsequent GetOrder calls see the new state
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.lastEvents = append(r.lastEvents, buf.events...)
	}
	return nil
}
