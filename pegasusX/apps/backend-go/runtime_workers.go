package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetryroutes"
	"github.com/pegasusx/pegasusx/apps/backend-go/warehouse"
	"github.com/pegasusx/pegasusx/apps/backend-go/workerlease"
	"github.com/pegasusx/pegasusx/apps/backend-go/ws"
)

func startLeasedWorker(ctx context.Context, app *bootstrap.App, leaseName string, ttl time.Duration, fn func(leaseCtx context.Context)) {
	if app == nil || app.LeaseManager == nil {
		slog.Warn("skipping background worker; lease manager unavailable", "lease", leaseName)
		return
	}
	workerlease.StartLeasedLoop(ctx, nil, app.LeaseManager, workerlease.Config{
		LeaseName: leaseName,
		HolderID:  app.WorkerHolderID,
		TTL:       ttl,
		Log:       slog.Default(),
	}, fn)
}

func startBackgroundWorkers(ctx context.Context, app *bootstrap.App) {
	if app == nil || (app.Config != nil && !app.Config.RunsWorkers()) {
		return
	}
	// Worker tier publishes a liveness heartbeat to Redis + Spanner WorkerLeases.
	bootstrap.StartWorkerHeartbeatWithLease(ctx, app.RedisClient, app.LeaseManager, app.WorkerHolderID, slog.Default())
	if app.OutboxRelay != nil {
		go app.OutboxRelay.Start(ctx)
		slog.Info("outbox relay started")
	}
	if app.Spanner != nil {
		startLeasedWorker(ctx, app, "worker:outbox_supplier_backfill", 2*time.Minute, func(leaseCtx context.Context) {
			outbox.StartSupplierIDBackfill(leaseCtx, app.Spanner, slog.Default())
		})
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
	if app.ClaimsEventConsumer != nil {
		go app.ClaimsEventConsumer.Start(ctx)
		slog.Info("claims event consumer started")
	}
	if app.ReturnsEventConsumer != nil {
		go app.ReturnsEventConsumer.Start(ctx)
		slog.Info("returns reverse-logistics consumer started")
	}
	if app.WarehouseService != nil {
		startLeasedWorker(ctx, app, "worker:warehouse_auto_dispatch", 60*time.Second, func(leaseCtx context.Context) {
			warehouse.StartAutoDispatchWorker(leaseCtx, app.WarehouseService, warehouse.AutoDispatchWorkerConfig{})
		})
		slog.Info("warehouse auto-dispatch worker started")
		startLeasedWorker(ctx, app, "worker:warehouse_dispatch_warmer", 60*time.Second, func(leaseCtx context.Context) {
			warehouse.StartDispatchPlanWarmer(leaseCtx, app.WarehouseService, warehouse.DispatchPlanWarmerConfig{})
		})
		slog.Info("dispatch plan warmer started")
	}
	if app.WebhookInbox != nil && app.PaymentService != nil {
		startLeasedWorker(ctx, app, "worker:webhook_inbox_reconciler", 60*time.Second, func(leaseCtx context.Context) {
			app.WebhookInbox.StartReconciler(leaseCtx, app.PaymentService, 0)
		})
		slog.Info("webhook inbox reconciler started")
	}
	// P1-8: settlement-vs-captured reconciliation of stuck payment sessions.
	// On by default; the reconciler skips stub gateway refs and only advances
	// sessions stuck >15min via a real provider status check. Set
	// PAYMENT_RECONCILE_DISABLED=1 to turn it off.
	if app.WebhookReconciler != nil && os.Getenv("PAYMENT_RECONCILE_DISABLED") != "1" {
		startLeasedWorker(ctx, app, "worker:payment_webhook_reconciler", 2*time.Minute, func(leaseCtx context.Context) {
			// Initial jitter so a multi-replica rollout doesn't stampede the gateway.
			timer := time.NewTimer(30 * time.Second)
			select {
			case <-leaseCtx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			ticker := time.NewTicker(5 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-leaseCtx.Done():
					return
				case <-ticker.C:
					if err := app.WebhookReconciler.ReconcileStuckSessions(leaseCtx); err != nil {
						slog.Warn("webhook reconciler pass failed", "err", err)
					}
				}
			}
		})
		slog.Info("webhook reconciler worker started", "interval", "5m")
	}
	if app.ReplenishmentEngine != nil && os.Getenv("REPLENISHMENT_CRON_DISABLED") != "1" {
		startLeasedWorker(ctx, app, "worker:replenishment_cron", 2*time.Minute, func(leaseCtx context.Context) {
			app.ReplenishmentEngine.StartCron(leaseCtx)
			<-leaseCtx.Done()
		})
		slog.Info("replenishment engine cron started")
	}
	if app.FactoryPlanning != nil {
		startLeasedWorker(ctx, app, "worker:factory_planning_cron", 2*time.Minute, func(leaseCtx context.Context) {
			app.FactoryPlanning.StartPlanningCron(leaseCtx)
			<-leaseCtx.Done()
		})
		slog.Info("factory planning cron started")
	}
	if app.LaborCapacityService != nil {
		startLeasedWorker(ctx, app, "worker:labor_driver_score", 2*time.Minute, func(leaseCtx context.Context) {
			app.LaborCapacityService.RunDriverScoreWorker(leaseCtx, 24*time.Hour)
		})
		startLeasedWorker(ctx, app, "worker:labor_capacity_snapshot", 2*time.Minute, func(leaseCtx context.Context) {
			app.LaborCapacityService.RunCapacitySnapshotWorker(leaseCtx, 1*time.Hour)
		})
		slog.Info("labor capacity workers started")
	}
	if app.RouteAnalyticsWorker != nil {
		startLeasedWorker(ctx, app, "worker:route_analytics", 2*time.Minute, func(leaseCtx context.Context) {
			app.RouteAnalyticsWorker.RunNightlyWorker(leaseCtx, 24*time.Hour)
		})
		slog.Info("route analytics worker started")
	}
	if app.OrderService != nil {
		startLeasedWorker(ctx, app, "worker:order_saga_recovery", 60*time.Second, func(leaseCtx context.Context) {
			order.StartSagaRecoveryWorker(leaseCtx, app.OrderService, 15*time.Second)
		})
		slog.Info("order saga recovery worker started", "interval", "15s")
		startLeasedWorker(ctx, app, "worker:order_shop_closed", 60*time.Second, func(leaseCtx context.Context) {
			app.OrderService.RunShopClosedWorker(leaseCtx, 30*time.Second)
		})
		slog.Info("order shop closed worker started", "interval", "30s")
	}
	supplierID := ""
	if app.Supplier.SupplierID != "" {
		supplierID = app.Supplier.SupplierID
	}
	if app.CashReconEscalation != nil {
		startLeasedWorker(ctx, app, "worker:cash_recon_escalation", 2*time.Minute, func(leaseCtx context.Context) {
			app.CashReconEscalation.RunNightlyWorker(leaseCtx, 24*time.Hour)
		})
		slog.Info("cash reconciliation escalation worker started")
	}
	if app.ReorderSuggestionWorker != nil && supplierID != "" {
		startLeasedWorker(ctx, app, "worker:reorder_suggestion_batch", 2*time.Minute, func(leaseCtx context.Context) {
			app.ReorderSuggestionWorker.RunBatchWorker(leaseCtx, supplierID, 12*time.Hour)
		})
		slog.Info("reorder suggestion batch worker started")
	}

	if app.DemandService != nil {
		startLeasedWorker(ctx, app, "worker:demand_sensing", 2*time.Minute, func(leaseCtx context.Context) {
			app.DemandService.RunDemandSensingWorker(leaseCtx, 12*time.Hour)
		})
		slog.Info("demand sensing worker started", "interval", "12h")
		startLeasedWorker(ctx, app, "worker:demand_density", 2*time.Minute, func(leaseCtx context.Context) {
			app.DemandService.RunDensityWorker(leaseCtx, 6*time.Hour)
		})
		slog.Info("demand density worker started")
	}

	if app.ControlTowerWorker != nil {
		startLeasedWorker(ctx, app, "worker:control_tower_playbook", 2*time.Minute, func(leaseCtx context.Context) {
			app.ControlTowerWorker.Run(leaseCtx)
		})
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
	if app.OntologyEventConsumer != nil {
		go app.OntologyEventConsumer.Start(ctx)
		slog.Info("ontology projector consumer started")
	}
	if app.BQSinkConsumer != nil {
		go app.BQSinkConsumer.Start(ctx)
		slog.Info("bigquery sink consumer started")
	}
	if app.PartnerWebhookDelivery != nil {
		startLeasedWorker(ctx, app, "worker:partner_webhook_delivery", 60*time.Second, func(leaseCtx context.Context) {
			app.PartnerWebhookDelivery.Start(leaseCtx, 15*time.Second)
		})
		slog.Info("partner webhook delivery worker started")
	}
	if app.PartnerExportWorker != nil {
		startLeasedWorker(ctx, app, "worker:partner_export", 60*time.Second, func(leaseCtx context.Context) {
			app.PartnerExportWorker.Start(leaseCtx, 20*time.Second)
		})
		slog.Info("partner export worker started")
	}
	if app.PartnerEdiInbound != nil {
		startLeasedWorker(ctx, app, "worker:partner_edi_inbound", 60*time.Second, func(leaseCtx context.Context) {
			app.PartnerEdiInbound.Start(leaseCtx, 30*time.Second)
		})
		slog.Info("partner edi inbound worker started")
	}
	if app.PartnerEdiOutbound != nil {
		startLeasedWorker(ctx, app, "worker:partner_edi_outbound", 60*time.Second, func(leaseCtx context.Context) {
			app.PartnerEdiOutbound.Start(leaseCtx, 20*time.Second)
		})
		slog.Info("partner edi outbound worker started")
	}
	if app.ARDunningWorker != nil {
		startLeasedWorker(ctx, app, "worker:ar_dunning", 2*time.Minute, func(leaseCtx context.Context) {
			app.ARDunningWorker.Start(leaseCtx, time.Hour)
		})
		slog.Info("ar dunning worker started", "enabled", os.Getenv("AR_DUNNING_ENABLED"))
	}
	if app.ARPromiseBreachWorker != nil {
		startLeasedWorker(ctx, app, "worker:ar_promise_breach", 2*time.Minute, func(leaseCtx context.Context) {
			app.ARPromiseBreachWorker.Start(leaseCtx, 15*time.Minute)
		})
		slog.Info("ar promise breach worker started")
	}
	// P1-6: Soliq EHF buyer-clearance poller (MySoliq path only; nil otherwise).
	if app.BuyerAcceptancePoller != nil {
		startLeasedWorker(ctx, app, "worker:buyer_acceptance_poller", 60*time.Second, func(leaseCtx context.Context) {
			app.BuyerAcceptancePoller.Run(leaseCtx)
		})
		slog.Info("buyer acceptance poller started")
	}
	// Gate-0: auto-confirm due AI preorders (default off until smoke).
	if app.OrderService != nil && os.Getenv("AUTO_CONFIRM_PREORDERS_ENABLED") == "1" {
		startLeasedWorker(ctx, app, "worker:auto_confirm_preorders", 60*time.Second, func(leaseCtx context.Context) {
			ticker := time.NewTicker(time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-leaseCtx.Done():
					return
				case <-ticker.C:
					if err := app.OrderService.AutoConfirmDueOrders(leaseCtx, 200); err != nil {
						slog.Warn("auto-confirm preorders sweep failed", "err", err)
					}
				}
			}
		})
		slog.Info("auto-confirm preorders sweeper started")
	}
	// Wave C3.2: expire parked POS carts past 24h TTL (no-op when POS_HOLDS_ENABLED off).
	if app.RetailerService != nil {
		startLeasedWorker(ctx, app, "worker:retailer_pos_holds", 2*time.Minute, func(leaseCtx context.Context) {
			app.RetailerService.RunPosHoldsSweeper(leaseCtx, 15*time.Minute)
		})
		slog.Info("retailer pos holds sweeper started")
		// Wave C4.1: assist SLA breach worker (no-op when ASSIST_SLA_ENABLED off).
		startLeasedWorker(ctx, app, "worker:retailer_assist_sla", 60*time.Second, func(leaseCtx context.Context) {
			app.RetailerService.RunAssistSLAWorker(leaseCtx, time.Minute)
		})
		slog.Info("retailer assist sla worker started")
		// L2.4: auto-order draft worker (no-op when AUTO_ORDER_WORKER_ENABLED off).
		startLeasedWorker(ctx, app, "worker:retailer_auto_order", 2*time.Minute, func(leaseCtx context.Context) {
			app.RetailerService.RunAutoOrderWorker(leaseCtx, 15*time.Minute)
		})
		slog.Info("retailer auto-order worker started")
	}
	if app.FactoryService != nil {
		startLeasedWorker(ctx, app, "worker:factory_sla_breach", 2*time.Minute, func(leaseCtx context.Context) {
			app.FactoryService.RunFactorySLABreachWorker(leaseCtx, 5*time.Minute)
		})
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

// locationBusEmitter returns the direct Kafka emitter for driver location when available,
// falling back to Spanner outbox only when Kafka publisher is unconfigured.
// This decouples high-frequency driver GPS telemetry from Spanner write mutations.
func locationBusEmitter(app *bootstrap.App) telemetryroutes.LocationBusEmitter {
	if app == nil {
		return nil
	}
	if app.OutboxPublisher != nil {
		return telemetryroutes.NewDirectKafkaLocationBusEmitter(app.OutboxPublisher, slog.Default())
	}
	if app.Spanner != nil {
		return telemetryroutes.NewSpannerLocationBusEmitter(app.Spanner)
	}
	return nil
}
