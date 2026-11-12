package main

import (
	"io"
	"log/slog"
	"net/http"
	"testing"
)

func TestAPIHTTPServerBoundsKeepAliveAndLeavesStreamingDeadlinesUnset(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	srv := newAPIHTTPServer(":8080", http.NewServeMux(), logger)

	if srv.Addr != ":8080" {
		t.Fatalf("addr = %q", srv.Addr)
	}
	if srv.Handler == nil {
		t.Fatal("handler is nil")
	}
	if srv.ReadHeaderTimeout != httpReadHeaderTimeout {
		t.Fatalf("ReadHeaderTimeout = %s", srv.ReadHeaderTimeout)
	}
	if srv.IdleTimeout != httpIdleTimeout {
		t.Fatalf("IdleTimeout = %s", srv.IdleTimeout)
	}
	if srv.MaxHeaderBytes != httpMaxHeaderBytes {
		t.Fatalf("MaxHeaderBytes = %d", srv.MaxHeaderBytes)
	}
	if srv.ErrorLog == nil {
		t.Fatal("ErrorLog is nil")
	}
	if srv.ReadTimeout != 0 || srv.WriteTimeout != 0 {
		t.Fatalf("ReadTimeout=%s WriteTimeout=%s; streaming deadlines belong on the hijacked socket", srv.ReadTimeout, srv.WriteTimeout)
	}
}
