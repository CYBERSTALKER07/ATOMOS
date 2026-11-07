package main

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/supplier"
)

const importWorkerConsumerName = "pegasusx-ai-worker-import"

type inventoryImportRuntime struct {
	reader      *kafka.Reader
	dlqWriter   *kafka.Writer
	repo        *supplier.ImportRepository
	opener      *supplier.ImportObjectOpener
	logger      *slog.Logger
}

func newInventoryImportRuntime(
	brokers []string,
	dialer *kafka.Dialer,
	dlqWriter *kafka.Writer,
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
	return &inventoryImportRuntime{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:        brokers,
			Dialer:         dialer,
			GroupID:        importWorkerConsumerName,
			Topic:          events.TopicInventoryImportEvents,
			MinBytes:       1e3,
			MaxBytes:       10e6,
			CommitInterval: 0,
			// FirstOffset: import lifecycle events are state transitions, not
			// telemetry. On the group's first-ever join (no committed offset)
			// LastOffset silently skips any UPLOADED event emitted before the
			// join, stranding sessions. Replay is safe: ProcessImportUploaded
			// is idempotent via the MarkSessionDiscovering acquire gate.
			StartOffset: kafka.FirstOffset,
		}),
		dlqWriter:   dlqWriter,
		repo:        repo,
		opener:      opener,
		logger:      logger,
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

		// Detached context with 4s deadline on SIGTERM so in-flight work completes before shutdown
		procCtx, cancelProc := detachedDrainContext(ctx, 4*time.Second)

		evt, parseErr := supplier.ParseInventoryImportUploadedEvent(msg.Value)
		if parseErr != nil {
			r.logger.Warn("inventory import event parse failed; routing poison pill to DLQ", "err", parseErr)
			if dlqErr := routeToDLQ(procCtx, r.dlqWriter, msg, "inventory_import_parse_error: "+parseErr.Error()); dlqErr != nil {
				r.logger.Error("failed to route import poison pill to DLQ", "err", dlqErr)
			}
			commitCtx, cancelCommit := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_ = r.reader.CommitMessages(commitCtx, msg)
			cancelCommit()
			cancelProc()
			continue
		}
		if strings.TrimSpace(evt.Type) != events.EventInventoryImportUploaded {
			commitCtx, cancelCommit := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			_ = r.reader.CommitMessages(commitCtx, msg)
			cancelCommit()
			cancelProc()
			continue
		}

		// Process synchronously using procCtx (detached context with 4s grace on SIGTERM)
		processErr := r.repo.ProcessImportUploaded(procCtx, r.opener, evt.SupplierID, evt.SessionID, evt.GCSPath, nil)
		cancelProc()
		if processErr != nil {
			r.logger.Error("inventory import processing failed",
				"session_id", evt.SessionID,
				"supplier_id", evt.SupplierID,
				"gcs_path", evt.GCSPath,
				"err", processErr,
			)
			// Do NOT commit on failure. This ensures infrastructure blips (Spanner/GCS down)
			// or context cancellations do not silently discard imports.
			if ctx.Err() != nil {
				return // Graceful shutdown, leave message uncommitted
			}
			// Sleep briefly to avoid tight crash looping on persistent infra failures
			time.Sleep(2 * time.Second)
			continue
		}

		commitCtx, cancelCommit := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
		if err := r.reader.CommitMessages(commitCtx, msg); err != nil {
			r.logger.Error("inventory import commit failed", "err", err, "partition", msg.Partition, "offset", msg.Offset)
		}
		cancelCommit()
	}
}
