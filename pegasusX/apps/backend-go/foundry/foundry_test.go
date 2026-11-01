package foundry_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

func TestOntologySearchAroundAndLotRecallBlastRadius(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()
	cell := "cell-uz"
	sup := "sup_nestle_uz"

	nodes := []foundry.OntologyObject{
		{CellID: cell, ObjectType: "RawMaterialLot", ObjectID: "rm_milk_88", SupplierID: sup, Properties: map[string]any{"batch": "M-88"}},
		{CellID: cell, ObjectType: "WorkOrder", ObjectID: "wo_101", SupplierID: sup, Properties: map[string]any{"line": "L1"}},
		{CellID: cell, ObjectType: "FinishedLot", ObjectID: "fl_yogurt_50", SupplierID: sup, Properties: map[string]any{"sku": "YOG-500G"}},
		{CellID: cell, ObjectType: "SSCCPallet", ObjectID: "sscc_00389", SupplierID: sup, Properties: map[string]any{"cases": 60}},
		{CellID: cell, ObjectType: "Tote", ObjectID: "tote_77", SupplierID: sup, Properties: map[string]any{"bin": "A-04"}},
		{CellID: cell, ObjectType: "Order", ObjectID: "ord_in_transit_1", SupplierID: sup, Properties: map[string]any{"status": "IN_TRANSIT"}},
		{CellID: cell, ObjectType: "Order", ObjectID: "ord_completed_2", SupplierID: sup, Properties: map[string]any{"status": "COMPLETED"}},
		{CellID: cell, ObjectType: "SoliqFactura", ObjectID: "ehf_9901", SupplierID: sup, Properties: map[string]any{"ikpu": "02104001001000000"}},
	}
	for _, n := range nodes {
		if _, err := eng.UpsertObject(ctx, n); err != nil {
			t.Fatalf("UpsertObject(%s/%s): %v", n.ObjectType, n.ObjectID, err)
		}
	}

	links := []foundry.OntologyGraphEdge{
		{CellID: cell, SourceObjectType: "RawMaterialLot", SourceObjectID: "rm_milk_88", LinkType: "CONSUMED_IN", TargetObjectType: "WorkOrder", TargetObjectID: "wo_101", SupplierID: sup, CreatedByActionID: "act_1"},
		{CellID: cell, SourceObjectType: "WorkOrder", SourceObjectID: "wo_101", LinkType: "PRODUCED", TargetObjectType: "FinishedLot", TargetObjectID: "fl_yogurt_50", SupplierID: sup, CreatedByActionID: "act_2"},
		{CellID: cell, SourceObjectType: "FinishedLot", SourceObjectID: "fl_yogurt_50", LinkType: "PALLETIZED_TO", TargetObjectType: "SSCCPallet", TargetObjectID: "sscc_00389", SupplierID: sup, CreatedByActionID: "act_3"},
		{CellID: cell, SourceObjectType: "SSCCPallet", SourceObjectID: "sscc_00389", LinkType: "PICKED_INTO", TargetObjectType: "Tote", TargetObjectID: "tote_77", SupplierID: sup, CreatedByActionID: "act_4"},
		{CellID: cell, SourceObjectType: "Tote", SourceObjectID: "tote_77", LinkType: "DELIVERED_IN", TargetObjectType: "Order", TargetObjectID: "ord_in_transit_1", SupplierID: sup, CreatedByActionID: "act_5"},
		{CellID: cell, SourceObjectType: "Tote", SourceObjectID: "tote_77", LinkType: "DELIVERED_IN", TargetObjectType: "Order", TargetObjectID: "ord_completed_2", SupplierID: sup, CreatedByActionID: "act_6"},
		{CellID: cell, SourceObjectType: "Order", SourceObjectID: "ord_completed_2", LinkType: "FISCALIZED_AS", TargetObjectType: "SoliqFactura", TargetObjectID: "ehf_9901", SupplierID: sup, CreatedByActionID: "act_7"},
	}
	for _, l := range links {
		if _, err := eng.LinkObjects(ctx, l); err != nil {
			t.Fatalf("LinkObjects(%s -> %s): %v", l.SourceObjectID, l.TargetObjectID, err)
		}
	}

	report, err := eng.TriggerContaminatedLotRecall(ctx, cell, sup, "rm_milk_88")
	if err != nil {
		t.Fatalf("TriggerContaminatedLotRecall: %v", err)
	}
	if len(report.AffectedWorkOrders) != 1 || report.AffectedWorkOrders[0] != "wo_101" {
		t.Fatalf("expected wo_101 in AffectedWorkOrders, got %v", report.AffectedWorkOrders)
	}
	if len(report.QuarantinedSSCCPallets) != 1 || report.QuarantinedSSCCPallets[0] != "sscc_00389" {
		t.Fatalf("expected sscc_00389 in QuarantinedSSCCPallets, got %v", report.QuarantinedSSCCPallets)
	}
	if len(report.InTransitDriverDropBlocks) != 1 || report.InTransitDriverDropBlocks[0] != "ord_in_transit_1" {
		t.Fatalf("expected ord_in_transit_1 in InTransitDriverDropBlocks, got %v", report.InTransitDriverDropBlocks)
	}
	if len(report.RetailerReversePickupOrders) != 1 || report.RetailerReversePickupOrders[0] != "ord_completed_2" {
		t.Fatalf("expected ord_completed_2 in RetailerReversePickupOrders, got %v", report.RetailerReversePickupOrders)
	}
	if len(report.ImpactedSoliqFacturas) != 1 || report.ImpactedSoliqFacturas[0] != "ehf_9901" {
		t.Fatalf("expected ehf_9901 in ImpactedSoliqFacturas, got %v", report.ImpactedSoliqFacturas)
	}

	// Verify propagating security marking reached the leaf SoliqFactura node.
	facturaObj, err := eng.GetObject(ctx, cell, "SoliqFactura", "ehf_9901")
	if err != nil {
		t.Fatalf("GetObject(SoliqFactura): %v", err)
	}
	if len(facturaObj.SecurityMarkings) == 0 || facturaObj.SecurityMarkings[0] != foundry.MarkingLotRecallQuarantine {
		t.Fatalf("expected propagated marking %s on SoliqFactura, got %v", foundry.MarkingLotRecallQuarantine, facturaObj.SecurityMarkings)
	}
}

