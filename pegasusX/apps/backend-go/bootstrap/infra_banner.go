package bootstrap

import (
	"log/slog"
	"os"
	"strings"
)

type infraBackendStatus struct {
	Spanner          bool
	RedisCache       bool
	IdempotencyRedis bool
	Kafka            bool
	SpannerOutbox    bool
}

func logInfraBackendBanner(log *slog.Logger, cfg *Config, status infraBackendStatus) {
	env := strings.TrimSpace(os.Getenv("PEGASUSX_ENV"))
	if env == "" {
		env = "development"
	}
	idemBackend := "in-memory"
	if status.IdempotencyRedis {
		idemBackend = "redis"
	}
	outboxBackend := "in-memory"
	if status.SpannerOutbox {
		outboxBackend = "spanner"
	}
	repoBackend := "in-memory"
	if status.Spanner {
		repoBackend = "spanner"
	}
	log.Info("pegasusX backend infrastructure banner",
		"pegasusx_env", env,
		"require_infra_adapters", cfg.RequireInfraAdapters,
		"spanner", status.Spanner,
		"redis_cache", status.RedisCache,
		"idempotency_store", idemBackend,
		"outbox_store", outboxBackend,
		"kafka_publisher", status.Kafka,
		"domain_repos", repoBackend,
	)
}
