package cache

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisBackend binds Cache Backend to a Redis client.
type RedisBackend struct {
	client *redis.Client
}

// NewRedisBackend constructs a Redis-backed cache backend using RedisConfig.
func NewRedisBackend(cfg RedisConfig) (*RedisBackend, error) {
	trimmed := strings.TrimSpace(cfg.Addr)
	if trimmed == "" {
		return nil, fmt.Errorf("redis backend: addr required")
	}

	opts := &redis.Options{
		Addr:            trimmed,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxIdleTime: cfg.MaxIdleTime,
		DialTimeout:     cfg.DialTimeout,
		ReadTimeout:     cfg.ReadTimeout,
		WriteTimeout:    cfg.WriteTimeout,
		MaxRetries:      cfg.MaxRetries,
		MinRetryBackoff: cfg.MinRetryBackoff,
		MaxRetryBackoff: cfg.MaxRetryBackoff,
	}

	if cfg.TLSEnabled {
		tlsCfg := &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
		if cfg.TLSInsecure {
			tlsCfg.InsecureSkipVerify = true
		} else if pem := strings.TrimSpace(cfg.CACertPEM); pem != "" {
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM([]byte(pem)) {
				return nil, fmt.Errorf("redis backend: failed to parse REDIS CA PEM")
			}
			tlsCfg.RootCAs = pool
			// Memorystore presents a cert for the instance host IP / internal name.
			if host, _, ok := strings.Cut(trimmed, ":"); ok && host != "" {
				tlsCfg.ServerName = host
			}
		}
		opts.TLSConfig = tlsCfg
	}

	client := redis.NewClient(opts)
	return &RedisBackend{client: client}, nil
}

// Ping verifies connectivity.
func (r *RedisBackend) Ping(ctx context.Context) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("redis backend: nil client")
	}
	return r.client.Ping(ctx).Err()
}

// Client exposes the underlying Redis client for shared infrastructure (rate limits, idempotency).
func (r *RedisBackend) Client() *redis.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// Close closes underlying client resources.
func (r *RedisBackend) Close() error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Close()
}

func (r *RedisBackend) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if r == nil || r.client == nil {
		return nil, false, nil
	}
	value, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return value, true, nil
}

func (r *RedisBackend) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Set(ctx, key, value, ttl).Err()
}

func (r *RedisBackend) Delete(ctx context.Context, keys ...string) error {
	if r == nil || r.client == nil || len(keys) == 0 {
		return nil
	}
	return r.client.Del(ctx, keys...).Err()
}

func (r *RedisBackend) IncrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if r == nil || r.client == nil {
		return 0, nil
	}
	return r.client.IncrBy(ctx, key, amount).Result()
}

func (r *RedisBackend) DecrBy(ctx context.Context, key string, amount int64) (int64, error) {
	if r == nil || r.client == nil {
		return 0, nil
	}
	return r.client.DecrBy(ctx, key, amount).Result()
}

func (r *RedisBackend) Publish(ctx context.Context, channel string, payload []byte) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.Publish(ctx, channel, payload).Err()
}

// ReplaceSet atomically replaces a Redis set with the provided members and
// applies the specified TTL. A TTL of 0 means no expiration.
func (r *RedisBackend) ReplaceSet(ctx context.Context, key string, members []string, ttl time.Duration) error {
	if r == nil || r.client == nil {
		return nil
	}
	pipe := r.client.TxPipeline()
	pipe.Del(ctx, key)
	if len(members) > 0 {
		sorted := append([]string(nil), members...)
		sort.Strings(sorted)
		args := make([]any, 0, len(sorted))
		for _, member := range sorted {
			if strings.TrimSpace(member) == "" {
				continue
			}
			args = append(args, member)
		}
		if len(args) > 0 {
			pipe.SAdd(ctx, key, args...)
		}
	}
	if ttl > 0 {
		pipe.Expire(ctx, key, ttl)
	} else {
		pipe.Persist(ctx, key)
	}
	_, err := pipe.Exec(ctx)
	return err
}

// SIsMember checks whether member belongs to a Redis set key.
func (r *RedisBackend) SIsMember(ctx context.Context, key string, member string) (bool, error) {
	if r == nil || r.client == nil {
		return false, nil
	}
	return r.client.SIsMember(ctx, key, member).Result()
}

// Exists reports whether the key is present in Redis.
func (r *RedisBackend) Exists(ctx context.Context, key string) (bool, error) {
	if r == nil || r.client == nil {
		return false, nil
	}
	count, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *RedisBackend) Subscribe(ctx context.Context, channel string) (<-chan []byte, func(), error) {
	if r == nil || r.client == nil {
		return nil, func() {}, fmt.Errorf("redis backend: nil client")
	}
	pubsub := r.client.Subscribe(ctx, channel)
	if _, err := pubsub.Receive(ctx); err != nil {
		_ = pubsub.Close()
		return nil, func() {}, fmt.Errorf("redis backend: subscribe receive: %w", err)
	}
	in := pubsub.Channel()
	out := make(chan []byte, 128)
	var once sync.Once
	cancel := func() {
		once.Do(func() {
			_ = pubsub.Close()
		})
	}
	go func() {
		defer close(out)
		for {
			select {
			case <-ctx.Done():
				cancel()
				return
			case msg, ok := <-in:
				if !ok {
					return
				}
				payload := []byte(msg.Payload)
				select {
				case out <- payload:
				default:
				}
			}
		}
	}()
	return out, cancel, nil
}

