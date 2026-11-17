package main

import (
	"log/slog"
	"net/http"
	"time"
)

const (
	// httpReadHeaderTimeout caps slow-header (slowloris) attacks before a
	// handler runs. It does not cover the request body; body size is capped
	// by MaxBytesMiddleware.
	httpReadHeaderTimeout = 10 * time.Second
	// httpIdleTimeout bounds a keep-alive connection waiting for the next
	// request. Hijacked WebSocket and SSE connections are no longer owned by
	// the server once the handler takes the socket, so this does not cut them.
	httpIdleTimeout = 120 * time.Second
	// httpMaxHeaderBytes matches net/http.DefaultMaxHeaderBytes and locks it
	// so a later edit cannot accept unbounded headers.
	httpMaxHeaderBytes = 1 << 20
)

// newAPIHTTPServer builds the process HTTP server for the public API and the
// worker health listener.
//
// ReadTimeout and WriteTimeout stay at zero on purpose. This process upgrades
// WebSockets and holds SSE streams. A server-wide read or write deadline would
// abort those connections. Streaming deadlines live on the hijacked socket
// (websocket pong wait and write wait). Request-class backpressure, rate
// limits, and circuit breaking live in ReliabilityMiddleware.
func newAPIHTTPServer(addr string, handler http.Handler, logger *slog.Logger) *http.Server {
	if logger == nil {
		logger = slog.Default()
	}
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: httpReadHeaderTimeout,
		IdleTimeout:       httpIdleTimeout,
		MaxHeaderBytes:    httpMaxHeaderBytes,
		ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
	}
}
