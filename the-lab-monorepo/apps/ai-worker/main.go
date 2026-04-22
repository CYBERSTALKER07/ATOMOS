package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"runtime"
	"sort"
	"strconv"
	"sync"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/segmentio/kafka-go"

	"lab-ai-worker/optimizer"
	contract "optimizercontract"
)

// ─── Global Configuration ────────────────────────────────────────────────────

var (
	backendURL     string
	internalAPIKey string
	logger         *slog.Logger

	// Configurable AI constants (env vars with defaults)
	aiDedupThreshold  time.Duration = 1 * time.Hour
	aiRejectionWeight float64       = 0.5
	aiRatioBlendOld   float64       = 0.7
	aiRatioBlendNew   float64       = 0.3
	aiMinTriggerWaitH float64       = 2.0
	aiMaxConcurrent   int           = 10
)

// ─── Event Schemas ── matches JSON emitted by the main backend ───────────────

type OrderCompletedEvent struct {
	OrderID     string `json:"order_id"`
	RetailerID  string `json:"retailer_id"`
	WarehouseId string `json:"warehouse_id"`
	Timestamp   string `json:"timestamp"`
}

type PredictionCorrectedEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id"`
	FieldChanged string `json:"field_changed"`
	OldValue     string `json:"old_value"`
	NewValue     string `json:"new_value"`
	Timestamp    int64  `json:"timestamp"`
}

type AIPlanDateShiftEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id"`
	OldDate      string `json:"old_date"`
	NewDate      string `json:"new_date"`
	Timestamp    int64  `json:"timestamp"`
}

type AIPlanSkuModifiedEvent struct {
	PredictionID string `json:"prediction_id"`
	RetailerID   string `json:"retailer_id"`
	WarehouseId  string `json:"warehouse_id"`
	SkuID        string `json:"sku_id"`
	Field        string `json:"field"`
	OldValue     string `json:"old_value"`
	NewValue     string `json:"new_value"`
	Timestamp    int64  `json:"timestamp"`
}

// ─── SKU-Level Line Item History ─────────────────────────────────────────────

type HistoryItem struct {
	SkuID           string `json:"skuId"`
	ProductID       string `json:"productId"`
	CategoryID      string `json:"categoryId"`
	SupplierID      string `json:"supplierId"`
	WarehouseId     string `json:"warehouseId"`
	Quantity        int64  `json:"quantity"`
	UnitPrice       int64  `json:"unitPrice"`
	OrderDate       string `json:"orderDate"`
	MinimumOrderQty int64  `json:"minimumOrderQty"`
	StepSize        int64  `json:"stepSize"`
}

// ─── Auto-Order Settings ─────────────────────────────────────────────────────

type OverrideEntry struct {
	SupplierID         string  `json:"supplierId"`
	CategoryID         string  `json:"categoryId"`
	ProductID          string  `json:"productId"`
	SkuID              string  `json:"skuId"`
	AnalyticsStartDate *string `json:"analyticsStartDate"`
}

type AutoOrderSettings struct {
	GlobalEnabled      bool            `json:"globalEnabled"`
	AnalyticsStartDate *string         `json:"analyticsStartDate"`
	SupplierOverrides  []OverrideEntry `json:"supplierOverrides"`
	CategoryOverrides  []OverrideEntry `json:"categoryOverrides"`
	ProductOverrides   []OverrideEntry `json:"productOverrides"`
	VariantOverrides   []OverrideEntry `json:"variantOverrides"`
}

// ─── RLHF Correction Weights ────────────────────────────────────────────────
// Implementation lives in correction_store.go (Spanner-backed, write-through cache).

var corrections *correctionStore

// ─── Prediction Deduplication ────────────────────────────────────────────────

type deduplicator struct {
	mu   sync.Mutex
	last map[string]time.Time // retailerID:warehouseID → last prediction time
}

var dedup = &deduplicator{last: make(map[string]time.Time)}

