package bootstrap

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	backendkafka "github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/redis/go-redis/v9"
)

type redisRuntimeAdapter interface {
	cache.Backend
	Client() *redis.Client
	Ping(ctx context.Context) error
	Close() error
}

type kafkaRuntimePublisher interface {
	outbox.Publisher
	Close() error
}

type kafkaRuntimeDLQWriter interface {
	backendkafka.DLQWriter
}

var (
	newRedisRuntimeAdapter = func(cfg cache.RedisConfig) (redisRuntimeAdapter, error) {
		return cache.NewRedisBackend(cfg)
	}
	newKafkaRuntimePublisher = func(brokersCSV string, cfg outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return outbox.NewKafkaPublisherFromCSV(brokersCSV, cfg)
	}
	newKafkaRuntimeDLQWriter = func(brokersCSV, topic string, auth kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return backendkafka.NewDLQWriterFromCSVWithAuth(brokersCSV, topic, auth)
	}
	newSpannerRuntimeClient = func(ctx context.Context, database string) (*spanner.Client, error) {
		return spanner.NewClient(ctx, database)
	}
	newSpannerRuntimeStore = func(client *spanner.Client) *outbox.SpannerStore {
		return outbox.NewSpannerStore(client)
	}
)

func spannerDatabasePath(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	project := strings.TrimSpace(cfg.SpannerProject)
	instance := strings.TrimSpace(cfg.SpannerInstance)
	database := strings.TrimSpace(cfg.SpannerDatabase)
	if project == "" || instance == "" || database == "" {
		return ""
	}
	return fmt.Sprintf("projects/%s/instances/%s/databases/%s", project, instance, database)
}

func tryNewSpannerOutboxStore(ctx context.Context, cfg *Config) (*spanner.Client, *outbox.SpannerStore, error) {
	database := spannerDatabasePath(cfg)
	if database == "" {
		return nil, nil, fmt.Errorf("spanner database path is empty")
	}

	client, err := newSpannerRuntimeClient(ctx, database)
	if err != nil {
		return nil, nil, fmt.Errorf("create spanner client: %w", err)
	}

	store := newSpannerRuntimeStore(client)
	probeCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := store.Fetch(probeCtx, 1); err != nil {
		client.Close()
		return nil, nil, fmt.Errorf("probe outboxevents read: %w", err)
	}

	return client, store, nil
}
