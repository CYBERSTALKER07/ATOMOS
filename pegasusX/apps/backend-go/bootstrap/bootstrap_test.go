package bootstrap

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/redis/go-redis/v9"
	segmentkafka "github.com/segmentio/kafka-go"
)

type fakeRedisAdapter struct {
	pingErr    error
	pingCalled bool
	closed     bool
}

func (f *fakeRedisAdapter) Ping(_ context.Context) error {
	f.pingCalled = true
	return f.pingErr
}

func (f *fakeRedisAdapter) Close() error {
	f.closed = true
	return nil
}

func (f *fakeRedisAdapter) Client() *redis.Client {
	return nil
}

func (f *fakeRedisAdapter) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, nil
}

func (f *fakeRedisAdapter) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return nil
}

func (f *fakeRedisAdapter) Delete(_ context.Context, _ ...string) error {
	return nil
}

func (f *fakeRedisAdapter) Publish(_ context.Context, _ string, _ []byte) error {
	return nil
}

func (f *fakeRedisAdapter) Subscribe(_ context.Context, _ string) (<-chan []byte, func(), error) {
	ch := make(chan []byte)
	cancel := func() {
		close(ch)
	}
	return ch, cancel, nil
}

type fakeKafkaPublisher struct {
	closed bool
}

func (f *fakeKafkaPublisher) Publish(_ context.Context, _ string, _ []byte, _ []byte) error {
	return nil
}

func (f *fakeKafkaPublisher) Close() error {
	f.closed = true
	return nil
}

type fakeKafkaDLQWriter struct {
	closed bool
}

func (f *fakeKafkaDLQWriter) WriteMessages(_ context.Context, _ ...segmentkafka.Message) error {
	return nil
}

func (f *fakeKafkaDLQWriter) Close() error {
	f.closed = true
	return nil
}

func testConfig() *Config {
	return &Config{
		HTTPPort:               "8080",
		SpannerEmulatorHost:    "localhost:9010",
		SpannerProject:         "pegasusx-local",
		SpannerInstance:        "pegasusx-instance",
		SpannerDatabase:        "pegasusx-db",
		RedisAddr:              "localhost:6379",
		KafkaBrokers:           "localhost:9092",
		KafkaTopicMain:         "pegasusx-main",
		KafkaTopicMainDLQ:      "pegasusx-main-dlq",
		JWTSecret:              "test-secret",
		JWTIssuer:              "pegasusx-test",
		GlobalPayWebhookSecret: "gp-test",
		AdyenWebhookSecret:     "ad-test",
		StripeWebhookSecret:    "st-test",
		TestingMode:            true,
		SeedSupplierName:       "pegasusX Supplier",
		SeedSupplierCountry:    "UZ",
		SeedSupplierCurrency:   "UZS",
	}
}

func stubRuntimeConstructors(t *testing.T) {
	t.Helper()
	originalRedis := newRedisRuntimeAdapter
	originalKafka := newKafkaRuntimePublisher
	originalKafkaDLQ := newKafkaRuntimeDLQWriter
	originalSpannerClient := newSpannerRuntimeClient
	originalSpannerStore := newSpannerRuntimeStore
	t.Cleanup(func() {
		newRedisRuntimeAdapter = originalRedis
		newKafkaRuntimePublisher = originalKafka
		newKafkaRuntimeDLQWriter = originalKafkaDLQ
		newSpannerRuntimeClient = originalSpannerClient
		newSpannerRuntimeStore = originalSpannerStore
	})
}

func TestNewApp_StrictModeFailsWhenRedisUnavailable(t *testing.T) {
	stubRuntimeConstructors(t)

	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return nil, errors.New("redis unavailable")
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(_ context.Context, _ string) (*spanner.Client, error) {
		return nil, errors.New("skip spanner in strict-mode test")
	}

	cfg := testConfig()
	cfg.RequireInfraAdapters = true

	app, err := NewApp(context.Background(), cfg)
	if err == nil {
		if app != nil {
			app.Close()
		}
		t.Fatalf("expected strict startup error when redis is unavailable")
	}
	if !strings.Contains(err.Error(), "redis unavailable") {
		t.Fatalf("expected redis strict-mode error, got: %v", err)
	}
}

