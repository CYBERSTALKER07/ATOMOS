import re

# 1. Update EventDedupStore interface
with open('apps/backend-go/kafka/event_dedup_store.go', 'r') as f:
    content = f.read()

content = content.replace(
    """type EventDedupStore interface {
	ShouldProcess(ctx context.Context, key string) (bool, error)
}""",
    """type EventDedupStore interface {
	ShouldProcess(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}"""
)

# Add Release to InMemoryEventDedup
content += """
// Release removes the key from the seen map.
func (d *InMemoryEventDedup) Release(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	delete(d.seen, key)
	return nil
}
"""

with open('apps/backend-go/kafka/event_dedup_store.go', 'w') as f:
    f.write(content)

# 2. Update RedisEventDedup
with open('apps/backend-go/kafka/redis_event_dedup.go', 'r') as f:
    content = f.read()

content += """
// Release deletes the key from Redis, allowing future retries.
func (d *RedisEventDedup) Release(ctx context.Context, key string) error {
	if d == nil || d.client == nil || key == "" {
		return nil
	}
	_, err := d.client.Del(ctx, eventDedupKeyPrefix+key).Result()
	return err
}
"""

with open('apps/backend-go/kafka/redis_event_dedup.go', 'w') as f:
    f.write(content)

# 3. Update WithEventDedup
with open('apps/backend-go/kafka/event_dedup_middleware.go', 'r') as f:
    content = f.read()

old_block = """		ok, err := store.ShouldProcess(ctx, key)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		return handler(ctx, msg)"""

new_block = """		ok, err := store.ShouldProcess(ctx, key)
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}
		err = handler(ctx, msg)
		if err != nil {
			_ = store.Release(ctx, key)
			return err
		}
		return nil"""

content = content.replace(old_block, new_block)

with open('apps/backend-go/kafka/event_dedup_middleware.go', 'w') as f:
    f.write(content)

