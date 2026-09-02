import re

with open("apps/backend-go/driver/service.go", "r") as f:
    content = f.read()

pattern = re.compile(r'type Service struct \{\n\trepo\s+Repository\n\tlog\s+\*slog\.Logger\n\tdriverHub\s+\*ws\.Hub\n\tcache\s+cache\.Backend\n\toutbox\s+events\.Outbox\n\}')

replacement = r"""type Service struct {
	repo         Repository
	log          *slog.Logger
	driverHub    *ws.Hub
	warehouseHub *ws.Hub
	cache        cache.Backend
	outbox       events.Outbox
	redisClient  interface{} // Assuming it uses an interface or *redis.Client, but we'll import redis if needed
}"""
content = pattern.sub(replacement, content)

with open("apps/backend-go/driver/service.go", "w") as f:
    f.write(content)
