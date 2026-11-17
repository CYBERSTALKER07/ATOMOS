package e2ep0

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/anomalies"
	"github.com/pegasusx/pegasusx/apps/backend-go/audit"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/gl"
	"github.com/pegasusx/pegasusx/apps/backend-go/iam"
	"github.com/pegasusx/pegasusx/apps/backend-go/proposals"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
	"github.com/pegasusx/pegasusx/apps/backend-go/whatif"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
)

func openWave4BSpannerClient(t *testing.T) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	requireSpanner := strings.TrimSpace(os.Getenv("PARITY_REQUIRE_SPANNER")) == "1"
	if host == "" {
		if requireSpanner {
			t.Fatal("PARITY_REQUIRE_SPANNER=1 but SPANNER_EMULATOR_HOST is unset")
		}
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping Wave 4B live Spanner E2E test")
	}
	envOr := func(k, def string) string {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
		return def
	}
	dsn := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	client, err := spanner.NewClient(ctx, dsn,
		option.WithEndpoint(host),
		option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithTransportCredentials(insecure.NewCredentials())),
	)
	if err != nil {
		t.Fatalf("spanner.NewClient: %v", err)
	}
	for _, spec := range []struct {
		tbl string
		col string
		key spanner.Key
	}{
		{"AnomalyCases", "Status", spanner.Key{"__w4b_e2e__", "__probe__"}},
		{"WhatIfSimulations", "Status", spanner.Key{"__w4b_e2e__", "__probe__"}},
		{"CustomRoles", "Status", spanner.Key{"__w4b_e2e__", "__probe__"}},
		{"ScimIdentities", "ResourceType", spanner.Key{"__w4b_e2e__", "User", "__probe__"}},
		{"Proposals", "AutonomyLevel", spanner.Key{"__w4b_e2e__", "__probe__"}},
	} {
		ro := client.ReadOnlyTransaction()
		_, pErr := ro.ReadRow(ctx, spec.tbl, spec.key, []string{spec.col})
		ro.Close()
		if pErr != nil && spanner.ErrCode(pErr) != codes.NotFound {
			client.Close()
			t.Fatalf("Wave 4B table %s.%s unavailable: %v", spec.tbl, spec.col, pErr)
		}
	}
	t.Cleanup(client.Close)
	return client
}

func cleanupWave4BTenant(t *testing.T, client *spanner.Client, tid string, extraDeletes ...*spanner.Mutation) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		var muts []*spanner.Mutation
		ro := client.ReadOnlyTransaction()
		it := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT EventId FROM OutboxEvents WHERE SupplierId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			r, err := it.Next()
			if err != nil {
				break
			}
			var eid string
			if r.Column(0, &eid) == nil {
				muts = append(muts, spanner.Delete("OutboxEvents", spanner.Key{eid}))
			}
		}
		it.Stop()
		nIt := ro.Query(ctx, spanner.Statement{
			SQL:    `SELECT NotificationId FROM Notifications WHERE RecipientId = @tid`,
			Params: map[string]any{"tid": tid},
		})
		for {
			r, err := nIt.Next()
			if err != nil {
				break
			}
			var nid string
			if r.Column(0, &nid) == nil {
				muts = append(muts, spanner.Delete("Notifications", spanner.Key{nid}))
			}
		}
		nIt.Stop()
		ro.Close()

		kr := spanner.KeyRange{Start: spanner.Key{tid}, End: spanner.Key{tid}, Kind: spanner.ClosedClosed}
		muts = append(muts,
			spanner.Delete("AnomalyCases", kr),
			spanner.Delete("WhatIfSimulations", kr),
			spanner.Delete("CustomRoles", kr),
			spanner.Delete("ScimIdentities", kr),
			spanner.Delete("RolePermissions", kr),
			spanner.Delete("SupplierUserRoles", kr),
			spanner.Delete("SodPolicies", kr),
			spanner.Delete("SodViolations", kr),
			spanner.Delete("Proposals", kr),
			spanner.Delete("ApprovalDecisions", kr),
			spanner.Delete("ApprovalRequests", kr),
			spanner.Delete("RoiEvents", kr),
			spanner.Delete("AuditEvents", kr),
			spanner.Delete("JournalPostings", kr),
			spanner.Delete("JournalEntries", kr),
			spanner.Delete("GlBalances", kr),
			spanner.Delete("GlAccounts", kr),
			spanner.Delete("AccountingPeriods", kr),
			spanner.Delete("ForecastValues", kr),
			spanner.Delete("InventoryCostLayers", kr),
			spanner.Delete("SupplierInventoryV2", kr),
		)
		muts = append(muts, extraDeletes...)
		_, _ = client.Apply(ctx, muts)
	})
}

