package cache

import (
	"fmt"
	"strings"
)

// KeyActiveOptimizationJobs returns the Redis key for a supplier's active optimization jobs.
// Uses a hash tag for Redis Cluster slot safety on supplier-scoped keys.
func KeyActiveOptimizationJobs(supplierID string) string {
	return fmt.Sprintf("{sup:%s}:jobs:active", supplierID)
}

// SupplierScopedKey prefixes a supplier id with the cluster hash-tag convention.
func SupplierScopedKey(supplierID, suffix string) string {
	return fmt.Sprintf("{sup:%s}:%s", supplierID, strings.TrimPrefix(suffix, ":"))
}
