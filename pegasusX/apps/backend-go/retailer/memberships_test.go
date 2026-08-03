package retailer

import (
	"context"
	"testing"
	"time"
)

func TestUpsertAndListMembershipsByUser_Memory(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   func() time.Time { return time.Date(2026, 8, 10, 12, 0, 0, 0, time.UTC) },
		NewID: func() string { return "uid-m1" },
	})
	ctx := context.Background()

	if err := svc.UpsertMembership(ctx, RetailerMembership{
		UserID: "person-1", RetailerID: "org-a", RetailerRole: "OWNER", IsActive: true,
		Phone: "+998901", Name: "Ada",
	}); err != nil {
		t.Fatal(err)
	}
	if err := svc.UpsertMembership(ctx, RetailerMembership{
		UserID: "person-1", RetailerID: "org-b", RetailerRole: "MANAGER", IsActive: true,
		Phone: "+998901", Name: "Ada",
	}); err != nil {
		t.Fatal(err)
	}
	// Inactive membership must not appear in active-only list
	if err := svc.UpsertMembership(ctx, RetailerMembership{
		UserID: "person-1", RetailerID: "org-c", RetailerRole: "CASHIER", IsActive: false,
	}); err != nil {
		t.Fatal(err)
	}

	ms, err := svc.ListMembershipsByUser(ctx, "person-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 2 {
		t.Fatalf("expected 2 active memberships, got %d: %+v", len(ms), ms)
	}
	if svc.MembershipCountForUser(ctx, "person-1") != 2 {
		t.Fatalf("MembershipCountForUser=%d", svc.MembershipCountForUser(ctx, "person-1"))
	}
}

func TestListMembershipsByPhone_MultiOrg_Memory(t *testing.T) {
	t.Parallel()
	ids := []string{"owner-a", "owner-b"}
	i := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			id := ids[i%len(ids)]
			i++
			return id
		},
	})
	ctx := context.Background()
	phone := "+998909990001"

	// Same phone as owner of two different orgs (legacy per-org user rows).
	if _, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "org-1", Phone: phone, Name: "Shop 1"}); err != nil {
		t.Fatal(err)
	}
	// Second org: force a second owner row with same phone via memory maps + membership dual-write.
	// EnsureOwnerUser would reuse owner for same RetailerID only; different retailer gets new owner.
	// Reset NewID for second ensure
	svc.newID = func() string { return "owner-b" }
	if _, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "org-2", Phone: phone, Name: "Shop 2"}); err != nil {
		t.Fatal(err)
	}

	ms, err := svc.ListMembershipsByPhone(ctx, phone)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) < 2 {
		t.Fatalf("expected multi-org memberships for phone, got %d: %+v", len(ms), ms)
	}
	seen := map[string]bool{}
	for _, m := range ms {
		if m.Phone != phone && m.Phone != "" {
			// phone may be enriched
		}
		seen[m.RetailerID] = true
	}
	if !seen["org-1"] || !seen["org-2"] {
		t.Fatalf("expected org-1 and org-2, got %+v", seen)
	}
}

func TestDualWrite_EnsureOwnerAndCreateMember(t *testing.T) {
	t.Parallel()
	ids := []string{"owner-1", "staff-1"}
	idx := 0
	svc := NewService(ServiceConfig{
		Now: time.Now,
		NewID: func() string {
			id := ids[idx]
			if idx < len(ids)-1 {
				idx++
			}
			return id
		},
	})
	ctx := context.Background()

	owner, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "ret-1", Phone: "+99890001", Name: "Main"})
	if err != nil {
		t.Fatal(err)
	}
	ms, err := svc.ListMembershipsByUser(ctx, owner.UserID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].RetailerID != "ret-1" || ms[0].RetailerRole != "OWNER" {
		t.Fatalf("owner membership: %+v", ms)
	}

	staff := RetailerUser{
		UserID: "staff-1", RetailerID: "ret-1", Phone: "+99890002", Name: "Cash",
		RetailerRole: "CASHIER", IsActive: true,
	}
	if err := svc.createOrgMember(ctx, staff); err != nil {
		t.Fatal(err)
	}
	ms2, err := svc.ListMembershipsByUser(ctx, "staff-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms2) != 1 || ms2[0].RetailerRole != "CASHIER" {
		t.Fatalf("staff membership: %+v", ms2)
	}
}

