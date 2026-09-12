package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/manifest"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap/memory"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/storage"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
)

// setupGCS initializes Google Cloud Storage bucket access.
func setupGCS(ctx context.Context, bucketName string, log *slog.Logger) {
	if err := storage.InitGCS(ctx, bucketName); err != nil {
		log.Warn("gcs init failed; catalog image uploads use placeholders", "err", err)
	}
}

// setupRedisCache initializes Redis cache backend and circuit breaker.
func setupRedisCache(ctx context.Context, cfg *Config, log *slog.Logger) (cache.Backend, redisRuntimeAdapter, bool, error) {
	cacheBackend := cache.Backend(cache.NewInMemoryBackend())
	redisEnabled := false
	var redisAdapter redisRuntimeAdapter
	redisCfg := cache.RedisConfig{
		Addr:            cfg.RedisAddr,
		Password:        cfg.RedisPassword,
		PoolSize:        cfg.RedisPoolSize,
		MaxRetries:      cfg.RedisMaxRetries,
		TLSEnabled:      cfg.RedisTLSEnabled,
		CACertPEM:       cfg.RedisCACertPEM,
		TLSInsecure:     cfg.RedisTLSInsecure,
		MinIdleConns:    10,
		MaxIdleTime:     5 * time.Minute,
		DialTimeout:     2 * time.Second,
		ReadTimeout:     5 * time.Second,
		WriteTimeout:    5 * time.Second,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	}
	if redisBackend, err := newRedisRuntimeAdapter(redisCfg); err != nil {
		if cfg.RequireInfraAdapters {
			return nil, nil, false, fmt.Errorf("redis backend init failed (REQUIRE_INFRA_ADAPTERS): %w", err)
		}
		log.Warn("redis backend init failed; using in-memory cache",
			"addr", cfg.RedisAddr,
			"err", err,
		)
	} else if err := redisBackend.Ping(ctx); err != nil {
		_ = redisBackend.Close()
		if cfg.RequireInfraAdapters {
			return nil, nil, false, fmt.Errorf("redis ping failed (REQUIRE_INFRA_ADAPTERS): %w", err)
		}
		log.Warn("redis ping failed; using in-memory cache",
			"addr", cfg.RedisAddr,
			"err", err,
		)
	} else {
		redisAdapter = redisBackend
		// Fail-closed circuit under strict infra: no silent memory cache under load.
		failClosedCache := cfg.RequireInfraAdapters || isProductionEnv()
		cbBackend := cache.NewCircuitBreakerBackendWithMode(redisBackend, cacheBackend, failClosedCache)
		cacheBackend = cbBackend
		redisEnabled = true
		log.Info("redis cache backend enabled",
			"addr", cfg.RedisAddr,
			"pool_size", cfg.RedisPoolSize,
			"tls", cfg.RedisTLSEnabled,
			"circuit_fail_closed", failClosedCache,
		)
	}
	if cfg.RequireInfraAdapters && !redisEnabled {
		return nil, nil, false, fmt.Errorf("require infra adapters: redis unavailable")
	}
	return cacheBackend, redisAdapter, redisEnabled, nil
}

// setupIdempotency initializes the idempotency store.
func setupIdempotency(cfg *Config, redisAdapter redisRuntimeAdapter, redisEnabled bool, log *slog.Logger) (idempotency.Store, error) {
	var idemStore idempotency.Store = idempotency.NewInMemoryStore()
	if redisAdapter != nil {
		if client := redisAdapter.Client(); client != nil {
			idemStore = idempotency.NewRedisStore(client)
			log.Info("idempotency redis store enabled", "addr", cfg.RedisAddr)
		}
	}
	if cfg.RequireInfraAdapters && redisEnabled {
		if _, ok := idemStore.(*idempotency.InMemoryStore); ok {
			return nil, fmt.Errorf("require infra adapters: idempotency redis store unavailable")
		}
	}
	return idemStore, nil
}