func TestScenarioCopyOnWriteAndOptimisticConcurrencyWatermark(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()
	cell := "cell-uz"
	sup := "sup_1"

	// 1. Seed base inventory bin object at Version 1.
	base, err := eng.UpsertObject(ctx, foundry.OntologyObject{
		CellID:     cell,
		ObjectType: "WarehouseBin",
		ObjectID:   "bin_A12",
		SupplierID: sup,
		Properties: map[string]any{"available_cases": 200},
	})
	if err != nil || base.Version != 1 {
		t.Fatalf("seed base object failed: v=%v err=%v", base, err)
	}

	// 2. Patch inside Scenario-1 without affecting live production state.
	_, err = eng.ApplyScenarioPatch(ctx, cell, "scen_1", sup, "WarehouseBin", "bin_A12", foundry.ScenarioOpUpsert, map[string]any{
		"available_cases": 120,
	})
	if err != nil {
		t.Fatalf("ApplyScenarioPatch: %v", err)
	}

	liveView, _ := eng.ResolveObjectState(ctx, cell, "", "WarehouseBin", "bin_A12")
	if liveView.Properties["available_cases"] != 200 {
		t.Fatalf("expected live production state to remain 200, got %v", liveView.Properties["available_cases"])
	}
	scenView, _ := eng.ResolveObjectState(ctx, cell, "scen_1", "WarehouseBin", "bin_A12")
	if scenView.Properties["available_cases"] != 120 {
		t.Fatalf("expected scenario state to resolve to 120, got %v", scenView.Properties["available_cases"])
	}

	// 3. Simulate concurrent production mutation (bumps live Version to 2) before Scenario-1 commits!
	_, err = eng.UpsertObject(ctx, foundry.OntologyObject{
		CellID:     cell,
		ObjectType: "WarehouseBin",
		ObjectID:   "bin_A12",
		SupplierID: sup,
		Properties: map[string]any{"available_cases": 150},
	})
	if err != nil {
		t.Fatalf("concurrent live update: %v", err)
	}

	commitRes, err := eng.CommitScenario(ctx, "scen_1")
	if !errors.Is(err, foundry.ErrScenarioWatermarkDrift) {
		t.Fatalf("expected ErrScenarioWatermarkDrift on stale BaseVersion, got %v", err)
	}
	if len(commitRes.Conflicts) != 1 || commitRes.Conflicts[0].BaseVersion != 1 || commitRes.Conflicts[0].LiveVersion != 2 {
		t.Fatalf("unexpected conflict report: %+v", commitRes.Conflicts)
	}

	// 4. Create Scenario-2 anchored to Version 2 and commit cleanly.
	_, err = eng.ApplyScenarioPatch(ctx, cell, "scen_2", sup, "WarehouseBin", "bin_A12", foundry.ScenarioOpUpsert, map[string]any{
		"available_cases": 90,
	})
	if err != nil {
		t.Fatalf("ApplyScenarioPatch scen_2: %v", err)
	}
	commitRes2, err := eng.CommitScenario(ctx, "scen_2")
	if err != nil || commitRes2.CommittedObjects != 1 {
		t.Fatalf("CommitScenario scen_2 failed: %v", err)
	}
	postCommitLive, _ := eng.GetObject(ctx, cell, "WarehouseBin", "bin_A12")
	if postCommitLive.Properties["available_cases"] != 90 || postCommitLive.Version != 3 {
		t.Fatalf("expected live available_cases=90 version=3, got %+v", postCommitLive)
	}
}

