package cache

import "time"

// RedisConfig defines enterprise pooling, retry, and security settings for Redis.
type RedisConfig struct {
	Addr            string
	Password        string        // AUTH password (empty = no auth)
	DB              int           // Redis DB index
	PoolSize        int           // Max active connections (default: 50)
	MinIdleConns    int           // Keep-warm connections (default: 10)
	MaxIdleTime     time.Duration // Idle connection reap (default: 5m)
	DialTimeout     time.Duration // Connection timeout (default: 5s)
	ReadTimeout     time.Duration // Read deadline (default: 3s)
	WriteTimeout    time.Duration // Write deadline (default: 3s)
	MaxRetries      int           // Retry count (default: 3)
	MinRetryBackoff time.Duration // Retry floor (default: 8ms)
	MaxRetryBackoff time.Duration // Retry ceiling (default: 512ms)
	TLSEnabled      bool          // Enforce TLS for production
	// CACertPEM is an optional PEM-encoded CA (e.g. Memorystore server CA).
	// When set with TLSEnabled, the client trusts this CA instead of the system pool.
	CACertPEM string
	// TLSInsecure skips certificate verification (staging only; prefer CACertPEM).
	TLSInsecure bool
}
