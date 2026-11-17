package bqexport

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	kgo "github.com/segmentio/kafka-go"
)

const (
	// ConsumerGroupID is the canonical Kafka consumer group for the BigQuery sink exporter.
	ConsumerGroupID = "void-bq-sink"
	// DefaultSinkID is the primary SinkId key stored in BigQuerySinkCheckpoints.
	DefaultSinkID = "void-bq-sink"
)

// DefaultSinkTopics returns the deduplicated list of canonical domain topics exported to BigQuery.
// Retired TopicWebhooks and dead-letter topics are intentionally excluded.
func DefaultSinkTopics() []string {
	candidates := []string{
		events.TopicMain,
		events.TopicOrders,
		events.TopicDispatch,
		events.TopicRealtime,
		events.TopicExceptions,
		events.TopicTelemetryLogistics,
		events.TopicDemand,
		events.TopicLedger,
		events.TopicIntel,
		events.TopicConnectors,
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

// Exporter maps Kafka domain messages to BigQuery rows, writes batches to Sink,
// and advances monotonic checkpoints in CheckpointStore.
type Exporter struct {
	SinkID        string
	DefaultCellID string
	Sink          Sink
	Checkpoints   CheckpointStore
	Log           *slog.Logger
}

// NewExporter constructs an Exporter with safe defaults.
func NewExporter(sink Sink, checkpoints CheckpointStore, defaultCellID string, log *slog.Logger) (*Exporter, error) {
	if sink == nil {
		return nil, errors.New("bqexport: nil sink")
	}
	if checkpoints == nil {
		return nil, errors.New("bqexport: nil checkpoint store")
	}
	if log == nil {
		log = slog.Default()
	}
	cellID := strings.TrimSpace(defaultCellID)
	if cellID == "" {
		cellID = "uz"
	}
	return &Exporter{
		SinkID:        DefaultSinkID,
		DefaultCellID: cellID,
		Sink:          sink,
		Checkpoints:   checkpoints,
		Log:           log,
	}, nil
}

// HandleMessage processes a single Kafka message idempotently using Spanner checkpoints.
func (e *Exporter) HandleMessage(ctx context.Context, msg kgo.Message) error {
	if e == nil || e.Sink == nil || e.Checkpoints == nil {
		return errors.New("bqexport: exporter not initialized")
	}
	sinkID := strings.TrimSpace(e.SinkID)
	if sinkID == "" {
		sinkID = DefaultSinkID
	}

	cp, found, err := e.Checkpoints.GetCheckpoint(ctx, sinkID, msg.Topic, int64(msg.Partition))
	if err != nil {
		return err
	}
	if found && msg.Offset <= cp.OffsetVal {
		return nil
	}

	eventID, rows, err := MapKafkaMessage(msg, e.DefaultCellID)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	if err := e.Sink.WriteBatch(ctx, rows); err != nil {
		return fmt.Errorf("write bq batch (%s:%d:%d): %w", msg.Topic, msg.Partition, msg.Offset, err)
	}

	if err := e.Checkpoints.SaveCheckpoint(ctx, Checkpoint{
		SinkID:       sinkID,
		Topic:        msg.Topic,
		PartitionNum: int64(msg.Partition),
		OffsetVal:    msg.Offset,
		LastEventID:  eventID,
		RowsDelta:    int64(len(rows)),
	}); err != nil {
		return err
	}
	return nil
}

// FlushBatch exports a slice of Kafka messages in a single Sink.WriteBatch call and
// advances the highest monotonic checkpoint per (topic, partition).
func (e *Exporter) FlushBatch(ctx context.Context, msgs []kgo.Message) error {
	if e == nil || e.Sink == nil || e.Checkpoints == nil {
		return errors.New("bqexport: exporter not initialized")
	}
	if len(msgs) == 0 {
		return nil
	}
	sinkID := strings.TrimSpace(e.SinkID)
	if sinkID == "" {
		sinkID = DefaultSinkID
	}

	type partKey struct {
		topic     string
		partition int64
	}
	type partWatermark struct {
		maxOffset   int64
		lastEventID string
		rowsDelta   int64
	}

	existingWatermarks := make(map[partKey]int64)
	rows := make([]Row, 0, len(msgs)*2)
	updates := make(map[partKey]*partWatermark)

	for _, msg := range msgs {
		pk := partKey{topic: msg.Topic, partition: int64(msg.Partition)}
		watermark, loaded := existingWatermarks[pk]
		if !loaded {
			cp, found, err := e.Checkpoints.GetCheckpoint(ctx, sinkID, msg.Topic, int64(msg.Partition))
			if err != nil {
				return err
			}
			if found {
				watermark = cp.OffsetVal
			} else {
				watermark = -1
			}
			existingWatermarks[pk] = watermark
		}
		if msg.Offset <= watermark {
			continue
		}

		eventID, mapped, err := MapKafkaMessage(msg, e.DefaultCellID)
		if err != nil {
			return err
		}
		rows = append(rows, mapped...)

		up, exists := updates[pk]
		if !exists {
			up = &partWatermark{maxOffset: msg.Offset, lastEventID: eventID, rowsDelta: int64(len(mapped))}
			updates[pk] = up
		} else {
			up.rowsDelta += int64(len(mapped))
			if msg.Offset >= up.maxOffset {
				up.maxOffset = msg.Offset
				up.lastEventID = eventID
			}
		}
	}

	if len(rows) == 0 {
		return nil
	}

	if err := e.Sink.WriteBatch(ctx, rows); err != nil {
		return err
	}

	for pk, up := range updates {
		if err := e.Checkpoints.SaveCheckpoint(ctx, Checkpoint{
			SinkID:       sinkID,
			Topic:        pk.topic,
			PartitionNum: pk.partition,
			OffsetVal:    up.maxOffset,
			LastEventID:  up.lastEventID,
			RowsDelta:    up.rowsDelta,
		}); err != nil {
			return err
		}
	}
	return nil
}

// ConsumerDeps configures the multi-topic Kafka BigQuery sink consumer (`void-bq-sink`).
type ConsumerDeps struct {
	Brokers        []string
	Topics         []string
	GroupID        string
	SinkID         string
	DefaultCellID  string
	Sink           Sink
	Checkpoints    CheckpointStore
	DLQWriter      kafka.DLQWriter
	OnDLQ          kafka.DLQHook
	Auth           kafkautil.ClientAuth
	MaxAttempts    int
	ProcessTimeout time.Duration
	Log            *slog.Logger
}

// NewKafkaConsumer wires a manual-commit Kafka multi-topic consumer for `void-bq-sink`.
func NewKafkaConsumer(deps ConsumerDeps) (*kafka.Consumer, *Exporter, error) {
	if len(deps.Brokers) == 0 {
		return nil, nil, errors.New("bqexport: kafka brokers required")
	}
	exporter, err := NewExporter(deps.Sink, deps.Checkpoints, deps.DefaultCellID, deps.Log)
	if err != nil {
		return nil, nil, err
	}
	if strings.TrimSpace(deps.SinkID) != "" {
		exporter.SinkID = strings.TrimSpace(deps.SinkID)
	}
	groupID := strings.TrimSpace(deps.GroupID)
	if groupID == "" {
		groupID = ConsumerGroupID
	}
	topics := deps.Topics
	if len(topics) == 0 {
		topics = DefaultSinkTopics()
	}
	maxAttempts := deps.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 4
	}
	procTimeout := deps.ProcessTimeout
	if procTimeout <= 0 {
		procTimeout = 30 * time.Second
	}

	consumer := kafka.NewMultiTopicConsumer(kafka.ConsumerDeps{
		Brokers:        deps.Brokers,
		GroupID:        groupID,
		Topics:         topics,
		Handler:        exporter.HandleMessage,
		DLQWriter:      deps.DLQWriter,
		OnDLQ:          deps.OnDLQ,
		Auth:           deps.Auth,
		MaxAttempts:    maxAttempts,
		ProcessTimeout: procTimeout,
	})
	return consumer, exporter, nil
}
