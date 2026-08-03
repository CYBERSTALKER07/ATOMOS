package auth

import "testing"

func TestHasRetailerPerm_ClaimFile(t *testing.T) {
	manager := Claims{
		Role: RoleRetailer, Subject: "user-mgr", RetailerOrgID: "ret-1",
		RetailerUserID: "user-mgr", RetailerRole: "MANAGER",
	}
	if !HasRetailerPerm(manager, PermClaimFile) {
		t.Fatal("MANAGER should have claim.file")
	}
	cashier := Claims{
		Role: RoleRetailer, Subject: "user-cash", RetailerOrgID: "ret-1",
		RetailerUserID: "user-cash", RetailerRole: "CASHIER",
	}
	if HasRetailerPerm(cashier, PermClaimFile) {
		t.Fatal("CASHIER must not have claim.file")
	}
	legacyOwner := Claims{Role: RoleRetailer, Subject: "ret-1"}
	if ResolveRetailerOrgID(legacyOwner) != "ret-1" {
		t.Fatal("legacy owner org should be subject")
	}
	if !HasRetailerPerm(legacyOwner, PermClaimFile) {
		t.Fatal("legacy OWNER should have claim.file")
	}
	buyer := Claims{
		Role: RoleRetailer, Subject: "u", RetailerOrgID: "ret-1", RetailerRole: "BUYER",
	}
	if HasRetailerPerm(buyer, PermClaimFile) {
		t.Fatal("BUYER must not have claim.file")
	}
}

func TestListRetailerPerms_IncludesClaimFileForManager(t *testing.T) {
	perms := ListRetailerPerms(Claims{Role: RoleRetailer, RetailerRole: "MANAGER"})
	found := false
	for _, p := range perms {
		if p == PermClaimFile {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("MANAGER perms missing claim.file: %v", perms)
	}
}
