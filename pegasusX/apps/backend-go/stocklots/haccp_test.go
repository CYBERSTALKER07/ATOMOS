package stocklots

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"google.golang.org/api/iterator"
)

func TestParseHACCPWindow_SafeCSVCell_And_Severity(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)

	from, to, explicitTo, errCode := parseHACCPWindow("2026-11-01", "2026-11-17", now)
	if errCode != "" || !explicitTo {
		t.Fatalf("parseHACCPWindow valid dates explicitTo=%v errCode=%q", explicitTo, errCode)
	}
	if from.Format("2006-01-02") != "2026-11-01" || to.Hour() != 23 || to.Minute() != 59 {
		t.Fatalf("unexpected window from=%s to=%s", from, to)
	}

	if _, _, _, code := parseHACCPWindow("2026-11-18", "2026-11-17", now); code != "invalid_date_range" {
		t.Fatalf("expected invalid_date_range, got %q", code)
	}
	if _, _, _, code := parseHACCPWindow("bad-date", "", now); code != "invalid_from_timestamp" {
		t.Fatalf("expected invalid_from_timestamp, got %q", code)
	}

	if got := safeCSVCell("-18000"); got != "-18000" {
		t.Fatalf("safeCSVCell must preserve signed ints, got %q", got)
	}
	if got := safeCSVCell("=HYPERLINK()"); got != "'=HYPERLINK()" {
		t.Fatalf("safeCSVCell must neutralize formula, got %q", got)
	}
	if got := safeCSVCell("-cmd"); got != "'-cmd" {
		t.Fatalf("safeCSVCell must neutralize non-numeric leading dash, got %q", got)
	}

	if got := classifyHACCPExcursionSeverity(5200); got != "CRITICAL" {
		t.Fatalf("5200 mC severity = %q, want CRITICAL", got)
	}
	if got := classifyHACCPExcursionSeverity(2000); got != "HIGH" {
		t.Fatalf("2000 mC severity = %q, want HIGH", got)
	}
	if got := classifyHACCPExcursionSeverity(500); got != "MEDIUM" {
		t.Fatalf("500 mC severity = %q, want MEDIUM", got)
	}
}

func TestHandleHACCPCompliance_ValidationAndCrossTenant(t *testing.T) {
	t.Parallel()
	hNil := &Handler{}
	reqNil := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp", nil)
	rrNil := httptest.NewRecorder()
	hNil.HandleHACCPCompliance(rrNil, reqNil)
	if rrNil.Code != http.StatusServiceUnavailable {
		t.Fatalf("nil spanner status = %d, want 503", rrNil.Code)
	}

	h := &Handler{Spanner: &spanner.Client{}, SupplierID: "sup-a"}
	ctxA := auth.WithClaims(context.Background(), auth.Claims{
		Subject:      "aud-1",
		Role:         auth.RoleAdmin,
		SupplierID:   "sup-a",
		SupplierRole: auth.SupplierRoleAuditor,
	})

	reqCross := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp?entity=sup-b", nil).WithContext(ctxA)
	rrCross := httptest.NewRecorder()
	h.HandleHACCPCompliance(rrCross, reqCross)
	if rrCross.Code != http.StatusForbidden {
		t.Fatalf("cross-tenant GET = %d, want 403 body=%s", rrCross.Code, rrCross.Body.String())
	}

	reqBadFmt := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp?format=xml", nil).WithContext(ctxA)
	rrBadFmt := httptest.NewRecorder()
	h.HandleHACCPCompliance(rrBadFmt, reqBadFmt)
	if rrBadFmt.Code != http.StatusBadRequest {
		t.Fatalf("bad format GET = %d, want 400 body=%s", rrBadFmt.Code, rrBadFmt.Body.String())
	}

	reqBadRange := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp?from=2026-11-18&to=2026-11-10", nil).WithContext(ctxA)
	rrBadRange := httptest.NewRecorder()
	h.HandleHACCPCompliance(rrBadRange, reqBadRange)
	if rrBadRange.Code != http.StatusBadRequest {
		t.Fatalf("bad range GET = %d, want 400 body=%s", rrBadRange.Code, rrBadRange.Body.String())
	}
}

