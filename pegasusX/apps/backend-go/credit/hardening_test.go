package credit

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

type multiCreditMemoryRepo struct {
	mu       sync.Mutex
	profiles map[string]Profile
}

func newMultiCreditMemoryRepo() *multiCreditMemoryRepo {
	return &multiCreditMemoryRepo{profiles: make(map[string]Profile)}
}

func (r *multiCreditMemoryRepo) key(rid, sid string) string { return rid + "|" + sid }

func (r *multiCreditMemoryRepo) GetProfile(_ context.Context, retailerID, supplierID string) (Profile, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	p, ok := r.profiles[r.key(retailerID, supplierID)]
	return p, ok, nil
}

func (r *multiCreditMemoryRepo) UpsertProfile(_ context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
	if emit != nil {
		if err := emit(&memoryCreditTxnBuf{}); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.profiles[r.key(p.RetailerID, p.SupplierID)] = p
	return nil
}

func (r *multiCreditMemoryRepo) ListBySupplier(ctx context.Context, supplierID, status string, limit int) ([]Profile, error) {
	page, _, err := r.ListBySupplierPage(ctx, supplierID, status, limit, "")
	return page, err
}

func (r *multiCreditMemoryRepo) ListBySupplierPage(_ context.Context, supplierID, status string, limit int, cursor string) ([]Profile, string, error) {
	if limit <= 0 {
		limit = 50
	}
	cursorTs, cursorRid, cursorSid, hasCursor, err := decodeProfileCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	r.mu.Lock()
	items := make([]Profile, 0)
	for _, p := range r.profiles {
		if p.SupplierID != supplierID {
			continue
		}
		if status != "" && string(p.Status) != strings.ToUpper(status) {
			continue
		}
		items = append(items, p)
	}
	r.mu.Unlock()
	return paginateTestProfiles(items, limit, cursorTs, cursorRid, cursorSid, hasCursor), nextCursorIfMore(items, limit, cursorTs, cursorRid, cursorSid, hasCursor), nil
}

func (r *multiCreditMemoryRepo) ListDelinquentCandidates(ctx context.Context, limit int) ([]Profile, error) {
	page, _, err := r.ListDelinquentCandidatesPage(ctx, limit, "")
	return page, err
}

func (r *multiCreditMemoryRepo) ListDelinquentCandidatesPage(_ context.Context, limit int, cursor string) ([]Profile, string, error) {
	if limit <= 0 {
		limit = 100
	}
	cursorTs, cursorRid, cursorSid, hasCursor, err := decodeProfileCursor(cursor)
	if err != nil {
		return nil, "", err
	}
	r.mu.Lock()
	items := make([]Profile, 0)
	for _, p := range r.profiles {
		if p.Status == StatusActive && (p.DelinquencyCount > 0 || p.CurrentBalanceMinor > 0) {
			items = append(items, p)
		}
	}
	r.mu.Unlock()
	return paginateTestProfiles(items, limit, cursorTs, cursorRid, cursorSid, hasCursor), nextCursorIfMore(items, limit, cursorTs, cursorRid, cursorSid, hasCursor), nil
}

func sortTestProfiles(items []Profile) {
	sort.Slice(items, func(i, j int) bool {
		if !items[i].UpdatedAt.Equal(items[j].UpdatedAt) {
			return items[i].UpdatedAt.After(items[j].UpdatedAt)
		}
		if items[i].RetailerID != items[j].RetailerID {
			return items[i].RetailerID > items[j].RetailerID
		}
		return items[i].SupplierID > items[j].SupplierID
	})
}

func filterTestProfiles(items []Profile, cursorTs time.Time, cursorRid, cursorSid string, hasCursor bool) []Profile {
	sortTestProfiles(items)
	if !hasCursor {
		return items
	}
	out := make([]Profile, 0, len(items))
	for _, p := range items {
		ts := p.UpdatedAt.UTC()
		if ts.Before(cursorTs) || (ts.Equal(cursorTs) && (p.RetailerID < cursorRid || (p.RetailerID == cursorRid && p.SupplierID < cursorSid))) {
			out = append(out, p)
		}
	}
	return out
}

func paginateTestProfiles(items []Profile, limit int, cursorTs time.Time, cursorRid, cursorSid string, hasCursor bool) []Profile {
	filtered := filterTestProfiles(items, cursorTs, cursorRid, cursorSid, hasCursor)
	if len(filtered) > limit {
		return filtered[:limit]
	}
	return filtered
}

func nextCursorIfMore(items []Profile, limit int, cursorTs time.Time, cursorRid, cursorSid string, hasCursor bool) string {
	filtered := filterTestProfiles(items, cursorTs, cursorRid, cursorSid, hasCursor)
	if len(filtered) > limit {
		last := filtered[limit-1]
		return encodeProfileCursor(last.UpdatedAt, last.RetailerID, last.SupplierID)
	}
	return ""
}

func (r *multiCreditMemoryRepo) AdjustBalance(context.Context, string, string, int64, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *multiCreditMemoryRepo) GetScoresForRetailers(context.Context, []string) (map[string]RetailerCreditScore, error) {
	return nil, nil
}
func (r *multiCreditMemoryRepo) ReserveOrder(context.Context, OrderReservation, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *multiCreditMemoryRepo) ReleaseOrderReservation(context.Context, string, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *multiCreditMemoryRepo) ConvertOrderReservation(context.Context, string, func(outbox.TxnBuffer) error) error {
	return nil
}
func (r *multiCreditMemoryRepo) GetOrderReservation(context.Context, string) (OrderReservation, bool, error) {
	return OrderReservation{}, false, nil
}

type memoryCreditTxnBuf struct{}

func (*memoryCreditTxnBuf) BufferOutbox(context.Context, outbox.Event) error { return nil }

func TestHandleGetRetailerProfile_UsesResolveRetailerOrgID(t *testing.T) {
	repo := newMultiCreditMemoryRepo()
	repo.profiles[repo.key("ret-org-1", "sup-1")] = Profile{
		RetailerID:       "ret-org-1",
		SupplierID:       "sup-1",
		CreditLimitMinor: 900000,
		Status:           StatusActive,
	}
	svc := NewService(repo)

	req := httptest.NewRequest(http.MethodGet, "/v1/retailer/credit-profile", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject:       "staff-buyer-77",
		Role:          auth.RoleRetailer,
		RetailerOrgID: "ret-org-1",
		SupplierID:    "sup-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleGetRetailerProfile(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var got Profile
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.RetailerID != "ret-org-1" || got.CreditLimitMinor != 900000 {
		t.Fatalf("unexpected profile: %+v", got)
	}
}

func TestHandleUpsertSupplierProfile_PointerLimitAndAuthz(t *testing.T) {
	repo := newMultiCreditMemoryRepo()
	repo.profiles[repo.key("ret-1", "sup-1")] = Profile{
		RetailerID:       "ret-1",
		SupplierID:       "sup-1",
		CreditLimitMinor: 500000,
		Status:           StatusActive,
	}
	svc := NewService(repo)

	patch := func(supID, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/retailer-credit-profile", strings.NewReader(body))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "admin-1",
			Role:       auth.RoleAdmin,
			SupplierID: supID,
		}))
		rr := httptest.NewRecorder()
		svc.HandleUpsertSupplierProfile(rr, req)
		return rr
	}

	// Missing supplier scope fails closed.
	if rr := patch("", `{"retailer_id":"ret-1","status":"FROZEN"}`); rr.Code != http.StatusForbidden {
		t.Fatalf("missing supplier status=%d want 403", rr.Code)
	}

	// Negative limit rejected.
	if rr := patch("sup-1", `{"retailer_id":"ret-1","credit_limit_minor":-5}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("negative limit status=%d want 400", rr.Code)
	}

	// Omitted credit_limit_minor preserves 500000 while patching status.
	if rr := patch("sup-1", `{"retailer_id":"ret-1","status":"FROZEN","reason":"audit"}`); rr.Code != http.StatusOK {
		t.Fatalf("omit limit status=%d body=%s", rr.Code, rr.Body.String())
	}
	got, _, _ := repo.GetProfile(context.Background(), "ret-1", "sup-1")
	if got.CreditLimitMinor != 500000 || got.Status != StatusFrozen {
		t.Fatalf("omitted limit clobbered existing limit: %+v", got)
	}

	// Explicit 0 zeroes out CreditLimitMinor.
	if rr := patch("sup-1", `{"retailer_id":"ret-1","credit_limit_minor":0,"reason":"zero"}`); rr.Code != http.StatusOK {
		t.Fatalf("zero limit status=%d body=%s", rr.Code, rr.Body.String())
	}
	got, _, _ = repo.GetProfile(context.Background(), "ret-1", "sup-1")
	if got.CreditLimitMinor != 0 {
		t.Fatalf("explicit 0 limit got %d, want 0", got.CreditLimitMinor)
	}
}

func TestBumpDelinquency_ConcurrentNoLostIncrements(t *testing.T) {
	t.Setenv("CREDIT_MAX_DELINQUENCY_TOLERANCE", "10")
	repo := newMultiCreditMemoryRepo()
	repo.profiles[repo.key("ret-1", "sup-1")] = Profile{
		RetailerID: "ret-1",
		SupplierID: "sup-1",
		Status:     StatusActive,
	}
	svc := NewService(repo)

	var wg sync.WaitGroup
	for range 5 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := svc.BumpDelinquency(context.Background(), "ret-1", "sup-1"); err != nil {
				t.Errorf("BumpDelinquency: %v", err)
			}
		}()
	}
	wg.Wait()

	got, _, _ := repo.GetProfile(context.Background(), "ret-1", "sup-1")
	if got.DelinquencyCount != 5 {
		t.Fatalf("DelinquencyCount=%d want 5 (lost update!)", got.DelinquencyCount)
	}
}

func TestCreditCursorPaginationAndFullSweeper(t *testing.T) {
	t.Setenv("CREDIT_OVERDUE_BATCH_SIZE", "2")
	t.Setenv("CREDIT_MAX_DELINQUENCY_TOLERANCE", "2")
	now := time.Date(2026, 9, 1, 10, 0, 0, 0, time.UTC)
	repo := newMultiCreditMemoryRepo()
	for i := 1; i <= 5; i++ {
		rid := "ret-" + string(rune('0'+i))
		repo.profiles[repo.key(rid, "sup-1")] = Profile{
			RetailerID:       rid,
			SupplierID:       "sup-1",
			CreditLimitMinor: 100000,
			DelinquencyCount: 2,
			Status:           StatusActive,
			UpdatedAt:        now.Add(time.Duration(i) * time.Minute),
		}
	}
	svc := NewService(repo)
	svc.SetNow(func() time.Time { return now.Add(10 * time.Minute) })

	// Cursor paginate ListSupplierProfilesPage 2 at a time -> 3 pages covering all 5 profiles.
	seen := make([]string, 0, 5)
	cursor := ""
	for range 5 {
		page, next, err := svc.ListSupplierProfilesPage(context.Background(), "sup-1", "", 2, cursor)
		if err != nil {
			t.Fatalf("ListSupplierProfilesPage: %v", err)
		}
		for _, p := range page {
			seen = append(seen, p.RetailerID)
		}
		if next == "" {
			break
		}
		cursor = next
	}
	if len(seen) != 5 {
		t.Fatalf("expected 5 paginated profiles, got %v", seen)
	}
	if _, _, err := svc.ListSupplierProfilesPage(context.Background(), "sup-1", "", 2, "bad-cursor"); !errors.Is(err, ErrInvalidCursor) {
		t.Fatalf("expected ErrInvalidCursor, got %v", err)
	}

	// Sweeper with batch size 2 must sweep ALL 5 delinquent profiles across multiple cursor pages.
	frozen, err := svc.RunOverdueCreditSweeperOnce(context.Background())
	if err != nil || frozen != 5 {
		t.Fatalf("frozen=%d err=%v (want 5 across multiple cursor pages)", frozen, err)
	}
}

func TestPolicyService_PaginationAndUUIDs(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	polRepo := NewMemoryPolicyRepository()
	svc := NewPolicyService(polRepo, nil)
	if id := svc.newID(); len(id) != 36 || !strings.HasPrefix(id, "cpa_") {
		t.Fatalf("unexpected PolicyService default ID %q", id)
	}
	creditSvc := NewService(newMultiCreditMemoryRepo())
	if id := creditSvc.newID(); len(id) != 36 || !strings.HasPrefix(id, "crp_") {
		t.Fatalf("unexpected Service default ID %q", id)
	}

	for i := 1; i <= 3; i++ {
		rid := "ret-" + string(rune('0'+i))
		if err := polRepo.UpsertTerms(context.Background(), RetailerPaymentTerms{
			RetailerID: rid,
			SupplierID: "sup-1",
			UpdatedAt:  now.Add(time.Duration(i) * time.Minute),
		}, nil); err != nil {
			t.Fatal(err)
		}
	}

	page1, next1, err := svc.ListRelationshipsPage(context.Background(), "sup-1", 2, "")
	if err != nil || len(page1) != 2 || next1 == "" {
		t.Fatalf("page1=%d next=%q err=%v", len(page1), next1, err)
	}
	page2, next2, err := svc.ListRelationshipsPage(context.Background(), "sup-1", 2, next1)
	if err != nil || len(page2) != 1 || next2 != "" || page2[0].RetailerID != "ret-1" {
		t.Fatalf("page2=%+v next=%q err=%v", page2, next2, err)
	}
}

func TestHandleUpsertSupplierProfile_PrepayOnlyLockedAndRejectInvalidStatus(t *testing.T) {
	t.Setenv("CREDIT_SCORING_ENABLED", "true")
	repo := newMultiCreditMemoryRepo()
	repo.profiles[repo.key("ret-1", "sup-1")] = Profile{
		RetailerID:       "ret-1",
		SupplierID:       "sup-1",
		CreditLimitMinor: 500000,
		RiskScore:        5,
		Status:           StatusActive,
	}
	svc := NewService(repo)

	patch := func(body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodPatch, "/v1/supplier/retailer-credit-profile", strings.NewReader(body))
		req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
			Subject:    "admin-1",
			Role:       auth.RoleAdmin,
			SupplierID: "sup-1",
		}))
		rr := httptest.NewRecorder()
		svc.HandleUpsertSupplierProfile(rr, req)
		return rr
	}

	// Legacy pseudo-status PREPAY_ONLY must be rejected; PrepayOnlyLocked bool is authoritative.
	if rr := patch(`{"retailer_id":"ret-1","status":"PREPAY_ONLY"}`); rr.Code != http.StatusBadRequest {
		t.Fatalf("PREPAY_ONLY status=%d want 400 body=%s", rr.Code, rr.Body.String())
	}

	// Locking prepay_only_locked preserves CreditLimitMinor and StatusActive.
	if rr := patch(`{"retailer_id":"ret-1","prepay_only_locked":true}`); rr.Code != http.StatusOK {
		t.Fatalf("lock prepay_only_locked status=%d body=%s", rr.Code, rr.Body.String())
	}
	got, _, _ := repo.GetProfile(context.Background(), "ret-1", "sup-1")
	if !got.PrepayOnlyLocked || got.CreditLimitMinor != 500000 || got.Status != StatusActive {
		t.Fatalf("unexpected profile after lock: %+v", got)
	}

	// Supplier collections list marks NeedsAttention when PrepayOnlyLocked is true,
	// and never solely due to critical RiskScore when unlocked.
	listReq := httptest.NewRequest(http.MethodGet, "/v1/supplier/credit-profiles", nil)
	listReq = listReq.WithContext(auth.WithClaims(listReq.Context(), auth.Claims{
		Subject:    "admin-1",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-1",
	}))
	listRR := httptest.NewRecorder()
	svc.HandleListSupplierProfiles(listRR, listReq)
	if listRR.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", listRR.Code, listRR.Body.String())
	}
	var listResp struct {
		Profiles []struct {
			NeedsAttention   bool `json:"needs_attention"`
			PrepayOnlyLocked bool `json:"prepay_only_locked"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(listRR.Body.Bytes(), &listResp); err != nil {
		t.Fatal(err)
	}
	if len(listResp.Profiles) != 1 || !listResp.Profiles[0].PrepayOnlyLocked || !listResp.Profiles[0].NeedsAttention {
		t.Fatalf("expected PrepayOnlyLocked + NeedsAttention=true, got %+v", listResp.Profiles)
	}

	// Unlock prepay_only_locked: NeedsAttention must be false despite RiskScore=5.
	if rr := patch(`{"retailer_id":"ret-1","prepay_only_locked":false}`); rr.Code != http.StatusOK {
		t.Fatalf("unlock prepay_only_locked status=%d body=%s", rr.Code, rr.Body.String())
	}
	listRR2 := httptest.NewRecorder()
	svc.HandleListSupplierProfiles(listRR2, listReq)
	var listResp2 struct {
		Profiles []struct {
			NeedsAttention   bool `json:"needs_attention"`
			PrepayOnlyLocked bool `json:"prepay_only_locked"`
		} `json:"profiles"`
	}
	if err := json.Unmarshal(listRR2.Body.Bytes(), &listResp2); err != nil {
		t.Fatal(err)
	}
	if len(listResp2.Profiles) != 1 || listResp2.Profiles[0].PrepayOnlyLocked || listResp2.Profiles[0].NeedsAttention {
		t.Fatalf("expected PrepayOnlyLocked=false and NeedsAttention=false despite low risk score, got %+v", listResp2.Profiles)
	}
}
