package supplier

import (
	"net/http"
	"strings"
)

type demandAnalyticsQuery struct {
	Granularity string
	WarehouseID string
	RetailerID  string
}

func parseDemandAnalyticsQuery(r *http.Request) demandAnalyticsQuery {
	q := demandAnalyticsQuery{
		Granularity: strings.ToLower(strings.TrimSpace(r.URL.Query().Get("granularity"))),
		WarehouseID: strings.TrimSpace(r.URL.Query().Get("warehouse_id")),
		RetailerID:  strings.TrimSpace(r.URL.Query().Get("retailer_id")),
	}
	if q.WarehouseID == "" {
		q.WarehouseID = strings.TrimSpace(r.URL.Query().Get("region_id"))
	}
	switch q.Granularity {
	case "", "macro":
		q.Granularity = "macro"
	case "regional", "micro":
	default:
		q.Granularity = "macro"
	}
	if q.Granularity == "micro" && q.RetailerID == "" {
		q.Granularity = "macro"
	}
	if q.Granularity == "regional" && q.WarehouseID == "" {
		q.Granularity = "macro"
	}
	return q
}
