package telemetry

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"cloud.google.com/go/spanner"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

func newSLOEmulatorClient(t *testing.T, ctx context.Context) *spanner.Client {
	t.Helper()
	host := strings.TrimSpace(os.Getenv("SPANNER_EMULATOR_HOST"))
	if host == "" {
		t.Skip("SPANNER_EMULATOR_HOST not set; skipping")
	}
	envOr := func(k, def string) string {
		if v := os.Getenv(k); v != "" {
			return v
		}
		return def
	}
	dbPath := fmt.Sprintf("projects/%s/instances/%s/databases/%s",
		envOr("SPANNER_PROJECT", "pegasusx-local"),
		envOr("SPANNER_INSTANCE", "pegasusx-instance"),
		envOr("SPANNER_DATABASE", "pegasusx-db"))
	client, err := spanner.NewClient(ctx, dbPath,
		option.WithEndpoint(host), option.WithoutAuthentication(),
		option.WithGRPCDialOption(grpc.WithInsecure()))
	if err != nil {
		t.Fatalf("spanner client: %v", err)
	}
	return client
}

func TestSLOCollector_EmitsMetrics(t *testing.T) {
	ctx := context.Background()
	client := newSLOEmulatorClient(t, ctx)
	defer client.Close()

	reg := prometheus.NewRegistry()
	c := NewSLOCollector(client, reg, nil)
	c.collect(ctx)

	fams, err := reg.Gather()
	if err != nil {
		t.Fatalf("gather: %v", err)
	}
	found := map[string]bool{}
	for _, f := range fams {
		found[f.GetName()] = true
	}
	for _, want := range []string{
		"void_outbox_lag_seconds", "void_fiscal_success_ratio", "void_capture_success_ratio",
		"void_outbox_dlq_depth", "void_partner_webhook_success_ratio",
	} {
		if !found[want] {
			t.Errorf("metric %s not emitted", want)
		}
	}
}

func TestSLOCollector_RatioNoTraffic(t *testing.T) {
	ctx := context.Background()
	client := newSLOEmulatorClient(t, ctx)
	defer client.Close()
	reg := prometheus.NewRegistry()
	c := NewSLOCollector(client, reg, nil)
	// Empty tables → no traffic → ratio must be 1.0 (not a breach).
	r, err := c.ratio(ctx, `SELECT COUNTIF(Status='SUCCESS'), COUNT(*) FROM OrderFiscalReceipts
		WHERE CreatedAt >= TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 HOUR)`)
	if err != nil {
		t.Fatalf("ratio: %v", err)
	}
	if r != 1.0 {
		t.Fatalf("no-traffic ratio = %v, want 1.0", r)
	}
}
