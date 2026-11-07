// Package bootstrap provides uber-go/dig dependency injection containers for
// CLI tools, test harnesses, and composition-root graph validation.
package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/cache"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafkautil"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
	"go.uber.org/dig"
)

// HubGraphOut is a dig.Out result object that exposes each role-scoped *ws.Hub
// both as a named dependency and as a member of the "ws_hubs" value group.
type HubGraphOut struct {
	dig.Out

	RetailerHub      *ws.Hub   `name:"retailer_hub"`
	SupplierHub      *ws.Hub   `name:"supplier_hub"`
	DriverHub        *ws.Hub   `name:"driver_hub"`
	PayloadHub       *ws.Hub   `name:"payload_hub"`
	WarehouseHub     *ws.Hub   `name:"warehouse_hub"`
	FactoryHub       *ws.Hub   `name:"factory_hub"`
	TelemetryHub     *ws.Hub   `name:"telemetry_hub"`
	PlatformAdminHub *ws.Hub   `name:"platform_admin_hub"`
	AllHubs          []*ws.Hub `group:"ws_hubs,flatten"`
}

// HubConsumerParams is a dig.In parameter object for consumers that require
// either specific named WebSocket hubs or the complete "ws_hubs" value group.
type HubConsumerParams struct {
	dig.In

	Logger           *slog.Logger
	CacheBackend     cache.Backend `optional:"true"`
	RetailerHub      *ws.Hub       `name:"retailer_hub"`
	SupplierHub      *ws.Hub       `name:"supplier_hub"`
	DriverHub        *ws.Hub       `name:"driver_hub"`
	PayloadHub       *ws.Hub       `name:"payload_hub"`
	WarehouseHub     *ws.Hub       `name:"warehouse_hub"`
	FactoryHub       *ws.Hub       `name:"factory_hub"`
	TelemetryHub     *ws.Hub       `name:"telemetry_hub"`
	PlatformAdminHub *ws.Hub       `name:"platform_admin_hub"`
	AllHubs          []*ws.Hub     `group:"ws_hubs"`
}

// ProvideHubGraph constructs all role-scoped WebSocket hubs from a shared
// cache.Backend and *slog.Logger, returning a HubGraphOut result struct.
func ProvideHubGraph(backend cache.Backend, log *slog.Logger) HubGraphOut {
	if log == nil {
		log = slog.Default()
	}
	retailer := ws.NewHub("retailer", backend, log)
	supplier := ws.NewHub("supplier", backend, log)
	driver := ws.NewHub("driver", backend, log)
	payload := ws.NewHub("payload", backend, log)
	warehouse := ws.NewHub("warehouse", backend, log)
	factory := ws.NewHub("factory", backend, log)
	telemetry := ws.NewHub("telemetry", backend, log)
	platformAdmin := ws.NewHub("platform-admin", backend, log)

	return HubGraphOut{
		RetailerHub:      retailer,
		SupplierHub:      supplier,
		DriverHub:        driver,
		PayloadHub:       payload,
		WarehouseHub:     warehouse,
		FactoryHub:       factory,
		TelemetryHub:     telemetry,
		PlatformAdminHub: platformAdmin,
		AllHubs: []*ws.Hub{
			retailer,
			supplier,
			driver,
			payload,
			warehouse,
			factory,
			telemetry,
			platformAdmin,
		},
	}
}

// ExtractAppGraph exposes singletons from a constructed *App into a dig.Out
// result struct so downstream dig/fx consumers can inject narrow dependencies.
type AppGraphOut struct {
	dig.Out

	Cache            *cache.Cache
	IdempotencyStore idempotency.Store
	RetailerHub      *ws.Hub   `name:"retailer_hub"`
	SupplierHub      *ws.Hub   `name:"supplier_hub"`
	DriverHub        *ws.Hub   `name:"driver_hub"`
	PayloadHub       *ws.Hub   `name:"payload_hub"`
	WarehouseHub     *ws.Hub   `name:"warehouse_hub"`
	FactoryHub       *ws.Hub   `name:"factory_hub"`
	TelemetryHub     *ws.Hub   `name:"telemetry_hub"`
	PlatformAdminHub *ws.Hub   `name:"platform_admin_hub"`
	AllHubs          []*ws.Hub `group:"ws_hubs,flatten"`
}

