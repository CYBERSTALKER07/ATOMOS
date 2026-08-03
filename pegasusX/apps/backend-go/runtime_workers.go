package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
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
	if app.WebhookInbox != nil && app.PaymentService != nil {
		go app.WebhookInbox.StartReconciler(ctx, app.PaymentService, 0)
		slog.Info("webhook inbox reconciler started")
	}
	if app.ReplenishmentEngine != nil && os.Getenv("REPLENISHMENT_CRON_DISABLED") != "1" {
		app.ReplenishmentEngine.StartCron(ctx)
		slog.Info("replenishment engine cron started")
	}
	if app.LaborCapacityService != nil {
		go app.LaborCapacityService.RunDriverScoreWorker(ctx, 24*time.Hour)
		go app.LaborCapacityService.RunCapacitySnapshotWorker(ctx, 1*time.Hour)
		slog.Info("labor capacity workers started")
	}
	if app.CreditScoreWorker != nil {
		go app.CreditScoreWorker.RunNightlyWorker(ctx, 24*time.Hour)
		slog.Info("retailer credit score worker started")
	}
	if app.RouteAnalyticsWorker != nil {
		go app.RouteAnalyticsWorker.RunNightlyWorker(ctx, 24*time.Hour)
		slog.Info("route analytics worker started")
	}
	supplierID := ""
	if app.Supplier.SupplierID != "" {
		supplierID = app.Supplier.SupplierID
	}
	if app.CashReconEscalation != nil {
		go app.CashReconEscalation.RunNightlyWorker(ctx, 24*time.Hour)
		slog.Info("cash reconciliation escalation worker started")
	}
	if app.ReorderSuggestionWorker != nil && supplierID != "" {
		go app.ReorderSuggestionWorker.RunBatchWorker(ctx, supplierID, 12*time.Hour)
		slog.Info("reorder suggestion batch worker started")
	}
	if app.Config.WeatherWorkerEnabled && app.DemandService != nil {
		// Use globally configured center. For Phase 1, treating this as city-level default region.
		weatherCfg := demand.WeatherConfig{
			BaseURL:        app.Config.WeatherBaseURL,
			UpdateInterval: 6 * time.Hour,
			LookaheadDays:  14,
			Locations: []demand.Location{
				{
					Scope: "city:Tashkent", // Default to Tashkent logic for all retailers
					Lat:   app.Config.DeliveryZoneCenterLat,
					Lng:   app.Config.DeliveryZoneCenterLng,
				},
			},
		}
		go app.DemandService.RunWeatherIngestionWorker(ctx, weatherCfg)
		slog.Info("weather ingestion worker started", "lookahead_days", weatherCfg.LookaheadDays)
	}

	if app.ControlTowerWorker != nil {
		go app.ControlTowerWorker.Run(ctx)
		slog.Info("control tower playbook worker started")
	}
	if app.BillingTierConsumer != nil {
		go app.BillingTierConsumer.Start(ctx)
		slog.Info("billing tier consumer started")
	}
	// Wave C3.2: expire parked POS carts past 24h TTL (no-op when POS_HOLDS_ENABLED off).
	if app.RetailerService != nil {
		go app.RetailerService.RunPosHoldsSweeper(ctx, 15*time.Minute)
		slog.Info("retailer pos holds sweeper started")
		go app.RetailerService.RunAssistSLAWorker(ctx, time.Minute)
		slog.Info("retailer assist sla worker started")
	}
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
