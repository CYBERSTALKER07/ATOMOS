// Package cache — Centralized Redis key constants and member helpers.
//
// Every Redis key used across the backend MUST be defined here.
// Raw string keys elsewhere in the codebase should reference these constants
// to prevent typos, enable auditing, and ensure consistent naming.
package cache

import "time"

// ─── Key Constants ─────────────────────────────────────────────────────────────
//
// Naming convention: service:entity:id[:field]
// Prefix grouping:
//   geo:       — GEO sorted sets (proximity, warehouses)
//   proximity: — proximity detection state
//   wh:        — warehouse operational data
//   whcell:    — warehouse grid cell index
//   whdetail:  — warehouse detail hashes
//   rl:        — rate limiting
//   idem:      — idempotency
//   desert:    — Desert Protocol (offline sync)
//   lb:        — load balancer
//   dlq:       — dead letter queue

const (
	// ── GEO & Proximity ────────────────────────────────────────────────────
	KeyGeoProximity  = "geo:proximity"      // GEO sorted set: driver/retailer positions
	KeyArrivingSet   = "proximity:arriving" // SET: order IDs already processed for approach alerts
	KeyGeoWarehouses = "geo:warehouses"     // GEO sorted set: warehouse positions

	// ── Warehouse ──────────────────────────────────────────────────────────
	PrefixWarehouseCell   = "whcell:"   // SET per grid cell: whcell:<cellId> → {warehouseId, ...}
	PrefixWarehouseDetail = "whdetail:" // HASH: whdetail:<warehouseId> → {...}
	PrefixWarehouseQueue  = "wh:queue:" // STRING (counter): wh:queue:<warehouseId>

	// ── Order ──────────────────────────────────────────────────────────────
	PrefixDeliveryToken  = "delivery_token:" // STRING: delivery_token:<orderId>
	PrefixActiveOrders   = "active_orders:"  // STRING (cached): active_orders:<retailerId>
	PrefixEarlyComplete  = "early_complete:" // STRING (JSON): early_complete:<driverId>
	PrefixManifestDetail = "manifest_detail:"
	PrefixManifestOrders = "manifest_orders:"

	PrefixPowerOutage = "power_outage:" // STRING (flag): power_outage:<orderId>

	// ── Idempotency ────────────────────────────────────────────────────────
	PrefixIdempotency     = "idem:" // STRING (JSON): idem:<key>
	SuffixIdempotencyLock = ":lock" // Appended to idem key: idem:<key>:lock

	// ── Rate Limiting ──────────────────────────────────────────────────────
	PrefixRateLimit = "rl:" // STRING (counter): rl:ip:<addr>

	// ── Token Bucket (Priority Load Shedder) ───────────────────────────────
	PrefixTokenBucket = "tb:" // STRING (atomic counter): tb:<priority>:<identity>

	// ── Desert Protocol (Offline Sync) ─────────────────────────────────────
	PrefixDesertSync = "desert:sync:" // STRING (SETNX lock): desert:sync:<driverId>:<orderId>

	// ── Payment ────────────────────────────────────────────────────────────
	PrefixWebhookSigFail = "webhook_sigfail:" // STRING (counter): webhook_sigfail:<provider>:<ref>

	// ── Load Balancer ──────────────────────────────────────────────────────
	PrefixLBCooldown = "lb:cooldown:" // STRING (flag): lb:cooldown:<warehouseId>

	// ── DLQ Resolution Ledger ──────────────────────────────────────────────
	KeyDLQResolved = "dlq:resolved_offsets" // SET: resolved DLQ offsets

	// ── HTTP Cache ─────────────────────────────────────────────────────────
	PrefixCacheCategories = "cache:categories" // STRING (JSON): cache:categories:<query>
	PrefixCacheProducts   = "cache:products"   // STRING (JSON): cache:products:<query>
	// ── Profile Cache ──────────────────────────────────────────────────
	PrefixSupplierProfile = "profile:supplier:" // STRING (JSON): profile:supplier:<supplierId>
	PrefixRetailerProfile = "profile:retailer:" // STRING (JSON): profile:retailer:<retailerId>
	PrefixDriverProfile   = "profile:driver:"   // STRING (JSON): profile:driver:<driverId>
	PrefixFactoryProfile  = "profile:factory:"  // STRING (JSON): profile:factory:<factoryId>

	// ── Catalog Search Cache ───────────────────────────────────────────
	PrefixCatalogSearch     = "cache:catalog:search:"     // STRING (JSON): cache:catalog:search:<query>
	PrefixCategorySuppliers = "cache:category:suppliers:" // STRING (JSON): cache:category:suppliers:<query>

	// ── Settings Cache ─────────────────────────────────────────────────
	PrefixSettings = "settings:" // STRING (JSON): settings:<retailerId>:<scope>

	// ── Analytics Cache ────────────────────────────────────────────────
	PrefixAnalytics = "analytics:" // STRING (JSON): analytics:<scope>:<id>

	// ── Country Config Invalidation ────────────────────────────────────
	PrefixCountryConfigCache   = "countrycfg:config:"   // signal-only invalidation key: countrycfg:config:<countryCode>
	PrefixCountryOverrideCache = "countrycfg:override:" // signal-only invalidation key: countrycfg:override:<supplierId>:<countryCode>
)

