package cache

import "time"

// RedisConfig defines enterprise pooling, retry, and security settings for Redis.
type RedisConfig struct {
	Addr            string
	Username        string        // Redis 6+ ACL username (empty = default user)
	Password        string        // AUTH password (empty = no auth)
	DB              int           // Redis DB index
	Protocol        int           // RESP protocol version (2 or 3; 0 defaults to 2/driver default)
	ReadOnly        bool          // Route read commands to replicas in cluster/replica deployments
	PoolSize        int           // Max active connections (default: 50)
	MinIdleConns    int           // Keep-warm connections (default: 10)
	MaxIdleTime     time.Duration // Idle connection reap (default: 5m)
	DialTimeout     time.Duration // Connection timeout (default: 2s)
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

func (c *RedisConfig) applyDefaults() {
	if c.PoolSize <= 0 {
		c.PoolSize = 50
	}
	if c.MinIdleConns <= 0 {
		c.MinIdleConns = 10
	}
	if c.MaxIdleTime <= 0 {
		c.MaxIdleTime = 5 * time.Minute
	}
	if c.DialTimeout <= 0 {
		c.DialTimeout = 2 * time.Second
	}
	if c.ReadTimeout <= 0 {
		c.ReadTimeout = 3 * time.Second
	}
	if c.WriteTimeout <= 0 {
		c.WriteTimeout = 3 * time.Second
	}
	if c.MaxRetries <= 0 {
		c.MaxRetries = 3
	}
	if c.MinRetryBackoff <= 0 {
		c.MinRetryBackoff = 8 * time.Millisecond
	}
	if c.MaxRetryBackoff <= 0 {
		c.MaxRetryBackoff = 512 * time.Millisecond
	}
}
