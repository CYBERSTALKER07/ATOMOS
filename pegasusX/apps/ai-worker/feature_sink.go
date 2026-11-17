package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/ai-worker/predictivepush"
	"github.com/pegasusx/pegasusx/apps/ai-worker/synthesis"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bqexport"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/segmentio/kafka-go"
)

const ontologyQuarantineFreezeTTL = 30 * time.Minute

var quarantineOntologyMarkings = map[string]struct{}{
	"MARKING:AML_INVESTIGATION":          {},
	"MARKING:DISCREPANT_QUARANTINE":      {},
	"MARKING:THERMAL_QUARANTINE_SPOILED": {},
	"MARKING:MICRO_SKIMMING_SUSPECT":     {},
	"MARKING:LOT_RECALL_QUARANTINE":      {},
}

func initFeatureSink(ctx context.Context, cfg *Config, logger *slog.Logger) (bqexport.Sink, error) {
	sink, err := bqexport.NewSinkFromEnv(ctx)
	if err != nil {
		if cfg != nil && (cfg.AIWorkerBigQueryEnabled || auth.IsProduction()) {
			return nil, err
		}
		if logger != nil {
			logger.Warn("ai-worker feature sink unavailable; feature export disabled", "err", err)
		}
		return nil, nil
	}
	if sink == nil && cfg != nil && cfg.AIWorkerBigQueryEnabled && auth.IsProduction() {
		return nil, fmt.Errorf("%w: AI_WORKER_BIGQUERY_ENABLED=true requires configured sink", bqexport.ErrBigQueryConfigRequired)
	}
	return sink, nil
}