func TestCashierDualBlindRatchetMicroSkimmingAndRRNReclassification(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// 1. Start blind count session for 1,000,000 UZS = 100,000,000 Tiyin.
	sess, err := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
		CellID:            "cell-uz",
		SupplierID:        "sup_1",
		DepotID:           "depot_tashkent_1",
		DriverID:          "drv_01",
		RouteID:           "rt_100",
		BagSealBarcode:    "SEAL-9001",
		ExpectedCashMinor: 100_000_000,
	})
	if err != nil {
		t.Fatalf("StartBlindCountSession: %v", err)
	}
	if sess.ExpectedCashMinor != nil || sess.VarianceMinor != nil {
		t.Fatalf("expected ExpectedCashMinor to be nil (masked) on session start")
	}

	// Duplicate bag seal must fail closed.
	if _, err := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
		DepotID:        "depot_tashkent_1",
		DriverID:       "drv_01",
		BagSealBarcode: "SEAL-9001",
	}); !errors.Is(err, foundry.ErrDuplicateBagSeal) {
		t.Fatalf("expected ErrDuplicateBagSeal, got %v", err)
	}

	// 2. Cashier submits Attempt #1 with 500,000 UZS (5 x 100,000 UZS notes) -> 500,000 UZS shortage!
	// Dual-blind ratchet must trigger BLIND_RECOUNT_REQUIRED and STILL mask ExpectedCashMinor.
	view1, err := eng.SubmitBlindDenominationCount(ctx, sess.SessionID, map[int64]int64{100000: 5}, 500_000)
	if !errors.Is(err, foundry.ErrBlindRecountRequired) {
		t.Fatalf("expected ErrBlindRecountRequired on Attempt #1 shortage, got %v", err)
	}
	if view1.Status != foundry.CashierStatusBlindRecountReq || view1.ExpectedCashMinor != nil {
		t.Fatalf("expected BLIND_RECOUNT_REQUIRED with masked ExpectedCashMinor, got %+v", view1)
	}

	// 3. Driver explains Stop #9 paid 500,000 UZS by Humo card; verify RRN tender reclassification!
	epodTime := time.Now().UTC()
	// First test invalid RRN outside 15m window -> must fail.
	if _, err := eng.ReclassifyCashToCardByRRN(ctx, foundry.RRNTenderReclassificationRequest{
		SessionID:         sess.SessionID,
		OrderID:           "ord_9",
		DisputedCashMinor: 50_000_000,
		GatewayRRN:        "RRN-882910",
		RRNAmountMinor:    50_000_000,
		EPODCapturedAt:    epodTime,
		RRNCapturedAt:     epodTime.Add(25 * time.Minute),
	}); !errors.Is(err, foundry.ErrInvalidRRNProof) {
		t.Fatalf("expected ErrInvalidRRNProof for >15m timestamp drift, got %v", err)
	}

	// Valid RRN within 3 minutes -> balances session to zero variance!
	reclassified, err := eng.ReclassifyCashToCardByRRN(ctx, foundry.RRNTenderReclassificationRequest{
		SessionID:         sess.SessionID,
		OrderID:           "ord_9",
		DisputedCashMinor: 50_000_000,
		GatewayRRN:        "RRN-882910",
		RRNAmountMinor:    50_000_000,
		EPODCapturedAt:    epodTime,
		RRNCapturedAt:     epodTime.Add(3 * time.Minute),
		ActorID:           "sup_fin_1",
	})
	if err != nil {
		t.Fatalf("ReclassifyCashToCardByRRN: %v", err)
	}
	if reclassified.Status != foundry.CashierStatusBalanced || reclassified.ClearanceTokenHash == nil || *reclassified.ClearanceTokenHash == "" {
		t.Fatalf("expected BALANCED status and ClearanceTokenHash after RRN proof, got %+v", reclassified)
	}

	// 4. Execute CIT Armored Pickup from Depot Vault.
	citEntry, err := eng.ExecuteCITArmoredHandover(ctx, foundry.CITHandoverRequest{
		DepotID:        "depot_tashkent_1",
		SupplierID:     "sup_1",
		SessionIDs:     []string{sess.SessionID},
		CourierBadgeID: "INKASS-774",
		CITSealBarcode: "CIT-SEAL-01",
		AmountMinor:    50_000_000,
	})
	if err != nil || citEntry.RunningBalanceMinor != 0 {
		t.Fatalf("ExecuteCITArmoredHandover failed: entry=%+v err=%v", citEntry, err)
	}

	// 5. Verify Micro-Skimming Detection (driver hits negative minor tolerance across 4 consecutive shifts).
	for i := 1; i <= 4; i++ {
		s, err := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
			CellID:            "cell-uz",
			SupplierID:        "sup_1",
			DepotID:           "depot_tashkent_1",
			DriverID:          "drv_skimmer",
			RouteID:           "rt_skim",
			BagSealBarcode:    "SEAL-SKIM-" + string(rune('A'+i)),
			ExpectedCashMinor: 10_400_000, // 104,000 UZS
		})
		if err != nil {
			t.Fatalf("StartBlindCountSession skim %d: %v", i, err)
		}
		// Submit 100,000 UZS (-4,000 UZS = -400,000 Tiyin, within 500,000 Tiyin tolerance).
		v, err := eng.SubmitBlindDenominationCount(ctx, s.SessionID, map[int64]int64{100000: 1}, 500_000)
		if err != nil {
			t.Fatalf("SubmitBlindDenominationCount skim %d: %v", i, err)
		}
		if i == 4 && (!v.MicroSkimmingFlagged || v.Status != foundry.CashierStatusDiscrepantQuarantine) {
			t.Fatalf("expected 4th consecutive negative tolerance shift to flag micro-skimming, got %+v", v)
		}
	}
}

func TestFleetDVIRRedTagAndMidRouteBreakdownRescue(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// 1. Vehicle 1 fails brakes on Pre-Trip DVIR -> Red-Tag Quarantine!
	eng.PairDriverWithVehicle("veh_broken_1", "drv_1")
	dvirFail, err := eng.SubmitDVIRInspection(ctx, foundry.FleetDVIRInspection{
		CellID:            "cell-uz",
		SupplierID:        "sup_1",
		VehicleID:         "veh_broken_1",
		DriverID:          "drv_1",
		RequiresColdChain: true,
		OdometerKm:        29900,
		Checklist: foundry.DVIRChecklist8Point{
			SteeringAndHornOK:             true,
			ServiceAndParkingBrakesOK:     false, // Brake defect!
			TiresTreadAndPressureOK:       true,
			LightsAndHazardsOK:            true,
			WipersAndFluidsOK:             true,
			EmergencyKitAndExtinguisherOK: true,
			CargoBoxSanitationOK:          true,
			ReeferTempCalibratedOK:        true,
		},
	})
	if err != nil {
		t.Fatalf("SubmitDVIRInspection: %v", err)
	}
	if !dvirFail.RedTagApplied || dvirFail.Status != foundry.DVIRStatusRedTagQuarantine || dvirFail.MaintenanceWorkOrderID == nil || *dvirFail.MaintenanceWorkOrderID == "" {
		t.Fatalf("expected RED_TAG_QUARANTINE with MaintenanceWorkOrderID, got %+v", dvirFail)
	}
	if err := eng.AssertVehicleDispatchAuthorized(ctx, "veh_broken_1"); !errors.Is(err, foundry.ErrVehicleRedTagged) {
		t.Fatalf("expected ErrVehicleRedTagged for red-tagged vehicle, got %v", err)
	}

	// 2. Rescue Vehicle 2 passes all 8 DVIR points.
	_, err = eng.SubmitDVIRInspection(ctx, foundry.FleetDVIRInspection{
		CellID:            "cell-uz",
		SupplierID:        "sup_1",
		VehicleID:         "veh_rescue_2",
		DriverID:          "drv_2",
		RequiresColdChain: true,
		OdometerKm:        15000,
		Checklist: foundry.DVIRChecklist8Point{
			SteeringAndHornOK:             true,
			ServiceAndParkingBrakesOK:     true,
			TiresTreadAndPressureOK:       true,
			LightsAndHazardsOK:            true,
			WipersAndFluidsOK:             true,
			EmergencyKitAndExtinguisherOK: true,
			CargoBoxSanitationOK:          true,
			ReeferTempCalibratedOK:        true,
		},
	})
	if err != nil {
		t.Fatalf("SubmitDVIRInspection veh_rescue_2: %v", err)
	}

	// 3. Execute Mid-Route Breakdown Rescue after Stop #2.
	// Tote 3 (chilled) suffered 65m thermal excursion (> 45m limit) -> must be quarantined!
	// Tote 4 (ambient/within limit) transfers to Rescue Driver 2.
	rescueRes, err := eng.ExecuteMidRouteBreakdownRescue(ctx, foundry.MidRouteRescueRequest{
		CellID:               "cell-uz",
		SupplierID:           "sup_1",
		OriginalRouteID:      "rt_1",
		BrokenVehicleID:      "veh_broken_1",
		OriginalDriverID:     "drv_1",
		RescueVehicleID:      "veh_rescue_2",
		RescueDriverID:       "drv_2",
		LastCompletedStopSeq: 2,
		NewRescueBoltSeal:    "SEAL-RESCUE-99",
		Totes: []foundry.RescueToteItem{
			{ToteID: "tote_1", StopSequence: 1, ExpectedCashMinor: 10_000_000},
			{ToteID: "tote_2", StopSequence: 2, ExpectedCashMinor: 15_000_000},
			{ToteID: "tote_3_spoiled", StopSequence: 3, IsColdChain: true, ExpectedCashMinor: 20_000_000, ThermalExcursionMinutes: 65, MaxAllowedExcursionMins: 45},
			{ToteID: "tote_4_ok", StopSequence: 4, IsColdChain: true, ExpectedCashMinor: 30_000_000, ThermalExcursionMinutes: 15, MaxAllowedExcursionMins: 45},
		},
	})
	if err != nil {
		t.Fatalf("ExecuteMidRouteBreakdownRescue: %v", err)
	}
	if rescueRes.OriginalDriverLiabilityMinor != 25_000_000 {
		t.Fatalf("expected OriginalDriverLiabilityMinor=25000000, got %d", rescueRes.OriginalDriverLiabilityMinor)
	}
	if rescueRes.RescueDriverLiabilityMinor != 30_000_000 {
		t.Fatalf("expected RescueDriverLiabilityMinor=30000000, got %d", rescueRes.RescueDriverLiabilityMinor)
	}
	if len(rescueRes.QuarantinedSpoiledToteIDs) != 1 || rescueRes.QuarantinedSpoiledToteIDs[0] != "tote_3_spoiled" {
		t.Fatalf("expected tote_3_spoiled quarantined, got %v", rescueRes.QuarantinedSpoiledToteIDs)
	}
	if len(rescueRes.TransferredToteIDs) != 1 || rescueRes.TransferredToteIDs[0] != "tote_4_ok" {
		t.Fatalf("expected tote_4_ok transferred, got %v", rescueRes.TransferredToteIDs)
	}
}

