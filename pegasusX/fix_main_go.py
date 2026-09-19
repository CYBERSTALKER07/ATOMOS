import re

with open("apps/ai-worker/main.go", "r") as f:
    content = f.read()

old_loop = """			g.Go(func() error {
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
			})"""

new_loop = """			// Process synchronously to maintain offset commit ordering per partition
			func() {
				defer func() {
					if r := recover(); r != nil {
						slog.Error("panic in event handler", "panic", r, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
					}
				}()

				if err := processMessage(gCtx, msg, spannerClient, frozen); err != nil {
					slog.Error("failed to process message", "err", err, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
					cb.RecordFailure()
					
					// Do not commit on failure to avoid data loss from implicit offset commits
					if gCtx.Err() == nil {
						time.Sleep(2 * time.Second)
					}
					return
				}
				cb.RecordSuccess()

				if err := reader.CommitMessages(gCtx, msg); err != nil {
					slog.Error("failed to commit message", "err", err, "topic", msg.Topic, "partition", msg.Partition, "offset", msg.Offset)
				}
			}()"""

if old_loop in content:
    content = content.replace(old_loop, new_loop)
    with open("apps/ai-worker/main.go", "w") as f:
        f.write(content)
    print("Patched main.go successfully")
else:
    print("Could not find loop to patch in main.go")