func TestHACCPCompliance_LiveSpanner_Readings_Excursions_ZoneMismatch_CSV_And_Outbox(t *testing.T) {
	client := openStocklotsSpannerTestClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sid := "sup_haccp_" + uuid.NewString()[:8]
	wid := "wh_haccp_" + uuid.NewString()[:8]
	prodChilled := "prod_ch_" + uuid.NewString()[:8]
	prodFrozen := "prod_fr_" + uuid.NewString()[:8]
	lotReserved := "lot_res_" + uuid.NewString()[:8]
	lotMismatch := "lot_mis_" + uuid.NewString()[:8]
	ordChilled := "ord_ch_" + uuid.NewString()[:8]
	ordFrozen := "ord_fr_" + uuid.NewString()[:8]
	manChilled := "man_ch_" + uuid.NewString()[:8]
	manFrozen := "man_fr_" + uuid.NewString()[:8]
	rd1 := "rd_1_" + uuid.NewString()[:8]
	rd2 := "rd_2_" + uuid.NewString()[:8]
	rd3 := "rd_3_" + uuid.NewString()[:8]
	rd4 := "rd_4_" + uuid.NewString()[:8]
	repID := "rep_tb_" + uuid.NewString()[:8]

	t.Cleanup(func() {
		cleanCtx, cleanCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanCancel()
		muts := []*spanner.Mutation{
			spanner.Delete("TemperatureReadings", spanner.Key{rd1}),
			spanner.Delete("TemperatureReadings", spanner.Key{rd2}),
			spanner.Delete("TemperatureReadings", spanner.Key{rd3}),
			spanner.Delete("TemperatureReadings", spanner.Key{rd4}),
			spanner.Delete("OrderConditionReports", spanner.Key{repID}),
			spanner.Delete("SupplierTruckManifests", spanner.Key{manChilled}),
			spanner.Delete("SupplierTruckManifests", spanner.Key{manFrozen}),
			spanner.Delete("Orders", spanner.Key{ordChilled}),
			spanner.Delete("Orders", spanner.Key{ordFrozen}),
			spanner.Delete("StockLots", spanner.Key{lotReserved}),
			spanner.Delete("StockLots", spanner.Key{lotMismatch}),
			spanner.Delete("WarehouseLocations", spanner.KeyRange{Start: spanner.Key{wid}, End: spanner.Key{wid}, Kind: spanner.ClosedClosed}),
			spanner.Delete("Products", spanner.Key{prodChilled}),
			spanner.Delete("Products", spanner.Key{prodFrozen}),
		}
		iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(cleanCtx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @sid`,
			Params: map[string]any{"sid": sid},
		})
		for {
			row, err := iter.Next()
			if err != nil {
				break
			}
			var eid string
			if row.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		iter.Stop()
		_, _ = client.Apply(cleanCtx, muts)
	})

	lineChilled, _ := json.Marshal([]map[string]any{{"product_id": prodChilled, "quantity": 5}})
	lineFrozen, _ := json.Marshal([]map[string]any{{"product_id": prodFrozen, "quantity": 3}})
	now := time.Now().UTC().Add(-10 * time.Minute)

	_, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("Products", map[string]any{
			"ProductId":         prodChilled,
			"SupplierId":        sid,
			"CategoryId":        "cat-dairy",
			"Name":              "Fresh Milk 1L",
			"PriceMinor":        int64(15000),
			"Currency":          "UZS",
			"HandlingClass":     "CHILLED",
			"RequiresColdChain": true,
			"IsPerishable":      true,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId":         prodFrozen,
			"SupplierId":        sid,
			"CategoryId":        "cat-frozen",
			"Name":              "Frozen Dumplings",
			"PriceMinor":        int64(32000),
			"Currency":          "UZS",
			"HandlingClass":     "FROZEN",
			"RequiresColdChain": true,
			"IsPerishable":      true,
			"CreatedAt":         spanner.CommitTimestamp,
			"UpdatedAt":         spanner.CommitTimestamp,
		}),
		spanner.InsertMap("WarehouseLocations", map[string]any{
			"WarehouseId":  wid,
			"LocationId":   "bin-chilled-1",
			"Zone":         "CHILL",
			"LocationType": "PICK",
			"PickSequence": int64(1),
			"IsActive":     true,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
		spanner.InsertMap("WarehouseLocations", map[string]any{
			"WarehouseId":  wid,
			"LocationId":   "bin-ambient-bad",
			"Zone":         "AMBIENT",
			"LocationType": "PICK",
			"PickSequence": int64(2),
			"IsActive":     true,
			"CreatedAt":    spanner.CommitTimestamp,
			"UpdatedAt":    spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId":            lotReserved,
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        prodChilled,
			"LocationId":       "bin-chilled-1",
			"LotCode":          "LOT-CH-01",
			"QuantityOnHand":   int64(15),
			"QuantityReserved": int64(5),
			"Status":           "AVAILABLE",
			"ReceivedAt":       now,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId":            lotMismatch,
			"SupplierId":       sid,
			"WarehouseId":      wid,
			"ProductId":        prodFrozen,
			"LocationId":       "bin-ambient-bad",
			"LotCode":          "LOT-FR-BAD",
			"QuantityOnHand":   int64(8),
			"QuantityReserved": int64(0),
			"Status":           "AVAILABLE",
			"ReceivedAt":       now,
			"CreatedAt":        spanner.CommitTimestamp,
			"UpdatedAt":        spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierTruckManifests", map[string]any{
			"ManifestId":  manChilled,
			"SupplierId":  sid,
			"WarehouseId": wid,
			"TruckId":     "trk-1",
			"DriverId":    "drv-1",
			"State":       "IN_TRANSIT",
			"CreatedAt":   spanner.CommitTimestamp,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierTruckManifests", map[string]any{
			"ManifestId":  manFrozen,
			"SupplierId":  sid,
			"WarehouseId": wid,
			"TruckId":     "trk-2",
			"DriverId":    "drv-2",
			"State":       "IN_TRANSIT",
			"CreatedAt":   spanner.CommitTimestamp,
			"UpdatedAt":   spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            ordChilled,
			"SupplierId":         sid,
			"RetailerId":         "ret-1",
			"WarehouseId":        wid,
			"ManifestId":         manChilled,
			"Status":             "DISPATCHED",
			"OrderSource":        "RETAILER",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      lineChilled,
			"TotalMinor":         int64(75000),
			"OriginalTotalMinor": int64(75000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId":            ordFrozen,
			"SupplierId":         sid,
			"RetailerId":         "ret-2",
			"WarehouseId":        wid,
			"ManifestId":         manFrozen,
			"Status":             "DISPATCHED",
			"OrderSource":        "RETAILER",
			"ConfirmationStatus": "CONFIRMED",
			"LineItemsJson":      lineFrozen,
			"TotalMinor":         int64(96000),
			"OriginalTotalMinor": int64(96000),
			"Currency":           "UZS",
			"Version":            int64(1),
			"CreatedAt":          spanner.CommitTimestamp,
			"UpdatedAt":          spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderLotReservations", map[string]any{
			"OrderId":   ordChilled,
			"LotId":     lotReserved,
			"Quantity":  int64(5),
			"CreatedAt": spanner.CommitTimestamp,
		}),
		// Chilled compliant (3.5 C) and critical excursion (13.2 C => 13200 mC vs 8000 max)
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId":  rd1,
			"ManifestId": manChilled,
			"SensorId":   "sens-ch-1",
			"RecordedAt": now.Add(time.Minute),
			"TempC":      3.5,
			"CreatedAt":  spanner.CommitTimestamp,
		}),
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId":  rd2,
			"ManifestId": manChilled,
			"SensorId":   "sens-ch-1",
			"RecordedAt": now.Add(2 * time.Minute),
			"TempC":      13.2,
			"CreatedAt":  spanner.CommitTimestamp,
		}),
		// Frozen compliant (-18.0 C) and HIGH warm excursion (-12.5 C => -12500 mC vs -15000 max)
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId":  rd3,
			"ManifestId": manFrozen,
			"SensorId":   "sens-fr-1",
			"RecordedAt": now.Add(time.Minute),
			"TempC":      -18.0,
			"CreatedAt":  spanner.CommitTimestamp,
		}),
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId":  rd4,
			"ManifestId": manFrozen,
			"SensorId":   "sens-fr-1",
			"RecordedAt": now.Add(2 * time.Minute),
			"TempC":      -12.5,
			"CreatedAt":  spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderConditionReports", map[string]any{
			"ReportId":         repID,
			"OrderId":          ordChilled,
			"SupplierId":       sid,
			"RetailerId":       "ret-1",
			"SKU":              prodChilled,
			"ConditionType":    "TEMPERATURE_BREACH",
			"Severity":         "CRITICAL",
			"Description":      "Chilled cabin rose to 13.2C",
			"ReportedBy":       "coldchain-sensor",
			"ReportedByRole":   "SYSTEM",
			"ResolutionStatus": "OPEN",
			"CreatedAt":        spanner.CommitTimestamp,
		}),
	})
	if err != nil {
		t.Fatalf("seed HACCP scenario: %v", err)
	}

	var mu sync.Mutex
	var auditActions []string
	h := &Handler{
		Spanner:    client,
		SupplierID: sid,
		AuditRecord: func(_ context.Context, in HACCPAuditInput) error {
			mu.Lock()
			auditActions = append(auditActions, in.Action)
			mu.Unlock()
			return nil
		},
	}

	auditorCtx := auth.WithClaims(context.Background(), auth.Claims{
		Subject:      "aud-haccp-1",
		Role:         auth.RoleAdmin,
		SupplierID:   sid,
		SupplierRole: auth.SupplierRoleAuditor,
		ActorType:    auth.ActorTypeHuman,
	})

	// 1. JSON report evaluation (AUDITOR read access permitted).
	reqJSON := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp", nil).WithContext(auditorCtx)
	rrJSON := httptest.NewRecorder()
	h.HandleHACCPCompliance(rrJSON, reqJSON)
	if rrJSON.Code != http.StatusOK {
		t.Fatalf("GET /v1/compliance/haccp json = %d body=%s", rrJSON.Code, rrJSON.Body.String())
	}

	var rep HACCPReport
	if err := json.Unmarshal(rrJSON.Body.Bytes(), &rep); err != nil {
		t.Fatalf("decode HACCPReport: %v", err)
	}
	if rep.TotalReadings != 4 || rep.ExcursionCount != 2 || rep.ComplianceRateBps != 5000 {
		t.Fatalf("unexpected telemetry metrics: total=%d exc=%d bps=%d", rep.TotalReadings, rep.ExcursionCount, rep.ComplianceRateBps)
	}
	if rep.OpenBreaches != 1 || len(rep.ConditionReports) != 1 {
		t.Fatalf("unexpected condition breaches: open=%d reports=%d", rep.OpenBreaches, len(rep.ConditionReports))
	}
	if len(rep.ColdZoneMismatches) != 1 || rep.ColdZoneMismatches[0].LotID != lotMismatch {
		t.Fatalf("expected 1 cold zone mismatch on %s, got %+v", lotMismatch, rep.ColdZoneMismatches)
	}
	if rep.AffectedLotsCount != 2 || len(rep.AffectedLots) != 2 {
		t.Fatalf("expected 2 affected lots, got %+v", rep.AffectedLots)
	}

	// 2. CSV stream export (?format=csv).
	reqCSV := httptest.NewRequest(http.MethodGet, "/v1/compliance/haccp?format=csv", nil).WithContext(auditorCtx)
	rrCSV := httptest.NewRecorder()
	h.HandleHACCPCompliance(rrCSV, reqCSV)
	if rrCSV.Code != http.StatusOK {
		t.Fatalf("GET /v1/compliance/haccp csv = %d body=%s", rrCSV.Code, rrCSV.Body.String())
	}
	if ct := rrCSV.Header().Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("expected text/csv, got %q", ct)
	}
	rows, err := csv.NewReader(strings.NewReader(rrCSV.Body.String())).ReadAll()
	if err != nil || len(rows) < 7 {
		t.Fatalf("expected >= 7 CSV rows (header + summary + 2 exc + 2 lots + mismatch + cond), got %d err=%v", len(rows), err)
	}

	mu.Lock()
	actions := append([]string(nil), auditActions...)
	mu.Unlock()
	if len(actions) != 2 || actions[0] != "compliance.haccp.evaluated" || actions[1] != "compliance.haccp.exported" {
		t.Fatalf("expected evaluated+exported audit actions, got %+v", actions)
	}

	// 3. Verify OutboxEvents captured coldchain.haccp.excursion on telemetry.logistics.
	iter := client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @sid`,
		Params: map[string]any{"sid": sid},
	})
	defer iter.Stop()
	var foundOutbox bool
	for {
		row, rerr := iter.Next()
		if rerr == iterator.Done {
			break
		}
		if rerr != nil {
			t.Fatalf("query OutboxEvents: %v", rerr)
		}
		var topic string
		var raw []byte
		if serr := row.Columns(&topic, &raw); serr != nil {
			t.Fatalf("scan OutboxEvents: %v", serr)
		}
		if topic == events.TopicTelemetryLogistics && strings.Contains(string(raw), events.EventHaccpExcursionReported) {
			foundOutbox = true
		}
	}
	if !foundOutbox {
		t.Fatalf("expected coldchain.haccp.excursion event in OutboxEvents on %s", events.TopicTelemetryLogistics)
	}
}
