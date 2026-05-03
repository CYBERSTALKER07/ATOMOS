package countrycfg

import (
	"testing"
	"time"

	"backend-go/cache"
)

func TestMergeCountryConfigForUpsertPreservesExistingFields(t *testing.T) {
	existing := &CountryConfig{
		CountryCode:                 "KZ",
		CountryName:                 "Kazakhstan",
		Timezone:                    "Asia/Almaty",
		CurrencyCode:                "KZT",
		CurrencyDecimalPlaces:       0,
		DistanceUnit:                "km",
		DefaultVUConversion:         2.5,
		MapsProvider:                "YANDEX",
		LLMProvider:                 "GEMINI",
		PaymentGateways:             []string{"KASPI"},
		SMSProvider:                 "TWILIO",
		NotificationFallbackOrder:   []string{"SMS", "FCM"},
		LegalRetentionDays:          90,
		GridSystem:                  "H3",
		BreachRadiusMeters:          275,
		ShopClosedGraceMinutes:      8,
		ShopClosedEscalationMinutes: 12,
		OfflineModeDurationMinutes:  45,
		CashCustodyAlertHours:       6,
	}

	incoming := &CountryConfig{
		CountryCode:        "KZ",
		CountryName:        "Kazakhstan Updated",
		BreachRadiusMeters: 320,
		PaymentGateways:    []string{"KASPI", "CASH"},
	}

	merged := mergeCountryConfigForUpsert(existing, incoming)

	if merged.CountryName != "Kazakhstan Updated" {
		t.Fatalf("CountryName = %q, want updated value", merged.CountryName)
	}
	if merged.BreachRadiusMeters != 320 {
		t.Fatalf("BreachRadiusMeters = %v, want 320", merged.BreachRadiusMeters)
	}
	if merged.CurrencyCode != "KZT" {
		t.Fatalf("CurrencyCode = %q, want existing value", merged.CurrencyCode)
	}
	if merged.MapsProvider != "YANDEX" {
		t.Fatalf("MapsProvider = %q, want existing value", merged.MapsProvider)
	}
	if merged.ShopClosedGraceMinutes != 8 {
		t.Fatalf("ShopClosedGraceMinutes = %d, want existing value", merged.ShopClosedGraceMinutes)
	}
	if merged.LegalRetentionDays != 90 {
		t.Fatalf("LegalRetentionDays = %d, want existing value", merged.LegalRetentionDays)
	}
	if len(merged.PaymentGateways) != 2 || merged.PaymentGateways[0] != "KASPI" || merged.PaymentGateways[1] != "CASH" {
		t.Fatalf("PaymentGateways = %#v, want updated slice", merged.PaymentGateways)
	}
}

func TestMergeCountryConfigForUpsertFallsBackToDefaultWhenMissing(t *testing.T) {
	merged := mergeCountryConfigForUpsert(nil, &CountryConfig{
		CountryCode: "TJ",
	})

	if merged.CountryCode != "TJ" {
		t.Fatalf("CountryCode = %q, want TJ", merged.CountryCode)
	}
	if merged.CountryName != "Uzbekistan" {
		t.Fatalf("CountryName = %q, want default fallback", merged.CountryName)
	}
	if merged.SMSProvider != "ESKIZ" {
		t.Fatalf("SMSProvider = %q, want default fallback", merged.SMSProvider)
	}
	if merged.ShopClosedGraceMinutes != 5 {
		t.Fatalf("ShopClosedGraceMinutes = %d, want default fallback", merged.ShopClosedGraceMinutes)
	}
}

func TestHandleInvalidation_CountryConfigClearsCountryAndOverrideCaches(t *testing.T) {
	svc := NewService(nil)
	svc.cache.Store("KZ", &cacheEntry{
		config:    &CountryConfig{CountryCode: "KZ"},
		expiresAt: time.Now().Add(time.Minute),
	})
	svc.overrideCache.Store("supplier-1:KZ", &overrideCacheEntry{
		override:  &SupplierOverride{SupplierId: "supplier-1", CountryCode: "KZ"},
		expiresAt: time.Now().Add(time.Minute),
	})
	svc.overrideCache.Store("supplier-2:US", &overrideCacheEntry{
		override:  &SupplierOverride{SupplierId: "supplier-2", CountryCode: "US"},
		expiresAt: time.Now().Add(time.Minute),
	})

	svc.handleInvalidation([]string{cache.CountryConfigCacheKey("KZ")})

	if _, ok := svc.cache.Load("KZ"); ok {
		t.Fatal("country cache entry should be evicted")
	}
	if _, ok := svc.overrideCache.Load("supplier-1:KZ"); ok {
		t.Fatal("country-specific override should be evicted")
	}
	if _, ok := svc.overrideCache.Load("supplier-2:US"); !ok {
		t.Fatal("unrelated override should remain cached")
	}
}

func TestHandleInvalidation_SupplierOverrideClearsOnlyTargetOverride(t *testing.T) {
	svc := NewService(nil)
	svc.cache.Store("KZ", &cacheEntry{
		config:    &CountryConfig{CountryCode: "KZ"},
		expiresAt: time.Now().Add(time.Minute),
	})
	svc.overrideCache.Store("supplier-1:KZ", &overrideCacheEntry{
		override:  &SupplierOverride{SupplierId: "supplier-1", CountryCode: "KZ"},
		expiresAt: time.Now().Add(time.Minute),
	})
	svc.overrideCache.Store("supplier-2:KZ", &overrideCacheEntry{
		override:  &SupplierOverride{SupplierId: "supplier-2", CountryCode: "KZ"},
		expiresAt: time.Now().Add(time.Minute),
	})

	svc.handleInvalidation([]string{cache.CountryOverrideCacheKey("supplier-1", "KZ")})

	if _, ok := svc.overrideCache.Load("supplier-1:KZ"); ok {
		t.Fatal("target override should be evicted")
	}
	if _, ok := svc.overrideCache.Load("supplier-2:KZ"); !ok {
		t.Fatal("other supplier override should remain cached")
	}
	if _, ok := svc.cache.Load("KZ"); !ok {
		t.Fatal("base country cache should remain cached")
	}
}