func (d *deduplicator) shouldSkip(retailerID, warehouseID string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	key := correctionKey(retailerID, warehouseID)
	if t, ok := d.last[key]; ok && time.Since(t) < aiDedupThreshold {
		return true
	}
	d.last[key] = time.Now()
	return false
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func resolveStartDateForSku(s *AutoOrderSettings, skuID, productID, categoryID, supplierID string) string {
	for _, v := range s.VariantOverrides {
		if v.SkuID == skuID && v.AnalyticsStartDate != nil && *v.AnalyticsStartDate != "" {
			return *v.AnalyticsStartDate
		}
	}
	for _, v := range s.ProductOverrides {
		if v.ProductID == productID && v.AnalyticsStartDate != nil && *v.AnalyticsStartDate != "" {
			return *v.AnalyticsStartDate
		}
	}
	for _, v := range s.CategoryOverrides {
		if v.CategoryID == categoryID && v.AnalyticsStartDate != nil && *v.AnalyticsStartDate != "" {
			return *v.AnalyticsStartDate
		}
	}
	for _, v := range s.SupplierOverrides {
		if v.SupplierID == supplierID && v.AnalyticsStartDate != nil && *v.AnalyticsStartDate != "" {
			return *v.AnalyticsStartDate
		}
	}
	if s.AnalyticsStartDate != nil && *s.AnalyticsStartDate != "" {
		return *s.AnalyticsStartDate
	}
	return ""
}

func calculateMedianHours(intervals []float64) float64 {
	if len(intervals) == 0 {
		return 0
	}
	sort.Float64s(intervals)
	mid := len(intervals) / 2
	if len(intervals)%2 == 0 {
		return (intervals[mid-1] + intervals[mid]) / 2.0
	}
	return intervals[mid]
}

func calculateMedianInt64(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
	n := len(values)
	if n%2 == 0 {
		return (values[n/2-1] + values[n/2]) / 2
	}
	return values[n/2]
}

func applyPackagingConstraint(rawQty, moq, stepSize int64) int64 {
	if stepSize <= 0 {
		stepSize = 1
	}
	if moq <= 0 {
		moq = stepSize
	}
	if rawQty <= 0 {
		return moq
	}
	constrained := int64(math.Ceil(float64(rawQty)/float64(stepSize))) * stepSize
	if constrained < moq {
		constrained = int64(math.Ceil(float64(moq)/float64(stepSize))) * stepSize
	}
	return constrained
}

// ─── Authenticated HTTP Client ───────────────────────────────────────────────

func authGet(url string) (*http.Response, error) {
	return authRequest("GET", url, nil)
}

func authPost(url string, body []byte) (*http.Response, error) {
	return authRequest("POST", url, body)
}

func authRequest(method, url string, body []byte) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Internal-Key", internalAPIKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return http.DefaultClient.Do(req)
}

// authRequestWithRetry retries the HTTP call up to maxAttempts with exponential backoff.
func authRequestWithRetry(method, url string, body []byte, maxAttempts int) (*http.Response, error) {
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		resp, err := authRequest(method, url, body)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}
		if err != nil {
			lastErr = err
		} else {
			lastErr = fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
			resp.Body.Close()
		}
		if attempt < maxAttempts {
			backoff := time.Duration(attempt*attempt) * 500 * time.Millisecond
			logger.Warn("retrying backend call", "attempt", attempt, "url", url, "err", lastErr, "backoff", backoff)
			time.Sleep(backoff)
		}
	}
	return nil, fmt.Errorf("all %d attempts failed: %w", maxAttempts, lastErr)
}

// ─── Main ────────────────────────────────────────────────────────────────────