// ─── TTL Constants ─────────────────────────────────────────────────────────────
const (
	TTLDeliveryToken    = 4 * time.Hour
	TTLArrivingSet      = 24 * time.Hour
	TTLWarehouseGeo     = 24 * time.Hour
	TTLWarehouseQueue   = 24 * time.Hour
	TTLIdempotency      = 24 * time.Hour
	TTLIdempotencyLock  = 30 * time.Second
	TTLDesertSync       = 24 * time.Hour
	TTLWebhookSigFail   = 1 * time.Hour
	TTLLBCooldown       = 5 * time.Minute
	TTLEarlyComplete    = 2 * time.Hour
	TTLPowerOutage      = 2 * time.Hour
	TTLDLQResolved      = 7 * 24 * time.Hour // 7 days — auto-evict old DLQ offsets
	TTLRateLimitAuth    = 1 * time.Minute
	TTLRateLimitDefault = 1 * time.Minute

	// ── Profile + settings + analytics cache TTLs ──────────────────────
	TTLProfile           = 5 * time.Minute  // profile reads change infrequently
	TTLCatalogSearch     = 30 * time.Second // search results refresh quickly
	TTLCategorySuppliers = 2 * time.Minute  // category-supplier listings
	TTLSettings          = 2 * time.Minute  // retailer settings hierarchy
	TTLAnalytics         = 60 * time.Second // dashboard metrics
)

// ─── GEO Member Helpers ────────────────────────────────────────────────────────
// These MUST be used everywhere a GEO member name is constructed.
// The canonical prefixes are "d:" for drivers and "r:" for retailers.

// DriverGeoMember returns the Redis GEO member name for a driver.
func DriverGeoMember(driverID string) string { return "d:" + driverID }

// RetailerGeoMember returns the Redis GEO member name for a retailer.
func RetailerGeoMember(retailerID string) string { return "r:" + retailerID }

// WarehouseGeoMember returns the Redis GEO member name for a warehouse.
func WarehouseGeoMember(warehouseID string) string { return "wh:" + warehouseID }

// ─── Profile Cache Key Helpers ─────────────────────────────────────────────────

// SupplierProfile returns the cache key for a supplier's profile.
func SupplierProfile(supplierID string) string { return PrefixSupplierProfile + supplierID }

// RetailerProfile returns the cache key for a retailer's profile.
func RetailerProfile(retailerID string) string { return PrefixRetailerProfile + retailerID }

// DriverProfile returns the cache key for a driver's profile.
func DriverProfile(driverID string) string { return PrefixDriverProfile + driverID }

// FactoryProfile returns the cache key for a factory's profile.
func FactoryProfile(factoryID string) string { return PrefixFactoryProfile + factoryID }

// SettingsKey returns the cache key for a retailer's resolved settings.
func SettingsKey(retailerID, scope string) string { return PrefixSettings + retailerID + ":" + scope }

// CountryConfigCacheKey returns the invalidation key for a country config row.
func CountryConfigCacheKey(countryCode string) string { return PrefixCountryConfigCache + countryCode }

// CountryOverrideCacheKey returns the invalidation key for a supplier country override row.
func CountryOverrideCacheKey(supplierID, countryCode string) string {
	return PrefixCountryOverrideCache + supplierID + ":" + countryCode
}
