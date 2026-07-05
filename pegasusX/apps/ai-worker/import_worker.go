package main

import (
	"context"
	"log/slog"
	"runtime"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

const importWorkerConsumerName = "pegasusx-ai-worker-import"

type inventoryImportRuntime struct {
	reader      *kafka.Reader
	repo        *supplier.ImportRepository
	opener      *supplier.ImportObjectOpener
	logger      *slog.Logger
	concurrency int
}

func newInventoryImportRuntime(
	brokers []string,
	repo *supplier.ImportRepository,
	opener *supplier.ImportObjectOpener,
	logger *slog.Logger,
) *inventoryImportRuntime {
	if repo == nil || opener == nil {
		return nil
	}
	if logger == nil {
		logger = slog.Default()
	}
	concurrency := runtime.GOMAXPROCS(0)
	if concurrency < 1 {
		concurrency = 1
	}
	return &inventoryImportRuntime{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			GroupID:        importWorkerConsumerName,
			Topic:          events.TopicInventoryImportEvents,
			MinBytes:       1e3,
			MaxBytes:       10e6,
			CommitInterval: time.Second,
			// FirstOffset: import lifecycle events are state transitions, not
			// telemetry. On the group's first-ever join (no committed offset)
			// LastOffset silently skips any UPLOADED event emitted before the
			// join, stranding sessions. Replay is safe: ProcessImportUploaded
			// is idempotent via the MarkSessionDiscovering acquire gate.
			StartOffset: kafka.FirstOffset,
		}),
		repo:        repo,
		opener:      opener,
		logger:      logger,
		concurrency: concurrency,
	}
}

func (r *inventoryImportRuntime) Close() {
	if r == nil || r.reader == nil {
		return
	}
	_ = r.reader.Close()
}

func (r *inventoryImportRuntime) Run(ctx context.Context, metrics *consumerLagMetrics) {
	if r == nil || r.reader == nil {
		return
	}

	importSem := make(chan struct{}, r.concurrency)
	for {
		msg, err := r.reader.FetchMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			r.logger.Error("inventory import fetch failed", "err", err)
			time.Sleep(250 * time.Millisecond)
			continue
		}

		metrics.observe(importWorkerConsumerName, msg)
		evt, parseErr := supplier.ParseInventoryImportUploadedEvent(msg.Value)
		if parseErr != nil {
			r.logger.Warn("inventory import event parse failed", "err", parseErr)
			_ = r.reader.CommitMessages(ctx, msg)
			continue
		}
		if strings.TrimSpace(evt.Type) != events.EventInventoryImportUploaded {
			_ = r.reader.CommitMessages(ctx, msg)
			continue
		}

		importSem <- struct{}{}
		go func(m kafka.Message, e events.InventoryImportEvent) {
			defer func() { <-importSem }()

			processErr := r.repo.ProcessImportUploaded(ctx, r.opener, e.SupplierID, e.SessionID, e.GCSPath)
			if processErr != nil {
				r.logger.Error("inventory import processing failed",
					"session_id", e.SessionID,
					"supplier_id", e.SupplierID,
					"gcs_path", e.GCSPath,
					"err", processErr,
				)
				// Retry mechanisms or DLQ should be handled by the repo or later.
				// We commit the message to avoid poison pills blocking the partition.
			}

			if err := r.reader.CommitMessages(context.Background(), m); err != nil {
				r.logger.Error("inventory import commit failed", "err", err, "partition", m.Partition, "offset", m.Offset)
			}
		}(msg, evt)
	}
}