func TestEdgeGuardsSplitTenderAMLSmurfingFiscalClawbackCrushabilityFXAndToteTakeover(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	tiers := []foundry.VolumeBreakTier{
		{MinCases: 10, DiscountBps: 500},  // 5%
		{MinCases: 50, DiscountBps: 1200}, // 12%
	}

	// Guard #1A: DAMAGED_IN_TRANSIT preserves the 50-case 12% discount tier even when only 46 cases are accepted,
	// and reduces the Credit reservation leg first!
	reconciledDamage, err := eng.ReconcileSplitTenderPartialRejection(ctx, foundry.SplitTenderRejectionInput{
		OrderID:                     "ord_split_1",
		OrderedCases:                50,
		AcceptedCases:               46,
		BaseCasePriceMinor:          10_000_00, // 10,000 UZS
		RejectionReason:             "DAMAGED_IN_TRANSIT",
		Tiers:                       tiers,
		OriginalCreditReservedMinor: 22_000_000, // 220,000 UZS
		OriginalCashPlannedMinor:    22_000_000, // 220,000 UZS (Total = 440,000 UZS @ 12% off)
	})
	if err != nil {
		t.Fatalf("ReconcileSplitTenderPartialRejection damage: %v", err)
	}
	if !reconciledDamage.TierProtected || reconciledDamage.AppliedDiscountBps != 1200 {
		t.Fatalf("expected 1200 bps tier protection on DAMAGED_IN_TRANSIT, got %+v", reconciledDamage)
	}
	// 46 cases * 8,800 UZS = 404,800 UZS (40,480,000 Tiyin). Reduction of 3,520,000 Tiyin comes off Credit first!
	if reconciledDamage.FinalCreditBookedMinor != 18_480_000 || reconciledDamage.FinalCashDueMinor != 22_000_000 {
		t.Fatalf("expected Credit reduced to 18480000 and Cash preserved at 22000000, got credit=%d cash=%d",
			reconciledDamage.FinalCreditBookedMinor, reconciledDamage.FinalCashDueMinor)
	}

	// Guard #1B: BUYER_REMORSE_REFUSAL drops discount from 12% (50 cases) to 5% (10-49 cases).
	reconciledRemorse, err := eng.ReconcileSplitTenderPartialRejection(ctx, foundry.SplitTenderRejectionInput{
		OrderID:                     "ord_split_2",
		OrderedCases:                50,
		AcceptedCases:               46,
		BaseCasePriceMinor:          10_000_00,
		RejectionReason:             "BUYER_REMORSE_REFUSAL",
		Tiers:                       tiers,
		OriginalCreditReservedMinor: 22_000_000,
		OriginalCashPlannedMinor:    22_000_000,
	})
	if err != nil || reconciledRemorse.TierProtected || reconciledRemorse.AppliedDiscountBps != 500 {
		t.Fatalf("expected 500 bps tier recalculation on BUYER_REMORSE_REFUSAL, got %+v err=%v", reconciledRemorse, err)
	}

	// Guard #2 & #9: Statutory 25,000,000 UZS (2,500,000,000 Tiyin) AML Ceiling across same LegalEntityTIN (Anti-Smurfing).
	if _, err := eng.CheckAndRecordAMLCashExposure(ctx, "TIN-309988776", "sup_1", "2026-10-29", 1_500_000_000); err != nil {
		t.Fatalf("first 15M UZS order should pass: %v", err)
	}
	// Second 12M UZS order on the same day for another store under the same TIN (15M + 12M = 27M > 25M) must fail!
	if _, err := eng.CheckAndRecordAMLCashExposure(ctx, "TIN-309988776", "sup_1", "2026-10-29", 1_200_000_000); !errors.Is(err, foundry.ErrAMLCeilingExceeded) {
		t.Fatalf("expected ErrAMLCeilingExceeded on smurfing attempt across same TIN, got %v", err)
	}

	// Policy Override: When AML_LIMITS_DISABLED=1 (No Limits Policy), uncapped cash/card payments are permitted.
	t.Setenv("AML_LIMITS_DISABLED", "1")
	if _, err := eng.CheckAndRecordAMLCashExposure(ctx, "TIN-309988776", "sup_1", "2026-10-29", 100_000_000_000); err != nil {
		t.Fatalf("uncapped payment should succeed with AML_LIMITS_DISABLED=1, got: %v", err)
	}

	// Guard #3: Pre-Wave Fiscal Readiness (blocks expiring cert < 72h or missing IKPU) & Payout Slice Lock.
	if err := eng.ValidatePreWaveFiscalReadiness(ctx, []foundry.FiscalLineItemCheck{
		{SKU: "COLA-1.5L", IKPUCode: "02202001001000000", PackageCode: "123"},
	}, time.Now().UTC().Add(24*time.Hour)); !errors.Is(err, foundry.ErrFiscalPreFlightBlocked) {
		t.Fatalf("expected ErrFiscalPreFlightBlocked when PKCS#12 cert expires in 24h, got %v", err)
	}
	if err := eng.ValidateSlicePayoutEligibility(ctx, "slice_1", "SIGNED_PROVISIONAL"); !errors.Is(err, foundry.ErrSlicePayoutUnfiscalized) {
		t.Fatalf("expected ErrSlicePayoutUnfiscalized when FiscalState != REGISTERED_SOLIQ, got %v", err)
	}

	// Guard #8: Post-Settlement Clawback Slice backed by Supplier Rolling Reserve.
	eng.SetSupplierReserveEscrow("sup_1", 5_000_000)
	clawback, err := eng.CreatePostSettlementClawbackSlice(ctx, foundry.SettlementSliceRecord{
		SliceID:        "slice_batched_10",
		OrderID:        "ord_500",
		SupplierID:     "sup_1",
		NetPayoutMinor: 40_000_000,
		Status:         "BATCHED",
	}, 2_000_000)
	if err != nil || clawback.NetPayoutMinor != -2_000_000 || clawback.ParentSliceID != "slice_batched_10" {
		t.Fatalf("CreatePostSettlementClawbackSlice failed: %+v err=%v", clawback, err)
	}

	// Guard #6: 3D Volumetric Crushability & Returnable Container Ledger.
	_, err = eng.Validate3DLoadAndRTIReturns(ctx, []foundry.LoadStackLayer{
		{SKU: "CHIPS-FRAGILE", LayerIndex: 0, GrossWeightKg: 4, MaxStackCrushKg: 8, VolumeDm3: 50},
		{SKU: "WATER-HEAVY", LayerIndex: 1, GrossWeightKg: 18, MaxStackCrushKg: 40, VolumeDm3: 30},
	}, 2000, "ret_1", "sup_1", "CRATE-PLASTIC-20L", 10, 5, 25, 500_00)
	if !errors.Is(err, foundry.ErrCrushabilityViolation) {
		t.Fatalf("expected ErrCrushabilityViolation when stacking 18kg water on 8kg max-crush chips, got %v", err)
	}

	// Guard #9: FX Volatility Corridor Cap vs Wave Freeze.
	if _, err := eng.EvaluateFXCorridorBeforeWave(ctx, "FLOAT_WITH_CORRIDOR_CAP", 10_000_00, 12500_000000, 13200_000000, 300, false); !errors.Is(err, foundry.ErrFXReconfirmationRequired) {
		t.Fatalf("expected ErrFXReconfirmationRequired on +560 bps FX jump (> 300 bps cap), got %v", err)
	}
	frozenPrice, err := eng.EvaluateFXCorridorBeforeWave(ctx, "FLOAT_WITH_CORRIDOR_CAP", 10_000_00, 12500_000000, 13200_000000, 300, true)
	if err != nil || frozenPrice != 10_000_00 {
		t.Fatalf("expected hard-frozen price 1000000 once wave is released, got %d err=%v", frozenPrice, err)
	}

	// Guard #10: Abandoned Pick-Wave Tote Heartbeat Takeover (>= 20m stall required).
	eng.RegisterPickToteState(foundry.PickToteState{
		ToteID:              "tote_stalled_1",
		WaveID:              "wave_9",
		ActivePickerID:      "picker_old",
		Status:              "PICKER_IN_HAND",
		CompletedLineCount:  4,
		TotalLineCount:      10,
		PickedLotIDs:        []string{"lot_A", "lot_B"},
		LastScanHeartbeatAt: time.Now().UTC().Add(-25 * time.Minute),
	})
	takenOver, err := eng.TakeoverAbandonedPickTote(ctx, "tote_stalled_1", "picker_relief")
	if err != nil || takenOver.ActivePickerID != "picker_relief" || len(takenOver.PickedLotIDs) != 2 {
		t.Fatalf("TakeoverAbandonedPickTote failed: %+v err=%v", takenOver, err)
	}
}

