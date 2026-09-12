package outbox

import "strings"

// FairInterleave returns up to limit events by round-robin across SupplierID
// buckets (empty SupplierID is treated as one shared bucket). Preserves relative
// order within each tenant.
func FairInterleave(events []Event, limit int) []Event {
	if limit <= 0 || len(events) == 0 {
		return nil
	}
	if len(events) <= limit {
		// Still interleave when oversubscribed later; early exit when already small.
		buckets := bucketBySupplier(events)
		if len(buckets) <= 1 {
			if len(events) > limit {
				return events[:limit]
			}
			return events
		}
	}

	buckets := bucketBySupplier(events)
	keys := make([]string, 0, len(buckets))
	for k := range buckets {
		keys = append(keys, k)
	}
	// Stable key order: empty last, then lexicographic for determinism in tests.
	sortTenantKeys(keys)

	out := make([]Event, 0, limit)
	indexes := make(map[string]int, len(keys))
	for len(out) < limit {
		progress := false
		for _, k := range keys {
			i := indexes[k]
			if i >= len(buckets[k]) {
				continue
			}
			out = append(out, buckets[k][i])
			indexes[k] = i + 1
			progress = true
			if len(out) >= limit {
				break
			}
		}
		if !progress {
			break
		}
	}
	return out
}

func bucketBySupplier(events []Event) map[string][]Event {
	out := make(map[string][]Event)
	for _, e := range events {
		k := strings.TrimSpace(e.SupplierID)
		out[k] = append(out[k], e)
	}
	return out
}

func sortTenantKeys(keys []string) {
	// insertion sort — small N (distinct tenants in a fetch window)
	for i := 1; i < len(keys); i++ {
		j := i
		for j > 0 {
			a, b := keys[j-1], keys[j]
			if tenantKeyLess(b, a) {
				keys[j-1], keys[j] = b, a
				j--
				continue
			}
			break
		}
	}
}

func tenantKeyLess(a, b string) bool {
	if a == "" {
		return false
	}
	if b == "" {
		return true
	}
	return a < b
}