// ProvideAppSubgraph projects narrow interfaces and named hubs from *App.
func ProvideAppSubgraph(app *App) AppGraphOut {
	if app == nil {
		return AppGraphOut{}
	}
	return AppGraphOut{
		Cache:            app.Cache,
		IdempotencyStore: app.Idempotency,
		RetailerHub:      app.RetailerHub,
		SupplierHub:      app.SupplierHub,
		DriverHub:        app.DriverHub,
		PayloadHub:       app.PayloadHub,
		WarehouseHub:     app.WarehouseHub,
		FactoryHub:       app.FactoryHub,
		TelemetryHub:     app.TelemetryHub,
		PlatformAdminHub: app.PlatformAdminHub,
		AllHubs: []*ws.Hub{
			app.RetailerHub,
			app.SupplierHub,
			app.DriverHub,
			app.PayloadHub,
			app.WarehouseHub,
			app.FactoryHub,
			app.TelemetryHub,
			app.PlatformAdminHub,
		},
	}
}

// CLIInfraParams groups dependencies injected into CLI subcommands via dig.In.
type CLIInfraParams struct {
	dig.In

	Config      *Config
	Logger      *slog.Logger
	KafkaAuth   kafkautil.ClientAuth
	Brokers     []string          `name:"kafka_brokers"`
	Cache       *cache.Cache      `optional:"true"`
	CacheEngine cache.Backend     `optional:"true"`
}

// BuildCLIContainer constructs a lazy uber-go/dig container for CLI utilities
// (such as cmd/replay-dlq, cmd/apply-migration, and test harnesses).
// Expensive resources like *spanner.Client are registered lazily and only
// instantiated if an invoked command explicitly requests them.
func BuildCLIContainer(ctx context.Context, opts ...dig.Option) (*dig.Container, error) {
	baseOpts := []dig.Option{
		dig.RecoverFromPanics(),
		dig.DeferAcyclicVerification(),
	}
	baseOpts = append(baseOpts, opts...)
	c := dig.New(baseOpts...)

	providers := []struct {
		ctor any
		opts []dig.ProvideOption
	}{
		{ctor: func() context.Context { return ctx }},
		{ctor: func() *slog.Logger { return slog.Default() }},
		{ctor: LoadConfig},
		{
			ctor: cache.NewInMemoryBackend,
			opts: []dig.ProvideOption{
				dig.As(new(cache.Backend), new(cache.EnterpriseBackend)),
			},
		},
		{
			ctor: func(b cache.Backend, log *slog.Logger) *cache.Cache {
				return cache.New(b, log)
			},
		},
		{ctor: ProvideHubGraph},
		{
			ctor: func(cfg *Config) kafkautil.ClientAuth {
				return kafkautil.ClientAuth{
					Mode:     cfg.KafkaAuthMode,
					Username: cfg.KafkaSASLUsername,
				}
			},
		},
		{
			ctor: func(cfg *Config) ([]string, error) {
				raw := strings.Split(cfg.KafkaBrokers, ",")
				brokers := make([]string, 0, len(raw))
				for _, b := range raw {
					if trimmed := strings.TrimSpace(b); trimmed != "" {
						brokers = append(brokers, trimmed)
					}
				}
				if len(brokers) == 0 {
					return nil, fmt.Errorf("bootstrap: KafkaBrokers not configured")
				}
				return brokers, nil
			},
			opts: []dig.ProvideOption{dig.Name("kafka_brokers")},
		},
		{
			// Lazy Spanner provider: only runs when an Invoke requests *spanner.Client.
			ctor: func(ctx context.Context, cfg *Config) (*spanner.Client, error) {
				if strings.TrimSpace(cfg.SpannerDatabase) == "" {
					return nil, fmt.Errorf("bootstrap: SpannerDatabase not configured")
				}
				return spanner.NewClient(ctx, cfg.SpannerDatabase)
			},
		},
	}

	for _, p := range providers {
		if err := c.Provide(p.ctor, p.opts...); err != nil {
			return nil, fmt.Errorf("dig provide failed: %w", err)
		}
	}
	return c, nil
}

// VisualizeContainer renders the dig container's dependency graph in Graphviz DOT format.
func VisualizeContainer(c *dig.Container) (string, error) {
	if c == nil {
		return "", fmt.Errorf("nil dig container")
	}
	var buf bytes.Buffer
	if err := dig.Visualize(c, &buf); err != nil {
		return "", err
	}
	return buf.String(), nil
}
