package bootstrap

import (
	"context"
	"errors"
	"log/slog"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/bqexport"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
)

func TestStartSinkWorker_APIModeIsNoOp(t *testing.T) {
	orig := newBQSinkFromEnv
	t.Cleanup(func() { newBQSinkFromEnv = orig })
	called := false
	newBQSinkFromEnv = func(context.Context) (bqexport.Sink, error) {
		called = true
		return nil, errors.New("must not run in api mode")
	}

	cfg := testConfig()
	cfg.RunMode = RunModeAPI

	cleanup, err := StartSinkWorker(context.Background(), cfg, nil, slog.Default())
	if err != nil {
		t.Fatalf("unexpected error in API mode: %v", err)
	}
	cleanup()
	if called {
		t.Fatal("expected StartSinkWorker to skip sink initialization in RunModeAPI")
	}
}

func TestStartSinkWorker_ProductionFailsClosedWithoutBigQueryConfig(t *testing.T) {
	t.Setenv("PEGASUSX_ENV", "production")
	t.Setenv("BIGQUERY_PROJECT_ID", "")
	t.Setenv("BIGQUERY_LOCAL_SPOOL_DIR", t.TempDir())

	cfg := testConfig()
	cfg.RunMode = RunModeWorker
	cfg.RequireInfraAdapters = true

	if _, err := StartSinkWorker(context.Background(), cfg, nil, slog.Default()); err == nil {
		t.Fatal("expected fail-closed error in production when BIGQUERY_PROJECT_ID is unset")
	}
}

func TestNewBQSinkConsumer_WorkerModeWithJSONLSpoolAndMemoryCheckpoints(t *testing.T) {
	stubRuntimeConstructors(t)
	newKafkaRuntimeDLQWriter = func(string, string, kafkautil.ClientAuth) (kafkaRuntimeDLQWriter, error) {
		return &fakeKafkaDLQWriter{}, nil
	}
	t.Setenv("PEGASUSX_ENV", "local")
	t.Setenv("BIGQUERY_PROJECT_ID", "")
	t.Setenv("BIGQUERY_LOCAL_SPOOL_DIR", t.TempDir())

	cfg := testConfig()
	cfg.RunMode = RunModeWorker
	cfg.RequireInfraAdapters = false
	cfg.AllowMemoryFallback = true
	cfg.TestingMode = true

	consumer, exporter, cleanup, err := NewBQSinkConsumer(context.Background(), cfg, nil, slog.Default())
	if err != nil {
		t.Fatalf("NewBQSinkConsumer failed: %v", err)
	}
	if consumer == nil || exporter == nil || cleanup == nil {
		t.Fatalf("expected initialized consumer/exporter/cleanup, got consumer=%t exporter=%t cleanup=%t",
			consumer != nil, exporter != nil, cleanup != nil)
	}
	cleanup()
}