// PoolStats returns the underlying Redis connection pool statistics.
func (r *RedisBackend) PoolStats() *redis.PoolStats {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.PoolStats()
}

// ── Hash Support ─────────────────────────────────────────────────────────────

func (r *RedisBackend) HGet(ctx context.Context, key, field string) ([]byte, bool, error) {
	if r == nil || r.client == nil {
		return nil, false, nil
	}
	val, err := r.client.HGet(ctx, key, field).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return val, true, nil
}

func (r *RedisBackend) HSet(ctx context.Context, key, field string, value []byte) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.HSet(ctx, key, field, value).Err()
}

func (r *RedisBackend) HGetAll(ctx context.Context, key string) (map[string][]byte, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	res, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	out := make(map[string][]byte, len(res))
	for k, v := range res {
		out[k] = []byte(v)
	}
	return out, nil
}

func (r *RedisBackend) HDel(ctx context.Context, key string, fields ...string) error {
	if r == nil || r.client == nil || len(fields) == 0 {
		return nil
	}
	return r.client.HDel(ctx, key, fields...).Err()
}

// ── Set Support ──────────────────────────────────────────────────────────────

func (r *RedisBackend) SAdd(ctx context.Context, key string, members ...string) error {
	if r == nil || r.client == nil || len(members) == 0 {
		return nil
	}
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SAdd(ctx, key, args...).Err()
}

func (r *RedisBackend) SMembers(ctx context.Context, key string) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return r.client.SMembers(ctx, key).Result()
}

func (r *RedisBackend) SRem(ctx context.Context, key string, members ...string) error {
	if r == nil || r.client == nil || len(members) == 0 {
		return nil
	}
	args := make([]any, len(members))
	for i, m := range members {
		args[i] = m
	}
	return r.client.SRem(ctx, key, args...).Err()
}

// ── Sorted Set Support ───────────────────────────────────────────────────────

func (r *RedisBackend) ZAdd(ctx context.Context, key string, score float64, member string) error {
	if r == nil || r.client == nil {
		return nil
	}
	return r.client.ZAdd(ctx, key, redis.Z{Score: score, Member: member}).Err()
}

func (r *RedisBackend) ZRangeByScore(ctx context.Context, key string, min, max float64, offset, count int64) ([]string, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	opt := &redis.ZRangeBy{
		Min:    fmt.Sprintf("%f", min),
		Max:    fmt.Sprintf("%f", max),
		Offset: offset,
		Count:  count,
	}
	return r.client.ZRangeByScore(ctx, key, opt).Result()
}

func (r *RedisBackend) ZRemRangeByScore(ctx context.Context, key string, min, max float64) (int64, error) {
	if r == nil || r.client == nil {
		return 0, nil
	}
	return r.client.ZRemRangeByScore(ctx, key, fmt.Sprintf("%f", min), fmt.Sprintf("%f", max)).Result()
}

func (r *RedisBackend) ZCard(ctx context.Context, key string) (int64, error) {
	if r == nil || r.client == nil {
		return 0, nil
	}
	return r.client.ZCard(ctx, key).Result()
}

// ── Scan Support ─────────────────────────────────────────────────────────────

func (r *RedisBackend) Scan(ctx context.Context, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if r == nil || r.client == nil {
		return nil, 0, nil
	}
	return r.client.Scan(ctx, cursor, match, count).Result()
}

func (r *RedisBackend) HScan(ctx context.Context, key string, cursor uint64, match string, count int64) ([]string, uint64, error) {
	if r == nil || r.client == nil {
		return nil, 0, nil
	}
	return r.client.HScan(ctx, key, cursor, match, count).Result()
}

// ── Pipelining Support (redis-connections) ───────────────────────────────────

func (r *RedisBackend) Pipelined(ctx context.Context, fn func(pipe redis.Pipeliner) error) error {
	if r == nil || r.client == nil {
		return nil
	}
	_, err := r.client.Pipelined(ctx, fn)
	return err
}

// ── Diagnostics & Triage (redis-observability) ────────────────────────────────

func (r *RedisBackend) SlowLog(ctx context.Context, count int64) ([]redis.SlowLog, error) {
	if r == nil || r.client == nil {
		return nil, nil
	}
	return r.client.SlowLogGet(ctx, count).Result()
}

func (r *RedisBackend) ServerInfo(ctx context.Context, section ...string) (string, error) {
	if r == nil || r.client == nil {
		return "", nil
	}
	return r.client.Info(ctx, section...).Result()
}

var _ EnterpriseBackend = (*RedisBackend)(nil)

