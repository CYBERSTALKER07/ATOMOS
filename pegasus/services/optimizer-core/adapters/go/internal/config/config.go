package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	defaultKafkaBrokers      = "localhost:9092"
	defaultKafkaTopic        = "pegasus-optimizer-jobs"
	defaultKafkaGroupID      = "pegasus-optimizer-core-worker"
	defaultOptimizerCoreAddr = "localhost:50055"
	defaultMetricsAddr       = ":8082"
	defaultMaxAttempts       = 3
	defaultRetryBaseMS       = 200
)

type Config struct {
	KafkaBrokers []string
	KafkaTopic   string
	KafkaGroupID string

	SpannerDatabase     string
	RedisAddress        string
	OptimizerCoreAddr   string
	MetricsAddr         string
	SolverMaxAttempts   int
	SolverRetryBaseTime time.Duration
}

func LoadFromEnv() (Config, error) {
	cfg := Config{}

	cfg.KafkaBrokers = splitCSV(getEnv("KAFKA_BROKERS", defaultKafkaBrokers))
	cfg.KafkaTopic = getEnv("OPTIMIZER_JOB_TOPIC", defaultKafkaTopic)
	cfg.KafkaGroupID = getEnv("OPTIMIZER_WORKER_GROUP_ID", defaultKafkaGroupID)

	cfg.SpannerDatabase = os.Getenv("SPANNER_DATABASE")
	if strings.TrimSpace(cfg.SpannerDatabase) == "" {
		return Config{}, fmt.Errorf("SPANNER_DATABASE is required")
	}
	cfg.RedisAddress = strings.TrimSpace(os.Getenv("REDIS_ADDRESS"))

	cfg.OptimizerCoreAddr = getEnv("OPTIMIZER_CORE_GRPC_ADDR", defaultOptimizerCoreAddr)
	cfg.MetricsAddr = getEnv("OPTIMIZER_WORKER_METRICS_ADDR", defaultMetricsAddr)
	cfg.SolverMaxAttempts = getEnvInt("SOLVER_MAX_ATTEMPTS", defaultMaxAttempts)
	cfg.SolverRetryBaseTime = time.Duration(getEnvInt("SOLVER_RETRY_BASE_MS", defaultRetryBaseMS)) * time.Millisecond

	if cfg.SolverMaxAttempts < 1 {
		cfg.SolverMaxAttempts = defaultMaxAttempts
	}
	if cfg.SolverRetryBaseTime <= 0 {
		cfg.SolverRetryBaseTime = time.Duration(defaultRetryBaseMS) * time.Millisecond
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getEnvInt(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return parsed
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return []string{defaultKafkaBrokers}
	}
	return out
}
