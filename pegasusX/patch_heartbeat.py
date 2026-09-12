import re

with open('apps/backend-go/bootstrap/worker_heartbeat.go', 'r') as f:
    content = f.read()

old_block = """			case <-ctx.Done():
				return
			case <-ticker.C:
				beat()"""

new_block = """			case <-ctx.Done():
				c, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				_ = client.Del(c, workerHeartbeatKey).Err()
				cancel()
				return
			case <-ticker.C:
				beat()"""

if old_block in content:
    content = content.replace(old_block, new_block)
    with open('apps/backend-go/bootstrap/worker_heartbeat.go', 'w') as f:
        f.write(content)
    print("Patched StartWorkerHeartbeat successfully.")
else:
    print("Could not find the target block to replace.")

