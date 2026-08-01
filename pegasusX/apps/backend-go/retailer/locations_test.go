package retailer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestEnsurePrimaryLocationMemory(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return time.Date(2026, 8, 2, 0, 0, 0, 0, time.UTC) },
		NewID: func() string { return "loc-primary-1" },
	})
	loc, err := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	if err != nil {
		t.Fatal(err)
	}
	if !loc.IsPrimary || loc.LocationID != "loc-primary-1" {
		t.Fatalf("unexpected primary: %+v", loc)
	}
	// Idempotent second call
	loc2, err := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	if err != nil {
		t.Fatal(err)
	}
	if loc2.LocationID != loc.LocationID {
		t.Fatalf("expected same primary, got %s vs %s", loc2.LocationID, loc.LocationID)
	}
}

func TestLocationsCreateAndList(t *testing.T) {
	t.Parallel()
	n := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			n++
			return "loc-" + string(rune('a'+n-1))
		},
	})
	// seed primary
	_, _ = svc.EnsurePrimaryLocation(t.Context(), "ret-1")

	body := `{"name":"Branch North","lat":41.4,"lng":69.3,"delivery_address":"North St 1"}`
	req := httptest.NewRequest(http.MethodPost, "/v1/retailer/locations", bytes.NewBufferString(body))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
	}))
	rr := httptest.NewRecorder()
	svc.HandleLocations(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", rr.Code, rr.Body.String())
	}

	reqGet := httptest.NewRequest(http.MethodGet, "/v1/retailer/locations", nil)
	reqGet = reqGet.WithContext(auth.WithClaims(reqGet.Context(), auth.Claims{
		Subject: "owner", Role: auth.RoleRetailer, RetailerOrgID: "ret-1", RetailerRole: "OWNER",
	}))
	rrGet := httptest.NewRecorder()
	svc.HandleLocations(rrGet, reqGet)
	if rrGet.Code != http.StatusOK {
		t.Fatalf("list status=%d body=%s", rrGet.Code, rrGet.Body.String())
	}
	var resp locationsResponse
	if err := json.Unmarshal(rrGet.Body.Bytes(), &resp); err != nil {
		t.Fatal(err)
	}
	if len(resp.Items) < 2 {
		t.Fatalf("expected primary+branch, got %d", len(resp.Items))
	}
}

func TestSwitchLocationForbiddenOutOfScope(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "loc-x" }, JWTSecret: "s", JWTIssuer: "t"})
	primary, _ := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	// bind staff to empty set then only primary via replace
	_ = svc.replaceUserLocations(t.Context(), "ret-1", "staff-1", []string{primary.LocationID})

	// create second location manually in memory
	branch := RetailerLocation{
		LocationID: "loc-branch", RetailerID: "ret-1", Name: "B", IsActive: true, IsPrimary: false,
	}
	_ = svc.insertLocation(t.Context(), branch)

	req := httptest.NewRequest(http.MethodPost, "/v1/auth/retailer/switch-location",
		bytes.NewBufferString(`{"location_id":"loc-branch"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Subject: "staff-1", Role: auth.RoleRetailer, RetailerOrgID: "ret-1",
		RetailerRole: "CASHIER", RetailerUserID: "staff-1",
	}))
	rr := httptest.NewRecorder()
	svc.HandleSwitchLocation(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestMemberLocationBind(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{Now: time.Now, NewID: func() string { return "loc-p" }})
	primary, _ := svc.EnsurePrimaryLocation(t.Context(), "ret-1")
	if err := svc.replaceUserLocations(t.Context(), "ret-1", "u1", []string{primary.LocationID}); err != nil {
		t.Fatal(err)
	}
	ids, err := svc.listUserLocationIDs(t.Context(), "u1")
	if err != nil || len(ids) != 1 || ids[0] != primary.LocationID {
		t.Fatalf("bind failed: %v %v", ids, err)
	}
}
