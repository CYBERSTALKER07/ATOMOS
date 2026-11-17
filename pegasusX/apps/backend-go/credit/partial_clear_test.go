package credit

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type reservationMemoryRepo struct {
	mu           sync.Mutex
	profiles     map[string]Profile
	reservations map[string]OrderReservation
	outbox       []outbox.Event
}

func newReservationMemoryRepo() *reservationMemoryRepo {
	return &reservationMemoryRepo{
		profiles:     map[string]Profile{},
		reservations: map[string]OrderReservation{},
	}
}

func (r *reservationMemoryRepo) key(rid, sid string) string { return rid + "|" + sid }

func (r *reservationMemoryRepo) GetProfile(_ context.Context, rid, sid string) (Profile, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[r.key(rid, sid)]
	return p, ok, nil
}

func (r *reservationMemoryRepo) ListBySupplier(_ context.Context, sid, status string, limit int) ([]Profile, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var out []Profile
	for _, p := range r.profiles {
		if p.SupplierID == sid && (status == "" || string(p.Status) == status) {
			out = append(out, p)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *reservationMemoryRepo) UpsertProfile(_ context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.outbox = append(r.outbox, buf.events...)
	}
	r.profiles[r.key(p.RetailerID, p.SupplierID)] = p
	return nil
}

func (r *reservationMemoryRepo) AdjustBalance(_ context.Context, rid, sid string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.profiles[r.key(rid, sid)]
	p.CurrentBalanceMinor += deltaMinor
	if p.CurrentBalanceMinor < 0 {
		p.CurrentBalanceMinor = 0
	}
	p.AvailableCreditMinor = p.Available()
	p.Version++
	r.profiles[r.key(rid, sid)] = p
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.outbox = append(r.outbox, buf.events...)
	}
	return nil
}

func (r *reservationMemoryRepo) GetScoresForRetailers(_ context.Context, _ []string) (map[string]RetailerCreditScore, error) {
	return map[string]RetailerCreditScore{}, nil
}

func (r *reservationMemoryRepo) ReserveOrder(_ context.Context, res OrderReservation, emit func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	p := r.profiles[r.key(res.RetailerID, res.SupplierID)]
	p.ReservedMinor += res.AmountMinor
	p.AvailableCreditMinor = p.Available()
	r.profiles[r.key(res.RetailerID, res.SupplierID)] = p
	res.Status = ReservationReserved
	r.reservations[res.OrderID] = res
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf); err != nil {
			return err
		}
		r.outbox = append(r.outbox, buf.events...)
	}
	return nil
}

func (r *reservationMemoryRepo) ReleaseOrderReservation(_ context.Context, orderID string, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	if !ok || res.Status != ReservationReserved {
		return nil
	}
	p := r.profiles[r.key(res.RetailerID, res.SupplierID)]
	p.ReservedMinor -= res.AmountMinor
	if p.ReservedMinor < 0 {
		p.ReservedMinor = 0
	}
	p.AvailableCreditMinor = p.Available()
	res.Status = ReservationReleased
	r.profiles[r.key(res.RetailerID, res.SupplierID)] = p
	r.reservations[orderID] = res
	return nil
}

func (r *reservationMemoryRepo) ConvertOrderReservation(_ context.Context, orderID string, _ func(outbox.TxnBuffer) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	if !ok || res.Status != ReservationReserved {
		return nil
	}
	p := r.profiles[r.key(res.RetailerID, res.SupplierID)]
	p.ReservedMinor -= res.AmountMinor
	if p.ReservedMinor < 0 {
		p.ReservedMinor = 0
	}
	p.CurrentBalanceMinor += res.AmountMinor
	p.AvailableCreditMinor = p.Available()
	res.Status = ReservationConverted
	r.profiles[r.key(res.RetailerID, res.SupplierID)] = p
	r.reservations[orderID] = res
	return nil
}

func (r *reservationMemoryRepo) GetOrderReservation(_ context.Context, orderID string) (OrderReservation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	return res, ok, nil
}

func (r *reservationMemoryRepo) ListDelinquentCandidates(_ context.Context, _ int) ([]Profile, error) {
	return nil, nil
}