// setupSpannerAndRouting initializes Spanner outbox store, client, manifest store, and routing clients.
func setupSpannerAndRouting(ctx context.Context, cfg *Config, outboundCircuits *OutboundCircuits, log *slog.Logger) (
	*spanner.Client,
	outbox.Store,
	outboxEventAppender,
	*manifest.Store,
	*routing.GeometryBuilder,
	*routing.OSRMClient,
	bool,
	func(),
	error,
) {
	memoryOutboxStore := memory.NewOutboxStore()
	relayStore := outbox.Store(memoryOutboxStore)
	outboxAppender := outboxEventAppender(memoryOutboxStore)
	spannerOutboxEnabled := false
	var spannerClient *spanner.Client
	var manifestStore *manifest.Store
	var routeGeometryBuilder *routing.GeometryBuilder
	var osrmClient *routing.OSRMClient
	var cleanup func()

	if client, store, err := tryNewSpannerOutboxStore(ctx, cfg); err != nil {
		if !cfg.allowsRepoMemoryFallback() {
			return nil, nil, nil, nil, nil, nil, false, nil, fmt.Errorf("spanner outbox store unavailable (memory fallback disabled): %w", err)
		}
		log.Warn("spanner outbox store unavailable; using in-memory outbox store",
			"database", spannerDatabasePath(cfg),
			"err", err,
		)
	} else {
		spannerClient = client
		relayStore = store
		outboxAppender = store
		spannerOutboxEnabled = true
		cleanup = func() {
			client.Close()
		}
		log.Info("spanner outbox store enabled", "database", spannerDatabasePath(cfg))
		manifestStore = manifest.NewStore(spannerClient)
		googleRoutesClient := routing.NewGoogleRoutesClient(cfg.GoogleMapsAPIKey, "", outboundCircuits.GoogleRoutes)
		osrmClient = routing.NewOSRMClient(cfg.RoutingOSRMURL, outboundCircuits.OSRM)
		routeGeometryBuilder = routing.NewGeometryBuilder(
			googleRoutesClient,
			osrmClient,
			routing.ParseRoutingProviderMode(cfg.RoutingProvider),
		)
		manifestStore.SetGeometryBuilder(routeGeometryBuilder)
		if googleRoutesClient != nil {
			log.Info("Google Routes geometry enabled", "provider_mode", cfg.RoutingProvider)
		}
		if osrmClient != nil {
			log.Info("OSRM routing enabled", "base_url", cfg.RoutingOSRMURL)
		}
	}
	return spannerClient, relayStore, outboxAppender, manifestStore, routeGeometryBuilder, osrmClient, spannerOutboxEnabled, cleanup, nil
}

// setupKafkaPublisher initializes Kafka publisher for transactional outbox relay.
func setupKafkaPublisher(cfg *Config, kafkaAuth kafkautil.ClientAuth, log *slog.Logger) (outbox.Publisher, bool, func(), error) {
	outboxPublisher := outbox.Publisher(&loggingOutboxPublisher{log: log})
	kafkaEnabled := false
	var cleanup func()

	if kafkaPublisher, err := newKafkaRuntimePublisher(cfg.KafkaBrokers, outbox.KafkaPublisherConfig{
		Auth: kafkaAuth,
	}); err != nil {
		if cfg.RequireInfraAdapters {
			return nil, false, nil, fmt.Errorf("kafka publisher init failed (REQUIRE_INFRA_ADAPTERS): %w", err)
		}
		log.Warn("kafka publisher init failed; using logging publisher",
			"brokers", cfg.KafkaBrokers,
			"err", err,
		)
	} else {
		outboxPublisher = kafkaPublisher
		kafkaEnabled = true
		cleanup = func() {
			if err := kafkaPublisher.Close(); err != nil {
				log.Warn("kafka publisher close failed", "err", err)
			}
		}
		log.Info("kafka outbox publisher enabled", "brokers", cfg.KafkaBrokers)
	}
	if cfg.RequireInfraAdapters && !kafkaEnabled {
		return nil, false, nil, fmt.Errorf("require infra adapters: kafka unavailable")
	}
	return outboxPublisher, kafkaEnabled, cleanup, nil
}

// setupPushBridge initializes Firebase Cloud Messaging client and notifications bridge.
func setupPushBridge(cfg *Config, spannerClient *spanner.Client, tokenRepo platform.DeviceTokenRepository, log *slog.Logger) (*notifications.PushBridge, error) {
	var fcmClient *notifications.FCMClient
	var err error
	if strings.TrimSpace(cfg.FirebaseProjectID) != "" || strings.TrimSpace(cfg.FirebaseCredentialsPath) != "" {
		fcmClient, err = notifications.InitFCM(cfg.FirebaseCredentialsPath, cfg.FirebaseProjectID, spannerClient, log)
		if err != nil {
			if !fcmAllowNoOp() {
				return nil, fmt.Errorf("FCM init failed and FCM_ALLOW_NOOP is not set (push must not be silent in this env): %w", err)
			}
			log.Error("FCM init failed; push_degraded no-op", "err", err, "push_degraded", true, "alert", "fcm_noop")
			fcmClient = notifications.NewNoOpFCMClient(log)
		}
	} else {
		if !fcmAllowNoOp() {
			return nil, fmt.Errorf("FCM required: set FIREBASE_PROJECT_ID and/or FIREBASE_CREDENTIALS_PATH, or FCM_ALLOW_NOOP=true for explicit degraded push")
		}
		fcmClient = notifications.NewNoOpFCMClient(log)
	}
	return notifications.NewPushBridge(fcmClient, tokenRepo, log), nil
}

// setupDriverLocations initializes the telemetry driver location store.
func setupDriverLocations(cacheClient *cache.Cache) telemetry.LastLocationStore {
	return telemetry.NewCacheLastLocationStore(cacheClient, telemetry.DefaultLastLocationTTL)
}
