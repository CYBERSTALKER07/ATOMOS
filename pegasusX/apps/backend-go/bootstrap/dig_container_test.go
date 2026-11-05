package bootstrap

import (
	"context"
	"strings"
	"testing"

	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"go.uber.org/dig"
)

func TestDigContainer_DryRunAndGraphResolution(t *testing.T) {
	t.Parallel()

	// 1. DryRun validation of the graph without executing constructors
	dryContainer, err := BuildCLIContainer(context.Background(), dig.DryRun(true))
	if err != nil {
		t.Fatalf("BuildCLIContainer(DryRun) error: %v", err)
	}
	if err := dryContainer.Invoke(func(p HubConsumerParams) {}); err != nil {
		t.Fatalf("DryRun Invoke(HubConsumerParams) failed: %v", err)
	}

	// 2. Real container resolution with Decorate override for Config
	c, err := BuildCLIContainer(context.Background())
	if err != nil {
		t.Fatalf("BuildCLIContainer error: %v", err)
	}

	if err := c.Decorate(func() *Config {
		return &Config{
			KafkaBrokers:      "broker-1:9092, broker-2:9092",
			KafkaAuthMode:     "sasl_scram",
			KafkaSASLUsername: "pegasus-cli",
		}
	}); err != nil {
		t.Fatalf("Decorate(*Config) error: %v", err)
	}

	if err := c.Invoke(func(p HubConsumerParams, enterprise cache.EnterpriseBackend, cli CLIInfraParams) {
		if len(p.AllHubs) != 8 {
			t.Fatalf("expected 8 hubs in group:\"ws_hubs\", got %d", len(p.AllHubs))
		}
		if p.RetailerHub == nil || p.DriverHub == nil || p.TelemetryHub == nil || p.PlatformAdminHub == nil {
			t.Fatal("expected all named hubs to be non-nil")
		}
		if enterprise == nil {
			t.Fatal("expected dig.As(cache.EnterpriseBackend) to resolve")
		}
		if len(cli.Brokers) != 2 || cli.Brokers[0] != "broker-1:9092" || cli.Brokers[1] != "broker-2:9092" {
			t.Fatalf("unexpected kafka_brokers: %v", cli.Brokers)
		}
		if cli.KafkaAuth.Username != "pegasus-cli" {
			t.Fatalf("unexpected KafkaAuth username: %q", cli.KafkaAuth.Username)
		}
	}); err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}

	// 3. Visualize DOT graph
	dot, err := VisualizeContainer(c)
	if err != nil {
		t.Fatalf("VisualizeContainer error: %v", err)
	}
	if !strings.Contains(dot, "digraph") {
		t.Fatalf("expected DOT digraph output, got %q", dot)
	}
}
