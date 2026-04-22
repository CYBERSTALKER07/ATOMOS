package bootstrap

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"cloud.google.com/go/spanner"

	"backend-go/notifications"
	"backend-go/order"
	"backend-go/telemetry"
	"backend-go/ws"
	"config"
)

// initFCM constructs the FCM client honoring FIREBASE_CREDENTIALS_PATH. An
// empty path or a failed init both yield a NoOp client — mirroring the
// graceful-degradation semantics used in production prior to this refactor.
func initFCM(spannerClient *spanner.Client) *notifications.FCMClient {
	credPath := os.Getenv("FIREBASE_CREDENTIALS_PATH")
	var client *notifications.FCMClient
	if credPath == "" {
		client = notifications.NewNoOpFCMClient()
	} else {
		c, err := notifications.InitFCM(credPath)
		if err != nil {
			log.Printf("[COMMUNICATION SPINE] FCM init failed (%v) — falling back to no-op mode", err)
			client = notifications.NewNoOpFCMClient()
		} else {
			client = c
		}
	}
	client.SpannerClient = spannerClient // enables stale-token auto-purge
	return client
}

// buildOrderDeps assembles the three order-flow dep bundles that close over
// FCM, the retailer hub, the driver hub, and the telemetry fleet broadcast.
// These closures were previously inlined inside main(); extracting them is
// mechanical (behaviour preserved verbatim).
func buildOrderDeps(
	fcm *notifications.FCMClient,
	retailerHub *ws.RetailerHub,
	driverHub *ws.DriverHub,
) (order.ShopClosedDeps, order.EarlyCompleteDeps, order.NegotiationDeps) {

	adminBroadcast := func(payload interface{}) {
		data, _ := json.Marshal(payload)
		telemetry.FleetHub.BroadcastToAdmins(data)
	}
	supplierPush := func(supplierID string, payload interface{}) bool {
		data, _ := json.Marshal(payload)
		telemetry.FleetHub.BroadcastToAdmins(data)
		return true
	}
	notifyUser := func(ctx context.Context, userID, role, title, body string, data map[string]string) {
		fcm.SendDataMessage(userID, data)
	}

	shopClosed := order.ShopClosedDeps{
		RetailerPush:   retailerHub.PushToRetailer,
		DriverPush:     driverHub.PushToDriver,
		AdminBroadcast: adminBroadcast,
		NotifyUser:     notifyUser,
	}

	earlyComplete := order.EarlyCompleteDeps{
		SupplierPush: supplierPush,
		DriverPush:   driverHub.PushToDriver,
		NotifyUser:   notifyUser,
	}

	negotiation := order.NegotiationDeps{
		SupplierPush: supplierPush,
		DriverPush:   driverHub.PushToDriver,
		NotifyUser:   notifyUser,
	}

	return shopClosed, earlyComplete, negotiation
}

// resolveCORSAllowlist returns the origin allowlist. Priority order:
//  1. cfg.CORSAllowedOrigins (env CORS_ALLOWED_ORIGINS, comma-separated)
//  2. localhost dev defaults (only when #1 is empty and IsDevelopment())
//
// In non-development environments with an empty cfg list, the allowlist is
// empty — EnableCORS will still pass pattern-matched origins (ngrok, expo,
// LAN) through, but no bare hosts are trusted.
func resolveCORSAllowlist(cfg *config.EnvConfig) map[string]bool {
	out := make(map[string]bool, len(cfg.CORSAllowedOrigins))
	for _, o := range cfg.CORSAllowedOrigins {
		if o != "" {
			out[o] = true
		}
	}
	if len(out) == 0 && cfg.IsDevelopment() {
		for _, o := range []string{
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:3002",
			"http://localhost:8081",
			"http://localhost:19006",
		} {
			out[o] = true
		}
	}
	return out
}
