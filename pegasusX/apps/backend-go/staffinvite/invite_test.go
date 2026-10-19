package staffinvite

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestMintParseRoundTrip(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	tok, exp, err := Mint("secret", RoleFactory, "sup-1", "fac-1", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	inv, err := Parse("secret", tok, now.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if inv.SupplierID != "sup-1" || inv.NodeID != "fac-1" || inv.Role != RoleFactory {
		t.Fatalf("%+v", inv)
	}
	if !exp.Equal(inv.ExpiresAt) {
		t.Fatalf("exp %v vs %v", exp, inv.ExpiresAt)
	}
}

func TestParseExpired(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	tok, _, err := Mint("secret", RoleWarehouse, "sup-1", "wh-1", time.Minute, now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Parse("secret", tok, now.Add(2*time.Minute)); err != ErrInviteExpired {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveRegister_RequiresInvite(t *testing.T) {
	_, err := ResolveRegister(ResolveOpts{WantRole: RoleFactory, Secret: "s"})
	if err != ErrInviteRequired {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveRegister_Invite(t *testing.T) {
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	tok, _, err := Mint("s", RoleFactory, "sup-minted", "fac-9", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	scope, err := ResolveRegister(ResolveOpts{
		Secret: "s", InviteToken: tok, WantRole: RoleFactory, Now: now, SeedSupplierID: "seed-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if scope.SupplierID != "sup-minted" || scope.NodeID != "fac-9" {
		t.Fatalf("%+v", scope)
	}
}

func TestResolveRegister_RejectsSeedOutsideSSMR(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	tok, _, err := Mint("s", RoleFactory, "seed-1", "fac-1", time.Hour, now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ResolveRegister(ResolveOpts{
		Secret: "s", InviteToken: tok, WantRole: RoleFactory, Now: now, SeedSupplierID: "seed-1",
	})
	if err != ErrSeedStaffForbidden {
		t.Fatalf("err=%v", err)
	}
}

func TestResolveRegister_AdminOwnsNode(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	scope, err := ResolveRegister(ResolveOpts{
		WantRole:       RoleWarehouse,
		RequestedNode:  "wh-1",
		SeedSupplierID: "seed-1",
		Admin:          &auth.Claims{Role: auth.RoleAdmin, SupplierID: "sup-minted"},
		NodeOwned: func(_ context.Context, sid, role, nid string) (bool, error) {
			return sid == "sup-minted" && role == RoleWarehouse && nid == "wh-1", nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if scope.SupplierID != "sup-minted" || scope.NodeID != "wh-1" {
		t.Fatalf("%+v", scope)
	}
}

func TestHandleCreate_MintsForAdmin(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	now := time.Date(2026, 8, 16, 8, 0, 0, 0, time.UTC)
	h := &Handler{
		Secret:         "s",
		SeedSupplierID: "seed-1",
		Now:            func() time.Time { return now },
		NodeOwned: func(_ context.Context, _, _, _ string) (bool, error) {
			return true, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/supplier/staff-invites",
		strings.NewReader(`{"role":"factory","node_id":"fac-1"}`))
	req = req.WithContext(auth.WithClaims(req.Context(), auth.Claims{
		Role: auth.RoleAdmin, SupplierID: "sup-minted",
	}))
	rr := httptest.NewRecorder()
	h.HandleCreate(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["supplier_id"] != "sup-minted" || body["token"] == "" {
		t.Fatalf("%v", body)
	}
}
