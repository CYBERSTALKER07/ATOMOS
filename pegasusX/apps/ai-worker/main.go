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
	"github.com/segmentio/kafka-go"
	"golang.org/x/sync/errgroup"

	"github.com/pegasusx/pegasusx/apps/ai-worker/optimizer"
	"github.com/pegasusx/pegasusx/apps/ai-worker/planningingest"
	"github.com/pegasusx/pegasusx/apps/ai-worker/predictivepush"
	"github.com/pegasusx/pegasusx/apps/ai-worker/synthesis"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
	contract "github.com/pegasusx/pegasusx/packages/optimizer-contract"
)

type EventEnvelope struct {
	Type      string `json:"type"`
	TraceID   string `json:"trace_id"`
	Timestamp string `json:"timestamp"`
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

type CircuitBreaker struct {
	mu          sync.Mutex
	failures    int
	threshold   int
}

func (cb *CircuitBreaker) RecordFailure() {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
}

func (cb *CircuitBreaker) RecordSuccess() {
	if cb == nil {
		return
	}
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.failures > 0 {
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) WaitIfTripped(ctx context.Context) error {
	if cb == nil {
		return nil
	}
	cb.mu.Lock()
	failures := cb.failures
	cb.mu.Unlock()

	if failures >= cb.threshold {
		slog.Warn("circuit breaker tripped, pausing fetches for 15s", "failures", failures)
		select {
		case <-time.After(15 * time.Second):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
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
	if lagSeconds > 300 {
		slog.Warn("consumer lag alert", "consumer", consumer, "topic", msg.Topic, "partition", msg.Partition, "lag_seconds", lagSeconds)
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

	cfg, err := loadConfig()
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

	if os.Getenv("AI_WORKER_MODE") == "predictive-push-cron" {
		if err := predictivepush.Run(ctx, spannerClient); err != nil {
			slog.Error("predictive push cron failed", "err", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

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
	g.Go(func() error {
		frozen.RunCleanup(gCtx)
		return nil
	})

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

	if planningIngest := planningingest.NewRuntime(brokers, spannerClient, logger); planningIngest != nil {
		g.Go(func() error {
			defer planningIngest.Close()
			planningIngest.Run(gCtx)
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

	cb := &CircuitBreaker{threshold: 5}

	g.Go(func() error {
		for {
			if err := cb.WaitIfTripped(gCtx); err != nil {
				return nil
			}

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
					cb.RecordFailure()
				} else {
					cb.RecordSuccess()
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
	case events.EventOrderCreated,
		"ORDER_COMPLETED",
		"ORDER_STATUS_CHANGED",
		"ORDER_DELIVERED":
		if synthesisDisabled() {
			logger.Debug("ai synthesis disabled via AI_SYNTHESIS_ENABLED=false")
			return nil
		}
		engine := synthesis.New(spannerClient, logger)
		if err := engine.HandleOrderEvent(ctx, env.Type, msg.Value); err != nil {
			logger.Error("ai synthesis failed", "err", err)
			return err
		}
		return nil
	default:
		logger.Debug("unhandled event type")
	}

	return nil
}

func synthesisDisabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("AI_SYNTHESIS_ENABLED")))
	return v == "0" || v == "false" || v == "off" || v == "no"
}
