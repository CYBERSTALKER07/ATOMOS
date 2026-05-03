package warehouse

import (
	"net/http/httptest"
	"testing"
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
