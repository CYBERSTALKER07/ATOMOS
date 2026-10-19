package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/segmentio/kafka-go"
	"google.golang.org/api/iterator"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	maxMsgs := flag.Int("max", 100, "Maximum number of DLQ messages to inspect/replay")
	dryRun := flag.Bool("dry-run", false, "Inspect messages without writing to main topic or committing DLQ offsets")
	tenantID := flag.String("tenant-id", "", "Filter messages by supplier/tenant ID")
	source := flag.String("source", "kafka", "DLQ source: 'kafka' (DLQ topic) or 'spanner' (OutboxDeadLetters table)")
	reEmit := flag.Bool("re-emit", false, "Required to re-emit messages to main topic when dry-run=false")
	flag.Parse()

	if !*dryRun && !*reEmit {
		slog.Error("Safety check: must specify --dry-run to inspect or --re-emit to execute replay")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	cfg, err := bootstrap.LoadConfig()
	if err != nil {
		slog.Error("Failed to load config", "err", err)
		os.Exit(1)
	}

	brokers := strings.Split(cfg.KafkaBrokers, ",")
	if len(brokers) == 0 || brokers[0] == "" {
		slog.Error("KafkaBrokers not configured")
		os.Exit(1)
	}

	switch strings.ToLower(*source) {
	case "spanner":
		replayFromSpanner(ctx, cfg, brokers, *tenantID, *maxMsgs, *dryRun)
	case "kafka":
		replayFromKafka(ctx, cfg, brokers, *tenantID, *maxMsgs, *dryRun)
	default:
		slog.Error("Invalid source: must be 'kafka' or 'spanner'", "source", *source)
		os.Exit(1)
	}
}

func replayFromKafka(ctx context.Context, cfg *bootstrap.Config, brokers []string, tenantFilter string, maxMsgs int, dryRun bool) {
	dlqTopic := cfg.KafkaTopicMainDLQ
	mainTopic := cfg.KafkaTopicMain
	if dlqTopic == "" || mainTopic == "" {
		slog.Error("Kafka DLQ or Main topic not configured")
		os.Exit(1)
	}

	slog.Info("Starting Kafka DLQ processing",
		"brokers", brokers,
		"dlq_topic", dlqTopic,
		"main_topic", mainTopic,
		"tenant_filter", tenantFilter,
		"max", maxMsgs,
		"dry_run", dryRun,
	)

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  brokers,
		GroupID:  fmt.Sprintf("dlq-replayer-%d", time.Now().Unix()),
		Topic:    dlqTopic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer reader.Close()

	var writer *kafka.Writer
	if !dryRun {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  mainTopic,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: false,
		}
		defer writer.Close()
	}

	var processed, replayed int
	for processed < maxMsgs {
		readCtx, readCancel := context.WithTimeout(ctx, 5*time.Second)
		msg, err := reader.FetchMessage(readCtx)
		readCancel()
		if err != nil {
			if err == context.DeadlineExceeded {
				slog.Info("Reached timeout waiting for DLQ messages")
				break
			}
			slog.Error("Error fetching message from DLQ", "err", err)
			break
		}
		processed++

		// Filter by tenant if requested
		if tenantFilter != "" {
			tenantHeader := ""
			for _, h := range msg.Headers {
				if h.Key == "supplier_id" || h.Key == "tenant_id" {
					tenantHeader = string(h.Value)
					break
				}
			}
			if tenantHeader != "" && tenantHeader != tenantFilter {
				continue
			}
		}

		if dryRun {
			slog.Info("[DRY-RUN] Inspecting DLQ message",
				"offset", msg.Offset,
				"partition", msg.Partition,
				"key", string(msg.Key),
				"time", msg.Time,
				"bytes", len(msg.Value),
			)
			continue
		}

		slog.Info("Replaying message to main topic", "offset", msg.Offset, "key", string(msg.Key))
		err = writer.WriteMessages(ctx, kafka.Message{
			Key:     msg.Key,
			Value:   msg.Value,
			Headers: msg.Headers,
		})
		if err != nil {
			slog.Error("Error writing message to main topic", "err", err)
			break
		}

		if err := reader.CommitMessages(ctx, msg); err != nil {
			slog.Error("Error committing DLQ message offset", "err", err)
			break
		}
		replayed++
	}

	slog.Info("Kafka DLQ replay finished", "processed", processed, "replayed", replayed, "dry_run", dryRun)
}