func TestKineticActionPreconditionsAndConcurrency(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	eng.RegisterPrecondition("AUTHORIZE_CREDIT_OVERRIDE", func(ctx context.Context, req foundry.ActionExecutionRequest) foundry.PreconditionResult {
		role := req.ActorRole
		if role != "PLATFORM_ADMIN" && role != "SUPPLIER_FINANCE_DIRECTOR" {
			return foundry.PreconditionResult{
				RuleName: "RoleAuthorizationGate",
				Passed:   false,
				Reason:   "actor role lacks credit override authority",
			}
		}
		return foundry.PreconditionResult{RuleName: "RoleAuthorizationGate", Passed: true}
	})

	// Unauthorized role must fail closed.
	_, err := eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_1",
		ActionType:     "AUTHORIZE_CREDIT_OVERRIDE",
		ActorID:        "usr_driver_1",
		ActorRole:      "DRIVER",
		IdempotencyKey: "idem_reject_1",
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ErrPreconditionFailed for unauthorized role, got %v", err)
	}

	// Concurrent idempotent calls with the same IdempotencyKey must execute exactly once.
	var wg sync.WaitGroup
	var mu sync.Mutex
	var execIDs []string
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rec, err := eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
				CellID:         "cell-uz",
				SupplierID:     "sup_1",
				ActionType:     "AUTHORIZE_CREDIT_OVERRIDE",
				ActorID:        "usr_admin_1",
				ActorRole:      "PLATFORM_ADMIN",
				IdempotencyKey: "idem_concurrent_override_99",
				Parameters:     map[string]any{"new_limit_minor": int64(500_000_000)},
			}, func(ctx context.Context) (map[string]any, error) {
				return map[string]any{"applied": true}, nil
			})
			if err != nil {
				t.Errorf("concurrent ExecuteKineticAction failed: %v", err)
				return
			}
			mu.Lock()
			execIDs = append(execIDs, rec.ActionExecutionID)
			mu.Unlock()
		}()
	}
	wg.Wait()

	if len(execIDs) != 16 {
		t.Fatalf("expected 16 responses, got %d", len(execIDs))
	}
	for i := 1; i < len(execIDs); i++ {
		if execIDs[i] != execIDs[0] {
			t.Fatalf("idempotency violation: got distinct ActionExecutionIDs %s vs %s", execIDs[0], execIDs[i])
		}
	}
}

