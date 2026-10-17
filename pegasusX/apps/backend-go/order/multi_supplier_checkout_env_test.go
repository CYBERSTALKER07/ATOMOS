package order

import "testing"

func TestMultiSupplierCheckoutEnabled_SandboxDefault(t *testing.T) {
	t.Setenv("MULTI_SUPPLIER_CHECKOUT_ENABLED", "")
	t.Setenv("PEGASUSX_ENV", "sandbox")
	if !MultiSupplierCheckoutEnabled() {
		t.Fatal("sandbox must default multi-supplier checkout on")
	}
	t.Setenv("PEGASUSX_ENV", "ssmr")
	if !MultiSupplierCheckoutEnabled() {
		t.Fatal("ssmr alias must default multi-supplier checkout on")
	}
	t.Setenv("PEGASUSX_ENV", "production")
	if MultiSupplierCheckoutEnabled() {
		t.Fatal("production must not default multi-supplier checkout on")
	}
}
