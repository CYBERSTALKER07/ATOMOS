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

func TestPurgeStaleToken_NilAndNoOpClient(t *testing.T) {
	var nilClient *FCMClient
	if err := nilClient.PurgeStaleToken(nil, "token-1", "session-1"); err != nil {
		t.Fatalf("nil client PurgeStaleToken should return nil, got %v", err)
	}

	noOpClient := NewNoOpFCMClient(nil)
	if err := noOpClient.PurgeStaleToken(nil, "token-1", "session-1"); err != nil {
		t.Fatalf("no-op client PurgeStaleToken without spanner should return nil, got %v", err)
	}
	if err := noOpClient.PurgeStaleToken(nil, "", ""); err != nil {
		t.Fatalf("empty token PurgeStaleToken should return nil, got %v", err)
	}
}
