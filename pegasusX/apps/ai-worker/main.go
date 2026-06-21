// Package main is the pegasusX AI worker entrypoint.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/ai-worker/optimizer"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

type EventEnvelope struct {
	Type      string `json:"type"`
	TraceID   string `json:"trace_id"`
	Timestamp string `json:"timestamp"`
}

type OrderCreatedData struct {
	Type                  string           `json:"type"`
	TraceID               string           `json:"trace_id"`
	Timestamp             string           `json:"timestamp"`
	OrderID               string           `json:"order_id"`
	SupplierID            string           `json:"supplier_id"`
	RetailerID            string           `json:"retailer_id"`
	WarehouseID           string           `json:"warehouse_id"`
	Status                string           `json:"status"`
	OrderSource           string           `json:"order_source"`
	ConfirmationStatus    string           `json:"confirmation_status"`
	TotalMinor            int64            `json:"total_minor"`
	Currency              string           `json:"currency"`
	H3Cell                string           `json:"h3_cell"`
	Lat                   float64          `json:"lat"`
	Lng                   float64          `json:"lng"`
	RequestedDeliveryDate string           `json:"requested_delivery_date"`
	ReceivingWindowOpen   string           `json:"receiving_window_open"`
	ReceivingWindowClose  string           `json:"receiving_window_close"`
	LineItems             []order.LineItem `json:"line_items"`
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

const (
	aiWorkerConsumerName          = "pegasusx-ai-worker"
	defaultAIWorkerHTTPPort       = "8081"
	aiWorkerShutdownGraceInterval = 5 * time.Second
)

type consumerLagMetrics struct {
	mu      sync.RWMutex
	seconds map[string]float64
}

func newConsumerLagMetrics() *consumerLagMetrics {
	return &consumerLagMetrics{seconds: make(map[string]float64)}
}

func (m *consumerLagMetrics) observe(consumer string, msg kafka.Message) {
	if m == nil {
		return
	}
	lagSeconds := 0.0
	if !msg.Time.IsZero() {
		lagSeconds = time.Since(msg.Time).Seconds()
		if lagSeconds < 0 {
			lagSeconds = 0
		}
	}
	key := consumer + "\xff" + msg.Topic + "\xff" + strconv.Itoa(msg.Partition)
	m.mu.Lock()
	m.seconds[key] = lagSeconds
	m.mu.Unlock()
}

func (m *consumerLagMetrics) writePrometheus(w io.Writer) {
	if m == nil {
		return
	}
	m.mu.RLock()
	defer m.mu.RUnlock()
	fmt.Fprintln(w, "# HELP void_kafka_consumer_lag_seconds Approximate age of the last consumed Kafka message by topic, partition, and consumer.")
	fmt.Fprintln(w, "# TYPE void_kafka_consumer_lag_seconds gauge")
	for key, value := range m.seconds {
		parts := strings.Split(key, "\xff")
		if len(parts) != 3 {
			continue
		}
		fmt.Fprintf(w, "void_kafka_consumer_lag_seconds{consumer=%q,topic=%q,partition=%q} %.6f\n", parts[0], parts[1], parts[2], value)
	}
}

func aiWorkerHTTPPort(getenv func(string) string) string {
	if port := strings.TrimSpace(getenv("AI_WORKER_HTTP_PORT")); port != "" {
		return port
	}
	if port := strings.TrimSpace(getenv("HEALTH_PORT")); port != "" {
		return port
	}
	return defaultAIWorkerHTTPPort
}

func newMonitoringServer(addr string, healthy, ready *atomic.Bool, metrics *consumerLagMetrics, internalAPIKey string, logger *slog.Logger) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		if healthy != nil && healthy.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ok"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"shutting_down"}`))
	})
	mux.HandleFunc("/ready", func(w http.ResponseWriter, _ *http.Request) {
		if healthy != nil && ready != nil && healthy.Load() && ready.Load() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"ready"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"status":"not_ready"}`))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, _ *http.Request) {
		up := int32(0)
		if healthy != nil && healthy.Load() {
			up = 1
		}
		readyState := int32(0)
		if healthy != nil && ready != nil && healthy.Load() && ready.Load() {
			readyState = 1
		}
		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintln(w, "# HELP void_ai_worker_up AI worker process health state.")
		fmt.Fprintln(w, "# TYPE void_ai_worker_up gauge")
		fmt.Fprintf(w, "void_ai_worker_up %d\n", up)
		fmt.Fprintln(w, "# HELP void_ai_worker_ready AI worker readiness state.")
		fmt.Fprintln(w, "# TYPE void_ai_worker_ready gauge")
		fmt.Fprintf(w, "void_ai_worker_ready %d\n", readyState)
		metrics.writePrometheus(w)
	})
	if strings.TrimSpace(internalAPIKey) != "" {
		mux.HandleFunc(contract.SolvePath, optimizer.Handler(internalAPIKey, logger, 2*time.Second))
	}

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	spannerDB := ""
	if cfg.SpannerEmulatorHost != "" {
		os.Setenv("SPANNER_EMULATOR_HOST", cfg.SpannerEmulatorHost)
		spannerDB = "projects/" + cfg.SpannerProject + "/instances/" + cfg.SpannerInstance + "/databases/" + cfg.SpannerDatabase
	}

	if spannerDB == "" {
		spannerDB = "projects/my-project/instances/my-instance/databases/my-database"
	}

	spannerClient, err := spanner.NewClient(ctx, spannerDB)
	if err != nil {
		slog.Error("failed to create spanner client", "err", err)
		os.Exit(1)
	}
	defer spannerClient.Close()

	orderRepo := order.NewSpannerRepository(spannerClient)
	_ = orderRepo // retained for future import worker hooks; AI order synthesis disabled
	_ = order.NewService(order.ServiceConfig{
		Repo:     orderRepo,
		Currency: cfg.SeedSupplierCurrency,
		Log:      logger,
	})

	brokers := []string{"localhost:9092"}
	if cfg.KafkaBrokers != "" {
		brokers = []string{cfg.KafkaBrokers}
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        aiWorkerConsumerName,
		Topic:          events.TopicMain,
		MinBytes:       10e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer reader.Close()

	freezeReader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		GroupID:        aiWorkerConsumerName + "-freeze",
		Topic:          events.TopicFreezeLocks,
		MinBytes:       1e3,
		MaxBytes:       10e6,
		CommitInterval: time.Second,
		StartOffset:    kafka.LastOffset,
	})
	defer freezeReader.Close()

	frozen := newFreezeRegistry()
	metrics := newConsumerLagMetrics()
	workerHTTPPort := aiWorkerHTTPPort(os.Getenv)
	var healthy atomic.Bool
	var ready atomic.Bool
	healthy.Store(true)
	monitoringServer := newMonitoringServer(":"+workerHTTPPort, &healthy, &ready, metrics, cfg.InternalAPIKey, logger)

	slog.Info("ai-worker starting",
		"topic", events.TopicMain,
		"freeze_topic", events.TopicFreezeLocks,
		"import_topic", events.TopicInventoryImportEvents,
		"brokers", brokers,
		"health_port", workerHTTPPort,
	)

	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(runtime.GOMAXPROCS(0) * 2)

	importRepo := supplier.NewImportRepository(spannerClient)
	if importOpener, importOpenerErr := supplier.NewImportObjectOpenerFromEnv(ctx); importOpenerErr != nil {
		slog.Warn("inventory import worker disabled", "err", importOpenerErr)
	} else if importRuntime := newInventoryImportRuntime(brokers, importRepo, importOpener, logger); importRuntime != nil {
		g.Go(func() error {
			defer importRuntime.Close()
			defer importOpener.Close()
			importRuntime.Run(gCtx, metrics)
			return nil
		})
	}

	g.Go(func() error {
		if err := monitoringServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("monitoring server: %w", err)
		}
		return nil
	})
	ready.Store(true)

	g.Go(func() error {
		for {
			m, err := reader.FetchMessage(gCtx)
			if err != nil {
				if gCtx.Err() != nil {
					return nil // graceful shutdown
				}
				slog.Error("failed to fetch message", "err", err)
				time.Sleep(250 * time.Millisecond)
				continue
			}

			metrics.observe(aiWorkerConsumerName, m)
			msg := m

			g.Go(func() error {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in event handler", "panic", r, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
					}
				}()

				if err := processMessage(gCtx, msg, spannerClient, frozen); err != nil {
					slog.Error("failed to process message", "err", err, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
				}

				if err := reader.CommitMessages(context.Background(), msg); err != nil {
					slog.Error("failed to commit message", "err", err, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
				}
				return nil
			})
		}
	})

	g.Go(func() error {
		for {
			m, err := freezeReader.FetchMessage(gCtx)
			if err != nil {
				if gCtx.Err() != nil {
					return nil
				}
				slog.Error("failed to fetch freeze message", "err", err)
				time.Sleep(250 * time.Millisecond)
				continue
			}
			metrics.observe(aiWorkerConsumerName+"-freeze", m)
			frozen.applyPayload(m.Value)
			if err := freezeReader.CommitMessages(context.Background(), m); err != nil {
				slog.Error("failed to commit freeze message", "err", err, "partition", m.Partition, "offset", m.Offset)
			}
		}
	})

	g.Go(func() error {
		<-gCtx.Done()
		return nil
	})

	<-ctx.Done()
	ready.Store(false)
	healthy.Store(false)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), aiWorkerShutdownGraceInterval)
	defer shutdownCancel()
	if err := monitoringServer.Shutdown(shutdownCtx); err != nil && !errors.Is(err, context.Canceled) {
		slog.Warn("monitoring server shutdown failed", "err", err)
	}
	slog.Info("ai-worker shutting down, waiting for workers")

	if err := g.Wait(); err != nil && err != context.Canceled {
		slog.Error("workers exited with error", "err", err)
	}
	slog.Info("ai-worker shutdown complete")
}

func processMessage(ctx context.Context, msg kafka.Message, spannerClient *spanner.Client, frozen *freezeRegistry) error {
	var env EventEnvelope
	if err := json.Unmarshal(msg.Value, &env); err != nil {
		slog.Warn("failed to parse event envelope", "err", err, "body", string(msg.Value))
		return nil
	}

	logger := slog.With("trace_id", env.TraceID, "event_type", env.Type)
	logger.Info("processing event")

	switch env.Type {
	case events.EventOrderCreated:
		// AI preorder/recommendation synthesis removed from PegasusX; optimizer-only worker.
		logger.Debug("order_created ignored (ai synthesis disabled)")
		return nil
	default:
		logger.Debug("unhandled event type")
	}

	return nil
}

func handleOrderCreated(ctx context.Context, logger *slog.Logger, payload []byte, cl *spanner.Client, frozen *freezeRegistry) error {
	var data OrderCreatedData
	if err := json.Unmarshal(payload, &data); err != nil {
		logger.Warn("failed to parse order_created data", "err", err)
		return nil
	}

	if data.OrderID == "" {
		return nil // skip invalid
	}
	if frozen != nil {
		if frozen.isFrozen("ORDER", data.OrderID) || frozen.isFrozen("WAREHOUSE", data.WarehouseID) {
			logger.Info("freeze-locked, skipping ai synthesis",
				"order_id", data.OrderID,
				"warehouse_id", data.WarehouseID,
			)
			return nil
		}
	}
	ctx = outbox.WithTraceID(ctx, data.TraceID)
	if data.OrderSource != "" && data.OrderSource != string(order.OrderSourceManual) {
		logger.Debug("skipping non-manual order for ai synthesis", "order_id", data.OrderID, "order_source", data.OrderSource)
		return nil
	}

	_, err := cl.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		mutations := make([]*spanner.Mutation, 0, 4)

		predictionID := uuid.NewString()
		createdAt := time.Now().UTC()
		expiresAt := createdAt.Add(2 * time.Hour)
		predictionData, err := json.Marshal(map[string]any{
			"action":       "assign_to_nearest",
			"confidence":   0.95,
			"source":       "ai_worker.order_created",
			"explanation":  "New order is ready for supplier dispatch review; nearest-driver assignment is recommended when capacity and route policy allow it.",
			"reason_codes": []string{"order_created", "dispatch_candidate", "human_review_required"},
			"evidence": []map[string]string{
				{"label": "Order", "value": data.OrderID},
				{"label": "Supplier", "value": data.SupplierID},
				{"label": "Order source", "value": data.OrderSource},
			},
			"expires_at": expiresAt.Format(time.RFC3339Nano),
		})
		if err != nil {
			return fmt.Errorf("encode ai recommendation data: %w", err)
		}

		mHeader, err := spanner.InsertOrUpdateStruct("AIPredictions", struct {
			PredictionId   string
			AggregateId    string
			AggregateType  string
			SupplierId     string
			PredictionData []byte
			Score          float64
			Status         string
			CreatedAt      time.Time
			UpdatedAt      time.Time
		}{
			PredictionId:   predictionID,
			AggregateId:    data.OrderID,
			AggregateType:  "Order",
			SupplierId:     data.SupplierID,
			PredictionData: predictionData,
			Score:          0.95,
			Status:         "PENDING",
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		})

		if err != nil {
			return err
		}
		mutations = append(mutations, mHeader)

		if err := outbox.EmitJSON(ctx, buf, events.AggregateAIRecommendation, predictionID, events.TopicMain, map[string]any{
			"type":              events.EventAIRecommendationCreated,
			"trace_id":          data.TraceID,
			"recommendation_id": predictionID,
			"supplier_id":       data.SupplierID,
			"aggregate_id":      data.OrderID,
			"aggregate_type":    "Order",
			"action":            "assign_to_nearest",
			"status":            "PENDING",
			"score":             0.95,
			"confidence":        0.95,
			"explanation":       "New order is ready for supplier dispatch review; nearest-driver assignment is recommended when capacity and route policy allow it.",
			"reason_codes":      []string{"order_created", "dispatch_candidate", "human_review_required"},
			"timestamp":         createdAt.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}

		preorderMutation, preorderCreated, err := buildRetailerAIPreorderMutation(ctx, txn, buf, data, createdAt)
		if err != nil {
			return err
		}
		if preorderCreated {
			mutations = append(mutations, preorderMutation)
		}
		for _, e := range buf.events {
			eventCreatedAt := e.CreatedAt.UTC()
			if eventCreatedAt.IsZero() {
				eventCreatedAt = time.Now().UTC()
			}

			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     eventCreatedAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}

			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}

		return txn.BufferWrite(mutations)
	})

	if err != nil {
		logger.Error("failed to create ai artifacts", "err", err)
		return err
	}

	logger.Info("handled order_created idempotently with ai artifacts")
	return nil
}

func buildRetailerAIPreorderMutation(ctx context.Context, txn *spanner.ReadWriteTransaction, buf *spannerTxnBuffer, data OrderCreatedData, createdAt time.Time) (*spanner.Mutation, bool, error) {
	if strings.TrimSpace(data.RetailerID) == "" || strings.TrimSpace(data.WarehouseID) == "" || len(data.LineItems) == 0 {
		return nil, false, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId
			FROM Orders@{FORCE_INDEX=Idx_Orders_ByDerivedSource}
			WHERE DerivedFromOrderId = @derived_from_order_id
			AND OrderSource = @order_source
			LIMIT 1`,
		Params: map[string]any{
			"derived_from_order_id": data.OrderID,
			"order_source":          string(order.OrderSourceAIPreorder),
		},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == nil {
		return nil, false, nil
	}
	if !errors.Is(err, iterator.Done) {
		return nil, false, fmt.Errorf("query existing ai preorder for order %s: %w", data.OrderID, err)
	}

	requestedDelivery, _ := deriveRetailerAIPreorderSchedule(data, createdAt)
	lineItemsRaw, err := json.Marshal(data.LineItems)
	if err != nil {
		return nil, false, fmt.Errorf("marshal ai preorder line items: %w", err)
	}
	preorderID := uuid.NewString()
	mutation := spanner.InsertMap("Orders", map[string]any{
		"OrderId":               preorderID,
		"SupplierId":            data.SupplierID,
		"RetailerId":            data.RetailerID,
		"WarehouseId":           data.WarehouseID,
		"Status":                string(order.StatusPending),
		"OrderSource":           string(order.OrderSourceAIPreorder),
		"ConfirmationStatus":    string(order.ConfirmationStatusConfirmed),
		"LineItemsJson":         lineItemsRaw,
		"TotalMinor":            data.TotalMinor,
		"Currency":              data.Currency,
		"H3Cell":                data.H3Cell,
		"Lat":                   data.Lat,
		"Lng":                   data.Lng,
		"RequestedDeliveryDate": requestedDelivery,
		"AutoConfirmAt":         nil,
		"DerivedFromOrderId":    data.OrderID,
		"Version":               int64(1),
		"CreatedAt":             createdAt,
		"UpdatedAt":             createdAt,
	})
	if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, preorderID, events.TopicMain, map[string]any{
		"type":                    events.EventOrderCreated,
		"order_id":                preorderID,
		"supplier_id":             data.SupplierID,
		"retailer_id":             data.RetailerID,
		"warehouse_id":            data.WarehouseID,
		"status":                  string(order.StatusPending),
		"order_source":            string(order.OrderSourceAIPreorder),
		"confirmation_status":     string(order.ConfirmationStatusConfirmed),
		"total_minor":             data.TotalMinor,
		"currency":                data.Currency,
		"h3_cell":                 data.H3Cell,
		"lat":                     data.Lat,
		"lng":                     data.Lng,
		"requested_delivery_date": requestedDelivery.Format(time.RFC3339Nano),
		"line_items":              data.LineItems,
		"derived_from_order_id":   data.OrderID,
		"timestamp":               createdAt.Format(time.RFC3339Nano),
	}); err != nil {
		return nil, false, fmt.Errorf("emit ai preorder created event: %w", err)
	}
	return mutation, true, nil
}

func deriveRetailerAIPreorderSchedule(data OrderCreatedData, createdAt time.Time) (time.Time, time.Time) {
	requestedDelivery := createdAt.AddDate(0, 0, 7)
	if parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(data.RequestedDeliveryDate)); err == nil && parsed.After(createdAt) {
		requestedDelivery = parsed.UTC().AddDate(0, 0, 7)
	}
	autoConfirmAt := createdAt.Add(24 * time.Hour)
	if !autoConfirmAt.Before(requestedDelivery) {
		autoConfirmAt = requestedDelivery.Add(-6 * time.Hour)
	}
	if !autoConfirmAt.After(createdAt) {
		autoConfirmAt = createdAt.Add(time.Hour)
	}
	return requestedDelivery.UTC(), autoConfirmAt.UTC()
}
