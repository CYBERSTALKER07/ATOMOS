import re

with open("apps/ai-worker/planningingest/worker.go", "r") as f:
    content = f.read()

old_logic = """		if err := r.handle(ctx, svc, msg.Value); err != nil {
			r.log.Warn("planning ingest project failed", "err", err)
		}
		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			r.log.Warn("planning ingest commit failed", "err", err)
		}"""

new_logic = """		if err := r.handle(ctx, svc, msg.Value); err != nil {
			r.log.Warn("planning ingest project failed", "err", err)
			if ctx.Err() != nil {
				return
			}
			time.Sleep(2 * time.Second)
			continue // Do not commit, let it be retried
		}
		if err := r.reader.CommitMessages(ctx, msg); err != nil {
			r.log.Warn("planning ingest commit failed", "err", err)
		}"""

if old_logic in content:
    content = content.replace(old_logic, new_logic)
    with open("apps/ai-worker/planningingest/worker.go", "w") as f:
        f.write(content)
    print("Patched planningingest/worker.go successfully")
else:
    print("Could not find logic to patch in planningingest/worker.go")