func TestDefaultPreExecutionLifecycleGuards(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// 1. Step 5 Wave Release blocked when PKCS#12 cert expires in < 72h.
	_, err := eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_1",
		ActionType:     foundry.ActionReleasePickWave,
		ActorID:        "wms_mgr_1",
		ActorRole:      "WAREHOUSE",
		IdempotencyKey: "wave_rel_fail_cert",
		Parameters: map[string]any{
			"pkcs12_cert_hours_remaining": int64(48),
			"all_skus_ikpu_valid":         true,
		},
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ActionReleasePickWave to fail closed on 48h PKCS#12 cert window, got %v", err)
	}

	// 2. Step 7 Driver Route Start blocked when vehicle lacks Pre-Trip DVIR.
	_, err = eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_1",
		ActionType:     foundry.ActionStartDriverRoute,
		ActorID:        "drv_1",
		ActorRole:      "DRIVER",
		IdempotencyKey: "route_start_no_dvir",
		Parameters: map[string]any{
			"vehicle_id": "veh_uninspected",
		},
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ActionStartDriverRoute to fail closed without DVIR, got %v", err)
	}

	// 3. Step 10 Supplier Payout Batch blocked when FiscalState != REGISTERED_SOLIQ.
	_, err = eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
		CellID:         "cell-uz",
		SupplierID:     "sup_1",
		ActionType:     foundry.ActionBatchSupplierPayout,
		ActorID:        "fin_1",
		ActorRole:      "SUPPLIER",
		IdempotencyKey: "payout_batch_unfiscalized",
		Parameters: map[string]any{
			"fiscal_state": "PENDING_SUBMISSION",
		},
	}, nil)
	if !errors.Is(err, foundry.ErrPreconditionFailed) {
		t.Fatalf("expected ActionBatchSupplierPayout to fail closed on unfiscalized slice, got %v", err)
	}
}

func TestDeduplicationAndMutationCeiling(t *testing.T) {
	ctx := context.Background()

	// 1. Deduplication of OntologyObjects by PK (CellId, ObjectType, ObjectId).
	objs := []*foundry.OntologyObject{
		{CellID: "cell-uz", ObjectType: "WarehouseBin", ObjectID: "bin_1", SupplierID: "sup_1", Version: 1},
		{CellID: "cell-uz", ObjectType: "WarehouseBin", ObjectID: "bin_2", SupplierID: "sup_1", Version: 1},
		{CellID: "cell-uz", ObjectType: "WarehouseBin", ObjectID: "bin_1", SupplierID: "sup_1", Version: 2}, // Duplicate PK
	}
	dedupedObjs := foundry.DeduplicateOntologyObjects(objs)
	if len(dedupedObjs) != 2 {
		t.Fatalf("expected 2 deduped objects, got %d", len(dedupedObjs))
	}
	if dedupedObjs[0].ObjectID != "bin_1" || dedupedObjs[0].Version != 2 {
		t.Fatalf("expected last write to win for bin_1 with version 2, got %+v", dedupedObjs[0])
	}
	if dedupedObjs[1].ObjectID != "bin_2" {
		t.Fatalf("expected bin_2 preserved at index 1, got %+v", dedupedObjs[1])
	}

	// 2. Deduplication of ScenarioObjectPatches by PK (ScenarioId, ObjectType, ObjectId).
	patches := []*foundry.ScenarioObjectPatch{
		{ScenarioID: "scen_1", ObjectType: "Order", ObjectID: "ord_1", Operation: "UPSERT", BaseVersion: 1},
		{ScenarioID: "scen_1", ObjectType: "Order", ObjectID: "ord_2", Operation: "UPSERT", BaseVersion: 1},
		{ScenarioID: "scen_1", ObjectType: "Order", ObjectID: "ord_1", Operation: "DELETE", BaseVersion: 2}, // Duplicate PK
	}
	dedupedPatches := foundry.DeduplicateScenarioPatches(patches)
	if len(dedupedPatches) != 2 {
		t.Fatalf("expected 2 deduped patches, got %d", len(dedupedPatches))
	}
	if dedupedPatches[0].Operation != "DELETE" || dedupedPatches[0].BaseVersion != 2 {
		t.Fatalf("expected last patch to win with DELETE, got %+v", dedupedPatches[0])
	}

	// 3. Mutation ceiling (> 1000 items fails fast with ErrTransactionMutationLimitExceeded).
	repo := foundry.NewSpannerRepository(nil)
	overflowObjs := make([]*foundry.OntologyObject, 1001)
	for i := 0; i < 1001; i++ {
		overflowObjs[i] = &foundry.OntologyObject{
			CellID:     "cell-uz",
			ObjectType: "Item",
			ObjectID:   fmt.Sprintf("item_%d", i),
			SupplierID: "sup_1",
		}
	}
	err := repo.PersistOntologyObjects(ctx, overflowObjs)
	if !errors.Is(err, foundry.ErrTransactionMutationLimitExceeded) {
		t.Fatalf("expected ErrTransactionMutationLimitExceeded for > 1000 objects, got %v", err)
	}

	overflowPatches := make([]*foundry.ScenarioObjectPatch, 1001)
	for i := 0; i < 1001; i++ {
		overflowPatches[i] = &foundry.ScenarioObjectPatch{
			ScenarioID: "scen_overflow",
			ObjectType: "Item",
			ObjectID:   fmt.Sprintf("item_%d", i),
			SupplierID: "sup_1",
			Operation:  "UPSERT",
		}
	}
	err = repo.PersistScenarioObjectPatches(ctx, overflowPatches)
	if !errors.Is(err, foundry.ErrTransactionMutationLimitExceeded) {
		t.Fatalf("expected ErrTransactionMutationLimitExceeded for > 1000 patches, got %v", err)
	}
}

