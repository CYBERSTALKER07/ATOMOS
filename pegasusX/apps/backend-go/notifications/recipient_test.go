package notifications

import (
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestRecipientIDFromClaims_UsesSupplierScopeForAdminPortal(t *testing.T) {
	claims := auth.Claims{
		Subject:    "supplier-1",
		SupplierID: "supplier-1",
		Role:       auth.RoleAdmin,
	}
	if got := RecipientIDFromClaims(claims); got != "supplier-1" {
		t.Fatalf("recipientID = %q, want %q", got, "supplier-1")
	}
}

func TestRecipientIDFromClaims_UsesSupplierScopeForPayloadStaff(t *testing.T) {
	claims := auth.Claims{
		Subject:    "worker-1",
		SupplierID: "supplier-1",
		Role:       auth.RolePayload,
	}
	if got := RecipientIDFromClaims(claims); got != "supplier-1" {
		t.Fatalf("recipientID = %q, want %q", got, "supplier-1")
	}
}

func TestRecipientIDFromClaims_UsesSubjectForRetailer(t *testing.T) {
	claims := auth.Claims{
		Subject: "retailer-1",
		Role:    auth.RoleRetailer,
	}
	if got := RecipientIDFromClaims(claims); got != "retailer-1" {
		t.Fatalf("recipientID = %q, want %q", got, "retailer-1")
	}
}
