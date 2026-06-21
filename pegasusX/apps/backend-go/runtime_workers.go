package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/kafka"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func startBackgroundWorkers(ctx context.Context, app *bootstrap.App) {
	if app.OutboxRelay != nil {
		go app.OutboxRelay.Start(ctx)
		slog.Info("outbox relay started")
	}
	if app.Cache != nil {
		go app.Cache.StartInvalidationSubscriber(ctx)
		slog.Info("cache invalidation subscriber started")
	}
	if app.NotificationConsumer != nil {
		go app.NotificationConsumer.Start(ctx)
		slog.Info("notification consumer started")
	}
	if app.OrderEventConsumer != nil {
		go app.OrderEventConsumer.Start(ctx)
		slog.Info("order event consumer started")
	}
	if app.WarehouseEventConsumer != nil {
		go app.WarehouseEventConsumer.Start(ctx)
		slog.Info("warehouse event consumer started")
	}
	if app.WarehouseService != nil {
		go warehouse.StartAutoDispatchWorker(ctx, app.WarehouseService, warehouse.AutoDispatchWorkerConfig{})
		slog.Info("warehouse auto-dispatch worker started")
		go warehouse.StartDispatchPlanWarmer(ctx, app.WarehouseService, warehouse.DispatchPlanWarmerConfig{})
		slog.Info("dispatch plan warmer started")
	}
	if app.ReplenishmentEngine != nil && os.Getenv("REPLENISHMENT_CRON_DISABLED") != "1" {
		app.ReplenishmentEngine.StartCron(ctx)
		slog.Info("replenishment engine cron started")
	}

	streamProcessor := kafka.NewAnalyticsStreamProcessor()
	dummyStream := make(chan []byte)
	go streamProcessor.Start(ctx, dummyStream)
	slog.Info("kafka stream processor started")
}

func startHubRelaySubscribers(ctx context.Context, hubs []*ws.Hub) {
	for _, hub := range hubs {
		if hub == nil {
			continue
		}
		go hub.StartRelaySubscriber(ctx)
	}
	slog.Info("ws relay subscribers started", "hub_count", len(hubs))
}

func hubList(app *bootstrap.App) []*ws.Hub {
	return []*ws.Hub{
		app.RetailerHub,
		app.SupplierHub,
		app.DriverHub,
		app.PayloadHub,
		app.WarehouseHub,
		app.FactoryHub,
		app.TelemetryHub,
	}
}