func TestNewApp_StrictModeFailsWhenKafkaUnavailable(t *testing.T) {
	stubRuntimeConstructors(t)

	redis := &fakeRedisAdapter{}
	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return redis, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return nil, errors.New("kafka unavailable")
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	newSpannerRuntimeClient = func(_ context.Context, _ string) (*spanner.Client, error) {
		return nil, errors.New("skip spanner in strict-mode test")
	}

	cfg := testConfig()
	cfg.RequireInfraAdapters = true

	app, err := NewApp(context.Background(), cfg)
	if err == nil {
		if app != nil {
			app.Close()
		}
		t.Fatalf("expected strict startup error when kafka is unavailable")
	}
	if !strings.Contains(err.Error(), "kafka unavailable") {
		t.Fatalf("expected kafka strict-mode error, got: %v", err)
	}
	if !redis.pingCalled {
		t.Fatalf("expected redis health check to run before kafka strict failure")
	}
}

func TestNewApp_StrictModePassesWhenAdaptersHealthy(t *testing.T) {
	stubRuntimeConstructors(t)

	redis := &fakeRedisAdapter{}
	kafka := &fakeKafkaPublisher{}
	dlq := &fakeKafkaDLQWriter{}

	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return redis, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return kafka, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string) (kafkaRuntimeDLQWriter, error) {
		return dlq, nil
	}
	newSpannerRuntimeClient = func(_ context.Context, _ string) (*spanner.Client, error) {
		return nil, errors.New("skip spanner in strict-mode test")
	}

	cfg := testConfig()
	cfg.RequireInfraAdapters = true

	app, err := NewApp(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected strict startup success with healthy adapters, got: %v", err)
	}
	if app == nil {
		t.Fatalf("expected app instance")
	}
	if app.OutboxRelay == nil {
		t.Fatalf("expected outbox relay to be initialized")
	}
	if !redis.pingCalled {
		t.Fatalf("expected redis ping check")
	}

	app.Close()

	if !redis.closed {
		t.Fatalf("expected redis adapter to be closed")
	}
	if !kafka.closed {
		t.Fatalf("expected kafka publisher to be closed")
	}
	if !dlq.closed {
		t.Fatalf("expected kafka dlq writer to be closed")
	}
}

func TestNewApp_StrictModeFailsWhenNotificationDLQUnavailable(t *testing.T) {
	stubRuntimeConstructors(t)

	redis := &fakeRedisAdapter{}
	newRedisRuntimeAdapter = func(_ cache.RedisConfig) (redisRuntimeAdapter, error) {
		return redis, nil
	}
	newKafkaRuntimePublisher = func(_ string, _ outbox.KafkaPublisherConfig) (kafkaRuntimePublisher, error) {
		return &fakeKafkaPublisher{}, nil
	}
	newKafkaRuntimeDLQWriter = func(_ string, _ string) (kafkaRuntimeDLQWriter, error) {
		return nil, errors.New("dlq unavailable")
	}
	newSpannerRuntimeClient = func(_ context.Context, _ string) (*spanner.Client, error) {
		return nil, errors.New("skip spanner in strict-mode test")
	}

	cfg := testConfig()
	cfg.RequireInfraAdapters = true

	app, err := NewApp(context.Background(), cfg)
	if err == nil {
		if app != nil {
			app.Close()
		}
		t.Fatalf("expected strict startup error when notification dlq is unavailable")
	}
	if !strings.Contains(err.Error(), "dlq unavailable") {
		t.Fatalf("expected dlq strict-mode error, got: %v", err)
	}
}
