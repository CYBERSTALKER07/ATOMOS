package notifications

import (
	"log/slog"
	"os"
	"testing"
)

func TestNoOpFCMClient_IsNoOp(t *testing.T) {
	c := NewNoOpFCMClient(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError})))
	if !c.IsNoOp() {
		t.Fatal("expected no-op client")
	}
}

func TestNilFCMClient_IsNoOp(t *testing.T) {
	var c *FCMClient
	if !c.IsNoOp() {
		t.Fatal("nil client should be no-op")
	}
}
