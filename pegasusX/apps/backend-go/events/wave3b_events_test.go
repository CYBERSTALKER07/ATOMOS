package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWave3BIntelProposalsRoiEventsContract(t *testing.T) {
	wantEvents := map[string]string{
		"EventForecastRunCompleted":       EventForecastRunCompleted,
		"EventCreditScoreComputed":        EventCreditScoreComputed,
		"EventCreditLimitChanged":         EventCreditLimitChanged,
		"EventCreditHoldApplied":          EventCreditHoldApplied,
		"EventCreditHoldReleased":         EventCreditHoldReleased,
		"EventProposalCreated":            EventProposalCreated,
		"EventProposalDecided":            EventProposalDecided,
		"EventProposalExecuted":           EventProposalExecuted,
		"EventInventoryRebalanceProposed": EventInventoryRebalanceProposed,
		"EventRoiEventRecorded":           EventRoiEventRecorded,
	}
	expectedValues := map[string]string{
		"EventForecastRunCompleted":       "forecast.run.completed",
		"EventCreditScoreComputed":        "credit.score.computed",
		"EventCreditLimitChanged":         "credit.limit.changed",
		"EventCreditHoldApplied":          "credit.hold.applied",
		"EventCreditHoldReleased":         "credit.hold.released",
		"EventProposalCreated":            "proposal.created",
		"EventProposalDecided":            "proposal.decided",
		"EventProposalExecuted":           "proposal.executed",
		"EventInventoryRebalanceProposed": "inventory.rebalance.proposed",
		"EventRoiEventRecorded":           "roi.event.recorded",
	}
	for name, got := range wantEvents {
		if got != expectedValues[name] {
			t.Fatalf("%s = %q, want %q", name, got, expectedValues[name])
		}
	}

	if AggregateForecastRun != "ForecastRun" {
		t.Fatalf("AggregateForecastRun = %q, want ForecastRun", AggregateForecastRun)
	}
	if AggregateCreditScore != "CreditScore" {
		t.Fatalf("AggregateCreditScore = %q, want CreditScore", AggregateCreditScore)
	}
	if AggregateProposal != "Proposal" {
		t.Fatalf("AggregateProposal = %q, want Proposal", AggregateProposal)
	}
	if AggregateRoiEvent != "RoiEvent" {
		t.Fatalf("AggregateRoiEvent = %q, want RoiEvent", AggregateRoiEvent)
	}

	fcEv := ForecastRunCompletedEvent{
		BaseEvent: BaseEvent{
			Type:      EventForecastRunCompleted,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		EntityID:           "sup-1",
		RunID:              "fc-run-1",
		SupplierID:         "sup-1",
		LocationID:         "wh-1",
		HorizonDays:        14,
		Status:             "COMPLETED",
		ChampionModel:      "holt_winters",
		ChallengerModel:    "global_xreg_challenger",
		WinnerModel:        "global_xreg_challenger",
		ChampionWapeBps:    1850,
		ChallengerWapeBps:  1420,
		WapeImprovementBps: 430,
		SkuCount:           25,
		LocationCount:      3,
		TriggeredBy:        "usr-planner",
		CompletedAt:        "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "ForecastRunCompletedEvent", fcEv, map[string]any{
		"type":                 EventForecastRunCompleted,
		"run_id":               "fc-run-1",
		"winner_model":         "global_xreg_challenger",
		"champion_wape_bps":    json.Number("1850"),
		"challenger_wape_bps":  json.Number("1420"),
		"wape_improvement_bps": json.Number("430"),
		"sku_count":            json.Number("25"),
	})

	csEv := CreditScoreComputedEvent{
		BaseEvent: BaseEvent{
			Type:      EventCreditScoreComputed,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		ScoreID:             "cs-1",
		RetailerID:          "ret-1",
		SupplierID:          "sup-1",
		Scope:               "RELATIONSHIP",
		ConsentVerified:     true,
		ModelVersion:        "ml-pd-v2",
		ScorerSource:        "ML_SCORER",
		Score:               82,
		PdBps:               310,
		RiskTier:            "LOW",
		SuggestedLimitMinor: 150000000,
		ReasonCodesJSON:     `["ON_TIME_PAYMENT_90D"]`,
		FeaturesHash:        strings.Repeat("a", 64),
		ComputedAt:          "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "CreditScoreComputedEvent", csEv, map[string]any{
		"type":                  EventCreditScoreComputed,
		"scope":                 "RELATIONSHIP",
		"consent_verified":      true,
		"score":                 json.Number("82"),
		"pd_bps":                json.Number("310"),
		"suggested_limit_minor": json.Number("150000000"),
	})

	clEv := CreditLimitChangedEvent{
		BaseEvent: BaseEvent{
			Type:      EventCreditLimitChanged,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		RetailerID:         "ret-1",
		SupplierID:         "sup-1",
		PreviousLimitMinor: 100000000,
		NewLimitMinor:      150000000,
		ApprovalID:         "apr-1",
		ActorID:            "usr-fin",
		Reason:             "Strong repayment velocity",
		OccurredAt:         "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "CreditLimitChangedEvent", clEv, map[string]any{
		"type":                 EventCreditLimitChanged,
		"previous_limit_minor": json.Number("100000000"),
		"new_limit_minor":      json.Number("150000000"),
		"approval_id":          "apr-1",
	})

	chEv := CreditHoldEvent{
		BaseEvent: BaseEvent{
			Type:      EventCreditHoldApplied,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		RetailerID:       "ret-2",
		SupplierID:       "sup-1",
		Action:           "APPLIED",
		Status:           "FROZEN",
		PrepayOnlyLocked: true,
		Score:            19,
		RiskTier:         "BLOCK",
		Reason:           "30d+ overdue AR invoice",
		ActorID:          "system",
		OccurredAt:       "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "CreditHoldEvent", chEv, map[string]any{
		"type":               EventCreditHoldApplied,
		"action":             "APPLIED",
		"status":             "FROZEN",
		"prepay_only_locked": true,
		"score":              json.Number("19"),
	})

	propEv := ProposalEvent{
		BaseEvent: BaseEvent{
			Type:      EventProposalExecuted,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		TenantID:            "sup-1",
		ProposalID:          "prop-1",
		AgentID:             "COLLECTIONS_AGENT",
		ProposalType:        "collections_nudge",
		TargetObjectType:    "ArInvoice",
		TargetObjectID:      "inv-1",
		Status:              "EXECUTED",
		PayloadHash:         strings.Repeat("b", 64),
		ApprovalRequestID:   "apr-2",
		ActorID:             "usr-fin",
		ActorType:           "human",
		RoiEventID:          "roi-1",
		ExpectedImpactMinor: 45000000,
		Currency:            "UZS",
		ConfidenceBps:       8400,
		OccurredAt:          "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "ProposalEvent", propEv, map[string]any{
		"type":                  EventProposalExecuted,
		"agent_id":              "COLLECTIONS_AGENT",
		"expected_impact_minor": json.Number("45000000"),
		"confidence_bps":        json.Number("8400"),
		"roi_event_id":          "roi-1",
	})

	rebEv := InventoryRebalanceProposedEvent{
		BaseEvent: BaseEvent{
			Type:      EventInventoryRebalanceProposed,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		TenantID:          "sup-1",
		ProposalID:        "prop-2",
		SourceWarehouseID: "wh-1",
		TargetWarehouseID: "wh-2",
		ProductID:         "sku-1",
		LotID:             "lot-1",
		LayerID:           "lay-1",
		Qty:               120,
		ValueAtRiskMinor:  24000000,
		Currency:          "UZS",
		ExpiryDate:        "2026-11-24",
		ConfidenceBps:     7900,
		OccurredAt:        "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "InventoryRebalanceProposedEvent", rebEv, map[string]any{
		"type":                EventInventoryRebalanceProposed,
		"source_warehouse_id": "wh-1",
		"qty":                 json.Number("120"),
		"value_at_risk_minor": json.Number("24000000"),
		"confidence_bps":      json.Number("7900"),
	})

	roiEv := RoiEventRecordedEvent{
		BaseEvent: BaseEvent{
			Type:      EventRoiEventRecorded,
			Version:   1,
			Timestamp: "2026-11-17T04:15:00Z",
		},
		TenantID:       "sup-1",
		EventID:        "roi-1",
		Category:       "EXPIRY_SAVED",
		SavingsMinor:   24000000,
		Currency:       "UZS",
		Methodology:    "Moved near-expiry FEFO stock prior to expiry cutoff",
		ConfidenceBps:  8500,
		IsEstimate:     false,
		HoldoutControl: false,
		ProposalID:     "prop-2",
		EvidenceRefID:  "lot-1",
		OccurredAt:     "2026-11-17T04:15:00Z",
	}
	assertJSONIntegerFields(t, "RoiEventRecordedEvent", roiEv, map[string]any{
		"type":            EventRoiEventRecorded,
		"category":        "EXPIRY_SAVED",
		"savings_minor":   json.Number("24000000"),
		"confidence_bps":  json.Number("8500"),
		"is_estimate":     false,
		"holdout_control": false,
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
