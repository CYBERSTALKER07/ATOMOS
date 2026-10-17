package partner

import (
	"context"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

func TestWithPartnerTenant_SupplierStampsIsolationKey(t *testing.T) {
	ctx := withPartnerTenant(context.Background(), Principal{
		TenantType: TenantSupplier, TenantID: "sup-1",
	})
	ten, ok := auth.TenantFromContext(ctx)
	if !ok || ten.SupplierID != "sup-1" {
		t.Fatalf("supplier partner must stamp tenant: %+v ok=%v", ten, ok)
	}
}

func TestWithPartnerTenant_RetailerDoesNotStampSupplierID(t *testing.T) {
	ctx := withPartnerTenant(context.Background(), Principal{
		TenantType: TenantRetailer, TenantID: "ret-1",
	})
	if _, ok := auth.TenantFromContext(ctx); ok {
		t.Fatal("retailer partner must not stamp TenantContext.SupplierID")
	}
	if _, ok := PrincipalFromContext(ctx); !ok {
		t.Fatal("retailer partner principal must still attach")
	}
}
