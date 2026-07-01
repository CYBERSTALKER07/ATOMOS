package supplier

import (
	"net/http/httptest"
	"testing"
)

func TestParseDemandAnalyticsQueryDefaultsMacro(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/supplier/analytics/demand/today", nil)
	q := parseDemandAnalyticsQuery(req)
	if q.Granularity != "macro" {
		t.Fatalf("granularity=%q want macro", q.Granularity)
	}
}

func TestParseDemandAnalyticsQueryRegionalRequiresWarehouse(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/supplier/analytics/demand/today?granularity=regional", nil)
	q := parseDemandAnalyticsQuery(req)
	if q.Granularity != "macro" {
		t.Fatalf("granularity=%q want macro fallback", q.Granularity)
	}

	req = httptest.NewRequest("GET", "/v1/supplier/analytics/demand/today?granularity=regional&warehouse_id=wh-1", nil)
	q = parseDemandAnalyticsQuery(req)
	if q.Granularity != "regional" || q.WarehouseID != "wh-1" {
		t.Fatalf("got %+v want regional wh-1", q)
	}
}

func TestParseDemandAnalyticsQueryMicroRequiresRetailer(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/supplier/analytics/demand/today?granularity=micro&retailer_id=r-1", nil)
	q := parseDemandAnalyticsQuery(req)
	if q.Granularity != "micro" || q.RetailerID != "r-1" {
		t.Fatalf("got %+v", q)
	}
}

func TestParseDemandAnalyticsQueryRegionAlias(t *testing.T) {
	req := httptest.NewRequest("GET", "/v1/supplier/analytics/demand/today?granularity=regional&region_id=wh-2", nil)
	q := parseDemandAnalyticsQuery(req)
	if q.WarehouseID != "wh-2" || q.Granularity != "regional" {
		t.Fatalf("got %+v", q)
	}
}
