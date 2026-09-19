import re

with open("apps/ai-worker/import_worker.go", "r") as f:
    content = f.read()

# Remove concurrency field
content = re.sub(r'\s*concurrency\s+int', '', content)
content = re.sub(r'\s*concurrency := runtime\.GOMAXPROCS\(0\)\n\s*if concurrency < 1 \{\n\s*concurrency = 1\n\s*\}', '', content)
content = re.sub(r'\s*concurrency:\s*concurrency,', '', content)

# Replace the run loop
old_run = """	importSem := make(chan struct{}, r.concurrency)
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

			processErr := r.repo.ProcessImportUploaded(ctx, r.opener, e.SupplierID, e.SessionID, e.GCSPath, nil)
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
	}"""

new_run = """	for {
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
			// Unparseable poison pills are safe to commit to unblock the partition
			_ = r.reader.CommitMessages(ctx, msg)
			continue
		}
		if strings.TrimSpace(evt.Type) != events.EventInventoryImportUploaded {
			_ = r.reader.CommitMessages(ctx, msg)
			continue
		}

		// Process synchronously to guarantee strict offset ordering and prevent
		// dropping messages on transient/infrastructure failures or pod shutdown.
		processErr := r.repo.ProcessImportUploaded(ctx, r.opener, evt.SupplierID, evt.SessionID, evt.GCSPath, nil)
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

		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			r.logger.Error("inventory import commit failed", "err", err, "partition", msg.Partition, "offset", msg.Offset)
		}
	}"""

content = content.replace(old_run, new_run)

# Also remove runtime package if not used anymore
content = re.sub(r'\t"runtime"\n', '', content)

with open("apps/ai-worker/import_worker.go", "w") as f:
    f.write(content)

print("Patched successfully")
