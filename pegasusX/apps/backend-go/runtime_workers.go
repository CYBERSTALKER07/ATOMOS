package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/demand"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func startBackgroundWorkers(ctx context.Context, app *bootstrap.App) {
	// Worker tier publishes a liveness heartbeat so an api-only tier can detect
	// it and avoid double-running the notification consumer (P1-9).
	bootstrap.StartWorkerHeartbeat(ctx, app.RedisClient, slog.Default())
	if app.OutboxRelay != nil {
		go app.OutboxRelay.Start(ctx)
		slog.Info("outbox relay started")
	}
	if app.Spanner != nil {
		go outbox.StartSupplierIDBackfill(ctx, app.Spanner, slog.Default())
		if outbox.SupplierBackfillEnabled() {
			slog.Info("outbox supplier_id backfill worker started")
		}
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
	if app.ReturnsEventConsumer != nil {
		go app.ReturnsEventConsumer.Start(ctx)
		slog.Info("returns reverse-logistics consumer started")
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
	// P1-8: settlement-vs-captured reconciliation of stuck payment sessions.
	// On by default; the reconciler skips stub gateway refs and only advances
	// sessions stuck >15min via a real provider status check. Set
	// PAYMENT_RECONCILE_DISABLED=1 to turn it off.
	if app.WebhookReconciler != nil && os.Getenv("PAYMENT_RECONCILE_DISABLED") != "1" {
		go func() {
			// Initial jitter so a multi-replica rollout doesn't stampede the gateway.
			timer := time.NewTimer(30 * time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := app.WebhookReconciler.ReconcileStuckSessions(ctx); err != nil {
						slog.Warn("webhook reconciler pass failed", "err", err)
					}
				}
			}
		}()
		slog.Info("webhook reconciler worker started", "interval", "5m")
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
	if app.DemandService != nil {
		go app.DemandService.RunDensityWorker(ctx, 6*time.Hour)
		slog.Info("demand density worker started")
	}

	if app.ControlTowerWorker != nil {
		go app.ControlTowerWorker.Run(ctx)
		slog.Info("control tower playbook worker started")
	}
	if app.BillingTierConsumer != nil {
		go app.BillingTierConsumer.Start(ctx)
		slog.Info("billing tier consumer started")
	}
	if app.PartnerEventConsumer != nil {
		go app.PartnerEventConsumer.Start(ctx)
		slog.Info("partner webhook event consumer started")
	}
	if app.TwinEventConsumer != nil {
		go app.TwinEventConsumer.Start(ctx)
		slog.Info("digital twin event consumer started")
	}
	if app.PartnerWebhookDelivery != nil {
		go app.PartnerWebhookDelivery.Start(ctx, 15*time.Second)
		slog.Info("partner webhook delivery worker started")
	}
	if app.PartnerExportWorker != nil {
		go app.PartnerExportWorker.Start(ctx, 20*time.Second)
		slog.Info("partner export worker started")
	}
	if app.PartnerEdiInbound != nil {
		go app.PartnerEdiInbound.Start(ctx, 30*time.Second)
		slog.Info("partner edi inbound worker started")
	}
	if app.PartnerEdiOutbound != nil {
		go app.PartnerEdiOutbound.Start(ctx, 20*time.Second)
		slog.Info("partner edi outbound worker started")
	}
	if app.ARDunningWorker != nil {
		go app.ARDunningWorker.Start(ctx, time.Hour)
		slog.Info("ar dunning worker started", "enabled", os.Getenv("AR_DUNNING_ENABLED"))
	}
	// P1-6: Soliq EHF buyer-clearance poller (MySoliq path only; nil otherwise).
	if app.BuyerAcceptancePoller != nil {
		go app.BuyerAcceptancePoller.Run(ctx)
		slog.Info("buyer acceptance poller started")
	}
	// Gate-0: auto-confirm due AI preorders (default off until smoke).
	if app.OrderService != nil && os.Getenv("AUTO_CONFIRM_PREORDERS_ENABLED") == "1" {
		go func() {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					if err := app.OrderService.AutoConfirmDueOrders(ctx, 200); err != nil {
						slog.Warn("auto-confirm preorders sweep failed", "err", err)
					}
				}
			}
		}()
		slog.Info("auto-confirm preorders sweeper started")
	}
	// Wave C3.2: expire parked POS carts past 24h TTL (no-op when POS_HOLDS_ENABLED off).
	if app.RetailerService != nil {
		go app.RetailerService.RunPosHoldsSweeper(ctx, 15*time.Minute)
		slog.Info("retailer pos holds sweeper started")
		// Wave C4.1: assist SLA breach worker (no-op when ASSIST_SLA_ENABLED off).
		go app.RetailerService.RunAssistSLAWorker(ctx, time.Minute)
		slog.Info("retailer assist sla worker started")
		// L2.4: auto-order draft worker (no-op when AUTO_ORDER_WORKER_ENABLED off).
		go app.RetailerService.RunAutoOrderWorker(ctx, 15*time.Minute)
		slog.Info("retailer auto-order worker started")
	}
	if app.FactoryService != nil {
		go app.FactoryService.RunFactorySLABreachWorker(ctx, 5*time.Minute)
		slog.Info("factory SLA breach worker started")
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

// startNotificationConsumerIfNoWorker starts the notification consumer on an
// api-tier only when no worker-tier heartbeat is present. This restores FCM
// push + inbox persistence for single-tier api deployments without
// double-firing when a worker tier is running (P1-9).
func startNotificationConsumerIfNoWorker(ctx context.Context, app *bootstrap.App) {
	if app.NotificationConsumer == nil {
		return
	}
	// RUN_MODE=all already started it via startBackgroundWorkers.
	if app.Config != nil && app.Config.RunsWorkers() {
		return
	}
	if bootstrap.WorkerLive(ctx, app.RedisClient) {
		slog.Info("worker tier live; api tier leaving notification consumer to worker")
		return
	}
	go app.NotificationConsumer.Start(ctx)
	slog.Warn("no worker tier heartbeat detected; api tier started notification consumer (push+inbox safety net)")
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
		app.PlatformAdminHub,
	}
}

// locationBusEmitter returns the throttled outbox emitter for driver location
// when Spanner is available, else nil (bus emit disabled, WS/Redis unaffected).
func locationBusEmitter(app *bootstrap.App) telemetryroutes.LocationBusEmitter {
	if app == nil || app.Spanner == nil {
		return nil
	}
	return telemetryroutes.NewSpannerLocationBusEmitter(app.Spanner)
}