func (r *reservationMemoryRepo) ClearOrderBalance(_ context.Context, retailerID, supplierID, orderID string, amountMinor int64, emit func(outbox.TxnBuffer, int64) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	res, ok := r.reservations[orderID]
	if !ok {
		return nil
	}
	if res.RetailerID != "" {
		retailerID = res.RetailerID
	}
	if res.SupplierID != "" {
		supplierID = res.SupplierID
	}
	p, ok := r.profiles[r.key(retailerID, supplierID)]
	if !ok {
		return ErrProfileNotFound
	}
	plan, changed := computePartialClearPlan(res.Status, p.CreditLimitMinor, p.CurrentBalanceMinor, p.ReservedMinor, res.AmountMinor, amountMinor)
	if !changed {
		return nil
	}
	if emit != nil {
		buf := &testTxnBuffer{}
		if err := emit(buf, plan.ClearAmountMinor); err != nil {
			return err
		}
		r.outbox = append(r.outbox, buf.events...)
	}
	p.CurrentBalanceMinor = plan.NewBalanceMinor
	p.AvailableCreditMinor = plan.AvailableCreditMinor
	p.Version++
	res.AmountMinor = plan.RemainingAmountMinor
	res.Status = plan.NextStatus
	r.profiles[r.key(retailerID, supplierID)] = p
	r.reservations[orderID] = res
	return nil
}

func TestComputePartialClearPlan_MultiStepAndIdempotent(t *testing.T) {
	// Step 1: partial clear 30k out of 100k reservation while another 50k sits on profile balance (150k total balance)
	p1, ok := computePartialClearPlan(ReservationConverted, 300_000, 150_000, 20_000, 100_000, 30_000)
	if !ok {
		t.Fatal("expected partial clear plan")
	}
	if p1.ClearAmountMinor != 30_000 || p1.RemainingAmountMinor != 70_000 || p1.NextStatus != ReservationConverted ||
		p1.NewBalanceMinor != 120_000 || p1.AvailableCreditMinor != 160_000 {
		t.Fatalf("unexpected step1 plan: %+v", p1)
	}

	// Step 2: overpay attempt 120k against remaining 70k caps clearAmt at 70k, preserving the other 50k order balance
	p2, ok := computePartialClearPlan(p1.NextStatus, 300_000, p1.NewBalanceMinor, 20_000, p1.RemainingAmountMinor, 120_000)
	if !ok {
		t.Fatal("expected final clear plan")
	}
	if p2.ClearAmountMinor != 70_000 || p2.RemainingAmountMinor != 0 || p2.NextStatus != ReservationCleared ||
		p2.NewBalanceMinor != 50_000 || p2.AvailableCreditMinor != 230_000 {
		t.Fatalf("unexpected step2 plan: %+v", p2)
	}

	// Step 3: already CLEARED -> idempotent no-op
	if _, ok := computePartialClearPlan(ReservationCleared, 300_000, 50_000, 20_000, 0, 10_000); ok {
		t.Fatal("expected cleared reservation to be no-op")
	}

	// Step 4: non-converted states and zero amounts -> no-op
	if _, ok := computePartialClearPlan(ReservationReserved, 300_000, 50_000, 20_000, 50_000, 10_000); ok {
		t.Fatal("expected RESERVED reservation to be no-op")
	}
	if _, ok := computePartialClearPlan(ReservationConverted, 300_000, 50_000, 20_000, 50_000, 0); ok {
		t.Fatal("expected 0 clear amount to be no-op")
	}
}