// aiWorkerInputTopics returns the Kafka topics consumed by the main ai-worker loop.
// When domain consumption or AI_WORKER_CONSUME_INTEL is enabled, it fans in
// TopicMain, TopicOrders, and TopicIntel; otherwise it stays on TopicMain.
func aiWorkerInputTopics(cfg *Config) []string {
	if cfg == nil {
		return []string{events.TopicMain}
	}
	if !cfg.KafkaConsumeDomainTopics && !cfg.KafkaConsumeIntel {
		return []string{events.TopicMain}
	}
	candidates := []string{events.TopicMain}
	if cfg.KafkaConsumeDomainTopics {
		candidates = append(candidates, cfg.KafkaTopicOrders)
	}
	if cfg.KafkaConsumeIntel {
		candidates = append(candidates, cfg.KafkaTopicIntel)
	}
	seen := make(map[string]struct{}, len(candidates))
	out := make([]string, 0, len(candidates))
	for _, t := range candidates {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

func newAIWorkerReaderConfig(brokers []string, dialer *kafka.Dialer, groupID string, topics []string) kafka.ReaderConfig {
	cfg := kafka.ReaderConfig{
		Brokers:        brokers,
		Dialer:         dialer,
		GroupID:        groupID,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: 0,
		StartOffset:    kafka.FirstOffset,
	}
	if len(topics) <= 1 {
		topic := events.TopicMain
		if len(topics) == 1 && strings.TrimSpace(topics[0]) != "" {
			topic = strings.TrimSpace(topics[0])
		}
		cfg.Topic = topic
	} else {
		cfg.GroupTopics = topics
	}
	return cfg
}

// handleOntologyChangedEvent synchronizes M0.5 ontology security markings into the
// worker's advisory freeze registry so quarantined entities are not auto-recommended.
func handleOntologyChangedEvent(payload []byte, frozen *freezeRegistry, logger *slog.Logger) error {
	var ev events.OntologyObjectChangedEvent
	if err := json.Unmarshal(payload, &ev); err != nil {
		return fmt.Errorf("parse ontology changed event: %w", err)
	}
	objType := strings.TrimSpace(ev.ObjectType)
	objID := strings.TrimSpace(ev.ObjectID)
	if objType == "" || objID == "" {
		return nil
	}

	var matched []string
	for _, m := range ev.SecurityMarkings {
		normalized := strings.ToUpper(strings.TrimSpace(m))
		if _, isQuarantine := quarantineOntologyMarkings[normalized]; isQuarantine {
			matched = append(matched, normalized)
		}
	}

	if len(matched) > 0 && frozen != nil {
		key := strings.ToUpper(objType) + ":" + objID
		frozen.mu.Lock()
		frozen.until[key] = time.Now().Add(ontologyQuarantineFreezeTTL)
		frozen.mu.Unlock()
		if logger != nil {
			logger.Info("applied ontology quarantine freeze in ai-worker",
				"cell_id", ev.CellID,
				"object_type", objType,
				"object_id", objID,
				"supplier_id", ev.SupplierID,
				"version", ev.Version,
				"markings", matched,
			)
		}
	}
	return nil
}

// BuildOrderSignalFeatureRows converts an order synthesis signal into canonical
// `features.sku_demand_daily` rows aligned with the BigQuery feature store schema.
func BuildOrderSignalFeatureRows(cellID string, signal synthesis.OrderSignal, forecast predictivepush.CrostonSBAResult, now time.Time) ([]bqexport.Row, error) {
	if strings.TrimSpace(signal.SupplierID) == "" || len(signal.LineItems) == 0 {
		return nil, nil
	}
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	if !signal.CreatedAt.IsZero() {
		now = signal.CreatedAt.UTC()
	}
	featureDate := now.Format("2006-01-02")
	whID := strings.TrimSpace(signal.WarehouseID)
	if whID == "" {
		whID = "default"
	}
	effCell := strings.TrimSpace(cellID)
	if effCell == "" {
		effCell = "uz"
	}

	// Aggregate per product_id within the order signal using integer minor units only.
	type skuAgg struct {
		sku          string
		unitsOrdered int64
		revenueMinor int64
	}
	byProduct := make(map[string]*skuAgg, len(signal.LineItems))
	order := make([]string, 0, len(signal.LineItems))
	for _, line := range signal.LineItems {
		pid := strings.TrimSpace(line.ProductID)
		if pid == "" {
			continue
		}
		agg, ok := byProduct[pid]
		if !ok {
			agg = &skuAgg{sku: strings.TrimSpace(line.SKU)}
			byProduct[pid] = agg
			order = append(order, pid)
		}
		if agg.sku == "" && strings.TrimSpace(line.SKU) != "" {
			agg.sku = strings.TrimSpace(line.SKU)
		}
		qty := line.Quantity
		if qty < 0 {
			qty = 0
		}
		agg.unitsOrdered += qty
		if line.UnitPrice > 0 && qty > 0 {
			agg.revenueMinor += qty * line.UnitPrice
		}
	}

	rows := make([]bqexport.Row, 0, len(order))
	var confPctPtr *int64
	var adiPtr, cv2Ptr *float64
	var baselinePtr *int64
	if forecast.SuggestedQty > 0 {
		b := forecast.SuggestedQty
		baselinePtr = &b
	}
	if forecast.Confidence > 0 {
		cp := int64(math.Round(forecast.Confidence * 100))
		confPctPtr = &cp
	}
	if forecast.ADI > 0 {
		a := forecast.ADI
		adiPtr = &a
	}
	if forecast.CV2 >= 0 && forecast.Category != "" {
		c := forecast.CV2
		cv2Ptr = &c
	}

	for _, pid := range order {
		agg := byProduct[pid]
		delivered := int64(0)
		if strings.EqualFold(signal.Status, "DELIVERED") || strings.EqualFold(signal.Status, "COMPLETED") {
			delivered = agg.unitsOrdered
		}
		row, err := bqexport.MapSKUDemandDailyRow(bqexport.SKUDemandDailyInput{
			FeatureDate:    featureDate,
			CellID:         effCell,
			SupplierID:     signal.SupplierID,
			WarehouseID:    whID,
			ProductID:      pid,
			SKU:            agg.sku,
			Currency:       strings.TrimSpace(signal.Currency),
			UnitsOrdered:   agg.unitsOrdered,
			UnitsDelivered: delivered,
			RevenueMinor:   agg.revenueMinor,
			BaselineQty:    baselinePtr,
			ConfidencePct:  confPctPtr,
			ADI:            adiPtr,
			CV2:            cv2Ptr,
			DemandCategory: forecast.Category,
			UpdatedAt:      now,
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// BuildSKUDemandFeatureRows converts predictivepush.DemandSignal items into canonical
// `features.sku_demand_daily` rows.
func BuildSKUDemandFeatureRows(cellID string, signals []predictivepush.DemandSignal, now time.Time) ([]bqexport.Row, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	} else {
		now = now.UTC()
	}
	effCell := strings.TrimSpace(cellID)
	if effCell == "" {
		effCell = "uz"
	}
	rows := make([]bqexport.Row, 0, len(signals))
	for _, sig := range signals {
		if strings.TrimSpace(sig.SupplierID) == "" || strings.TrimSpace(sig.ProductID) == "" {
			continue
		}
		target := sig.TargetDate
		if target.IsZero() {
			target = now
		}
		baseline := sig.Qty
		confPct := int64(math.Round(sig.Confidence * 100))
		row, err := bqexport.MapSKUDemandDailyRow(bqexport.SKUDemandDailyInput{
			FeatureDate:    target.UTC().Format("2006-01-02"),
			CellID:         effCell,
			SupplierID:     sig.SupplierID,
			WarehouseID:    sig.WarehouseID,
			ProductID:      sig.ProductID,
			UnitsOrdered:   sig.Qty,
			UnitsDelivered: 0,
			RevenueMinor:   0,
			BaselineQty:    &baseline,
			ConfidencePct:  &confPct,
			DemandCategory: sig.Source,
			UpdatedAt:      now,
		})
		if err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}