// runConsumer fans messages from r to runtime.GOMAXPROCS parallel workers
// sharded by partition index, preserving per-partition message ordering.
// Blocks until ctx is cancelled. The caller owns reader.Close().
func runConsumer(ctx context.Context, r *kafka.Reader, name string, handler func(m kafka.Message)) {
	n := runtime.GOMAXPROCS(0)
	chans := make([]chan kafka.Message, n)
	var wg sync.WaitGroup
	for i := range chans {
		chans[i] = make(chan kafka.Message, 32)
		wg.Add(1)
		go func(in <-chan kafka.Message) {
			defer wg.Done()
			for m := range in {
				handler(m)
				cCtx := ctx
				if cCtx.Err() != nil {
					var cancel context.CancelFunc
					cCtx, cancel = context.WithTimeout(context.Background(), 5*time.Second)
					defer cancel()
				}
				if err := r.CommitMessages(cCtx, m); err != nil {
					logger.Error(name+": commit failed",
						"partition", m.Partition, "offset", m.Offset, "err", err)
				}
			}
		}(chans[i])
	}
	defer func() {
		for _, c := range chans {
			close(c)
		}
		wg.Wait()
	}()
	streak := 0
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			d := time.Duration(100*1<<min(streak, 10)) * time.Millisecond
			if d > 30*time.Second {
				d = 30 * time.Second
			}
			streak++
			logger.Error(name+": fetch failed", "err", err, "streak", streak, "backoff", d)
			select {
			case <-ctx.Done():
				return
			case <-time.After(d):
			}
			continue
		}
		streak = 0
		idx := int(uint(m.Partition)) % n
		select {
		case <-ctx.Done():
			return
		case chans[idx] <- m:
		}
	}
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func main() {
	logger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	brokerAddress := os.Getenv("KAFKA_BROKER_ADDRESS")
	if brokerAddress == "" {
		brokerAddress = "localhost:9092"
	}

	backendURL = os.Getenv("BACKEND_URL")
	if backendURL == "" {
		backendURL = "http://localhost:8080"
	}

	internalAPIKey = os.Getenv("INTERNAL_API_KEY")
	if internalAPIKey == "" {
		internalAPIKey = "lab-internal-dev-key-2026"
	}

	healthPort := os.Getenv("HEALTH_PORT")
	if healthPort == "" {
		healthPort = "8081"
	}

	// ── Load configurable AI constants ──────────────────────────────
	aiDedupThreshold = envDuration("AI_DEDUP_THRESHOLD", 1*time.Hour)
	aiRejectionWeight = envFloat("AI_REJECTION_WEIGHT", 0.5)
	aiRatioBlendOld = envFloat("AI_RATIO_BLEND_OLD", 0.7)
	aiRatioBlendNew = envFloat("AI_RATIO_BLEND_NEW", 0.3)
	aiMinTriggerWaitH = envFloat("AI_MIN_TRIGGER_WAIT_HOURS", 2.0)
	maxConcurrentStr := os.Getenv("MAX_CONCURRENT_PREDICTIONS")
	aiMaxConcurrent = 10
	if maxConcurrentStr != "" {
		fmt.Sscanf(maxConcurrentStr, "%d", &aiMaxConcurrent)
	}

	// ── Spanner Client (optional — graceful degradation) ────────────
	spannerDB := os.Getenv("SPANNER_DATABASE")
	var spannerClient *spanner.Client
	if spannerDB != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		var err error
		spannerClient, err = spanner.NewClient(ctx, spannerDB)
		cancel()
		if err != nil {
			logger.Error("failed to create Spanner client — corrections will be volatile", "err", err)
		} else {
			logger.Info("Spanner client connected", "db", spannerDB)
		}
	} else {
		logger.Warn("SPANNER_DATABASE not set — RLHF corrections will be volatile (in-memory only)")
	}
	if spannerClient != nil {
		defer spannerClient.Close()
	}

	// ── Initialize correction store + boot load ─────────────────────
	corrections = newCorrectionStore(spannerClient)
	if spannerClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := corrections.loadFromSpanner(ctx); err != nil {
			logger.Error("failed to load corrections from Spanner — starting fresh", "err", err)
		}
		cancel()
	}

	fmt.Println("╔════════════════════════════════════════════════════════╗")
	fmt.Println("║  THE LAB INDUSTRIES — INTELLIGENCE ENGINE (AI WORKER) ║")
	fmt.Println("╚════════════════════════════════════════════════════════╝")
	logger.Info("starting AI worker",
		"broker", brokerAddress,
		"backend", backendURL,
		"health_port", healthPort,
		"max_concurrent", aiMaxConcurrent,
		"mode", "SKU-LEVEL MEDIAN PREDICTION (v3)",
		"dedup_threshold", aiDedupThreshold,
		"spanner_connected", spannerClient != nil,
	)

	// ── Health HTTP Server ──────────────────────────────────────────────
	var healthy int32 = 1
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if healthy == 1 {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			w.Write([]byte(`{"status":"shutting_down"}`))
		}
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	})

	// ── Correction Visibility Endpoint (Phase 2.3) ──────────────────
	mux.HandleFunc("/v1/internal/corrections", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Internal-Key") != internalAPIKey {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		retailerID := r.URL.Query().Get("retailer_id")
		weights := corrections.allWeights(retailerID)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(weights)
	})

	// ── Phase 2 Dispatch Optimiser (Clarke-Wright + 2-opt) ─────────
	// Mounted at the contract-locked path; auth via X-Internal-Api-Key.
	// The 2 s soft timeout fires fallback on the backend client.
	mux.HandleFunc(contract.SolvePath, optimizer.Handler(internalAPIKey, logger, 2*time.Second))

	go func() {
		logger.Info("health server listening", "port", healthPort)
		if err := http.ListenAndServe(":"+healthPort, mux); err != nil && err != http.ErrServerClosed {
			logger.Error("health server failed", "err", err)
		}
	}()

	// ── Kafka Reader (main events) ─────────────────────────────────────
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    "lab-logistics-events",
		GroupID:  "ai-worker-group",
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	// ── Kafka Reader (freeze locks — drop frozen entities from work queue) ──
	freezeReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  []string{brokerAddress},
		Topic:    "lab-freeze-locks",
		GroupID:  "ai-worker-freeze-group",
		MinBytes: 1e3,
		MaxBytes: 10e6,
	})
	defer freezeReader.Close()

	// ── Kafka Writer (demand forecast output) ───────────────────────────
	forecastWriter := &kafka.Writer{
		Addr:         kafka.TCP(brokerAddress),
		Topic:        "lab-demand-forecast",
		Balancer:     &kafka.Hash{},
		RequiredAcks: kafka.RequireAll,
		MaxAttempts:  5,
		BatchTimeout: 10 * time.Millisecond,
	}
	defer forecastWriter.Close()

	// ── Concurrency Semaphore ──────────────────────────────────────────
	sem := make(chan struct{}, aiMaxConcurrent)

	// ── Graceful Shutdown ──────────────────────────────────────────────
	ctx, cancel := context.WithCancel(context.Background())
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-stop
		logger.Info("SIGTERM received, shutting down...")
		healthy = 0
		cancel()
	}()

	// ── gRPC Optimizer Server (Phase 2 — internal mesh) ───────────────────
	// Listens on :8082 for backend-go gRPC calls (xDS or direct dial).
	// Uses the same in-process Solve() function as the HTTP handler above.
	if err := startGRPCServer(ctx); err != nil {
		logger.Error("gRPC server start failed", "err", err)
		// Non-fatal: HTTP solve path remains available as fallback.
	}

	// ── Frozen Entity Set (populated by freeze-lock consumer) ──────────
	var frozenMu sync.RWMutex
	frozenEntities := make(map[string]time.Time) // "entity_type:entity_id" → lock expiry

	isFrozen := func(entityType, entityID string) bool {
		frozenMu.RLock()
		defer frozenMu.RUnlock()
		exp, ok := frozenEntities[entityType+":"+entityID]
		return ok && time.Now().Before(exp)
	}

	// ── Freeze Lock Consumer (partition-parallel) ─────────────────────
	var consumerWg sync.WaitGroup
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		runConsumer(ctx, freezeReader, "freeze-consumer", func(m kafka.Message) {
			var evt struct {
				Type       string `json:"type"`
				EntityType string `json:"entity_type"`
				EntityID   string `json:"entity_id"`
				TTLSeconds int64  `json:"ttl_seconds"`
			}
			if err := json.Unmarshal(m.Value, &evt); err != nil {
				logger.Error("freeze event parse error", "err", err)
				return
			}
			key := evt.EntityType + ":" + evt.EntityID
			frozenMu.Lock()
			switch evt.Type {
			case "FREEZE_LOCK_ACQUIRED":
				ttl := time.Duration(evt.TTLSeconds) * time.Second
				if ttl <= 0 {
					ttl = 5 * time.Minute
				}
				frozenEntities[key] = time.Now().Add(ttl)
				logger.Info("entity frozen", "key", key, "ttl", ttl)
			case "FREEZE_LOCK_RELEASED":
				delete(frozenEntities, key)
				logger.Info("entity unfrozen", "key", key)
			}
			frozenMu.Unlock()
		})
	}()

	// ── Emit demand forecast to Kafka after prediction ─────────────────
	emitForecast := func(retailerID, warehouseID string, amount int64, triggerDate string) {
		payload, _ := json.Marshal(map[string]interface{}{
			"type":         "DEMAND_FORECAST_READY",
			"retailer_id":  retailerID,
			"warehouse_id": warehouseID,
			"amount":       amount,
			"trigger_date": triggerDate,
			"timestamp":    time.Now().UTC().Format(time.RFC3339),
		})
		wctx, wcancel := context.WithTimeout(ctx, 5*time.Second)
		defer wcancel()
		if err := forecastWriter.WriteMessages(wctx, kafka.Message{
			Key:   []byte(retailerID),
			Value: payload,
		}); err != nil {
			logger.Error("forecast emit failed", "retailer", retailerID, "err", err)
		}
	}
	_ = emitForecast // prevent unused if ORDER_COMPLETED path changes
	_ = isFrozen     // prevent unused

	// ── Main Consumer (partition-parallel) ───────────────────────────────
	consumerWg.Add(1)
	go func() {
		defer consumerWg.Done()
		runConsumer(ctx, reader, "main-consumer", func(m kafka.Message) {
			eventType := string(m.Key)
			logger.Info("event received", "type", eventType)

			switch eventType {
			case "ORDER_COMPLETED":
				var event OrderCompletedEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse ORDER_COMPLETED", "err", err)
					return
				}
				if isFrozen("RETAILER", event.RetailerID) || isFrozen("WAREHOUSE", event.WarehouseId) {
					logger.Info("freeze-locked, skipping prediction", "retailer", event.RetailerID, "warehouse", event.WarehouseId)
					return
				}
				if dedup.shouldSkip(event.RetailerID, event.WarehouseId) {
					logger.Info("dedup: skipping prediction (recent run exists)", "retailer", event.RetailerID, "warehouse", event.WarehouseId)
					return
				}
				sem <- struct{}{}
				go func() {
					defer func() { <-sem }()
					if pErr := runPredictionEngineV3(event.RetailerID, event.WarehouseId); pErr != nil {
						logger.Error("prediction engine failed", "retailer", event.RetailerID, "warehouse", event.WarehouseId, "err", pErr)
					}
				}()

			case "AI_PREDICTION_CORRECTED":
				var event PredictionCorrectedEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse AI_PREDICTION_CORRECTED", "err", err)
					return
				}
				logger.Info("RLHF correction received",
					"prediction", event.PredictionID,
					"retailer", event.RetailerID,
					"warehouse", event.WarehouseId,
					"field", event.FieldChanged,
					"old", event.OldValue,
					"new", event.NewValue,
				)
				corrections.recordCorrection(event.RetailerID, event.WarehouseId, event.PredictionID, event.FieldChanged, event.OldValue, event.NewValue)

			case "AI_PLAN_DATE_SHIFT":
				var event AIPlanDateShiftEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse AI_PLAN_DATE_SHIFT", "err", err)
					return
				}
				logger.Info("RLHF date shift",
					"prediction", event.PredictionID,
					"retailer", event.RetailerID,
					"warehouse", event.WarehouseId,
					"old_date", event.OldDate,
					"new_date", event.NewDate,
				)
				corrections.recordDateShift(event.RetailerID, event.WarehouseId, event.OldDate, event.NewDate)

			case "AI_PLAN_SKU_MODIFIED":
				var event AIPlanSkuModifiedEvent
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse AI_PLAN_SKU_MODIFIED", "err", err)
					return
				}
				logger.Info("RLHF SKU modification",
					"prediction", event.PredictionID,
					"retailer", event.RetailerID,
					"warehouse", event.WarehouseId,
					"sku", event.SkuID,
					"field", event.Field,
					"old", event.OldValue,
					"new", event.NewValue,
				)
				corrections.recordCorrection(event.RetailerID, event.WarehouseId, event.SkuID, event.Field, event.OldValue, event.NewValue)

			// ── Preorder Lifecycle Events → Forecast Refinement ───────────
			case "PRE_ORDER_AUTO_ACCEPTED":
				var event struct {
					OrderID      string `json:"order_id"`
					RetailerID   string `json:"retailer_id"`
					SupplierID   string `json:"supplier_id"`
					DeliveryDate string `json:"delivery_date"`
				}
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse PRE_ORDER_AUTO_ACCEPTED", "err", err)
					return
				}
				logger.Info("preorder auto-accepted, triggering forecast refresh",
					"order", event.OrderID, "retailer", event.RetailerID, "delivery", event.DeliveryDate)
				sem <- struct{}{}
				go func() {
					defer func() { <-sem }()
					if pErr := runPredictionEngineV3(event.RetailerID, ""); pErr != nil {
						logger.Error("prediction after auto-accept failed", "retailer", event.RetailerID, "err", pErr)
					}
				}()

			case "PRE_ORDER_CONFIRMED":
				var event struct {
					OrderID     string `json:"order_id"`
					ConfirmedBy string `json:"confirmed_by"`
				}
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse PRE_ORDER_CONFIRMED", "err", err)
					return
				}
				logger.Info("preorder explicitly confirmed", "order", event.OrderID, "confirmed_by", event.ConfirmedBy)

			case "PRE_ORDER_EDITED":
				var event struct {
					OrderID  string `json:"order_id"`
					EditedBy string `json:"edited_by"`
					NewDate  string `json:"new_date"`
				}
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse PRE_ORDER_EDITED", "err", err)
					return
				}
				logger.Info("preorder edited, demand signal shifted",
					"order", event.OrderID, "edited_by", event.EditedBy, "new_date", event.NewDate)
				if event.NewDate != "" {
					corrections.recordDateShift(event.EditedBy, "", event.NewDate, event.NewDate)
				}

			case "PRE_ORDER_CANCELLED":
				var event struct {
					OrderID     string `json:"order_id"`
					CancelledBy string `json:"cancelled_by"`
					Reason      string `json:"reason"`
				}
				if err := json.Unmarshal(m.Value, &event); err != nil {
					logger.Error("failed to parse PRE_ORDER_CANCELLED", "err", err)
					return
				}
				logger.Info("preorder cancelled, negative demand signal",
					"order", event.OrderID, "cancelled_by", event.CancelledBy, "reason", event.Reason)

			default:
				// Ignore unrelated events (FLEET_DISPATCHED, ORDER_CANCELLED, etc.)
			}
		})
	}()

	consumerWg.Wait()
	logger.Info("Intelligence Engine shut down cleanly.")
}

