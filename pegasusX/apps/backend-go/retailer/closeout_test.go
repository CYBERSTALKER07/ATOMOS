package retailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

func TestFamilyMigrateToTeam(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "fam-" + string(rune('A'+n%26))
		},
	})
	// Seed family in RAM
	svc.mu.Lock()
	svc.familyByRetailer["ret-fam"] = []FamilyMember{
		{MemberID: "f1", Name: "Helper One", Phone: "+998901111111", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)},
		{MemberID: "f2", Name: "No Phone", Phone: "", CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)},
	}
	svc.mu.Unlock()

	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-fam",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/family-members/migrate-to-team",
		bytes.NewBufferString(`{"retailer_role":"RECEIVER"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleFamilyMigrateToTeam(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("migrate status=%d body=%s", rr.Code, rr.Body.String())
	}
	var res FamilyMigrateResult
	_ = json.Unmarshal(rr.Body.Bytes(), &res)
	if len(res.Migrated) != 1 {
		t.Fatalf("migrated=%+v skipped=%+v", res.Migrated, res.Skipped)
	}
	if res.Migrated[0].TempPassword == "" {
		t.Fatal("expected temp password")
	}
	// f2 had no phone → skipped and left on family list
	if res.Remaining != 1 {
		t.Fatalf("remaining family=%d want 1 (no-phone skip)", res.Remaining)
	}
	if len(res.Skipped) < 1 {
		t.Fatalf("expected skip for no phone: %+v", res.Skipped)
	}

	// Family POST now 410
	reqPost := httptest.NewRequest(http.MethodPost, "/v1/retailer/family-members",
		bytes.NewBufferString(`{"name":"X","phone":"+1"}`))
	reqPost = reqPost.WithContext(auth.WithClaims(reqPost.Context(), owner))
	rrPost := httptest.NewRecorder()
	svc.HandleFamilyMembers(rrPost, reqPost)
	if rrPost.Code != http.StatusGone {
		t.Fatalf("expected 410, got %d body=%s", rrPost.Code, rrPost.Body.String())
	}

	// GET reports family_writes=gone
	reqGet := httptest.NewRequest(http.MethodGet, "/v1/retailer/family-members", nil)
	reqGet = reqGet.WithContext(auth.WithClaims(reqGet.Context(), owner))
	rrGet := httptest.NewRecorder()
	svc.HandleFamilyMembers(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("get status=%d", rrGet.Code)
	}
	var getBody map[string]any
	_ = json.Unmarshal(rrGet.Body.Bytes(), &getBody)
	if getBody["family_writes"] != "gone" {
		t.Fatalf("family_writes=%v want gone", getBody["family_writes"])
	}

	// Staff exists
	members, err := svc.listOrgMembers(t.Context(), "ret-fam")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range members {
		if m.Phone == "+998901111111" {
			found = true
			if m.RetailerRole != "RECEIVER" {
				t.Fatalf("role=%s", m.RetailerRole)
			}
		}
	}
	if !found {
		t.Fatalf("staff not created: %+v", members)
	}
}

func TestFamilyCreateAcceptsNickname(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "id1" }})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-nick",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/family-members",
		bytes.NewBufferString(`{"nickname":"Mobile Name","phone":"+998902222222"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleFamilyMembers(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	svc.mu.RLock()
	list := svc.familyByRetailer["ret-nick"]
	svc.mu.RUnlock()
	if len(list) != 1 || list[0].Name != "Mobile Name" || list[0].Phone != "+998902222222" {
		t.Fatalf("members=%+v", list)
	}
}

func TestFamilyMigrateForbiddenWithoutPerm(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "x" }})
	// VIEWER lacks staff.manage
	claims := auth.Claims{
		Subject: "v", Role: auth.RoleRetailer, RetailerOrgID: "ret-x",
		RetailerRole: "VIEWER", RetailerUserID: "v",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/family-members/migrate-to-team",
		bytes.NewBufferString(`{}`))
	req = req.WithContext(auth.WithClaims(req.Context(), claims))
	rr := httptest.NewRecorder()
	svc.HandleFamilyMigrateToTeam(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("status=%d want 403", rr.Code)
	}
}

