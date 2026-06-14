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
			StartOffset:    kafka.LastOffset,
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
		processErr := r.repo.ProcessImportUploaded(ctx, r.opener, evt.SupplierID, evt.SessionID, evt.GCSPath)
		<-importSem

		if processErr != nil {
			r.logger.Error("inventory import processing failed",
				"session_id", evt.SessionID,
				"supplier_id", evt.SupplierID,
				"gcs_path", evt.GCSPath,
				"err", processErr,
			)
			continue
		}

		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			r.logger.Error("inventory import commit failed", "err", err, "partition", msg.Partition, "offset", msg.Offset)
		}
	}
}