// ─── SKU-Level Prediction Engine (v3) ───────────────────────────────────────

func runPredictionEngineV3(retailerID, warehouseID string) error {
	logger.Info("analyzing SKU-level purchase patterns", "retailer", retailerID, "warehouse", warehouseID)

	// 1. Fetch auto-order settings
	settingsURL := backendURL + "/v1/retailer/settings/auto-order?retailer_id=" + url.QueryEscape(retailerID)
	var settings AutoOrderSettings

	resp, err := authRequestWithRetry("GET", settingsURL, nil, 3)
	if err != nil {
		return fmt.Errorf("fetch settings: %w", err)
	}
	defer resp.Body.Close()
	json.NewDecoder(resp.Body).Decode(&settings)

	predictionStatus := "DORMANT"
	if settings.GlobalEnabled {
		predictionStatus = "WAITING"
	}

	// 2. Fetch SKU-level line item history (warehouse-scoped when available)
	historyURL := backendURL + "/v1/orders/line-items/history?retailer_id=" + url.QueryEscape(retailerID)
	if warehouseID != "" {
		historyURL += "&warehouse_id=" + url.QueryEscape(warehouseID)
	}
	if settings.AnalyticsStartDate != nil && *settings.AnalyticsStartDate != "" {
		historyURL += "&since=" + url.QueryEscape(*settings.AnalyticsStartDate)
	}

	resp2, err := authRequestWithRetry("GET", historyURL, nil, 3)
	if err != nil {
		return fmt.Errorf("fetch history: %w", err)
	}
	defer resp2.Body.Close()

	var history []HistoryItem
	if err := json.NewDecoder(resp2.Body).Decode(&history); err != nil {
		return fmt.Errorf("decode history: %w", err)
	}

	// 3. Group by SKU with per-entity analytics date cutoff
	type skuData struct {
		quantities      []int64
		prices          []int64
		dates           []time.Time
		minimumOrderQty int64
		stepSize        int64
	}
	skuMap := make(map[string]*skuData)

	for _, h := range history {
		perSkuSince := resolveStartDateForSku(&settings, h.SkuID, h.ProductID, h.CategoryID, h.SupplierID)
		if perSkuSince != "" {
			cutOff, parseErr := time.Parse(time.RFC3339, perSkuSince)
			if parseErr == nil {
				orderDate, dateErr := time.Parse(time.RFC3339, h.OrderDate)
				if dateErr == nil && orderDate.Before(cutOff) {
					continue
				}
			}
		}

		sd, ok := skuMap[h.SkuID]
		if !ok {
			sd = &skuData{}
			skuMap[h.SkuID] = sd
		}
		sd.quantities = append(sd.quantities, h.Quantity)
		sd.prices = append(sd.prices, h.UnitPrice)
		if h.StepSize > 0 {
			sd.stepSize = h.StepSize
		}
		if h.MinimumOrderQty > 0 {
			sd.minimumOrderQty = h.MinimumOrderQty
		}
		if t, err := time.Parse(time.RFC3339, h.OrderDate); err == nil {
			sd.dates = append(sd.dates, t)
		}
	}

	if len(skuMap) == 0 {
		logger.Info("no SKU history, skipping prediction", "retailer", retailerID)
		return nil
	}

	// 4. Build SKU-level prediction items
	type predItem struct {
		SkuID    string `json:"sku_id"`
		Quantity int64  `json:"quantity"`
		Price    int64  `json:"price"`
	}

	var items []predItem
	var totalAmount int64

	for skuID, sd := range skuMap {
		if len(sd.quantities) < 2 {
			logger.Debug("SKU has insufficient history", "sku", skuID, "orders", len(sd.quantities))
			continue
		}

		medianQty := calculateMedianInt64(sd.quantities)
		medianPrice := calculateMedianInt64(sd.prices)
		if medianQty <= 0 || medianPrice <= 0 {
			continue
		}

		// Apply RLHF correction weight from retailer feedback
		medianQty = corrections.applyCorrection(retailerID, warehouseID, skuID, medianQty)

		// Apply packaging constraints (step size + MOQ)
		rawQty := medianQty
		medianQty = applyPackagingConstraint(rawQty, sd.minimumOrderQty, sd.stepSize)
		if medianQty != rawQty {
			logger.Debug("packaging constraint applied",
				"sku", skuID,
				"raw", rawQty,
				"constrained", medianQty,
				"step", sd.stepSize,
				"moq", sd.minimumOrderQty,
			)
		}

		items = append(items, predItem{
			SkuID:    skuID,
			Quantity: medianQty,
			Price:    medianPrice,
		})
		totalAmount += medianQty * medianPrice
	}

	if len(items) == 0 {
		logger.Info("no SKUs with 2+ orders, insufficient data", "retailer", retailerID)
		return nil
	}

	// 5. Calculate trigger date from order interval medians
	var allDates []time.Time
	for _, sd := range skuMap {
		allDates = append(allDates, sd.dates...)
	}
	sort.Slice(allDates, func(i, j int) bool { return allDates[i].Before(allDates[j]) })

	var intervals []float64
	for i := 1; i < len(allDates); i++ {
		diff := allDates[i].Sub(allDates[i-1]).Hours()
		if diff > 0 {
			intervals = append(intervals, diff)
		}
	}

	medianIntervalHours := calculateMedianHours(intervals)
	triggerWait := medianIntervalHours - 24.0
	if triggerWait < aiMinTriggerWaitH {
		triggerWait = aiMinTriggerWaitH
	}

	// Apply RLHF date-shift correction from retailer feedback
	dateShift := corrections.getTriggerDateShift(retailerID, warehouseID)
	if dateShift != 0 {
		triggerWait += dateShift
		if triggerWait < aiMinTriggerWaitH {
			triggerWait = aiMinTriggerWaitH
		}
		logger.Info("date-shift correction applied", "retailer", retailerID, "warehouse", warehouseID, "shift_h", dateShift, "adjusted_wait_h", triggerWait)
	}

	triggerDate := time.Now().Add(time.Duration(triggerWait) * time.Hour)

	logger.Info("prediction computed",
		"retailer", retailerID,
		"warehouse", warehouseID,
		"skus", len(items),
		"total", totalAmount,
		"status", predictionStatus,
		"trigger", triggerDate.Format(time.RFC3339),
		"interval_h", medianIntervalHours,
	)

	// 6. Inject the SKU-level prediction
	return injectPredictionV3(retailerID, warehouseID, totalAmount, triggerDate.Format(time.RFC3339), items, predictionStatus)
}

// ─── Prediction Injection (v3 — authenticated + retry) ──────────────────────

func injectPredictionV3(retailerID, warehouseID string, amount int64, triggerDate string, items interface{}, status string) error {
	payload := map[string]interface{}{
		"retailer_id":  retailerID,
		"warehouse_id": warehouseID,
		"amount":       amount,
		"trigger_date": triggerDate,
		"status":       status,
		"items":        items,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal prediction: %w", err)
	}

	resp, err := authRequestWithRetry("POST", backendURL+"/v1/prediction/create", jsonData, 3)
	if err != nil {
		return fmt.Errorf("inject prediction: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 201 {
		logger.Info("prediction locked",
			"retailer", retailerID,
			"warehouse", warehouseID,
			"trigger", triggerDate,
			"status", status,
		)
		return nil
	}

	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("backend rejected prediction (HTTP %d): %s", resp.StatusCode, string(body))
}
