package cache

import (
	"fmt"
	"strings"
	"time"
)

// Canonical named cache TTL constants.
// Eliminates magic numbers and ensures consistent retention across domains.
const (
	TTLDefault             = 5 * time.Minute
	TTLRateLimitDefault    = 1 * time.Minute
	TTLIdempotencyAPI      = 24 * time.Hour
	TTLIdempotencyWebhook  = 7 * 24 * time.Hour
	TTLSession             = 2 * time.Hour
	TTLTelemetry           = 10 * time.Minute
	TTLCatalog             = 15 * time.Minute
	TTLInventory           = 2 * time.Minute
	TTLSemanticCache       = 24 * time.Hour
)

// Standard domain cache prefixes.
const (
	PrefixRateLimit        = "ratelimit"
	PrefixIdempotency      = "idempotency"
	PrefixSupplierProfile  = "supplier:profile"
	PrefixRetailerProfile  = "retailer:profile"
	PrefixDriverProfile    = "driver:profile"
	PrefixFactoryProfile   = "factory:profile"
	PrefixCatalogSearch    = "catalog:search"
	PrefixAnalytics        = "analytics:dashboard"
	PrefixSettings         = "settings"
	PrefixTelemetry        = "telemetry"
	PrefixOrder            = "order"
	PrefixWarehouse        = "warehouse"
	PrefixInventory        = "inventory"
	PrefixSemanticCache    = "semantic"
)

// Key constructs a canonical colon-separated key using hash tags on the entity and ID.
// Format: {entity:id}:attribute1:attribute2...
// Hash tags ensure that all keys belonging to the same entity land in the same Redis Cluster
// slot, preventing CROSSSLOT errors during multi-key operations (MGET, pipelines, transactions).
func Key(entity, id string, attributes ...string) string {
	entity = strings.ToLower(strings.TrimSpace(entity))
	id = strings.TrimSpace(id)
	cleanAttrs := make([]string, 0, len(attributes))
	for _, attr := range attributes {
		trimmed := strings.ToLower(strings.TrimSpace(attr))
		if trimmed != "" {
			cleanAttrs = append(cleanAttrs, trimmed)
		}
	}

	tag := fmt.Sprintf("{%s:%s}", entity, id)
	if len(cleanAttrs) == 0 {
		return tag
	}
	return tag + ":" + strings.Join(cleanAttrs, ":")
}

// SimpleKey constructs a standard colon-separated key without cluster hash tags.
// Format: prefix:scope:id
func SimpleKey(prefix, scope, id string) string {
	parts := []string{strings.ToLower(strings.TrimSpace(prefix))}
	if s := strings.TrimSpace(scope); s != "" {
		parts = append(parts, s)
	}
	if i := strings.TrimSpace(id); i != "" {
		parts = append(parts, i)
	}
	return strings.Join(parts, ":")
}

// SupplierScopedKey prefixes a supplier ID with the cluster hash-tag convention.
func SupplierScopedKey(supplierID, suffix string) string {
	return fmt.Sprintf("{sup:%s}:%s", supplierID, strings.TrimPrefix(suffix, ":"))
}

// RetailerScopedKey prefixes a retailer ID with the cluster hash-tag convention.
func RetailerScopedKey(retailerID, suffix string) string {
	return fmt.Sprintf("{ret:%s}:%s", retailerID, strings.TrimPrefix(suffix, ":"))
}

// OrderScopedKey prefixes an order ID with the cluster hash-tag convention.
func OrderScopedKey(orderID, suffix string) string {
	return fmt.Sprintf("{ord:%s}:%s", orderID, strings.TrimPrefix(suffix, ":"))
}

// DriverScopedKey prefixes a driver ID with the cluster hash-tag convention.
func DriverScopedKey(driverID, suffix string) string {
	return fmt.Sprintf("{drv:%s}:%s", driverID, strings.TrimPrefix(suffix, ":"))
}