func TestAutoOrderWorkerDraft(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "ao-" + string(rune('A'+n%26))
		},
	})
	// Enable auto-order
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ao", "o", AutoOrderSettings{
		GlobalEnabled:     true,
		SupplierOverrides: []SupplierOverride{},
		CategoryOverrides: []CategoryOverride{},
		ProductOverrides:  []ProductOverride{},
		VariantOverrides:  []VariantOverride{},
	})
	svc.SeedAutoOrderCandidates("ret-ao", []AutoOrderCandidate{
		{SKU: "MILK", ProductID: "MILK", SupplierID: "sup-1", Qty: 3},
	})

	run1 := svc.RunAutoOrderForRetailer(t.Context(), "ret-ao", AutoOrderModeDraft)
	if run1.DraftLines != 1 || run1.Status != "OK" {
		t.Fatalf("run1=%+v", run1)
	}
	// Second tick same day → idempotent skip
	run2 := svc.RunAutoOrderForRetailer(t.Context(), "ret-ao", AutoOrderModeDraft)
	if run2.DraftLines != 0 {
		t.Fatalf("expected 0 second draft, got %+v", run2)
	}
	foundSkip := false
	for _, sk := range run2.Skipped {
		if sk.Reason == "already_processed_bucket" {
			foundSkip = true
		}
	}
	if !foundSkip {
		t.Fatalf("expected bucket skip: %+v", run2.Skipped)
	}

	// Disabled → skip all
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ao2", "o", AutoOrderSettings{GlobalEnabled: false})
	run3 := svc.RunAutoOrderForRetailer(t.Context(), "ret-ao2", AutoOrderModeDraft)
	if run3.Status != "SKIPPED_ALL" {
		t.Fatalf("run3=%+v", run3)
	}
}

func TestAutoOrderWorkerFromAIPredictions(t *testing.T) {
	t.Parallel()
	n := 0
	orders := &testOrderLifecycle{predictions: []order.RetailerAIPrediction{{
		OrderID:    "ord-pred-1",
		SupplierID: "sup-ai",
		Items: []order.LineItem{
			{SKU: "BREAD", Name: "Bread", Quantity: 2},
			{SKU: "MILK", Name: "Milk", Quantity: 1},
			{SKU: "BREAD", Name: "Bread", Quantity: 1}, // aggregate
		},
	}}}
	svc := NewService(ServiceConfig{
		Now:    time.Now,
		NewID:  func() string { n++; return "aop-" + string(rune('A'+n%26)) },
		Orders: orders,
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-ai", "o", AutoOrderSettings{GlobalEnabled: true})

	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-ai", AutoOrderModeDraft)
	if run.DraftLines != 2 {
		t.Fatalf("want 2 draft lines (BREAD+MILK), got %+v", run)
	}
	if run.Suggestions != 2 {
		t.Fatalf("suggestions=%d want 2", run.Suggestions)
	}
}

func TestAutoOrderHTTPRunAndRuns(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { n++; return "http-" + string(rune('A'+n%26)) },
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-http", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedAutoOrderCandidates("ret-http", []AutoOrderCandidate{
		{SKU: "EGG", ProductID: "EGG", SupplierID: "s1", Qty: 6},
	})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-http",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/settings/auto-order/run?mode=draft", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleAutoOrderRun(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("run status=%d body=%s", rr.Code, rr.Body.String())
	}
	var run AutoOrderRun
	_ = json.Unmarshal(rr.Body.Bytes(), &run)
	if run.DraftLines != 1 || run.Status != "OK" {
		t.Fatalf("run=%+v", run)
	}

	reqList := httptest.NewRequest(http.MethodGet, "/v1/retailer/settings/auto-order/runs", nil)
	reqList = reqList.WithContext(auth.WithClaims(reqList.Context(), owner))
	rrList := httptest.NewRecorder()
	svc.HandleAutoOrderRuns(rrList, reqList)
	if rrList.Code != http.StatusOK {
		t.Fatalf("runs status=%d", rrList.Code)
	}
	var list struct {
		Items []AutoOrderRun `json:"items"`
	}
	_ = json.Unmarshal(rrList.Body.Bytes(), &list)
	if len(list.Items) < 1 {
		t.Fatalf("items=%+v", list.Items)
	}
	if list.Items[0].RunID != run.RunID {
		t.Fatalf("newest run id=%s want %s", list.Items[0].RunID, run.RunID)
	}
}

type mockOrderCreator struct {
	calls int
	last  order.CreateRequest
	resp  order.CreateResponse
	err   error
}

func (m *mockOrderCreator) Create(_ context.Context, retailerID string, req order.CreateRequest) (order.CreateResponse, error) {
	m.calls++
	m.last = req
	if m.err != nil {
		return order.CreateResponse{}, m.err
	}
	if m.resp.OrderID == "" {
		m.resp.OrderID = "ord-placed-1"
		m.resp.TotalMinor = 4000
		m.resp.Status = order.StatusPending
	}
	return m.resp, nil
}