func TestClearBalance_MultiStepPartialPreservesOtherOrderBalance(t *testing.T) {
	ctx := context.Background()
	repo := newReservationMemoryRepo()
	repo.profiles[repo.key("ret-1", "sup-1")] = Profile{
		RetailerID:           "ret-1",
		SupplierID:           "sup-1",
		CreditLimitMinor:     500_000,
		CurrentBalanceMinor:  150_000, // ord-1 (100k) + ord-2 (50k)
		AvailableCreditMinor: 350_000,
		Status:               StatusActive,
		Version:              1,
	}
	repo.reservations["ord-1"] = OrderReservation{
		OrderID: "ord-1", RetailerID: "ret-1", SupplierID: "sup-1", AmountMinor: 100_000, Status: ReservationConverted,
	}
	svc := NewService(repo)
	svc.SetNow(func() time.Time { return time.Date(2026, 11, 13, 10, 0, 0, 0, time.UTC) })

	// 1st partial pay: 40_000
	if err := svc.ClearBalanceInTxn(ctx, nil, "ret-1", "sup-1", "ord-1", 40_000); err != nil {
		t.Fatalf("partial 1: %v", err)
	}
	res1, _, _ := repo.GetOrderReservation(ctx, "ord-1")
	p1, _, _ := repo.GetProfile(ctx, "ret-1", "sup-1")
	if res1.AmountMinor != 60_000 || res1.Status != ReservationConverted {
		t.Fatalf("after partial 1 reservation = %+v", res1)
	}
	if p1.CurrentBalanceMinor != 110_000 || p1.AvailableCreditMinor != 390_000 {
		t.Fatalf("after partial 1 profile = %+v", p1)
	}

	// 2nd partial pay: 90_000 (overpay for ord-1; must cap clear at 60_000 leaving ord-2's 50_000 untouched)
	if err := svc.ClearBalanceInTxn(ctx, nil, "ret-1", "sup-1", "ord-1", 90_000); err != nil {
		t.Fatalf("partial 2: %v", err)
	}
	res2, _, _ := repo.GetOrderReservation(ctx, "ord-1")
	p2, _, _ := repo.GetProfile(ctx, "ret-1", "sup-1")
	if res2.AmountMinor != 0 || res2.Status != ReservationCleared {
		t.Fatalf("after final clear reservation = %+v", res2)
	}
	if p2.CurrentBalanceMinor != 50_000 || p2.AvailableCreditMinor != 450_000 {
		t.Fatalf("after final clear profile = %+v", p2)
	}

	// Verify outbox event amounts (-40k, -60k)
	if len(repo.outbox) != 2 {
		t.Fatalf("outbox len = %d want 2", len(repo.outbox))
	}
	var ev2 events.CreditProfileEvent
	if err := json.Unmarshal(repo.outbox[1].Payload, &ev2); err != nil {
		t.Fatal(err)
	}
	if ev2.CurrentBalance != -60_000 {
		t.Fatalf("outbox delta = %d want -60000", ev2.CurrentBalance)
	}

	// 3rd call: idempotent no-op
	if err := svc.ClearBalanceInTxn(ctx, nil, "ret-1", "sup-1", "ord-1", 25_000); err != nil {
		t.Fatalf("idempotent retry: %v", err)
	}
	p3, _, _ := repo.GetProfile(ctx, "ret-1", "sup-1")
	if p3.CurrentBalanceMinor != 50_000 || len(repo.outbox) != 2 {
		t.Fatalf("expected idempotent retry noop, profile=%+v outbox=%d", p3, len(repo.outbox))
	}
}

func TestPolicyService_ResolveTermsForOrder(t *testing.T) {
	ctx := context.Background()
	polRepo := NewMemoryPolicyRepository()
	if err := polRepo.UpsertProgram(ctx, SupplierCreditProgram{
		SupplierID:              "sup-1",
		ProgramEnabled:          true,
		GlobalTermsDays:         14,
		GlobalGraceDays:         3,
		GlobalDefaultLimitMinor: 5_000_000,
		Timezone:                "Asia/Tashkent",
	}, nil); err != nil {
		t.Fatal(err)
	}
	if err := polRepo.UpsertTerms(ctx, RetailerPaymentTerms{
		RetailerID:       "ret-1",
		SupplierID:       "sup-1",
		CreditEnabled:    true,
		TermsDays:        21,
		GracePeriodDays:  5,
		CreditLimitMinor: 10_000_000,
	}, nil); err != nil {
		t.Fatal(err)
	}

	ps := NewPolicyService(polRepo, nil)
	resolved, err := ps.ResolveTermsForOrder(ctx, "ret-1", "sup-1")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.TermsDays != 21 || resolved.GracePeriodDays != 5 || resolved.CreditLimitMinor != 10_000_000 || resolved.Timezone != "Asia/Tashkent" {
		t.Fatalf("unexpected resolved terms: %+v", resolved)
	}
}