func TestDualRead_SynthesizeFromUserWhenMembershipMissing(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { return "legacy-owner" },
	})
	ctx := context.Background()

	// Seed owner without going through membership dual-write path: inject owner map only.
	svc.mu.Lock()
	svc.ownerByRetailer = map[string]RetailerUser{
		"ret-legacy": {
			UserID: "legacy-owner", RetailerID: "ret-legacy", Phone: "+1", Name: "Old",
			RetailerRole: "OWNER", IsOwner: true, IsActive: true,
		},
	}
	// Ensure memberships map is empty / nil so dual-read synthesizes
	svc.membershipsByUser = nil
	svc.mu.Unlock()

	ms, err := svc.ListMembershipsByUser(ctx, "legacy-owner")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 1 || ms[0].RetailerID != "ret-legacy" {
		t.Fatalf("expected synthesized membership, got %+v", ms)
	}
}

func TestBackfillMembershipsFromUsers_Memory(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { return "bf-owner" },
	})
	ctx := context.Background()

	// Seed users without membership dual-write
	svc.mu.Lock()
	svc.ownerByRetailer = map[string]RetailerUser{
		"r1": {UserID: "u1", RetailerID: "r1", Phone: "+a", Name: "A", RetailerRole: "OWNER", IsActive: true},
	}
	svc.staffByRetailer = map[string][]RetailerUser{
		"r1": {{UserID: "u2", RetailerID: "r1", Phone: "+b", Name: "B", RetailerRole: "CASHIER", IsActive: true}},
		"r2": {{UserID: "u3", RetailerID: "r2", Phone: "+c", Name: "C", RetailerRole: "MANAGER", IsActive: true}},
	}
	svc.membershipsByUser = nil
	svc.mu.Unlock()

	n, err := svc.BackfillMembershipsFromUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("backfill n=%d want 3", n)
	}
	// Idempotent
	n2, err := svc.BackfillMembershipsFromUsers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if n2 != 3 {
		t.Fatalf("re-backfill n=%d", n2)
	}
	if svc.MembershipCountForUser(ctx, "u1") != 1 {
		t.Fatal("u1")
	}
	if svc.MembershipCountForUser(ctx, "u2") != 1 {
		t.Fatal("u2")
	}
	if svc.MembershipCountForUser(ctx, "u3") != 1 {
		t.Fatal("u3")
	}
}

func TestUpdateOrgMember_DualWritesMembership(t *testing.T) {
	t.Parallel()
	svc := NewService(ServiceConfig{
		Now:   time.Now,
		NewID: func() string { return "own-upd" },
	})
	ctx := context.Background()
	owner, err := svc.EnsureOwnerUser(ctx, Retailer{RetailerID: "ret-u", Phone: "+upd1", Name: "S"})
	if err != nil {
		t.Fatal(err)
	}
	staff := RetailerUser{
		UserID: "staff-upd", RetailerID: "ret-u", Phone: "+upd2", Name: "Cash",
		RetailerRole: "CASHIER", IsActive: true,
	}
	if err := svc.createOrgMember(ctx, staff); err != nil {
		t.Fatal(err)
	}
	staff.IsActive = false
	staff.RetailerRole = "MANAGER"
	if err := svc.updateOrgMember(ctx, staff); err != nil {
		t.Fatal(err)
	}
	// Active-only list should exclude deactivated
	ms, err := svc.ListMembershipsByUser(ctx, "staff-upd")
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) != 0 {
		t.Fatalf("expected inactive membership filtered, got %+v", ms)
	}
	// All memberships (including inactive) via internal map
	svc.mu.RLock()
	m := svc.membershipsByUser["staff-upd"]["ret-u"]
	svc.mu.RUnlock()
	if m.IsActive || m.RetailerRole != "MANAGER" {
		t.Fatalf("membership not updated: %+v", m)
	}
	_ = owner
}