func TestAutoOrderWorkerPlaceMode(t *testing.T) {
	t.Parallel()
	// This test exercises place mechanics directly; bypass the soak-evidence gate
	// via the test seam (gate behavior is covered by auto_order_soak_gate_test.go).
	n := 0
	mock := &mockOrderCreator{}
	svc := NewService(ServiceConfig{
		Now:                   time.Now,
		NewID:                 func() string { n++; return fmt.Sprintf("pl-%d", n) },
		OrderCreator:          mock,
		AutoOrderPlaceEnabled: true,
	})
	svc.soakGateDisabled = true
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-place", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedReorderSuggestions("ret-place", []RetailerReorderSuggestion{
		{SKU: "SKU-A", SuggestedQty: 2, Sources: []string{"STORE_POS"}, Status: "OPEN"},
	})
	// Prefer explicit supplier on seed path via SeedAutoOrderCandidates for place grouping
	svc.SeedAutoOrderCandidates("ret-place", []AutoOrderCandidate{
		{SKU: "SKU-A", ProductID: "SKU-A", SupplierID: "sup-1", Qty: 2, Sources: []string{"STORE_POS"}},
	})

	// Seed wins over suggestions
	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-place", AutoOrderModePlace)
	if run.Status != "OK" && run.Status != "PARTIAL" {
		t.Fatalf("run=%+v", run)
	}
	if mock.calls != 1 {
		t.Fatalf("Create calls=%d want 1 run=%+v", mock.calls, run)
	}
	if run.PlacedLines != 1 || len(run.PlacedOrders) != 1 {
		t.Fatalf("placed=%+v lines=%d", run.PlacedOrders, run.PlacedLines)
	}
	if run.PlacedOrders[0].OrderID == "" {
		t.Fatal("missing order id")
	}
	if mock.last.SupplierID != "sup-1" {
		t.Fatalf("Create SupplierID=%q want sup-1", mock.last.SupplierID)
	}
	if mock.last.Source != order.OrderSourceAutoOrder {
		t.Fatalf("Create Source=%q want AUTO_ORDER", mock.last.Source)
	}
	if len(mock.last.H3Cell) != 15 {
		t.Fatalf("Create H3Cell len=%d want 15", len(mock.last.H3Cell))
	}
	// Second place same day → bucket
	run2 := svc.RunAutoOrderForRetailer(t.Context(), "ret-place", AutoOrderModePlace)
	if run2.PlacedLines != 0 {
		t.Fatalf("second place should be idempotent: %+v", run2)
	}
	// Draft never calls Create
	mock.calls = 0
	svc.SeedAutoOrderCandidates("ret-place2", []AutoOrderCandidate{
		{SKU: "SKU-B", SupplierID: "sup-1", Qty: 1},
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-place2", "o", AutoOrderSettings{GlobalEnabled: true})
	_ = svc.RunAutoOrderForRetailer(t.Context(), "ret-place2", AutoOrderModeDraft)
	if mock.calls != 0 {
		t.Fatalf("draft must not Create, calls=%d", mock.calls)
	}
}

func TestAutoOrderWorkerFromReorderSuggestions(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { n++; return "aos-" + string(rune('A'+n%26)) },
	})
	_ = svc.saveAutoOrderDurable(t.Context(), "ret-sug", "o", AutoOrderSettings{GlobalEnabled: true})
	svc.SeedReorderSuggestions("ret-sug", []RetailerReorderSuggestion{
		{SKU: "SKU-ST", SuggestedQty: 4, Sources: []string{"STORE_POS"}, Status: "OPEN"},
		{SKU: "local:bag", SuggestedQty: 99, Sources: []string{"STORE_POS"}, Status: "OPEN"}, // skipped
	})

	run := svc.RunAutoOrderForRetailer(t.Context(), "ret-sug", AutoOrderModeDraft)
	if run.CandidateSource != "reorder_suggestions" {
		t.Fatalf("candidate_source=%s want reorder_suggestions run=%+v", run.CandidateSource, run)
	}
	if run.DraftLines != 1 || run.Status != "OK" {
		t.Fatalf("run=%+v", run)
	}
	if run.Suggestions != 1 {
		// local: filtered → only one candidate counted
		t.Fatalf("suggestions_seen=%d", run.Suggestions)
	}
}

func TestAutoOrderRunInvalidMode(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "m" }})
	owner := auth.Claims{
		Subject: "o", Role: auth.RoleRetailer, RetailerOrgID: "ret-m",
		RetailerRole: "OWNER", RetailerUserID: "o",
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/settings/auto-order/run?mode=explode", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), owner))
	rr := httptest.NewRecorder()
	svc.HandleAutoOrderRun(rr, req)
	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status=%d want 422", rr.Code)
	}
}