func TestRingBufferBounds(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// 1. ActionHistory ring buffer: record 1050 actions, verify capped at 1000.
	for i := 0; i < 1050; i++ {
		_, _ = eng.ExecuteKineticAction(ctx, foundry.ActionExecutionRequest{
			CellID:         "cell-uz",
			SupplierID:     "sup_1",
			ActionType:     "TEST_ACTION",
			ActorID:        "actor_1",
			ActorRole:      "SYSTEM",
			IdempotencyKey: fmt.Sprintf("action_key_%d", i),
		}, nil)
	}
	if count := eng.ActionHistoryCount(); count > 1000 {
		t.Fatalf("expected ActionHistoryCount <= 1000, got %d", count)
	}

	// 2. GateEvents ring buffer: record 1050 events, verify capped at 1000.
	for i := 0; i < 1050; i++ {
		_, _ = eng.RecordGateCheckEvent(ctx, foundry.YardGateCheckEvent{
			DepotID:        "depot_1",
			SupplierID:     "sup_1",
			VehicleID:      fmt.Sprintf("veh_%d", i),
			LicensePlate:   fmt.Sprintf("01A%03dAA", i%1000),
			Direction:      "GATE_IN",
			VehiclePurpose: "INBOUND_FTL",
		})
	}
	if count := eng.GateEventsCount(); count > 1000 {
		t.Fatalf("expected GateEventsCount <= 1000, got %d", count)
	}

	// 3. VaultLedger ring buffer: record 1050 deposits, verify capped at 1000.
	for i := 0; i < 1050; i++ {
		sess, _ := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
			CellID:            "cell-uz",
			SupplierID:        "sup_1",
			DepotID:           "depot_ring",
			DriverID:          fmt.Sprintf("drv_ring_%d", i),
			RouteID:           "rt_1",
			BagSealBarcode:    fmt.Sprintf("SEAL_RING_%d", i),
			ExpectedCashMinor: 10_000_000,
		})
		_, _ = eng.SubmitBlindDenominationCount(ctx, sess.SessionID, map[int64]int64{100000: 1}, 0)
	}
	if count := eng.VaultLedgerCount("depot_ring"); count > 1000 {
		t.Fatalf("expected VaultLedgerCount <= 1000, got %d", count)
	}

	// 4. DriverShiftVariances ring buffer: record 8 shifts for one driver, verify capped at 5.
	for i := 0; i < 8; i++ {
		s, _ := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
			CellID:            "cell-uz",
			SupplierID:        "sup_1",
			DepotID:           "depot_drv",
			DriverID:          "drv_variance_bounded",
			RouteID:           "rt_1",
			BagSealBarcode:    fmt.Sprintf("SEAL_DRV_%d", i),
			ExpectedCashMinor: 10_000_000,
		})
		_, _ = eng.SubmitBlindDenominationCount(ctx, s.SessionID, map[int64]int64{100000: 1}, 500_000)
	}
	if count := eng.DriverShiftVarianceCount("drv_variance_bounded"); count > 5 {
		t.Fatalf("expected DriverShiftVarianceCount <= 5, got %d", count)
	}
}

func TestOfflineQuotaLeaseStatusAndExpirationValidation(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// 1. Issue an active lease for 2 hours.
	lease, err := eng.IssueOfflineQuotaLease(
		ctx,
		"ord_lease_status_test",
		"drv_test",
		"sup_1",
		"TIN-999000",
		"2026-10-31",
		100_000_000, // 1M UZS max cash
		50_000_000,  // 500k UZS max credit
		2*time.Hour,
	)
	if err != nil {
		t.Fatalf("IssueOfflineQuotaLease: %v", err)
	}
	if lease.Status != "ACTIVE" {
		t.Fatalf("expected ACTIVE status, got %s", lease.Status)
	}

	// 2. Validate collection within bounds -> PASS.
	if err := eng.ValidateOfflineLeaseCollection(ctx, lease.LeaseID, 80_000_000, 30_000_000); err != nil {
		t.Fatalf("ValidateOfflineLeaseCollection should pass within bounds: %v", err)
	}

	// 3. Validate collection exceeding cash limit -> FAIL.
	err = eng.ValidateOfflineLeaseCollection(ctx, lease.LeaseID, 120_000_000, 10_000_000)
	if !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded on excess cash, got %v", err)
	}

	// 4. Validate collection exceeding credit limit -> FAIL.
	err = eng.ValidateOfflineLeaseCollection(ctx, lease.LeaseID, 50_000_000, 70_000_000)
	if !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded on excess credit, got %v", err)
	}

	// 5. Test expired lease -> FAIL.
	expiredLease, err := eng.IssueOfflineQuotaLease(
		ctx,
		"ord_expired_test",
		"drv_test_exp",
		"sup_1",
		"TIN-999001",
		"2026-10-31",
		10_000_000,
		10_000_000,
		1*time.Hour,
	)
	if err != nil {
		t.Fatalf("IssueOfflineQuotaLease expired: %v", err)
	}
	expiredLease.ExpiresAt = time.Now().Add(-10 * time.Minute)
	err = eng.ValidateOfflineLeaseCollection(ctx, expiredLease.LeaseID, 5_000_000, 5_000_000)
	if !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded on expired lease, got %v", err)
	}

	// 6. Test non-ACTIVE lease status -> FAIL.
	expiredLease.ExpiresAt = time.Now().Add(1 * time.Hour)
	expiredLease.Status = "REVOKED"
	err = eng.ValidateOfflineLeaseCollection(ctx, expiredLease.LeaseID, 5_000_000, 5_000_000)
	if !errors.Is(err, foundry.ErrOfflineLeaseExceeded) {
		t.Fatalf("expected ErrOfflineLeaseExceeded on non-ACTIVE status, got %v", err)
	}
}

