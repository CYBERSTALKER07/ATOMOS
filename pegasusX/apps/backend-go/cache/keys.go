package cache

import "fmt"

// KeyActiveOptimizationJobs returns the Redis key for a supplier's active optimization jobs.
func KeyActiveOptimizationJobs(supplierID string) string {
	return fmt.Sprintf("supplier:%s:jobs:active", supplierID)
}
