package foundryroutes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
	"github.com/pegasusx/pegasusx/apps/backend-go/foundryroutes"
)

func newTestServer() *httptest.Server {
	r := chi.NewRouter()
	foundryroutes.RegisterRoutes(r, foundryroutes.Deps{
		Engine: foundry.NewEngine(),
	})
	return httptest.NewServer(r)
}

func postJSON(t *testing.T, url string, body any) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	resp, err := http.Post(url, "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	return resp
}

func TestFoundryRoutesEndToEnd(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	// 1. Create Ontology Objects (201 Created) & validation failure (422 Unprocessable Entity).
	badResp := postJSON(t, srv.URL+"/v1/foundry/ontology/objects", map[string]any{
		"cell_id": "",
	})
	defer badResp.Body.Close()
	if badResp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("expected 422 on missing object identity, got %d", badResp.StatusCode)
	}

	lotResp := postJSON(t, srv.URL+"/v1/foundry/ontology/objects", map[string]any{
		"cell_id":     "cell-uz",
		"object_type": "RawMaterialLot",
		"object_id":   "rm_10",
		"supplier_id": "sup_1",
		"properties":  map[string]any{"origin": "Samarkand"},
	})
	defer lotResp.Body.Close()
	if lotResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for RawMaterialLot, got %d", lotResp.StatusCode)
	}

	// 2. Cashier Blind Count Session (201) -> Short Count Attempt #1 triggers 409 Conflict (BLIND_RECOUNT_REQUIRED).
	sessResp := postJSON(t, srv.URL+"/v1/foundry/cashier/sessions", map[string]any{
		"cell_id":             "cell-uz",
		"supplier_id":         "sup_1",
		"depot_id":            "dep_1",
		"driver_id":           "drv_9",
		"route_id":            "rt_9",
		"bag_seal_barcode":    "BAG-771",
		"expected_cash_minor": int64(20_000_000), // 200,000 UZS
	})
	defer sessResp.Body.Close()
	if sessResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for cashier session, got %d", sessResp.StatusCode)
	}
	var sessView foundry.CashierSessionView
	if err := json.NewDecoder(sessResp.Body).Decode(&sessView); err != nil {
		t.Fatalf("decode cashier session: %v", err)
	}

	count1Resp := postJSON(t, srv.URL+"/v1/foundry/cashier/sessions/"+sessView.SessionID+"/blind-count", map[string]any{
		"denominations_uzs":         map[string]int64{"100000": 1}, // Only 100,000 UZS
		"tolerance_threshold_minor": int64(100_000),
	})
	defer count1Resp.Body.Close()
	if count1Resp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for blind recount ratchet on Attempt #1, got %d", count1Resp.StatusCode)
	}

	// 3. Fleet DVIR Red-Tag Grounding Gate (409 Conflict when critical defect found).
	dvirResp := postJSON(t, srv.URL+"/v1/foundry/fleet/dvir", map[string]any{
		"cell_id":             "cell-uz",
		"supplier_id":         "sup_1",
		"vehicle_id":          "veh_red_1",
		"driver_id":           "drv_9",
		"requires_cold_chain": false,
		"odometer_km":         40500,
		"checklist": map[string]bool{
			"steering_and_horn_ok":              true,
			"service_and_parking_brakes_ok":     true,
			"tires_tread_and_pressure_ok":       false, // Bald tire defect!
			"lights_and_hazards_ok":             true,
			"wipers_and_fluids_ok":              true,
			"emergency_kit_and_extinguisher_ok": true,
			"cargo_box_sanitation_ok":           true,
			"reefer_temp_calibrated_ok":         true,
		},
	})
	defer dvirResp.Body.Close()
	if dvirResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when vehicle is Red-Tagged, got %d", dvirResp.StatusCode)
	}

	// 4. AML Check endpoint (200 OK under 25M UZS, 409 Conflict above 25M UZS).
	amlOK := postJSON(t, srv.URL+"/v1/foundry/guards/aml-check", map[string]any{
		"legal_entity_tin": "TIN-555111",
		"supplier_id":      "sup_1",
		"calendar_date":    "2026-10-29",
		"cash_minor":       int64(2_000_000_000), // 20M UZS
	})
	defer amlOK.Body.Close()
	if amlOK.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for 20M UZS AML check, got %d", amlOK.StatusCode)
	}

	amlExceed := postJSON(t, srv.URL+"/v1/foundry/guards/aml-check", map[string]any{
		"legal_entity_tin": "TIN-555111",
		"supplier_id":      "sup_1",
		"calendar_date":    "2026-10-29",
		"cash_minor":       int64(600_000_000), // +6M UZS -> 26M UZS > 25M UZS ceiling
	})
	defer amlExceed.Body.Close()
	if amlExceed.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on AML ceiling breach, got %d", amlExceed.StatusCode)
	}

	// 5. Offline Quota Lease issue (201) & validation (200 OK within cap, 409 Conflict over cap).
	leaseResp := postJSON(t, srv.URL+"/v1/foundry/guards/offline-lease/issue", map[string]any{
		"order_id":               "ord_lease_1",
		"driver_id":              "drv_9",
		"supplier_id":            "sup_1",
		"legal_entity_tin":       "TIN-777888",
		"calendar_date":          "2026-10-29",
		"requested_cash_minor":   int64(500_000_000),
		"requested_credit_minor": int64(300_000_000),
		"ttl_minutes":            120,
	})
	defer leaseResp.Body.Close()
	if leaseResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for offline lease, got %d", leaseResp.StatusCode)
	}
	var lease foundry.OfflineQuotaLease
	if err := json.NewDecoder(leaseResp.Body).Decode(&lease); err != nil {
		t.Fatalf("decode lease: %v", err)
	}

	leaseExceed := postJSON(t, srv.URL+"/v1/foundry/guards/offline-lease/validate", map[string]any{
		"lease_id":               lease.LeaseID,
		"collected_cash_minor":   int64(600_000_000), // exceeds 500M cap!
		"collected_credit_minor": int64(100_000_000),
	})
	defer leaseExceed.Body.Close()
	if leaseExceed.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when offline collection exceeds lease cap, got %d", leaseExceed.StatusCode)
	}

	// 6. Predictive Reefer Health & Vehicle Dispatch Check.
	reeferHold := postJSON(t, srv.URL+"/v1/foundry/fleet/predictive-reefer", map[string]any{
		"vehicle_id":                    "veh_reefer_slow",
		"cooling_rate_deg_per_min":      0.15,
		"min_required_rate_deg_per_min": 0.45,
	})
	defer reeferHold.Body.Close()
	if reeferHold.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict when reefer cooling rate degrades, got %d", reeferHold.StatusCode)
	}

	dispCheck, err := http.Get(srv.URL + "/v1/foundry/fleet/vehicles/veh_reefer_slow/dispatch-check")
	if err != nil {
		t.Fatalf("GET dispatch-check: %v", err)
	}
	defer dispCheck.Body.Close()
	if dispCheck.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on dispatch-check for held reefer van, got %d", dispCheck.StatusCode)
	}

	// 7. Tote Heartbeat & Takeover (409 when active, 200 OK when stalled >= 20m).
	toteStalled := postJSON(t, srv.URL+"/v1/foundry/guards/tote-state", foundry.PickToteState{
		ToteID:              "tote_http_1",
		WaveID:              "wave_1",
		ActivePickerID:      "p_1",
		Status:              "PICKER_IN_HAND",
		CompletedLineCount:  3,
		TotalLineCount:      8,
		PickedLotIDs:        []string{"lot_1"},
		LastScanHeartbeatAt: time.Now().UTC().Add(-25 * time.Minute),
	})
	defer toteStalled.Body.Close()
	if toteStalled.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201 Created for tote-state, got %d", toteStalled.StatusCode)
	}

	takeoverOK := postJSON(t, srv.URL+"/v1/foundry/guards/tote-takeover", map[string]any{
		"tote_id":       "tote_http_1",
		"new_picker_id": "p_2",
	})
	defer takeoverOK.Body.Close()
	if takeoverOK.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for stalled tote takeover, got %d", takeoverOK.StatusCode)
	}
}

