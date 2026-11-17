package events

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWave3CConnectorsAuditorPackEventsContract(t *testing.T) {
	wantEvents := map[string]string{
		"EventConnectorSyncCompleted":  EventConnectorSyncCompleted,
		"EventConnectorSyncFailed":     EventConnectorSyncFailed,
		"EventConnectorShadowMismatch": EventConnectorShadowMismatch,
		"EventExportPackCreated":       EventExportPackCreated,
		"EventAuditorAccessGranted":    EventAuditorAccessGranted,
	}
	expectedValues := map[string]string{
		"EventConnectorSyncCompleted":  "connector.sync.completed",
		"EventConnectorSyncFailed":     "connector.sync.failed",
		"EventConnectorShadowMismatch": "connector.shadow.mismatch",
		"EventExportPackCreated":       "export.pack.created",
		"EventAuditorAccessGranted":    "iam.auditor_access.granted",
	}
	for name, got := range wantEvents {
		if got != expectedValues[name] {
			t.Fatalf("%s = %q, want %q", name, got, expectedValues[name])
		}
	}

	if AggregateExternalCredential != "ExternalCredential" {
		t.Fatalf("AggregateExternalCredential = %q, want ExternalCredential", AggregateExternalCredential)
	}
	if AggregateConnectorSyncRun != "ConnectorSyncRun" {
		t.Fatalf("AggregateConnectorSyncRun = %q, want ConnectorSyncRun", AggregateConnectorSyncRun)
	}
	if AggregateExportPack != "ExportPack" {
		t.Fatalf("AggregateExportPack = %q, want ExportPack", AggregateExportPack)
	}

	syncEv := ConnectorSyncEvent{
		BaseEvent: BaseEvent{
			Type:      EventConnectorSyncCompleted,
			Version:   1,
			Timestamp: "2026-11-17T05:20:00Z",
		},
		TenantID:             "sup-1",
		RunID:                "run-1",
		Provider:             "billz",
		AccountLabel:         "default",
		Direction:            "PULL",
		StreamName:           "products",
		Status:               "COMPLETED",
		RecordsProcessed:     142,
		MismatchCount:        0,
		MissingExternalCount: 0,
		MissingInternalCount: 0,
		StockDeltaCount:      0,
		BalanceDeltaCount:    0,
		TriggeredBy:          "usr-admin",
		StartedAt:            "2026-11-17T05:19:55Z",
		CompletedAt:          "2026-11-17T05:20:00Z",
		OccurredAt:           "2026-11-17T05:20:00Z",
	}
	assertJSONIntegerFields(t, "ConnectorSyncEvent", syncEv, map[string]any{
		"type":              EventConnectorSyncCompleted,
		"run_id":            "run-1",
		"provider":          "billz",
		"records_processed": json.Number("142"),
		"mismatch_count":    json.Number("0"),
	})

	mismatchEv := ConnectorShadowMismatchEvent{
		BaseEvent: BaseEvent{
			Type:      EventConnectorShadowMismatch,
			Version:   1,
			Timestamp: "2026-11-17T05:20:05Z",
		},
		TenantID:             "sup-1",
		RunID:                "run-2",
		Provider:             "billz",
		AccountLabel:         "tashkent-main",
		Direction:            "SHADOW_COMPARE",
		StreamName:           "all",
		Status:               "MISMATCH_DETECTED",
		ObjectType:           "Stock",
		RecordsProcessed:     250,
		MismatchCount:        4,
		MissingExternalCount: 1,
		MissingInternalCount: 1,
		StockDeltaCount:      1,
		BalanceDeltaCount:    1,
		BalanceDeltaMinor:    150000,
		OccurredAt:           "2026-11-17T05:20:05Z",
	}
	assertJSONIntegerFields(t, "ConnectorShadowMismatchEvent", mismatchEv, map[string]any{
		"type":                   EventConnectorShadowMismatch,
		"run_id":                 "run-2",
		"mismatch_count":         json.Number("4"),
		"missing_external_count": json.Number("1"),
		"missing_internal_count": json.Number("1"),
		"stock_delta_count":      json.Number("1"),
		"balance_delta_count":    json.Number("1"),
		"balance_delta_minor":    json.Number("150000"),
	})

	packEv := ExportPackCreatedEvent{
		BaseEvent: BaseEvent{
			Type:      EventExportPackCreated,
			Version:   1,
			Timestamp: "2026-11-17T05:20:10Z",
		},
		TenantID:   "sup-1",
		PackID:     "pack-1",
		PackType:   "AUDIT",
		EntityID:   "sup-1",
		PeriodID:   "2026-11",
		FromDate:   "2026-11-01T00:00:00Z",
		ToDate:     "2026-11-17T00:00:00Z",
		Format:     "zip",
		Status:     "READY",
		RowCount:   12400,
		Sha256Hex:  strings.Repeat("a", 64),
		CreatedBy:  "usr-fin",
		ExpiresAt:  "2026-11-24T05:20:10Z",
		CreatedAt:  "2026-11-17T05:20:10Z",
		OccurredAt: "2026-11-17T05:20:10Z",
	}
	assertJSONIntegerFields(t, "ExportPackCreatedEvent", packEv, map[string]any{
		"type":       EventExportPackCreated,
		"pack_id":    "pack-1",
		"pack_type":  "AUDIT",
		"format":     "zip",
		"row_count":  json.Number("12400"),
		"sha256_hex": strings.Repeat("a", 64),
	})

	auditorEv := AuditorAccessGrantedEvent{
		BaseEvent: BaseEvent{
			Type:      EventAuditorAccessGranted,
			Version:   1,
			Timestamp: "2026-11-17T05:20:15Z",
		},
		TenantID:   "sup-1",
		UserID:     "usr-ext-auditor-1",
		RoleName:   "AUDITOR",
		Action:     "GRANTED",
		GrantedBy:  "usr-owner",
		ExpiresAt:  "2026-11-20T05:20:15Z",
		Reason:     "Annual FY2026 external tax audit",
		TraceID:    "trace-w3c-1",
		OccurredAt: "2026-11-17T05:20:15Z",
	}
	assertJSONIntegerFields(t, "AuditorAccessGrantedEvent", auditorEv, map[string]any{
		"type":       EventAuditorAccessGranted,
		"user_id":    "usr-ext-auditor-1",
		"role_name":  "AUDITOR",
		"action":     "GRANTED",
		"granted_by": "usr-owner",
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
