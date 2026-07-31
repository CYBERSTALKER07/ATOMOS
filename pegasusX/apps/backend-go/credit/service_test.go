package credit

import (
	"bytes"
	"context"
	"errors"
	"testing"
	"time"

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

type testCreditRepo struct {
	profile        Profile
	found          bool
	getErr         error
	upsertErr      error
	adjustErr      error
	stored         *Profile
	adjustDelta    int64
	bufferedEvents int
	lastEvents     []outbox.Event
}

func (r *testCreditRepo) GetProfile(_ context.Context, _, _ string) (Profile, bool, error) {
	if r.getErr != nil {
		return Profile{}, false, r.getErr
	}
	return r.profile, r.found, nil
}

func (r *testCreditRepo) ListBySupplier(_ context.Context, supplierID, status string, limit int) ([]Profile, error) {
	if r.getErr != nil {
		return nil, r.getErr
	}
	if !r.found {
		return nil, nil
	}
	if r.profile.SupplierID != "" && r.profile.SupplierID != supplierID {
		return nil, nil
	}
	if status != "" && string(r.profile.Status) != status {
		return nil, nil
	}
	out := []Profile{r.profile}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (r *testCreditRepo) UpsertProfile(_ context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
	if r.upsertErr != nil {
		return r.upsertErr
	}
	r.stored = &p
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

func (r *testCreditRepo) AdjustBalance(_ context.Context, _, _ string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	if r.adjustErr != nil {
		return r.adjustErr
	}
	r.adjustDelta = deltaMinor
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

func (r *testCreditRepo) GetScoresForRetailers(_ context.Context, _ []string) (map[string]RetailerCreditScore, error) {
	return map[string]RetailerCreditScore{}, nil
}

func newTestService(repo Repository) *Service {
	s := NewService(repo)
	s.SetNow(func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) })
	s.SetNewID(func() string { return "test-id" })
	return s
}

func TestListSupplierProfiles_SortsCollectionsFirst(t *testing.T) {
	list := []Profile{
		{RetailerID: "r-idle", SupplierID: "s1", Status: StatusActive, CurrentBalanceMinor: 0},
		{RetailerID: "r-bal", SupplierID: "s1", Status: StatusActive, CurrentBalanceMinor: 50_000},
		{RetailerID: "r-frozen", SupplierID: "s1", Status: StatusFrozen, CurrentBalanceMinor: 10_000},
		{RetailerID: "r-bl", SupplierID: "s1", Status: StatusBlacklisted, CurrentBalanceMinor: 1},
	}
	sortProfilesForCollections(list)
	if list[0].RetailerID != "r-bl" || list[1].RetailerID != "r-frozen" || list[2].RetailerID != "r-bal" {
		t.Fatalf("order=%v %v %v %v", list[0].RetailerID, list[1].RetailerID, list[2].RetailerID, list[3].RetailerID)
	}
}

func TestListSupplierProfiles_ScopedViaRepo(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID: "ret-1", SupplierID: "sup-1",
			CreditLimitMinor: 100_000, CurrentBalanceMinor: 40_000,
			Status: StatusActive,
		},
	}
	svc := newTestService(repo)
	list, err := svc.ListSupplierProfiles(context.Background(), "sup-1", "", 50)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	list, err = svc.ListSupplierProfiles(context.Background(), "sup-OTHER", "", 50)
	if err != nil || len(list) != 0 {
		t.Fatalf("other supplier: %v len=%d", err, len(list))
	}
}

func TestCheckOrder_Allowed(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 20000,
			Status:              StatusActive,
			RiskTier:            RiskTierLow,
		},
	}
	svc := newTestService(repo)

	result, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 30000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Allowed {
		t.Fatalf("expected allowed, got %+v", result)
	}
	if result.Shortfall != 0 {
		t.Fatalf("expected zero shortfall, got %d", result.Shortfall)
	}
}

func TestCheckOrder_LimitBreached(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:          "ret-1",
			SupplierID:          "sup-1",
			CreditLimitMinor:    100000,
			CurrentBalanceMinor: 80000,
			Status:              StatusActive,
			RiskTier:            RiskTierLow,
		},
	}
	svc := newTestService(repo)

	result, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 30000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Fatal("expected blocked")
	}
	if result.Shortfall != 10000 {
		t.Fatalf("expected shortfall 10000, got %d", result.Shortfall)
	}
	if result.Reason != "credit_limit_breached" {
		t.Fatalf("expected credit_limit_breached reason, got %s", result.Reason)
	}
}

