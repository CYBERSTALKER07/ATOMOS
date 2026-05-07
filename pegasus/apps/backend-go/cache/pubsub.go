package cache

import (
	"context"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

// ── Redis Pub/Sub Backplane ─────────────────────────────────────────────────
// Enables multi-pod WebSocket broadcast. When a pod publishes to a channel,
// all other pods subscribed to that channel receive the message and relay it
// to their local WebSocket connections.
//
// Channel naming convention:
//   ws:supplier:{supplierID}  — fleet/order events scoped to a supplier
//   ws:retailer:{retailerID}  — payment/delivery events scoped to a retailer
//   ws:driver:{driverID}      — payment/order events scoped to a driver

// PubSubHandler is called when a message arrives on a subscribed channel.
type PubSubHandler func(channel string, payload []byte)

// relay manages a single Redis Pub/Sub subscription with multiple handlers.
type relay struct {
	mu       sync.RWMutex
	handlers map[string][]PubSubHandler // channel pattern → handlers
	subs     map[string]bool            // channel → redis subscription active
	pubsub   *redis.PubSub
	ctx      context.Context
	cancel   context.CancelFunc
}

var (
	globalRelay   *relay
	globalRelayMu sync.Mutex
)

// ensureRelay lazily boots the Pub/Sub subscriber loop and can recover if Redis
// wasn't available on the first call.
func ensureRelay() *relay {
	globalRelayMu.Lock()
	defer globalRelayMu.Unlock()

	if globalRelay != nil {
		return globalRelay
	}
	c := GetClient()
	if c == nil {
		slog.Warn("pubsub backplane disabled: redis client unavailable")
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	globalRelay = &relay{
		handlers: make(map[string][]PubSubHandler),
		subs:     make(map[string]bool),
		pubsub:   c.Subscribe(ctx), // initially no channels
		ctx:      ctx,
		cancel:   cancel,
	}
	go globalRelay.listen()
	slog.Info("pubsub backplane relay started")
	return globalRelay
}

// resetRelay cancels and clears the global relay so ensureRelay re-creates it
// after a Redis recovery. Called by setClient(nil) in the health monitor.
func resetRelay() {
	globalRelayMu.Lock()
	defer globalRelayMu.Unlock()
	if globalRelay != nil {
		globalRelay.cancel()
		globalRelay = nil
	}
}

// listen is the Pub/Sub receive loop. Runs until context is cancelled.
func (r *relay) listen() {
	ch := r.pubsub.Channel()
	for msg := range ch {
		r.mu.RLock()
		handlers := r.handlers[msg.Channel]
		r.mu.RUnlock()

		for _, h := range handlers {
			h(msg.Channel, []byte(msg.Payload))
		}
	}
}

// Publish sends a message to a Redis Pub/Sub channel.
// Non-blocking, fail-open: if Redis is unavailable the message is dropped
// (local WebSocket broadcast still works — this only affects cross-pod relay).
func Publish(ctx context.Context, channel string, payload []byte) {
	c := GetClient()
	if c == nil {
		return
	}
	if err := c.Publish(ctx, channel, payload).Err(); err != nil {
		slog.Warn("pubsub publish failed", "channel", channel, "err", err)
	}
}

// Subscribe registers a handler for messages on the given channel.
// The handler runs in the relay goroutine — keep it fast and non-blocking.
func Subscribe(channel string, handler PubSubHandler) {
	r := ensureRelay()
	if r == nil {
		return
	}

	r.mu.Lock()
	r.handlers[channel] = append(r.handlers[channel], handler)
	shouldSubscribe := !r.subs[channel]
	if shouldSubscribe {
		r.subs[channel] = true
	}
	r.mu.Unlock()

	if shouldSubscribe {
		if err := r.pubsub.Subscribe(r.ctx, channel); err != nil {
			slog.Warn("pubsub subscribe failed", "channel", channel, "err", err)
			r.mu.Lock()
			delete(r.subs, channel)
			r.mu.Unlock()
		}
	}
}

// Unsubscribe removes all handlers for a channel and unsubscribes from Redis.
func Unsubscribe(channel string) {
	r := ensureRelay()
	if r == nil {
		return
	}

	r.mu.Lock()
	delete(r.handlers, channel)
	hadSub := r.subs[channel]
	delete(r.subs, channel)
	r.mu.Unlock()

	if hadSub {
		if err := r.pubsub.Unsubscribe(r.ctx, channel); err != nil {
			slog.Warn("pubsub unsubscribe failed", "channel", channel, "err", err)
		}
	}
}

// StopRelay gracefully closes the Pub/Sub subscription.
// Call during server shutdown.
func StopRelay() {
	globalRelayMu.Lock()
	r := globalRelay
	globalRelay = nil
	globalRelayMu.Unlock()

	if r == nil {
		return
	}
	r.cancel()
	if r.pubsub != nil {
		r.pubsub.Close()
	}
	slog.Info("pubsub backplane relay stopped")
}
