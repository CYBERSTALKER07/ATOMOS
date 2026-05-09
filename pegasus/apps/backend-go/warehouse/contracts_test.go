package warehouse

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInventorySearchTerm(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{name: "prefers q", url: "/v1/warehouse/ops/inventory?q=cola&search=juice", want: "cola"},
		{name: "falls back to search", url: "/v1/warehouse/ops/inventory?search=juice", want: "juice"},
		{name: "empty", url: "/v1/warehouse/ops/inventory", want: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", test.url, nil)
			if got := inventorySearchTerm(request); got != test.want {
				t.Fatalf("inventorySearchTerm(%q) = %q, want %q", test.url, got, test.want)
			}
		})
	}
}

func TestInventoryMutationSKU(t *testing.T) {
	tests := []struct {
		name      string
		skuID     string
		productID string
		want      string
	}{
		{name: "prefers sku_id", skuID: "sku-1", productID: "prod-1", want: "sku-1"},
		{name: "falls back to product_id", skuID: "", productID: "prod-1", want: "prod-1"},
		{name: "trims whitespace", skuID: "  ", productID: " prod-1 ", want: "prod-1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := inventoryMutationSKU(test.skuID, test.productID); got != test.want {
				t.Fatalf("inventoryMutationSKU(%q, %q) = %q, want %q", test.skuID, test.productID, got, test.want)
			}
		})
	}
}

func TestNormalizeInventoryItemAliases(t *testing.T) {
	item := InventoryItem{SkuID: "sku-42"}
	normalizeInventoryItemAliases(&item)

	if item.ProductID != "sku-42" {
		t.Fatalf("ProductID = %q, want sku-42", item.ProductID)
	}
	if item.SKU != "sku-42" {
		t.Fatalf("SKU = %q, want sku-42", item.SKU)
	}
}

func TestNormalizeInventoryItemAliases_PreservesExistingAliases(t *testing.T) {
	item := InventoryItem{SkuID: "sku-42", ProductID: "product-9", SKU: "legacy-sku"}
	normalizeInventoryItemAliases(&item)

	if item.ProductID != "product-9" {
		t.Fatalf("ProductID = %q, want product-9", item.ProductID)
	}
	if item.SKU != "legacy-sku" {
		t.Fatalf("SKU = %q, want legacy-sku", item.SKU)
	}
}

func TestNormalizeWarehouseStaffRole(t *testing.T) {
	tests := []struct {
		name string
		role string
		want string
	}{
		{name: "warehouse staff", role: "WAREHOUSE_STAFF", want: "WAREHOUSE_STAFF"},
		{name: "payloader", role: "PAYLOADER", want: "PAYLOADER"},
		{name: "invalid defaults", role: "WAREHOUSE_ADMIN", want: "WAREHOUSE_STAFF"},
		{name: "empty defaults", role: "", want: "WAREHOUSE_STAFF"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := normalizeWarehouseStaffRole(test.role); got != test.want {
				t.Fatalf("normalizeWarehouseStaffRole(%q) = %q, want %q", test.role, got, test.want)
			}
		})
	}
}

func TestVehicleAvailabilityStatus(t *testing.T) {
	if got := vehicleAvailabilityStatus(true); got != "AVAILABLE" {
		t.Fatalf("vehicleAvailabilityStatus(true) = %q, want AVAILABLE", got)
	}
	if got := vehicleAvailabilityStatus(false); got != "INACTIVE" {
		t.Fatalf("vehicleAvailabilityStatus(false) = %q, want INACTIVE", got)
	}
}

func TestOpsPaymentConfigQueryTargetsSupplierConfigs(t *testing.T) {
	if !strings.Contains(opsPaymentConfigQuery, "FROM SupplierPaymentConfigs") {
		t.Fatalf("opsPaymentConfigQuery = %q, want SupplierPaymentConfigs source", opsPaymentConfigQuery)
	}
	if strings.Contains(opsPaymentConfigQuery, "SupplierPaymentGateways") {
		t.Fatalf("opsPaymentConfigQuery = %q, still references stale SupplierPaymentGateways", opsPaymentConfigQuery)
	}
}

func TestBuildOpsPaymentGatewayItem(t *testing.T) {
	updatedAt := time.Date(2026, time.May, 9, 7, 30, 0, 0, time.UTC)
	item := buildOpsPaymentGatewayItem("cfg-1", "GLOBAL_PAY", "merchant-1", "service-7", true, updatedAt)

	if item.ConfigID != "cfg-1" {
		t.Fatalf("ConfigID = %q, want cfg-1", item.ConfigID)
	}
	if item.GatewayID != "cfg-1" {
		t.Fatalf("GatewayID = %q, want cfg-1", item.GatewayID)
	}
	if item.GatewayName != "GLOBAL_PAY" {
		t.Fatalf("GatewayName = %q, want GLOBAL_PAY", item.GatewayName)
	}
	if item.Provider != "GLOBAL_PAY" {
		t.Fatalf("Provider = %q, want GLOBAL_PAY", item.Provider)
	}
	if item.Mode != "MANUAL_ONLY" {
		t.Fatalf("Mode = %q, want MANUAL_ONLY", item.Mode)
	}
	if item.MerchantID != "merchant-1" {
		t.Fatalf("MerchantID = %q, want merchant-1", item.MerchantID)
	}
	if item.ServiceID != "service-7" {
		t.Fatalf("ServiceID = %q, want service-7", item.ServiceID)
	}
	if item.LastUpdated != "2026-05-09T07:30:00Z" {
		t.Fatalf("LastUpdated = %q, want 2026-05-09T07:30:00Z", item.LastUpdated)
	}
}
