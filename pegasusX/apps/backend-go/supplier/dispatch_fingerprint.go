package supplier

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
)

const warehouseDispatchPlanCachePrefix = "warehouse:dispatch_plan:"

func computeSupplierPlanFingerprint(undispatched []map[string]any, proposedRoutes []map[string]any) string {
	parts := make([]string, 0, 2+len(proposedRoutes))
	orderIDs := make([]string, 0, len(undispatched))
	for _, row := range undispatched {
		if id, ok := row["order_id"].(string); ok && strings.TrimSpace(id) != "" {
			orderIDs = append(orderIDs, strings.TrimSpace(id))
		}
	}
	sort.Strings(orderIDs)
	parts = append(parts, "orders:"+strings.Join(orderIDs, ","))
	for i, route := range proposedRoutes {
		seq, _ := route["order_ids"].([]any)
		ids := make([]string, 0, len(seq))
		for _, item := range seq {
			if id, ok := item.(string); ok && strings.TrimSpace(id) != "" {
				ids = append(ids, strings.TrimSpace(id))
			}
		}
		sort.Strings(ids)
		driverID, _ := route["driver_id"].(string)
		parts = append(parts, "route:"+strconv.Itoa(i)+":"+strings.TrimSpace(driverID)+":"+strings.Join(ids, ","))
	}
	sum := sha256.Sum256([]byte(strings.Join(parts, "|")))
	return hex.EncodeToString(sum[:16])
}

func (s *Service) readWarehouseDispatchPlanFingerprint(ctx context.Context, warehouseID string) string {
	if s == nil || s.cache == nil {
		return ""
	}
	warehouseID = strings.TrimSpace(warehouseID)
	if warehouseID == "" {
		return ""
	}
	raw, found, err := s.cache.Get(ctx, warehouseDispatchPlanCachePrefix+warehouseID)
	if err != nil || !found || len(raw) == 0 {
		return ""
	}
	var entry struct {
		Fingerprint string `json:"fingerprint"`
	}
	if err := json.Unmarshal(raw, &entry); err != nil {
		return ""
	}
	return strings.TrimSpace(entry.Fingerprint)
}