func TestCheckOrder_NoProfileBlocks(t *testing.T) {
	repo := &testCreditRepo{found: false}
	svc := newTestService(repo)

	result, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed {
		t.Fatal("expected blocked when no profile")
	}
	if result.Reason != "no_credit_profile" {
		t.Fatalf("expected no_credit_profile, got %s", result.Reason)
	}
}

func TestCheckOrder_Blacklisted(t *testing.T) {
	repo := &testCreditRepo{
		found: true,
		profile: Profile{
			RetailerID:       "ret-1",
			SupplierID:       "sup-1",
			CreditLimitMinor: 100000,
			Status:           StatusBlacklisted,
		},
	}
	svc := newTestService(repo)

	result, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Allowed || result.Reason != "profile_blacklisted" {
		t.Fatalf("expected blacklisted block, got %+v", result)
	}
}

func TestMarkBalance(t *testing.T) {
	repo := &testCreditRepo{}
	svc := newTestService(repo)

	if err := svc.MarkBalance(context.Background(), "ret-1", "sup-1", 50000, "ord-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.adjustDelta != 50000 {
		t.Fatalf("expected delta 50000, got %d", repo.adjustDelta)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("expected 1 event, got %d", repo.bufferedEvents)
	}
	if !bytes.Contains(repo.lastEvents[0].Payload, []byte(events.EventRetailerCreditProfileChanged)) {
		t.Fatalf("expected profile changed payload, got %s", string(repo.lastEvents[0].Payload))
	}
}

func TestClearBalance(t *testing.T) {
	repo := &testCreditRepo{}
	svc := newTestService(repo)

	if err := svc.ClearBalance(context.Background(), "ret-1", "sup-1", 25000, "ord-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.adjustDelta != -25000 {
		t.Fatalf("expected delta -25000, got %d", repo.adjustDelta)
	}
}

func TestUpsertProfile(t *testing.T) {
	repo := &testCreditRepo{}
	svc := newTestService(repo)

	p := Profile{
		RetailerID:       "ret-1",
		SupplierID:       "sup-1",
		CreditLimitMinor: 200000,
	}
	if err := svc.UpsertProfile(context.Background(), p, "usr-1", "manual_review"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.stored == nil {
		t.Fatal("expected profile stored")
	}
	if repo.stored.Status != StatusActive {
		t.Fatalf("expected active status, got %s", repo.stored.Status)
	}
	if repo.stored.RiskTier != RiskTierMedium {
		t.Fatalf("expected medium risk tier, got %s", repo.stored.RiskTier)
	}
	if repo.stored.AvailableCreditMinor != 200000 {
		t.Fatalf("expected available credit 200000, got %d", repo.stored.AvailableCreditMinor)
	}
	if repo.bufferedEvents != 1 {
		t.Fatalf("expected 1 event, got %d", repo.bufferedEvents)
	}
}

func TestEvaluateRisk(t *testing.T) {
	svc := newTestService(&testCreditRepo{})
	cases := []struct {
		delinquency int64
		balance     int64
		limit       int64
		want        RiskTier
	}{
		{0, 0, 100000, RiskTierLow},
		{0, 30000, 100000, RiskTierMedium},
		{0, 60000, 100000, RiskTierHigh},
		{1, 0, 100000, RiskTierHigh},
		{3, 0, 100000, RiskTierBlock},
		{0, 110000, 100000, RiskTierBlock},
	}
	for _, tc := range cases {
		got := svc.EvaluateRisk(tc.delinquency, tc.balance, tc.limit)
		if got != tc.want {
			t.Errorf("EvaluateRisk(%d,%d,%d) = %s, want %s", tc.delinquency, tc.balance, tc.limit, got, tc.want)
		}
	}
}

func TestCheckOrder_RepositoryError(t *testing.T) {
	repo := &testCreditRepo{getErr: errors.New("db down")}
	svc := newTestService(repo)

	_, err := svc.CheckOrder(context.Background(), "ret-1", "sup-1", 1000)
	if err == nil {
		t.Fatal("expected error")
	}
}