func TestFoundryRoutesStructuredErrorsAndMutationCeiling(t *testing.T) {
	srv := newTestServer()
	defer srv.Close()

	// 1. UA-1: Cashier Blind Recount Ratchet -> 409 Conflict with code RECOUNT_REQUIRED.
	sessResp := postJSON(t, srv.URL+"/v1/foundry/cashier/sessions", map[string]any{
		"cell_id":             "cell-uz",
		"supplier_id":         "sup_1",
		"depot_id":            "dep_1",
		"driver_id":           "drv_recount",
		"route_id":            "rt_recount",
		"bag_seal_barcode":    "BAG-RECOUNT-99",
		"expected_cash_minor": int64(20_000_000), // 200,000 UZS
	})
	defer sessResp.Body.Close()
	var sessView foundry.CashierSessionView
	if err := json.NewDecoder(sessResp.Body).Decode(&sessView); err != nil {
		t.Fatalf("decode session: %v", err)
	}

	countFail := postJSON(t, srv.URL+"/v1/foundry/cashier/sessions/"+sessView.SessionID+"/blind-count", map[string]any{
		"denominations_uzs":         map[string]int64{"100000": 1}, // Only 100k UZS
		"tolerance_threshold_minor": int64(0),
	})
	defer countFail.Body.Close()
	if countFail.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict, got %d", countFail.StatusCode)
	}
	var countFailBody map[string]any
	if err := json.NewDecoder(countFail.Body).Decode(&countFailBody); err != nil {
		t.Fatalf("decode countFailBody: %v", err)
	}
	if countFailBody["code"] != "RECOUNT_REQUIRED" {
		t.Fatalf("expected code RECOUNT_REQUIRED, got %v", countFailBody["code"])
	}

	// 2. UA-2: AML Ceiling Breach -> 409 Conflict with code AML_DAILY_CASH_LIMIT_EXCEEDED and advice.
	amlExceed := postJSON(t, srv.URL+"/v1/foundry/guards/aml-check", map[string]any{
		"legal_entity_tin": "TIN-AML-999",
		"supplier_id":      "sup_1",
		"calendar_date":    "2026-10-31",
		"cash_minor":       int64(2_600_000_000), // > 25M UZS
	})
	defer amlExceed.Body.Close()
	if amlExceed.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for AML breach, got %d", amlExceed.StatusCode)
	}
	var amlBody map[string]any
	if err := json.NewDecoder(amlExceed.Body).Decode(&amlBody); err != nil {
		t.Fatalf("decode amlBody: %v", err)
	}
	if amlBody["code"] != "AML_DAILY_CASH_LIMIT_EXCEEDED" {
		t.Fatalf("expected code AML_DAILY_CASH_LIMIT_EXCEEDED, got %v", amlBody["code"])
	}
	if amlBody["advice"] == nil || amlBody["advice"] == "" {
		t.Fatalf("expected advice in AML error response, got %+v", amlBody)
	}

	// 3. UA-8: DVIR Red-Tag -> 409 Conflict with code VEHICLE_GROUNDED_RED_TAG.
	dvirResp := postJSON(t, srv.URL+"/v1/foundry/fleet/dvir", map[string]any{
		"cell_id":             "cell-uz",
		"supplier_id":         "sup_1",
		"vehicle_id":          "veh_dvir_red",
		"driver_id":           "drv_1",
		"requires_cold_chain": false,
		"odometer_km":         12000,
		"checklist": map[string]bool{
			"steering_and_horn_ok":              false, // Defect!
			"service_and_parking_brakes_ok":     true,
			"tires_tread_and_pressure_ok":       true,
			"lights_and_hazards_ok":             true,
			"wipers_and_fluids_ok":              true,
			"emergency_kit_and_extinguisher_ok": true,
			"cargo_box_sanitation_ok":           true,
			"reefer_temp_calibrated_ok":         true,
		},
	})
	defer dvirResp.Body.Close()
	if dvirResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict on Red-Tag, got %d", dvirResp.StatusCode)
	}
	var dvirBody map[string]any
	if err := json.NewDecoder(dvirResp.Body).Decode(&dvirBody); err != nil {
		t.Fatalf("decode dvirBody: %v", err)
	}
	if dvirBody["code"] != "VEHICLE_GROUNDED_RED_TAG" {
		t.Fatalf("expected code VEHICLE_GROUNDED_RED_TAG, got %v", dvirBody["code"])
	}

	// 4. UA-11: Active Tote Takeover Rejection -> 409 Conflict with code TOTE_TAKEOVER_NOT_PERMITTED.
	_ = postJSON(t, srv.URL+"/v1/foundry/guards/tote-state", foundry.PickToteState{
		ToteID:              "tote_active_now",
		WaveID:              "wave_1",
		ActivePickerID:      "picker_active",
		Status:              "PICKER_IN_HAND",
		CompletedLineCount:  1,
		TotalLineCount:      10,
		LastScanHeartbeatAt: time.Now().UTC().Add(-2 * time.Minute), // Only 2 mins idle
	})
	takeoverFail := postJSON(t, srv.URL+"/v1/foundry/guards/tote-takeover", map[string]any{
		"tote_id":       "tote_active_now",
		"new_picker_id": "picker_intruder",
	})
	defer takeoverFail.Body.Close()
	if takeoverFail.StatusCode != http.StatusConflict {
		t.Fatalf("expected 409 Conflict for active tote takeover, got %d", takeoverFail.StatusCode)
	}
	var toteBody map[string]any
	if err := json.NewDecoder(takeoverFail.Body).Decode(&toteBody); err != nil {
		t.Fatalf("decode toteBody: %v", err)
	}
	if toteBody["code"] != "TOTE_TAKEOVER_NOT_PERMITTED" {
		t.Fatalf("expected code TOTE_TAKEOVER_NOT_PERMITTED, got %v", toteBody["code"])
	}
}