func TestStructuredDomainErrorsUA1UA2UA8UA11(t *testing.T) {
	ctx := context.Background()
	eng := foundry.NewEngine()

	// UA-1: Cashier Recount Ratchet structured error
	sess, err := eng.StartBlindCountSession(ctx, foundry.StartCashierSessionRequest{
		CellID:            "cell-uz",
		SupplierID:        "sup_1",
		DepotID:           "depot_ua1",
		DriverID:          "drv_ua1",
		RouteID:           "rt_ua1",
		BagSealBarcode:    "BAG_UA1",
		ExpectedCashMinor: 50_000_000,
	})
	if err != nil {
		t.Fatalf("StartBlindCountSession: %v", err)
	}
	// Submit 10,000,000 with 0 tolerance -> Attempt #1 fails
	_, err = eng.SubmitBlindDenominationCount(ctx, sess.SessionID, map[int64]int64{100000: 1}, 0)
	if err == nil {
		t.Fatalf("expected error on Attempt #1 variance")
	}
	var recountErr *foundry.RecountRequiredError
	if !errors.As(err, &recountErr) {
		t.Fatalf("expected *foundry.RecountRequiredError, got %T", err)
	}
	if !errors.Is(err, foundry.ErrBlindRecountRequired) {
		t.Fatalf("expected error to unwrap to ErrBlindRecountRequired")
	}
	if recountErr.Session == nil || recountErr.Session.ExpectedCashMinor != nil || recountErr.Session.VarianceMinor != nil {
		t.Fatalf("expected redacted ExpectedCashMinor and VarianceMinor on recount session view, got %+v", recountErr.Session)
	}
	if recountErr.Session.ErrorCode != "RECOUNT_REQUIRED" {
		t.Fatalf("expected ErrorCode RECOUNT_REQUIRED, got %s", recountErr.Session.ErrorCode)
	}

	// UA-2: Statutory AML Cash Ceiling structured error
	_, err = eng.CheckAndRecordAMLCashExposure(ctx, "TIN-UA2", "sup_1", "2026-10-31", 2_000_000_000)
	if err != nil {
		t.Fatalf("CheckAndRecordAMLCashExposure: %v", err)
	}
	_, err = eng.CheckAndRecordAMLCashExposure(ctx, "TIN-UA2", "sup_1", "2026-10-31", 600_000_000)
	if err == nil {
		t.Fatalf("expected error exceeding 25M UZS AML ceiling")
	}
	var amlErr *foundry.AMLDailyCashLimitExceededError
	if !errors.As(err, &amlErr) {
		t.Fatalf("expected *foundry.AMLDailyCashLimitExceededError, got %T", err)
	}
	if !errors.Is(err, foundry.ErrAMLCeilingExceeded) {
		t.Fatalf("expected error to unwrap to ErrAMLCeilingExceeded")
	}
	if amlErr.LegalEntityTIN != "TIN-UA2" || amlErr.CurrentExposureMinor != 2_000_000_000 || amlErr.CeilingMinor != foundry.StatutoryAMLCashCeilingMinor {
		t.Fatalf("unexpected AML error details: %+v", amlErr)
	}
	if amlErr.Advice == "" {
		t.Fatalf("expected bank transfer advice in AML error")
	}

	// UA-8: Fleet Red-Tag Gate Rejection structured error
	_, err = eng.SubmitDVIRInspection(ctx, foundry.FleetDVIRInspection{
		CellID:            "cell-uz",
		SupplierID:        "sup_1",
		VehicleID:         "veh_ua8_defect",
		DriverID:          "drv_ua8",
		RequiresColdChain: false,
		OdometerKm:        10000,
		Checklist: foundry.DVIRChecklist8Point{
			SteeringAndHornOK:             true,
			ServiceAndParkingBrakesOK:     false, // Defect!
			TiresTreadAndPressureOK:       true,
			LightsAndHazardsOK:            true,
			WipersAndFluidsOK:             true,
			EmergencyKitAndExtinguisherOK: true,
			CargoBoxSanitationOK:          true,
			ReeferTempCalibratedOK:        true,
		},
	})
	if err != nil {
		t.Fatalf("SubmitDVIRInspection: %v", err)
	}
	err = eng.AssertVehicleDispatchAuthorized(ctx, "veh_ua8_defect")
	if err == nil {
		t.Fatalf("expected error on AssertVehicleDispatchAuthorized for red-tagged vehicle")
	}
	var redTagErr *foundry.VehicleGroundedRedTagError
	if !errors.As(err, &redTagErr) {
		t.Fatalf("expected *foundry.VehicleGroundedRedTagError, got %T", err)
	}
	if !errors.Is(err, foundry.ErrVehicleRedTagged) {
		t.Fatalf("expected error to unwrap to ErrVehicleRedTagged")
	}
	if redTagErr.VehicleID != "veh_ua8_defect" || len(redTagErr.CriticalDefects) == 0 {
		t.Fatalf("unexpected VehicleGroundedRedTagError: %+v", redTagErr)
	}

	// UA-11: Tote Takeover Rejection structured error
	eng.RegisterPickToteState(foundry.PickToteState{
		ToteID:              "tote_ua11_active",
		WaveID:              "wave_ua11",
		ActivePickerID:      "picker_active",
		Status:              "PICKER_IN_HAND",
		CompletedLineCount:  2,
		TotalLineCount:      5,
		LastScanHeartbeatAt: time.Now().UTC().Add(-5 * time.Minute), // Only 5m idle (< 20m required)
	})
	_, err = eng.TakeoverAbandonedPickTote(ctx, "tote_ua11_active", "picker_relief")
	if err == nil {
		t.Fatalf("expected error for active tote takeover")
	}
	var toteErr *foundry.ToteTakeoverNotPermittedError
	if !errors.As(err, &toteErr) {
		t.Fatalf("expected *foundry.ToteTakeoverNotPermittedError, got %T", err)
	}
	if !errors.Is(err, foundry.ErrToteHeartbeatActive) {
		t.Fatalf("expected error to unwrap to ErrToteHeartbeatActive")
	}
	if toteErr.ToteID != "tote_ua11_active" || toteErr.CurrentPickerID != "picker_active" || toteErr.RemainingIdleSeconds <= 0 {
		t.Fatalf("unexpected ToteTakeoverNotPermittedError: %+v", toteErr)
	}
}


