package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWave4BAnomalyWhatIfHaccpL2ScimEventsContract(t *testing.T) {
	wantEvents := map[string]string{
		"EventAnomalyDetected":        EventAnomalyDetected,
		"EventAnomalyResolved":        EventAnomalyResolved,
		"EventWhatIfSimulated":        EventWhatIfSimulated,
		"EventHaccpExcursionReported": EventHaccpExcursionReported,
		"EventProposalL2AutoExecuted": EventProposalL2AutoExecuted,
		"EventScimIdentitySynced":     EventScimIdentitySynced,
	}
	expectedValues := map[string]string{
		"EventAnomalyDetected":        "anomaly.case.detected",
		"EventAnomalyResolved":        "anomaly.case.resolved",
		"EventWhatIfSimulated":        "whatif.simulation.completed",
		"EventHaccpExcursionReported": "coldchain.haccp.excursion",
		"EventProposalL2AutoExecuted": "proposal.l2.auto_executed",
		"EventScimIdentitySynced":     "iam.scim.synced",
	}
	for name, got := range wantEvents {
		if got != expectedValues[name] {
			t.Fatalf("%s = %q, want %q", name, got, expectedValues[name])
		}
	}

	if AggregateAnomalyCase != "AnomalyCase" {
		t.Fatalf("AggregateAnomalyCase = %q, want AnomalyCase", AggregateAnomalyCase)
	}
	if AggregateWhatIfSimulation != "WhatIfSimulation" {
		t.Fatalf("AggregateWhatIfSimulation = %q, want WhatIfSimulation", AggregateWhatIfSimulation)
	}
	if AggregateCustomRole != "CustomRole" {
		t.Fatalf("AggregateCustomRole = %q, want CustomRole", AggregateCustomRole)
	}
	if AggregateScimIdentity != "ScimIdentity" {
		t.Fatalf("AggregateScimIdentity = %q, want ScimIdentity", AggregateScimIdentity)
	}

	anomalyEv := AnomalyCaseEvent{
		BaseEvent: BaseEvent{
			Type:      EventAnomalyDetected,
			Version:   1,
			Timestamp: "2026-11-17T18:00:00Z",
		},
		TenantID:       "sup-1",
		CaseID:         "anom-1",
		RuleCode:       "CASH_GEOFENCE_MISMATCH",
		Category:       "CASH",
		Severity:       "HIGH",
		Status:         "OPEN",
		ObjectType:     "CashierBlindCountSession",
		ObjectID:       "sess-1",
		ExposureMinor:  4500000,
		ScoreBps:       8400,
		PlaybookAction: "ESCALATE_AUDIT",
		DetectedAt:     "2026-11-17T18:00:00Z",
		OccurredAt:     "2026-11-17T18:00:00Z",
	}
	assertJSONIntegerFields(t, "AnomalyCaseEvent", anomalyEv, map[string]any{
		"type":           EventAnomalyDetected,
		"case_id":        "anom-1",
		"exposure_minor": json.Number("4500000"),
		"score_bps":      json.Number("8400"),
	})

	whatIfEv := WhatIfSimulatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventWhatIfSimulated,
			Version:   1,
			Timestamp: "2026-11-17T18:00:05Z",
		},
		TenantID:                 "sup-1",
		SimulationID:             "sim-1",
		ScenarioID:               "scen-1",
		Name:                     "FX +10% & Demand +15%",
		EntityID:                 "ent-hq",
		HorizonDays:              30,
		ForecastDeltaMinor:       15000000,
		ReplenishmentCostMinor:   6200000,
		DispatchCostDeltaMinor:   950000,
		CreditExposureDeltaMinor: 2100000,
		NetPnLDeltaMinor:         5750000,
		Status:                   "COMPLETED",
		CreatedBy:                "usr-planner",
		OccurredAt:               "2026-11-17T18:00:05Z",
	}
	assertJSONIntegerFields(t, "WhatIfSimulatedEvent", whatIfEv, map[string]any{
		"type":                        EventWhatIfSimulated,
		"simulation_id":               "sim-1",
		"horizon_days":                json.Number("30"),
		"forecast_delta_minor":        json.Number("15000000"),
		"replenishment_cost_minor":    json.Number("6200000"),
		"dispatch_cost_delta_minor":   json.Number("950000"),
		"credit_exposure_delta_minor": json.Number("2100000"),
		"net_pnl_delta_minor":         json.Number("5750000"),
	})

	haccpEv := HaccpExcursionReportedEvent{
		BaseEvent: BaseEvent{
			Type:      EventHaccpExcursionReported,
			Version:   1,
			Timestamp: "2026-11-17T18:00:10Z",
		},
		TenantID:          "sup-1",
		ManifestID:        "man-1",
		ReadingID:         "read-1",
		SensorID:          "sens-cold-1",
		TempMilliC:        11400,
		MinMilliC:         0,
		MaxMilliC:         8000,
		TotalReadings:     120,
		ExcursionCount:    3,
		ComplianceRateBps: 9750,
		OpenBreaches:      2,
		AffectedLotIDs:    []string{"lot-1", "lot-2"},
		Status:            "EXCURSION_OPEN",
		ActorID:           "usr-qa",
		OccurredAt:        "2026-11-17T18:00:10Z",
	}
	assertJSONIntegerFields(t, "HaccpExcursionReportedEvent", haccpEv, map[string]any{
		"type":                EventHaccpExcursionReported,
		"temp_milli_c":        json.Number("11400"),
		"max_milli_c":         json.Number("8000"),
		"total_readings":      json.Number("120"),
		"excursion_count":     json.Number("3"),
		"compliance_rate_bps": json.Number("9750"),
		"open_breaches":       json.Number("2"),
	})

	l2Ev := ProposalL2AutoExecutedEvent{
		BaseEvent: BaseEvent{
			Type:      EventProposalL2AutoExecuted,
			Version:   1,
			Timestamp: "2026-11-17T18:00:15Z",
		},
		TenantID:             "sup-1",
		ProposalID:           "prop-l2-1",
		AgentID:              "REORDER_AGENT",
		ProposalType:         "REORDER",
		TargetObjectType:     "WarehouseSku",
		TargetObjectID:       "wh-1:SKU-1",
		AutonomyLevel:        "L2",
		Status:               "EXECUTED",
		PayloadHash:          strings.Repeat("b", 64),
		EstimatedImpactMinor: 1200000,
		Currency:             "UZS",
		ConfidenceBps:        9200,
		SoakDays:             21,
		SoakExecutedCount:    18,
		RollbackRateBps:      150,
		RoiEventID:           "roi-1",
		ActorID:              "usr-ops",
		OccurredAt:           "2026-11-17T18:00:15Z",
	}
	assertJSONIntegerFields(t, "ProposalL2AutoExecutedEvent", l2Ev, map[string]any{
		"type":                   EventProposalL2AutoExecuted,
		"proposal_id":            "prop-l2-1",
		"autonomy_level":         "L2",
		"estimated_impact_minor": json.Number("1200000"),
		"confidence_bps":         json.Number("9200"),
		"soak_days":              json.Number("21"),
		"soak_executed_count":    json.Number("18"),
		"rollback_rate_bps":      json.Number("150"),
	})

	scimEv := ScimIdentitySyncedEvent{
		BaseEvent: BaseEvent{
			Type:      EventScimIdentitySynced,
			Version:   1,
			Timestamp: "2026-11-17T18:00:20Z",
		},
		TenantID:     "sup-1",
		ResourceType: "User",
		ResourceID:   "scim-usr-1",
		ExternalID:   "okta-001",
		UserName:     "dilnoza@acme.uz",
		DisplayName:  "Dilnoza Karimova",
		Email:        "dilnoza@acme.uz",
		Active:       true,
		Action:       "PROVISIONED",
		RolesSynced:  []string{"WAREHOUSE_ADMIN"},
		ActorID:      "scim-idp",
		TraceID:      "trace-scim-1",
		OccurredAt:   "2026-11-17T18:00:20Z",
	}
	assertJSONIntegerFields(t, "ScimIdentitySyncedEvent", scimEv, map[string]any{
		"type":          EventScimIdentitySynced,
		"resource_type": "User",
		"resource_id":   "scim-usr-1",
		"action":        "PROVISIONED",
	})

	envelopePath := filepath.Clean(filepath.Join("..", "..", "..", "packages", "types", "src", "envelope.ts"))
	envelopeSrc, err := os.ReadFile(envelopePath)
	if err != nil {
		t.Fatalf("read %s: %v", envelopePath, err)
	}
	for _, evVal := range expectedValues {
		if !strings.Contains(string(envelopeSrc), `"`+evVal+`"`) {
			t.Fatalf("%s missing %q in EventType union", envelopePath, evVal)
		}
	}
}
