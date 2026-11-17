package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pegasusx/pegasusx/apps/ai-worker/predictivepush"
	"github.com/pegasusx/pegasusx/apps/ai-worker/synthesis"
	"github.com/pegasusx/pegasusx/apps/backend-go/bqexport"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/segmentio/kafka-go"
)

func TestAIWorkerInputTopicsAndReaderConfig(t *testing.T) {
	defaultCfg := &Config{
		KafkaTopicOrders: events.TopicOrders,
		KafkaTopicIntel:  events.TopicIntel,
	}
	gotDefault := aiWorkerInputTopics(defaultCfg)
	if len(gotDefault) != 1 || gotDefault[0] != events.TopicMain {
		t.Fatalf("default topics = %v, want [%s]", gotDefault, events.TopicMain)
	}
	rcSingle := newAIWorkerReaderConfig([]string{"localhost:9092"}, nil, "ai-worker-test", gotDefault)
	if rcSingle.Topic != events.TopicMain || len(rcSingle.GroupTopics) != 0 {
		t.Fatalf("unexpected single-topic ReaderConfig: %+v", rcSingle)
	}

	fanInCfg := &Config{
		KafkaConsumeDomainTopics: true,
		KafkaConsumeIntel:        true,
		KafkaTopicOrders:         events.TopicOrders,
		KafkaTopicIntel:          events.TopicIntel,
	}
	gotFanIn := aiWorkerInputTopics(fanInCfg)
	if len(gotFanIn) != 3 {
		t.Fatalf("fan-in topics = %v, want 3 entries", gotFanIn)
	}
	rcMulti := newAIWorkerReaderConfig([]string{"localhost:9092"}, nil, "ai-worker-test", gotFanIn)
	if rcMulti.Topic != "" || len(rcMulti.GroupTopics) != 3 {
		t.Fatalf("unexpected multi-topic ReaderConfig: %+v", rcMulti)
	}
}

func TestHandleOntologyChangedEvent_QuarantineAndNormal(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	frozen := newFreezeRegistry()

	normalPayload, err := json.Marshal(events.OntologyObjectChangedEvent{
		BaseEvent:        events.BaseEvent{Type: events.EventOntologyObjectChanged, Version: 1},
		CellID:           "uz",
		ObjectType:       "RETAILER",
		ObjectID:         "ret-normal-1",
		Version:          1,
		SecurityMarkings: []string{"MARKING:PUBLIC"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := handleOntologyChangedEvent(normalPayload, frozen, logger); err != nil {
		t.Fatalf("normal marking returned err: %v", err)
	}
	if len(frozen.until) != 0 {
		t.Fatalf("expected no freeze entries for non-quarantine marking, got %v", frozen.until)
	}

	quarantineMsgPayload, err := json.Marshal(events.OntologyObjectChangedEvent{
		BaseEvent:        events.BaseEvent{Type: events.EventOntologyObjectChanged, Version: 2},
		CellID:           "uz",
		ObjectType:       "RETAILER",
		ObjectID:         "ret-quar-9",
		SupplierID:       "sup-1",
		Version:          2,
		SecurityMarkings: []string{"MARKING:DISCREPANT_QUARANTINE"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := processMessage(
		context.Background(),
		kafka.Message{Value: quarantineMsgPayload},
		nil,
		frozen,
		&Config{CellID: "uz"},
		nil,
		nil,
	); err != nil {
		t.Fatalf("processMessage returned err: %v", err)
	}

	frozen.mu.RLock()
	exp, ok := frozen.until["RETAILER:ret-quar-9"]
	frozen.mu.RUnlock()
	if !ok || time.Until(exp) <= 0 {
		t.Fatalf("expected freeze entry for RETAILER:ret-quar-9, got ok=%v exp=%v", ok, exp)
	}
}

func TestBuildOrderSignalAndSKUDemandFeatureRows_SpoolRoundTrip(t *testing.T) {
	now := time.Date(2026, 11, 17, 12, 0, 0, 0, time.UTC)
	sig := synthesis.OrderSignal{
		OrderID:     "ord-100",
		SupplierID:  "sup-42",
		RetailerID:  "ret-77",
		WarehouseID: "wh-tashkent-1",
		Status:      "DELIVERED",
		Currency:    "UZS",
		TotalMinor:  350000,
		CreatedAt:   now,
		LineItems: []synthesis.Line{
			{ProductID: "prod-1", SKU: "SKU-001", Quantity: 2, UnitPrice: 100000},
			{ProductID: "prod-1", SKU: "SKU-001", Quantity: 1, UnitPrice: 150000},
		},
	}
	forecast := predictivepush.CrostonSBAResult{
		SuggestedQty: 5,
		Confidence:   0.82,
		ADI:          1.4,
		CV2:          0.25,
		Category:     "SMOOTH",
	}

	orderRows, err := BuildOrderSignalFeatureRows("uz", sig, forecast, now)
	if err != nil {
		t.Fatalf("BuildOrderSignalFeatureRows: %v", err)
	}
	if len(orderRows) != 1 {
		t.Fatalf("len(orderRows) = %d, want 1 aggregated product row", len(orderRows))
	}
	if orderRows[0].Table != bqexport.TableFeaturesSKUDemandDaily {
		t.Fatalf("Table = %q, want %q", orderRows[0].Table, bqexport.TableFeaturesSKUDemandDaily)
	}
	if got := orderRows[0].Data["units_ordered"]; got != int64(3) {
		t.Fatalf("units_ordered = %#v, want int64(3)", got)
	}
	if got := orderRows[0].Data["units_delivered"]; got != int64(3) {
		t.Fatalf("units_delivered = %#v, want int64(3)", got)
	}
	if got := orderRows[0].Data["revenue_minor"]; got != int64(350000) {
		t.Fatalf("revenue_minor = %#v, want int64(350000)", got)
	}

	demandRows, err := BuildSKUDemandFeatureRows("uz", []predictivepush.DemandSignal{
		{
			SupplierID:  "sup-42",
			WarehouseID: "wh-tashkent-1",
			ProductID:   "prod-2",
			Qty:         12,
			Confidence:  0.75,
			Source:      "POS_FLYWHEEL",
			TargetDate:  now,
		},
	}, now)
	if err != nil {
		t.Fatalf("BuildSKUDemandFeatureRows: %v", err)
	}
	if len(demandRows) != 1 {
		t.Fatalf("len(demandRows) = %d, want 1", len(demandRows))
	}

	spoolDir := t.TempDir()
	sink, err := bqexport.NewJSONLSpoolSink(spoolDir)
	if err != nil {
		t.Fatalf("NewJSONLSpoolSink: %v", err)
	}
	allRows := append(orderRows, demandRows...)
	if err := sink.WriteBatch(context.Background(), allRows); err != nil {
		t.Fatalf("WriteBatch: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(spoolDir, "features.sku_demand_daily.jsonl"))
	if err != nil {
		t.Fatalf("read spool file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("spool line count = %d, want 2", len(lines))
	}
}
