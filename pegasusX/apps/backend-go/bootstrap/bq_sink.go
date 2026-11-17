package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bqexport"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
)

var newBQSinkFromEnv = bqexport.NewSinkFromEnv

// NewBQSinkConsumer constructs the Kafka BigQuery sink consumer (`void-bq-sink`) without
// starting its loop. Returns (nil, nil, nil, nil) in non-production when no sink env is configured.
func NewBQSinkConsumer(
	ctx context.Context,
	cfg *Config,
	spannerClient *spanner.Client,
	log *slog.Logger,
) (*kafka.Consumer, *bqexport.Exporter, func(), error) {
	if log == nil {
		log = slog.Default()
	}
	if cfg == nil || !cfg.RunsWorkers() {
		return nil, nil, nil, nil
	}

	sink, err := newBQSinkFromEnv(ctx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("init bigquery sink: %w", err)
	}
	if sink == nil {
		if isProductionEnv() {
			return nil, nil, nil, fmt.Errorf("init bigquery sink: %w", bqexport.ErrBigQueryConfigRequired)
		}
		log.Info("bigquery sink worker disabled (no BIGQUERY_PROJECT_ID or BIGQUERY_LOCAL_SPOOL_DIR configured)")
		return nil, nil, nil, nil
	}

	var checkpoints bqexport.CheckpointStore
	if spannerClient != nil {
		store, err := bqexport.NewSpannerCheckpointStore(spannerClient)
		if err != nil {
			return nil, nil, nil, err
		}
		checkpoints = store
	} else if cfg.allowsRepoMemoryFallback() {
		checkpoints = bqexport.NewMemoryCheckpointStore()
		log.Warn("bigquery sink using in-memory checkpoint store fallback")
	} else {
		return nil, nil, nil, errors.New("bigquery sink requires Spanner client for BigQuerySinkCheckpoints")
	}

	kafkaAuth := kafkautil.ClientAuth{
		Mode:     cfg.KafkaAuthMode,
		Username: cfg.KafkaSASLUsername,
	}
	dlqWriter, err := newKafkaRuntimeDLQWriter(cfg.KafkaBrokers, cfg.KafkaTopicMainDLQ, kafkaAuth)
	if err != nil {
		if cfg.RequireInfraAdapters {
			return nil, nil, nil, fmt.Errorf("init bq sink dlq writer: %w", err)
		}
		log.Warn("bigquery sink consumer disabled; dlq writer init failed", "err", err)
		return nil, nil, nil, nil
	}

	var dlqHook kafka.DLQHook
	if spannerClient != nil {
		dlqHook = kafka.NewSpannerDLQUpdater(spannerClient, log).Hook()
	}

	brokers := kafkautil.SplitBrokers(cfg.KafkaBrokers)
	if len(brokers) == 0 {
		brokers = []string{"localhost:9092"}
	}

	defaultCellID := strings.TrimSpace(os.Getenv("CELL_ID"))
	if defaultCellID == "" {
		defaultCellID = auth.DefaultHomeCellFromEnv()
	}

	consumer, exporter, err := bqexport.NewKafkaConsumer(bqexport.ConsumerDeps{
		Brokers:       brokers,
		Topics:        bqexport.DefaultSinkTopics(),
		GroupID:       bqexport.ConsumerGroupID,
		SinkID:        bqexport.DefaultSinkID,
		DefaultCellID: defaultCellID,
		Sink:          sink,
		Checkpoints:   checkpoints,
		DLQWriter:     dlqWriter,
		OnDLQ:         dlqHook,
		Auth:          kafkaAuth,
		Log:           log,
	})
	if err != nil {
		_ = dlqWriter.Close()
		return nil, nil, nil, fmt.Errorf("init bq sink kafka consumer: %w", err)
	}

	cleanup := func() {
		if err := consumer.Close(); err != nil {
			log.Warn("bigquery sink consumer close failed", "err", err)
		}
		if err := dlqWriter.Close(); err != nil {
			log.Warn("bigquery sink dlq writer close failed", "err", err)
		}
	}

	return consumer, exporter, cleanup, nil
}

// StartSinkWorker initializes and launches the `void-bq-sink` Kafka consumer in worker mode.
// Returns a cleanup callback that cancels and closes the consumer.
func StartSinkWorker(
	ctx context.Context,
	cfg *Config,
	spannerClient *spanner.Client,
	log *slog.Logger,
) (func(), error) {
	if log == nil {
		log = slog.Default()
	}
	consumer, exporter, closeResources, err := NewBQSinkConsumer(ctx, cfg, spannerClient, log)
	if err != nil {
		return nil, err
	}
	if consumer == nil {
		return func() {}, nil
	}

	workerCtx, cancel := context.WithCancel(ctx)
	go consumer.Start(workerCtx)

	log.Info("bigquery sink consumer started",
		"group_id", bqexport.ConsumerGroupID,
		"sink_id", exporter.SinkID,
		"topics", bqexport.DefaultSinkTopics(),
	)

	return func() {
		cancel()
		if closeResources != nil {
			closeResources()
		}
	}, nil
}

// StartBQSinkWorker is an alias for StartSinkWorker.
func StartBQSinkWorker(
	ctx context.Context,
	cfg *Config,
	spannerClient *spanner.Client,
	log *slog.Logger,
) (func(), error) {
	return StartSinkWorker(ctx, cfg, spannerClient, log)
}