func collectWave4BOutboxPayloads(t *testing.T, client *spanner.Client, tid string) string {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	ro := client.ReadOnlyTransaction().WithTimestampBound(spanner.StrongRead())
	defer ro.Close()
	iter := ro.Query(ctx, spanner.Statement{
		SQL:    `SELECT TopicName, Payload FROM OutboxEvents WHERE SupplierId = @tid`,
		Params: map[string]any{"tid": tid},
	})
	defer iter.Stop()
	var sb strings.Builder
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			t.Fatalf("query OutboxEvents: %v", err)
		}
		var topic string
		var payload []byte
		if serr := row.Columns(&topic, &payload); serr == nil {
			sb.WriteString(topic)
			sb.WriteString("::")
			sb.Write(payload)
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

func TestWave4B_AnomalyCatalogue_WhatIf_To_L1Proposals_And_RBAC(t *testing.T) {
	client := openWave4BSpannerClient(t)
	h := NewTestHarnessWithOptions(t, HarnessOptions{SpannerClient: client})
	ctx := context.Background()

	suffix := uuid.NewString()[:8]
	tid := "sup_w4b_intel_" + suffix
	whID := "wh_w4b_" + suffix
	catID := "cat_w4b_" + suffix
	skuID := "sku_w4b_" + suffix
	ordID := "ord_w4b_" + suffix
	retID := "ret_w4b_" + suffix

	cleanupWave4BTenant(t, client, tid,
		spanner.Delete("Orders", spanner.Key{ordID}),
		spanner.Delete("Products", spanner.Key{skuID}),
		spanner.Delete("ProductCategories", spanner.Key{catID}),
		spanner.Delete("RetailerCreditProfiles", spanner.Key{retID, tid}),
	)

	// Seed after-hours manual GL entry + Inventory + Forecast + Orders for both Anomaly detection & What-If simulation.
	uzt := time.FixedZone("UZT", 5*3600)
	now := time.Now().UTC().Truncate(time.Microsecond)
	yestUZT := now.In(uzt).AddDate(0, 0, -1)
	afterHoursUZT := time.Date(yestUZT.Year(), yestUZT.Month(), yestUZT.Day(), 23, 25, 0, 0, uzt).UTC()
	periodID := time.Now().UTC().Format("2006-01")

	glSvc := gl.NewService(client)
	if err := glSvc.SeedDefaults(ctx, tid, "UZ", "UZS"); err != nil {
		t.Fatalf("gl SeedDefaults: %v", err)
	}
	if _, err := glSvc.PostEntry(ctx, gl.JournalEntryInput{
		EntityID:       tid,
		PeriodID:       periodID,
		SourceType:     "MANUAL",
		SourceID:       "man-night-w4b-" + suffix,
		Memo:           "Wave 4B E2E after hours manual journal",
		Currency:       "UZS",
		IdempotencyKey: "w4b-man-" + suffix,
		PostedBy:       "acct-night-" + suffix,
		OccurredAt:     afterHoursUZT,
		Postings: []gl.PostingInput{
			{AccountCode: gl.AccountCashOffice, Direction: gl.DirectionDebit, AmountMinor: 1_250_000, Currency: "UZS"},
			{AccountCode: gl.AccountSalesRevenue, Direction: gl.DirectionCredit, AmountMinor: 1_250_000, Currency: "UZS"},
		},
	}); err != nil {
		t.Fatalf("gl PostEntry after hours: %v", err)
	}

	lineJSON := fmt.Sprintf(`[{"sku":%q,"name":"Kefir 1L","quantity":12,"unit_price_minor":50000}]`, skuID)
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ProductCategories", map[string]any{
			"CategoryId": catID, "SupplierId": tid, "Name": "Dairy", "SortOrder": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": skuID, "SupplierId": tid, "CategoryId": catID, "Name": "Kefir 1L",
			"PriceMinor": int64(50_000), "Currency": "UZS", "StockQuantity": int64(250),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("InventoryCostLayers", map[string]any{
			"EntityId": tid, "WarehouseId": whID, "ProductId": skuID, "LayerId": "lay-w4b-" + suffix,
			"QtyRemaining": int64(250), "OriginalQty": int64(250), "UnitCostMinor": int64(32_000),
			"Currency": "UZS", "ReceivedAt": spanner.CommitTimestamp, "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierInventoryV2", map[string]any{
			"SupplierId": tid, "WarehouseId": whID, "ProductId": skuID,
			"QuantityOnHand": int64(250), "QuantityReserved": int64(20), "ReorderThreshold": int64(60),
			"UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("ForecastValues", map[string]any{
			"EntityId": tid, "RunId": "run-w4b-" + suffix, "ForecastDate": "2026-11-22",
			"LocationId": whID, "Sku": skuID, "ProductId": skuID, "Model": "XGB",
			"IsWinner": true, "P50Milli": int64(400_000), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": tid, "RetailerId": retID, "WarehouseId": whID,
			"Status": "CONFIRMED", "OrderSource": "RETAILER", "ConfirmationStatus": "CONFIRMED",
			"LineItemsJson": []byte(lineJSON), "TotalMinor": int64(600_000),
			"OriginalTotalMinor": int64(600_000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("RetailerCreditProfiles", map[string]any{
			"RetailerId": retID, "SupplierId": tid, "CreditLimitMinor": int64(2_000_000),
			"CurrentBalanceMinor": int64(250_000), "ReservedMinor": int64(0),
			"AvailableCreditMinor": int64(1_750_000), "Status": "ACTIVE",
			"Version": int64(1), "CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed Wave 4B intelligence data: %v", err)
	}

	ownerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tid, "owner-w4b-"+suffix, nil)
	auditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tid, "aud-w4b-"+suffix, nil)
	expAt := time.Now().UTC().Add(-time.Minute)
	expiredAuditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tid, "aud-exp-"+suffix, &expAt)
	agentTok, err := auth.Issue(auth.Claims{
		Subject:      "agent-w4b-" + suffix,
		Role:         auth.RoleAdmin,
		SupplierID:   tid,
		SupplierRole: auth.SupplierRoleAgent,
		ActorType:    auth.ActorTypeAgent,
	}, auth.IssueOptions{Secret: h.JWTSecret, TTL: time.Hour, Issuer: h.JWTIssuer})
	if err != nil {
		t.Fatalf("issue agent token: %v", err)
	}

	// 1. Catalogue GET & RBAC enforcement.
	code, _, catRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/anomalies/catalogue", auditorTok, nil, "")
	if code != http.StatusOK {
		t.Fatalf("auditor GET /v1/anomalies/catalogue = %d body=%s", code, string(catRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, catRaw, "anomalies.catalogue")

	if codeExp, _, bodyExp := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/anomalies/catalogue", expiredAuditorTok, nil, ""); codeExp != http.StatusUnauthorized || !strings.Contains(string(bodyExp), "auditor_access_expired") {
		t.Fatalf("expired auditor GET /v1/anomalies/catalogue = %d body=%s", codeExp, string(bodyExp))
	}

	for _, tok := range []string{auditorTok, agentTok} {
		cScan, _, _ := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/anomalies/scan", tok, map[string]any{}, uuid.NewString())
		if cScan != http.StatusForbidden {
			t.Fatalf("forbidden POST /v1/anomalies/scan = %d, want 403", cScan)
		}
		cSim, _, _ := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/whatif/simulate", tok, map[string]any{"name": "Blocked", "demand_delta_bps": 500}, uuid.NewString())
		if cSim != http.StatusForbidden {
			t.Fatalf("forbidden POST /v1/whatif/simulate = %d, want 403", cSim)
		}
	}

	// 2. Owner runs anomaly scan -> persists AnomalyCase -> escalates into L1 PENDING Proposal.
	code, _, scanRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/anomalies/scan", ownerTok, map[string]any{
		"rule_codes": []string{anomalies.RuleManualJournalAfterHours},
	}, uuid.NewString())
	if code != http.StatusOK {
		t.Fatalf("POST /v1/anomalies/scan = %d body=%s", code, string(scanRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, scanRaw, "anomalies.scan")
	var scanRes anomalies.ScanResult
	if err := json.Unmarshal(scanRaw, &scanRes); err != nil || len(scanRes.Cases) == 0 {
		t.Fatalf("expected detected manual journal anomaly case, got (%s, %v)", string(scanRaw), err)
	}

	caseID := scanRes.Cases[0].CaseID
	code, _, resRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/anomalies/cases/"+caseID+"/resolve", ownerTok, map[string]any{
		"status":          "ESCALATED",
		"resolution_note": "Escalate after-hours journal for L1 governance review",
		"create_proposal": true,
	}, uuid.NewString())
	if code != http.StatusOK {
		t.Fatalf("POST /v1/anomalies/cases/%s/resolve = %d body=%s", caseID, code, string(resRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, resRaw, "anomalies.resolve")
	var resolveRes anomalies.ResolveResult
	if err := json.Unmarshal(resRaw, &resolveRes); err != nil || resolveRes.Proposal == nil || resolveRes.Proposal.Status != proposals.StatusPending {
		t.Fatalf("expected L1 PENDING proposal on anomaly escalation, got (%s, %v)", string(resRaw), err)
	}

	// 3. What-If simulation: reject float input (400), run pure int64 5-stage simulation, commit L1 proposals, verify domain unchanged.
	codeFloat, _, bodyFloat := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/whatif/simulate", ownerTok, map[string]any{
		"name":             "Float Reject",
		"demand_delta_bps": 12.5,
	}, uuid.NewString())
	if codeFloat != http.StatusBadRequest {
		t.Fatalf("float demand_delta_bps = %d body=%s, want 400", codeFloat, string(bodyFloat))
	}

	code, _, simRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/whatif/simulate", ownerTok, map[string]any{
		"name":                   "Q4 Demand & FX Shock",
		"horizon_days":           30,
		"demand_delta_bps":       2500,
		"factory_downtime_hours": 12,
		"fx_rate_shock_bps":      800,
		"credit_limit_delta_bps": 1000,
		"unit_price_delta_bps":   500,
	}, uuid.NewString())
	if code != http.StatusCreated {
		t.Fatalf("POST /v1/whatif/simulate = %d body=%s", code, string(simRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, simRaw, "whatif.simulate")
	var simEnvelope struct {
		Simulation whatif.WhatIfSimulation `json:"simulation"`
	}
	if err := json.Unmarshal(simRaw, &simEnvelope); err != nil || simEnvelope.Simulation.SimulationID == "" {
		t.Fatalf("decode simulation: %v body=%s", err, string(simRaw))
	}

	// Domain table StockQuantity in Products must NEVER be mutated by What-If simulate.
	var prodQty int64
	prodRow, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "Products", spanner.Key{skuID}, []string{"StockQuantity"})
	if err != nil || prodRow.Column(0, &prodQty) != nil || prodQty != 250 {
		t.Fatalf("What-If mutated Products or failed read: qty=%d err=%v", prodQty, err)
	}

	code, _, commitRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/whatif/simulations/"+simEnvelope.Simulation.SimulationID+"/commit-proposals", ownerTok, map[string]any{}, uuid.NewString())
	if code != http.StatusOK && code != http.StatusCreated {
		t.Fatalf("POST commit-proposals = %d body=%s", code, string(commitRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, commitRaw, "whatif.commit")
	var commitRes whatif.CommitProposalsResult
	if err := json.Unmarshal(commitRaw, &commitRes); err != nil || len(commitRes.CommittedProposalIDs) == 0 || commitRes.Simulation.Status != "COMMITTED" {
		t.Fatalf("unexpected commit-proposals response: %s err=%v", string(commitRaw), err)
	}
	for _, cp := range commitRes.Proposals {
		if cp.Status != proposals.StatusPending {
			t.Fatalf("committed What-If proposal must stay PENDING L1, got %q", cp.Status)
		}
	}

	// Verify OutboxEvents recorded anomaly & whatif events + intact AuditEvents hash chain.
	outboxLog := collectWave4BOutboxPayloads(t, client, tid)
	for _, wantEv := range []string{events.EventAnomalyDetected, events.EventAnomalyResolved, events.EventWhatIfSimulated} {
		if !strings.Contains(outboxLog, wantEv) {
			t.Fatalf("expected outbox event %q for tenant %s, got:\n%s", wantEv, tid, outboxLog)
		}
	}
	today := time.Now().UTC().Format("2006-01-02")
	code, _, verRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/audit/verify?date="+today, ownerTok, nil, "")
	if code != http.StatusOK {
		t.Fatalf("GET /v1/audit/verify = %d body=%s", code, string(verRaw))
	}
	var ver audit.ChainVerifyResult
	if err := json.Unmarshal(verRaw, &ver); err != nil || !ver.Valid || ver.EventCount < 3 {
		t.Fatalf("expected valid hash chain with >=3 events, got (%+v, %v)", ver, err)
	}
}

func TestWave4B_ColdChainHACCP_L2SoakAutoExec_CustomRoles_And_SCIMv2(t *testing.T) {
	client := openWave4BSpannerClient(t)
	suffix := uuid.NewString()[:8]
	tid := "sup_w4b_ops_" + suffix
	const scimToken = "wave4b-e2e-scim-bearer-2026"
	h := NewTestHarnessWithOptions(t, HarnessOptions{
		SpannerClient:    client,
		SCIMTenantTokens: map[string]string{tid: scimToken},
	})
	ctx := context.Background()

	whID := "wh_ops_" + suffix
	catID := "cat_ops_" + suffix
	prodChilled := "prod_ch_" + suffix
	lotRes := "lot_res_" + suffix
	lotBad := "lot_bad_" + suffix
	ordID := "ord_cc_" + suffix
	manID := "man_cc_" + suffix
	rdOK := "rd_ok_" + suffix
	rdExc := "rd_exc_" + suffix
	repID := "rep_cc_" + suffix

	cleanupWave4BTenant(t, client, tid,
		spanner.Delete("TemperatureReadings", spanner.Key{rdOK}),
		spanner.Delete("TemperatureReadings", spanner.Key{rdExc}),
		spanner.Delete("OrderConditionReports", spanner.Key{repID}),
		spanner.Delete("SupplierTruckManifests", spanner.Key{manID}),
		spanner.Delete("Orders", spanner.Key{ordID}),
		spanner.Delete("StockLots", spanner.Key{lotRes}),
		spanner.Delete("StockLots", spanner.Key{lotBad}),
		spanner.Delete("WarehouseLocations", spanner.KeyRange{Start: spanner.Key{whID}, End: spanner.Key{whID}, Kind: spanner.ClosedClosed}),
		spanner.Delete("Products", spanner.Key{prodChilled}),
		spanner.Delete("ProductCategories", spanner.Key{catID}),
		spanner.Delete("ReplenishmentInsights", spanner.Key{tid, whID, prodChilled}),
	)

	now := time.Now().UTC().Add(-15 * time.Minute)
	lineCC, _ := json.Marshal([]map[string]any{{"product_id": prodChilled, "quantity": 4}})
	if _, err := client.Apply(ctx, []*spanner.Mutation{
		spanner.InsertMap("ProductCategories", map[string]any{
			"CategoryId": catID, "SupplierId": tid, "Name": "Cold Chain", "SortOrder": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Products", map[string]any{
			"ProductId": prodChilled, "SupplierId": tid, "CategoryId": catID, "Name": "Yogurt 500g",
			"PriceMinor": int64(18_000), "Currency": "UZS", "StockQuantity": int64(30),
			"HandlingClass": "CHILLED", "RequiresColdChain": true, "IsPerishable": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("WarehouseLocations", map[string]any{
			"WarehouseId": whID, "LocationId": "bin-chill", "Zone": "CHILL", "LocationType": "PICK",
			"PickSequence": int64(1), "IsActive": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("WarehouseLocations", map[string]any{
			"WarehouseId": whID, "LocationId": "bin-amb-bad", "Zone": "AMBIENT", "LocationType": "PICK",
			"PickSequence": int64(2), "IsActive": true,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotRes, "SupplierId": tid, "WarehouseId": whID, "ProductId": prodChilled,
			"LocationId": "bin-chill", "LotCode": "LOT-CH-1", "QuantityOnHand": int64(20),
			"QuantityReserved": int64(4), "Status": "AVAILABLE", "ReceivedAt": now,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("StockLots", map[string]any{
			"LotId": lotBad, "SupplierId": tid, "WarehouseId": whID, "ProductId": prodChilled,
			"LocationId": "bin-amb-bad", "LotCode": "LOT-AMB-1", "QuantityOnHand": int64(10),
			"QuantityReserved": int64(0), "Status": "AVAILABLE", "ReceivedAt": now,
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("SupplierTruckManifests", map[string]any{
			"ManifestId": manID, "SupplierId": tid, "WarehouseId": whID,
			"TruckId": "trk-e2e", "DriverId": "drv-e2e", "State": "IN_TRANSIT",
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("Orders", map[string]any{
			"OrderId": ordID, "SupplierId": tid, "RetailerId": "ret-e2e", "WarehouseId": whID,
			"ManifestId": manID, "Status": "DISPATCHED", "OrderSource": "RETAILER",
			"ConfirmationStatus": "CONFIRMED", "LineItemsJson": lineCC, "TotalMinor": int64(72_000),
			"OriginalTotalMinor": int64(72_000), "Currency": "UZS", "Version": int64(1),
			"CreatedAt": spanner.CommitTimestamp, "UpdatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderLotReservations", map[string]any{
			"OrderId": ordID, "LotId": lotRes, "Quantity": int64(4), "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId": rdOK, "ManifestId": manID, "SensorId": "s-1",
			"RecordedAt": now.Add(time.Minute), "TempC": 4.0, "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("TemperatureReadings", map[string]any{
			"ReadingId": rdExc, "ManifestId": manID, "SensorId": "s-1",
			"RecordedAt": now.Add(2 * time.Minute), "TempC": 14.5, "CreatedAt": spanner.CommitTimestamp,
		}),
		spanner.InsertMap("OrderConditionReports", map[string]any{
			"ReportId": repID, "OrderId": ordID, "SupplierId": tid, "RetailerId": "ret-e2e",
			"SKU": prodChilled, "ConditionType": "TEMPERATURE_BREACH", "Severity": "CRITICAL",
			"Description": "Cabin excursion 14.5C", "ReportedBy": "coldchain-sensor", "ReportedByRole": "SYSTEM",
			"ResolutionStatus": "OPEN", "CreatedAt": spanner.CommitTimestamp,
		}),
	}); err != nil {
		t.Fatalf("seed HACCP E2E fixtures: %v", err)
	}

	ownerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tid, "owner-ops-"+suffix, nil)
	auditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tid, "aud-ops-"+suffix, nil)
	expAt := time.Now().UTC().Add(-time.Minute)
	expiredAuditorTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleAuditor, tid, "aud-exp-ops-"+suffix, &expAt)
	agentTok, _ := auth.Issue(auth.Claims{
		Subject:      "agent-ops-" + suffix,
		Role:         auth.RoleAdmin,
		SupplierID:   tid,
		SupplierRole: auth.SupplierRoleAgent,
		ActorType:    auth.ActorTypeAgent,
	}, auth.IssueOptions{Secret: h.JWTSecret, TTL: time.Hour, Issuer: h.JWTIssuer})

	// --- 1. Cold-Chain HACCP JSON + CSV + Expired Auditor 401 ---
	if cExp, _, bExp := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/compliance/haccp", expiredAuditorTok, nil, ""); cExp != http.StatusUnauthorized || !strings.Contains(string(bExp), "auditor_access_expired") {
		t.Fatalf("expired auditor GET /v1/compliance/haccp = %d body=%s", cExp, string(bExp))
	}
	if cCross, _, _ := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/compliance/haccp?entity=other-tenant", ownerTok, nil, ""); cCross != http.StatusForbidden {
		t.Fatalf("cross-tenant GET /v1/compliance/haccp = %d, want 403", cCross)
	}
	cJSON, _, haccpRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/compliance/haccp", auditorTok, nil, "")
	if cJSON != http.StatusOK {
		t.Fatalf("AUDITOR GET /v1/compliance/haccp = %d body=%s", cJSON, string(haccpRaw))
	}
	assertWave4AJSONNoFloatLiterals(t, haccpRaw, "compliance.haccp")
	var haccpRep stocklots.HACCPReport
	if err := json.Unmarshal(haccpRaw, &haccpRep); err != nil {
		t.Fatalf("decode HACCPReport: %v", err)
	}
	if haccpRep.TotalReadings != 2 || haccpRep.ExcursionCount != 1 || haccpRep.ComplianceRateBps != 5000 || haccpRep.OpenBreaches != 1 || haccpRep.AffectedLotsCount != 2 {
		t.Fatalf("unexpected HACCPReport metrics: %+v", haccpRep)
	}
	cCSV, hdrCSV, csvRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/compliance/haccp?format=csv", auditorTok, nil, "")
	if cCSV != http.StatusOK || !strings.Contains(hdrCSV.Get("Content-Type"), "text/csv") {
		t.Fatalf("AUDITOR GET /v1/compliance/haccp?format=csv = %d ct=%s", cCSV, hdrCSV.Get("Content-Type"))
	}
	csvRows, err := csv.NewReader(strings.NewReader(string(csvRaw))).ReadAll()
	if err != nil || len(csvRows) < 6 {
		t.Fatalf("unexpected HACCP CSV rows=%d err=%v", len(csvRows), err)
	}

	// --- 2. L2 Soak-Gated Auto-Execution ---
	// Before soak metrics exist -> ineligible.
	cPre, _, preRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/proposals/l2-evaluate?max_l2_amount_minor=100000", ownerTok, nil, "")
	if cPre != http.StatusOK {
		t.Fatalf("GET /v1/proposals/l2-evaluate (pre-soak) = %d body=%s", cPre, string(preRaw))
	}
	var preEval proposals.L2EvaluationResult
	_ = json.Unmarshal(preRaw, &preEval)
	if preEval.Eligible || len(preEval.Blockers) == 0 {
		t.Fatalf("expected pre-soak ineligible with blockers, got %+v", preEval)
	}

	// Seed 15 historical EXECUTED proposals starting 18 days ago -> satisfies soak >= 14d, count >= 10, rollback <= 500 bps.
	soakMuts := make([]*spanner.Mutation, 0, 15)
	for i := 0; i < 15; i++ {
		ts := time.Now().UTC().AddDate(0, 0, -18).Add(time.Duration(i) * time.Hour)
		soakMuts = append(soakMuts, spanner.InsertMap("Proposals", map[string]any{
			"TenantId":             tid,
			"ProposalId":           fmt.Sprintf("prop-hist-%d-%s", i, suffix),
			"AgentId":              "REORDER_AGENT",
			"ProposalType":         proposals.TypeReorder,
			"TargetObjectType":     "Product",
			"TargetObjectId":       prodChilled,
			"Title":                "Historical L1 executed reorder",
			"Summary":              "Soak evidence",
			"Rationale":            "Soak window",
			"PayloadJson":          `{"action":"reorder_suggestion","quantity":10}`,
			"PayloadHash":          "hist-hash",
			"EvidenceRefsJson":     "[]",
			"ConfidenceBps":        int64(9000),
			"EstimatedImpactMinor": int64(25_000),
			"Currency":             "UZS",
			"AutonomyLevel":        proposals.AutonomyLevelL1,
			"Status":               proposals.StatusExecuted,
			"CreatedBy":            "REORDER_AGENT",
			"ActorType":            auth.ActorTypeAgent,
			"CreatedAt":            ts,
			"UpdatedAt":            ts,
			"ExecutedAt":           ts,
		}))
	}
	if _, err := client.Apply(ctx, soakMuts); err != nil {
		t.Fatalf("seed L2 soak history: %v", err)
	}

	// Create 1 eligible low-value REORDER proposal AND 1 hard-blocked CREDIT_HOLD proposal via a separate maker actor (enforcing maker-checker SoD).
	makerTok := issueWave4ASupplierToken(t, h.JWTSecret, h.JWTIssuer, auth.RoleAdmin, auth.SupplierRoleOwner, tid, "maker-ops-"+suffix, nil)
	_, _, lowPropRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/proposals", makerTok, map[string]any{
		"agent_id":               "REORDER_AGENT",
		"proposal_type":          proposals.TypeReorder,
		"target_object_type":     "Product",
		"target_object_id":       prodChilled,
		"title":                  "Low-value milk reorder",
		"summary":                "L2 auto-exec candidate",
		"rationale":              "Within ceiling",
		"payload":                map[string]any{"action": "approve_reorder_suggestion", "warehouse_id": whID, "product_id": prodChilled, "suggested_qty": 5},
		"estimated_impact_minor": 45_000,
		"confidence_bps":         9200,
		"currency":               "UZS",
	}, uuid.NewString())
	var lowProp struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(lowPropRaw, &lowProp)

	_, _, creditPropRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/proposals", makerTok, map[string]any{
		"agent_id":               "COLLECTIONS_AGENT",
		"proposal_type":          proposals.TypeCreditHold,
		"target_object_type":     "Retailer",
		"target_object_id":       "ret-e2e",
		"title":                  "High-risk credit limit action",
		"summary":                "Must remain L1 human gated",
		"rationale":              "High-risk D4 invariant",
		"payload":                map[string]any{"action": "credit_limit", "new_limit_minor": 50_000},
		"estimated_impact_minor": 30_000,
		"confidence_bps":         9500,
		"currency":               "UZS",
	}, uuid.NewString())
	var creditProp struct {
		Proposal proposals.Proposal `json:"proposal"`
	}
	_ = json.Unmarshal(creditPropRaw, &creditProp)

	if lowProp.Proposal.ProposalID == "" || creditProp.Proposal.ProposalID == "" {
		t.Fatalf("failed to create L2 candidate proposals: low=%s credit=%s", string(lowPropRaw), string(creditPropRaw))
	}

	// AUDITOR and AGENT must receive 403 on POST /v1/proposals/l2-evaluate!
	for _, tok := range []string{auditorTok, agentTok} {
		cBlk, _, _ := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/proposals/l2-evaluate", tok, map[string]any{"auto_execute": true}, uuid.NewString())
		if cBlk != http.StatusForbidden {
			t.Fatalf("AUDITOR/AGENT POST /v1/proposals/l2-evaluate = %d, want 403", cBlk)
		}
	}

	// OWNER runs POST /v1/proposals/l2-evaluate {auto_execute:true} -> promotes & executes ONLY lowProp to L2!
	cL2, _, l2Raw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/proposals/l2-evaluate", ownerTok, map[string]any{
		"auto_execute":        true,
		"max_l2_amount_minor": 100_000,
	}, uuid.NewString())
	if cL2 != http.StatusOK {
		t.Fatalf("POST /v1/proposals/l2-evaluate = %d body=%s", cL2, string(l2Raw))
	}
	assertWave4AJSONNoFloatLiterals(t, l2Raw, "proposals.l2-evaluate")
	var l2Res proposals.L2EvaluationResult
	if err := json.Unmarshal(l2Raw, &l2Res); err != nil {
		t.Fatalf("decode L2EvaluationResult: %v", err)
	}
	if !l2Res.Eligible || l2Res.AutoExecutedCount != 1 || len(l2Res.AutoExecutedProposalIDs) != 1 || l2Res.AutoExecutedProposalIDs[0] != lowProp.Proposal.ProposalID {
		t.Fatalf("expected 1 auto-executed low-value REORDER proposal, got %+v", l2Res)
	}
	if l2Res.ExecutedProposals[0].AutonomyLevel != proposals.AutonomyLevelL2 || l2Res.ExecutedProposals[0].Status != proposals.StatusExecuted {
		t.Fatalf("expected AutonomyLevel=L2 & EXECUTED, got %+v", l2Res.ExecutedProposals[0])
	}
	if len(l2Res.BlockedProposals) == 0 {
		t.Fatalf("expected high-risk CREDIT_LIMIT_ADJUST in blocked_proposals")
	}
	var creditAfterStatus string
	credRow, err := client.Single().WithTimestampBound(spanner.StrongRead()).ReadRow(ctx, "Proposals", spanner.Key{tid, creditProp.Proposal.ProposalID}, []string{"Status"})
	if err != nil || credRow.Column(0, &creditAfterStatus) != nil || creditAfterStatus != proposals.StatusPending {
		t.Fatalf("CREDIT_HOLD must remain PENDING L1, got status=%q err=%v", creditAfterStatus, err)
	}

	// --- 3. Enterprise Custom Roles + SCIM 2.0 ---
	cRole, _, roleRaw := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/iam/custom-roles", ownerTok, map[string]any{
		"role_name":   "COLD_OPS_CLERK",
		"description": "Wave 4B E2E custom warehouse role",
		"permissions": []string{auth.PermStockCount, auth.PermAuditRead},
	}, uuid.NewString())
	if cRole != http.StatusCreated {
		t.Fatalf("POST /v1/iam/custom-roles = %d body=%s", cRole, string(roleRaw))
	}

	// Anti-escalation & AUDITOR/AGENT mutation blocks on Custom Roles.
	if cEsc, _, _ := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/iam/custom-roles", ownerTok, map[string]any{
		"role_name":   "BAD_AUD",
		"read_only":   true,
		"permissions": []string{auth.PermJournalPost},
	}, uuid.NewString()); cEsc != http.StatusForbidden {
		t.Fatalf("read-only mutating custom role = %d, want 403", cEsc)
	}
	if cAudPost, _, _ := doWave4AHTTP(t, http.MethodPost, h.Server.URL+"/v1/iam/custom-roles", auditorTok, map[string]any{"role_name": "X", "permissions": []string{auth.PermAuditRead}}, uuid.NewString()); cAudPost != http.StatusForbidden {
		t.Fatalf("AUDITOR POST custom-roles = %d, want 403", cAudPost)
	}

	// SCIM 2.0 provisioning via SCIMBearerToken + X-Tenant-Id header.
	doScim := func(method, path string, body any) (int, http.Header, []byte) {
		var raw []byte
		if body != nil {
			raw, _ = json.Marshal(body)
		}
		req, _ := http.NewRequest(method, h.Server.URL+path, strings.NewReader(string(raw)))
		req.Header.Set("Authorization", "Bearer "+scimToken)
		req.Header.Set("X-Tenant-Id", tid)
		if body != nil {
			req.Header.Set("Content-Type", "application/scim+json")
		}
		resp, derr := (&http.Client{Timeout: 15 * time.Second}).Do(req)
		if derr != nil {
			t.Fatalf("SCIM %s %s: %v", method, path, derr)
		}
		defer resp.Body.Close()
		out := make([]byte, 0)
		buf := make([]byte, 4096)
		for {
			n, rerr := resp.Body.Read(buf)
			if n > 0 {
				out = append(out, buf[:n]...)
			}
			if rerr != nil {
				break
			}
		}
		return resp.StatusCode, resp.Header, out
	}

	scU, scUHdr, scURaw := doScim(http.MethodPost, "/scim/v2/Users", map[string]any{
		"id":          "scim-e2e-u1-" + suffix,
		"userName":    "cold.clerk@example.uz",
		"displayName": "Cold Clerk",
		"active":      true,
		"roles":       []map[string]any{{"value": "COLD_OPS_CLERK"}},
	})
	if scU != http.StatusCreated || !strings.Contains(scUHdr.Get("Content-Type"), "application/scim+json") {
		t.Fatalf("SCIM POST /scim/v2/Users = %d ct=%s body=%s", scU, scUHdr.Get("Content-Type"), string(scURaw))
	}
	var scimUser iam.ScimUser
	_ = json.Unmarshal(scURaw, &scimUser)

	scG, _, scGRaw := doScim(http.MethodPost, "/scim/v2/Groups", map[string]any{
		"id":          "scim-e2e-g1-" + suffix,
		"displayName": "Cold Ops",
		"roles":       []map[string]any{{"value": "COLD_OPS_CLERK"}},
		"members":     []map[string]any{{"value": scimUser.ID}},
	})
	if scG != http.StatusCreated {
		t.Fatalf("SCIM POST /scim/v2/Groups = %d body=%s", scG, string(scGRaw))
	}

	// AUDITOR portal JWT can GET /scim/v2/Users (200) but cannot PATCH/DELETE (403); expired AUDITOR -> 401.
	if cAudGet, _, _ := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/scim/v2/Users", auditorTok, nil, ""); cAudGet != http.StatusOK {
		t.Fatalf("AUDITOR GET /scim/v2/Users = %d, want 200", cAudGet)
	}
	if cExpScim, _, bExpScim := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/scim/v2/Users", expiredAuditorTok, nil, ""); cExpScim != http.StatusUnauthorized || !strings.Contains(string(bExpScim), "auditor_access_expired") {
		t.Fatalf("expired AUDITOR GET /scim/v2/Users = %d body=%s", cExpScim, string(bExpScim))
	}
	if cAudPatch, _, _ := doWave4AHTTP(t, http.MethodPatch, h.Server.URL+"/scim/v2/Users/"+scimUser.ID, auditorTok, map[string]any{"Operations": []map[string]any{{"op": "replace", "path": "active", "value": false}}}, uuid.NewString()); cAudPatch != http.StatusForbidden {
		t.Fatalf("AUDITOR PATCH /scim/v2/Users = %d, want 403", cAudPatch)
	}

	// Deprovision User via DELETE -> 204 No Content.
	if scDelU, _, _ := doScim(http.MethodDelete, "/scim/v2/Users/"+scimUser.ID, nil); scDelU != http.StatusNoContent {
		t.Fatalf("SCIM DELETE /scim/v2/Users = %d, want 204", scDelU)
	}
	if scDelG, _, _ := doScim(http.MethodDelete, "/scim/v2/Groups/scim-e2e-g1-"+suffix, nil); scDelG != http.StatusNoContent {
		t.Fatalf("SCIM DELETE /scim/v2/Groups = %d, want 204", scDelG)
	}

	// --- 4. Verify all Wave 4B OutboxEvents + Intact AuditEvents Hash Chain ---
	outboxLog := collectWave4BOutboxPayloads(t, client, tid)
	for _, wantEv := range []string{
		events.EventHaccpExcursionReported,
		events.EventProposalL2AutoExecuted,
		events.EventScimIdentitySynced,
	} {
		if !strings.Contains(outboxLog, wantEv) {
			t.Fatalf("expected outbox event %q for tenant %s, got:\n%s", wantEv, tid, outboxLog)
		}
	}

	today := time.Now().UTC().Format("2006-01-02")
	cVer, _, verRaw := doWave4AHTTP(t, http.MethodGet, h.Server.URL+"/v1/audit/verify?date="+today, ownerTok, nil, "")
	if cVer != http.StatusOK {
		t.Fatalf("GET /v1/audit/verify = %d body=%s", cVer, string(verRaw))
	}
	var vr audit.ChainVerifyResult
	if err := json.Unmarshal(verRaw, &vr); err != nil || !vr.Valid || vr.EventCount < 5 {
		t.Fatalf("expected valid hash chain with >=5 AuditEvents, got (%+v, %v)", vr, err)
	}
}
