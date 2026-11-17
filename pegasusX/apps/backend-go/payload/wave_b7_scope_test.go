package payload

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// B7 PL-P0-6: list/mutate warehouse scope helpers.

func TestManifestMatchesWarehouseScope(t *testing.T) {
	cases := []struct {
		row, scope string
		want       bool
	}{
		{"", "", true},
		{"wh-1", "", true},
		{"", "wh-1", true}, // historical empty WH still visible
		{"wh-1", "wh-1", true},
		{"wh-2", "wh-1", false},
	}
	for _, tc := range cases {
		if got := manifestMatchesWarehouseScope(tc.row, tc.scope); got != tc.want {
			t.Fatalf("row=%q scope=%q got=%v want=%v", tc.row, tc.scope, got, tc.want)
		}
	}
}

func TestAssertManifestWarehouseScope_Mismatch(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:         auth.RolePayload,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-home",
	}))
	err := svc.assertManifestWarehouseScope(req.Context(), ManifestRow{WarehouseID: "wh-other"})
	if err == nil {
		t.Fatal("expected scope error")
	}
	if err.Error() != "warehouse_scope_forbidden" {
		t.Fatalf("err=%v", err)
	}
}

func TestAssertManifestWarehouseScope_EmptyRowOK(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:         auth.RolePayload,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-home",
	}))
	if err := svc.assertManifestWarehouseScope(req.Context(), ManifestRow{}); err != nil {
		t.Fatalf("empty row should pass: %v", err)
	}
}

func TestResolveWarehouseScope_FromJWT(t *testing.T) {
	svc := &Service{}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role:         auth.RolePayload,
		HomeNodeType: auth.HomeNodeWarehouse,
		HomeNodeID:   "wh-jwt",
	}))
	if got := svc.resolveWarehouseScope(req.Context()); got != "wh-jwt" {
		t.Fatalf("got %q", got)
	}
}

func TestListManifestWires_FiltersByWarehouse(t *testing.T) {
	svc := &Service{
		manifests: []ManifestRow{
			{ManifestID: "m1", WarehouseID: "wh-a", State: "DRAFT", UpdatedAt: "2026-01-02"},
			{ManifestID: "m2", WarehouseID: "wh-b", State: "DRAFT", UpdatedAt: "2026-01-03"},
			{ManifestID: "m3", WarehouseID: "", State: "DRAFT", UpdatedAt: "2026-01-04"},
		},
		manifestOrders: map[string][]ManifestOrder{},
		overflowCount:  map[string]int64{},
	}
	wires := svc.listManifestWiresLocked("", "", "wh-a")
	ids := map[string]bool{}
	for _, w := range wires {
		ids[w.ManifestID] = true
	}
	if !ids["m1"] {
		t.Fatal("expected m1 (matching WH)")
	}
	if ids["m2"] {
		t.Fatal("m2 foreign warehouse must be filtered")
	}
	if !ids["m3"] {
		t.Fatal("m3 empty warehouse remains visible (historical)")
	}
}
