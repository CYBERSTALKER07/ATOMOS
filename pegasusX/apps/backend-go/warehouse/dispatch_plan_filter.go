package warehouse

import (
	"sort"
	"strings"
)

func filterProposedRoutesByOrderIDs(routes []map[string]any, orderIDs []string) []map[string]any {
	if len(routes) == 0 || len(orderIDs) == 0 {
		return routes
	}
	allow := make(map[string]struct{}, len(orderIDs))
	for _, id := range orderIDs {
		if id = strings.TrimSpace(id); id != "" {
			allow[id] = struct{}{}
		}
	}
	if len(allow) == 0 {
		return routes
	}

	filtered := make([]map[string]any, 0, len(routes))
	for _, route := range routes {
		rawIDs, _ := route["order_ids"].([]any)
		if len(rawIDs) == 0 {
			continue
		}
		keptIDs := make([]string, 0, len(rawIDs))
		for _, raw := range rawIDs {
			id, _ := raw.(string)
			id = strings.TrimSpace(id)
			if id == "" {
				continue
			}
			if _, ok := allow[id]; ok {
				keptIDs = append(keptIDs, id)
			}
		}
		if len(keptIDs) == 0 {
			continue
		}
		sort.Strings(keptIDs)

		stops, _ := route["stops"].([]any)
		keptStops := make([]any, 0, len(keptIDs))
		loadedVU := 0.0
		for _, stopRaw := range stops {
			stop, ok := stopRaw.(map[string]any)
			if !ok {
				continue
			}
			oid, _ := stop["order_id"].(string)
			if _, ok := allow[strings.TrimSpace(oid)]; !ok {
				continue
			}
			keptStops = append(keptStops, stop)
			if vu, ok := stop["volume_vu"].(float64); ok {
				loadedVU += vu
			}
		}

		next := make(map[string]any, len(route)+1)
		for k, v := range route {
			next[k] = v
		}
		orderIDAny := make([]any, len(keptIDs))
		for i, id := range keptIDs {
			orderIDAny[i] = id
		}
		next["order_ids"] = orderIDAny
		next["stops"] = keptStops
		next["stop_count"] = len(keptStops)
		next["loaded_volume"] = loadedVU
		next["volume_vu"] = loadedVU
		if maxVU, ok := route["max_volume_vu"].(float64); ok && maxVU > 0 {
			next["util_pct"] = (loadedVU / maxVU) * 100
		}
		filtered = append(filtered, next)
	}
	return filtered
}
