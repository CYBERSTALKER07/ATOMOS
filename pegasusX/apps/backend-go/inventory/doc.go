// Package inventory owns per-warehouse stock levels backed by the
// InventoryLevels Spanner table. It provides CRUD, reservation, and
// threshold-breach detection. It does NOT own the product catalog (see catalog
// package) or the replenishment protocol (see replenishment package).
package inventory
