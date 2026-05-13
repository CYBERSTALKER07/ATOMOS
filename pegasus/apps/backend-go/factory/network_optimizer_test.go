package factory

import "testing"

func TestFactorySupportsProduct(t *testing.T) {
	tests := []struct {
		name         string
		productID    string
		productTypes []string
		want         bool
	}{
		{
			name:         "empty product id accepts all",
			productID:    "",
			productTypes: []string{"MILK"},
			want:         true,
		},
		{
			name:         "exact product match",
			productID:    "sku-123",
			productTypes: []string{"SKU-123"},
			want:         true,
		},
		{
			name:         "wildcard match",
			productID:    "sku-123",
			productTypes: []string{"ALL"},
			want:         true,
		},
		{
			name:         "no explicit mapping",
			productID:    "sku-123",
			productTypes: nil,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := factorySupportsProduct(tt.productID, tt.productTypes)
			if got != tt.want {
				t.Fatalf("factorySupportsProduct() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestChooseFallbackFactory_PrefersProductMappedFactories(t *testing.T) {
	candidates := []fallbackFactoryCandidate{
		{
			FactoryID:    "factory-near",
			Lat:          41.3111,
			Lng:          69.2797,
			ProductTypes: nil,
		},
		{
			FactoryID:    "factory-mapped",
			Lat:          41.4000,
			Lng:          69.3500,
			ProductTypes: []string{"SKU-42"},
		},
	}

	selected, ok := chooseFallbackFactory("sku-42", 41.31, 69.28, "", "", candidates)
	if !ok {
		t.Fatal("expected fallback candidate")
	}
	if selected.FactoryID != "factory-mapped" {
		t.Fatalf("selected factory = %q, want %q", selected.FactoryID, "factory-mapped")
	}
}

func TestChooseFallbackFactory_UsesNearestWhenNoProductMappingExists(t *testing.T) {
	candidates := []fallbackFactoryCandidate{
		{
			FactoryID:    "factory-near",
			Lat:          41.3111,
			Lng:          69.2797,
			ProductTypes: nil,
		},
		{
			FactoryID:    "factory-far",
			Lat:          42.0000,
			Lng:          70.0000,
			ProductTypes: []string{"OTHER-SKU"},
		},
	}

	selected, ok := chooseFallbackFactory("sku-42", 41.31, 69.28, "", "", candidates)
	if !ok {
		t.Fatal("expected fallback candidate")
	}
	if selected.FactoryID != "factory-near" {
		t.Fatalf("selected factory = %q, want %q", selected.FactoryID, "factory-near")
	}
}

func TestChooseFallbackFactory_UsesPrimaryFactoryWithoutCoordinates(t *testing.T) {
	candidates := []fallbackFactoryCandidate{
		{FactoryID: "factory-a"},
		{FactoryID: "factory-b"},
	}

	selected, ok := chooseFallbackFactory("sku-42", 0, 0, "factory-b", "", candidates)
	if !ok {
		t.Fatal("expected fallback candidate")
	}
	if selected.FactoryID != "factory-b" {
		t.Fatalf("selected factory = %q, want %q", selected.FactoryID, "factory-b")
	}
}