func replayFromSpanner(ctx context.Context, cfg *bootstrap.Config, brokers []string, tenantFilter string, maxMsgs int, dryRun bool) {
	if cfg.SpannerDatabase == "" {
		slog.Error("SpannerDatabase not configured")
		os.Exit(1)
	}

	client, err := spanner.NewClient(ctx, cfg.SpannerDatabase)
	if err != nil {
		slog.Error("Failed to initialize Spanner client", "err", err)
		os.Exit(1)
	}
	defer client.Close()

	slog.Info("Starting Spanner OutboxDeadLetters processing",
		"database", cfg.SpannerDatabase,
		"tenant_filter", tenantFilter,
		"max", maxMsgs,
		"dry_run", dryRun,
	)

	stmt := spanner.Statement{
		SQL: `SELECT EventId, AggregateType, AggregateId, TopicName, Payload, Attempts, LastError, SupplierId 
		      FROM OutboxDeadLetters 
		      WHERE (@tenant = '' OR SupplierId = @tenant)
		      ORDER BY DeadLetteredAt ASC 
		      LIMIT @limit`,
		Params: map[string]interface{}{
			"tenant": tenantFilter,
			"limit":  int64(maxMsgs),
		},
	}

	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	type dlqRow struct {
		EventID       string
		AggregateType string
		AggregateID   string
		TopicName     string
		Payload       []byte
		Attempts      int64
		LastError     spanner.NullString
		SupplierID    spanner.NullString
	}

	var rows []dlqRow
	for {
		r, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.Error("Query OutboxDeadLetters failed", "err", err)
			return
		}
		var row dlqRow
		if err := r.Columns(&row.EventID, &row.AggregateType, &row.AggregateID, &row.TopicName, &row.Payload, &row.Attempts, &row.LastError, &row.SupplierID); err != nil {
			slog.Warn("Failed to parse row", "err", err)
			continue
		}
		rows = append(rows, row)
	}

	slog.Info("Found OutboxDeadLetters rows", "count", len(rows))

	var writer *kafka.Writer
	if !dryRun && len(rows) > 0 {
		writer = &kafka.Writer{
			Addr:                   kafka.TCP(brokers...),
			Topic:                  cfg.KafkaTopicMain,
			Balancer:               &kafka.LeastBytes{},
			AllowAutoTopicCreation: false,
		}
		defer writer.Close()
	}

	var replayed int
	for _, row := range rows {
		if dryRun {
			slog.Info("[DRY-RUN] OutboxDeadLetters row",
				"event_id", row.EventID,
				"topic", row.TopicName,
				"aggregate_id", row.AggregateID,
				"supplier_id", row.SupplierID.StringVal,
				"attempts", row.Attempts,
				"last_error", row.LastError.StringVal,
			)
			continue
		}

		targetTopic := row.TopicName
		if targetTopic == "" {
			targetTopic = cfg.KafkaTopicMain
		}

		err := writer.WriteMessages(ctx, kafka.Message{
			Topic: targetTopic,
			Key:   []byte(row.AggregateID),
			Value: row.Payload,
			Headers: []kafka.Header{
				{Key: "event_id", Value: []byte(row.EventID)},
				{Key: "aggregate_type", Value: []byte(row.AggregateType)},
				{Key: "replayed", Value: []byte("true")},
			},
		})
		if err != nil {
			slog.Error("Failed to re-emit deadletter event", "event_id", row.EventID, "err", err)
			continue
		}

		// Delete from OutboxDeadLetters after successful re-emit
		_, err = client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.Delete("OutboxDeadLetters", spanner.Key{row.EventID}),
			})
		})
		if err != nil {
			slog.Error("Failed to remove row from OutboxDeadLetters after re-emit", "event_id", row.EventID, "err", err)
		} else {
			replayed++
		}
	}

	slog.Info("Spanner DLQ replay finished", "inspected", len(rows), "replayed", replayed, "dry_run", dryRun)
}
