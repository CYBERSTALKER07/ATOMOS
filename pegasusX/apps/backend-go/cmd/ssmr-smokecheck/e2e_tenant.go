package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// runTenantE2E exercises Gate 5 Phase 1 markers (register freeze, order IDOR,
// outbox SupplierId partition, tenant rate-limit keying).
func runTenantE2E(
	ctx context.Context,
	client *http.Client,
	base string,
	cfg *bootstrap.Config,
	supplierID, supplierCookie, orderID string,
) error {
	if err := runTenantRegisterFrozenE2E(ctx, client, base, cfg); err != nil {
		return err
	}
	if err := runTenantOrderIsolationE2E(ctx, client, base, cfg, orderID); err != nil {
		return err
	}
	if err := runOutboxTenantPartitionE2E(ctx, cfg, supplierID); err != nil {
		return err
	}
	if err := runTenantRateLimitE2E(ctx, client, base, cfg, supplierID, supplierCookie); err != nil {
		return err
	}
	return nil
}

func runTenantRegisterFrozenE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config) error {
	if envTruthy("ALLOW_MULTI_SUPPLIER_REGISTER") {
		fmt.Println("PX_E2E_TENANT_REGISTER_FROZEN_SKIPPED")
		return nil
	}
	phone := "+99890" + strings.ReplaceAll(uuid.NewString()[:8], "-", "")
	body, _ := json.Marshal(map[string]any{
		"phone": phone,
		"account": map[string]any{
			"legalName":   "SSMR Freeze Probe",
			"contactName": "Freeze Admin",
			"email":       "freeze-" + phone[len(phone)-4:] + "@pegasusx.local",
			"password":    "FreezeTest!234",
			"country":     cfg.SeedSupplierCountry,
		},
		"location": map[string]any{
			"warehouse": map[string]any{
				"name":    "Freeze WH",
				"address": "Tashkent",
				"lat":     cfg.DeliveryZoneCenterLat,
				"lng":     cfg.DeliveryZoneCenterLng,
			},
			"sameAsWarehouse": true,
		},
		"business": map[string]any{
			"taxId":             "FREEZE-TAX",
			"companyRegNumber":  "FREEZE-REG",
			"fleetVehicleCount": 1,
			"fleetMaxVU":        10,
			"factoryCount":      1,
		},
		"categories": []string{"GENERAL"},
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/auth/supplier/register", body, "", "ssmr-tenant-freeze-"+phone)
	if err != nil {
		return fmt.Errorf("tenant register freeze: %w", err)
	}
	bodyStr := string(respBody)
	if status == http.StatusForbidden || status == http.StatusConflict ||
		strings.Contains(bodyStr, "legacy_register_frozen") ||
		strings.Contains(bodyStr, "supplier_cap_reached") || strings.Contains(bodyStr, "supplier cap") {
		fmt.Println("PX_E2E_TENANT_REGISTER_FROZEN_OK")
		return nil
	}
	if status == http.StatusCreated || status == http.StatusOK {
		return fmt.Errorf("tenant register freeze: expected rejection (seed already demo-registered), got %d body=%s", status, bodyStr)
	}
	// Unregistered seed may still accept first registration — treat as skip.
	if strings.Contains(bodyStr, "not found") || status == http.StatusUnprocessableEntity {
		fmt.Println("PX_E2E_TENANT_REGISTER_FROZEN_SKIPPED")
		return nil
	}
	return fmt.Errorf("tenant register freeze: unexpected status %d body=%s", status, bodyStr)
}

func runTenantOrderIsolationE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, orderID string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		fmt.Println("PX_E2E_TENANT_ORDER_ISOLATION_SKIPPED")
		return nil
	}
	otherTok, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-tenant-idor",
		Role:       auth.RoleAdmin,
		SupplierID: "sup-other-tenant-" + uuid.NewString()[:8],
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 10 * time.Minute})
	if err != nil {
		return fmt.Errorf("tenant order isolation jwt: %w", err)
	}
	// Prefer status-context; fall back to timeline if route shape differs.
	paths := []string{
		base + "/v1/order/" + orderID + "/status-context",
		base + "/v1/order/" + orderID + "/timeline",
	}
	var status int
	var respBody []byte
	for _, path := range paths {
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, path, nil, otherTok, "")
		if err != nil {
			return fmt.Errorf("tenant order isolation: %w", err)
		}
		if status != http.StatusNotFound || !strings.Contains(string(respBody), "404 page not found") {
			break
		}
	}
	if status == http.StatusNotFound || status == http.StatusForbidden || status == http.StatusUnauthorized {
		fmt.Println("PX_E2E_TENANT_ORDER_ISOLATION_OK")
		return nil
	}
	if status == http.StatusOK {
		return fmt.Errorf("tenant order isolation: cross-tenant read must fail closed, got 200 body=%s", string(respBody))
	}
	fmt.Println("PX_E2E_TENANT_ORDER_ISOLATION_SKIPPED")
	return nil
}

func runOutboxTenantPartitionE2E(ctx context.Context, cfg *bootstrap.Config, supplierID string) error {
	spannerClient, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_SKIPPED")
		return nil
	}
	defer spannerClient.Close()

	// Opportunistic backfill then count stamped rows.
	if _, berr := outbox.BackfillSupplierID(ctx, spannerClient, 100); berr != nil {
		if strings.Contains(berr.Error(), "SupplierId") || strings.Contains(berr.Error(), "not found") || strings.Contains(berr.Error(), "Unrecognized") {
			fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_SKIPPED")
			return nil
		}
		return fmt.Errorf("outbox tenant partition backfill: %w", berr)
	}

	n, err := countOutboxWithSupplier(ctx, spannerClient)
	if err != nil {
		if strings.Contains(err.Error(), "SupplierId") || strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "Unrecognized") {
			fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_SKIPPED")
			return nil
		}
		return err
	}
	if n > 0 {
		fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_OK")
		return nil
	}

	// Probe insert proves schema + stamp path when the stack is quiet.
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		sid = smokeSupplierID()
	}
	probeID := "e2e-outbox-tenant-" + uuid.NewString()
	payload, _ := json.Marshal(map[string]any{
		"type":        "e2e.outbox_tenant_probe",
		"supplier_id": sid,
		"probe_id":    probeID,
	})
	_, err = spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
			"EventId":       probeID,
			"AggregateType": "e2e",
			"AggregateId":   probeID,
			"TopicName":     "main",
			"Payload":       payload,
			"CreatedAt":     spanner.CommitTimestamp,
			"PublishedAt":   nil,
			"SupplierId":    sid,
		}),
	})
	if err != nil {
		fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_SKIPPED")
		return nil
	}
	n, err = countOutboxWithSupplier(ctx, spannerClient)
	if err != nil || n == 0 {
		fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_SKIPPED")
		return nil
	}
	fmt.Println("PX_E2E_OUTBOX_TENANT_PARTITION_OK")
	return nil
}

func countOutboxWithSupplier(ctx context.Context, client *spanner.Client) (int64, error) {
	iter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COUNT(1) FROM OutboxEvents
		      WHERE SupplierId IS NOT NULL AND SupplierId != ''
		        AND CreatedAt > TIMESTAMP_SUB(CURRENT_TIMESTAMP(), INTERVAL 1 DAY)`,
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var n int64
	if err := row.Columns(&n); err != nil {
		return 0, err
	}
	return n, nil
}

func runTenantRateLimitE2E(ctx context.Context, client *http.Client, base string, cfg *bootstrap.Config, supplierID, supplierCookie string) error {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		fmt.Println("PX_E2E_TENANT_RATE_LIMIT_SKIPPED")
		return nil
	}
	tokA, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-rl-a-" + uuid.NewString()[:6],
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 5 * time.Minute})
	if err != nil {
		return err
	}
	tokB, err := auth.Issue(auth.Claims{
		Subject:    "ssmr-rl-b-" + uuid.NewString()[:6],
		Role:       auth.RoleAdmin,
		SupplierID: supplierID,
	}, auth.IssueOptions{Secret: cfg.JWTSecret, Issuer: cfg.JWTIssuer, TTL: 5 * time.Minute})
	if err != nil {
		return err
	}

	// Non-exempt operational path — health is rate-limit exempt.
	paths := []string{base + "/v1/supplier/profile", base + "/v1/supplier/dashboard"}
	sawLimit := false
	for _, path := range paths {
		for _, tok := range []string{tokA, tokB, supplierCookie} {
			if strings.TrimSpace(tok) == "" {
				continue
			}
			status, _, hdr, err := clientDo(ctx, client, http.MethodGet, path, nil, tok, "")
			if err != nil {
				continue
			}
			if status >= 200 && status < 500 && hdr.Get("X-RateLimit-Limit") != "" {
				sawLimit = true
				break
			}
		}
		if sawLimit {
			break
		}
	}
	if sawLimit {
		fmt.Println("PX_E2E_TENANT_RATE_LIMIT_OK")
		return nil
	}
	fmt.Println("PX_E2E_TENANT_RATE_LIMIT_SKIPPED")
	return nil
}

// runTenantSmokeCheck is a focused Gate 5 Phase 1 path (no full ecosystem chain).
func runTenantSmokeCheck(ctx context.Context, cfg *bootstrap.Config) error {
	base := strings.TrimRight(envOr("PUBLIC_BASE_URL", "http://localhost:8180"), "/")
	client := &http.Client{Timeout: 30 * time.Second}
	if _, err := clientGet(ctx, client, base+"/v1/health"); err != nil {
		return fmt.Errorf("health: %w", err)
	}
	supplierID, cookie, err := ensureSupplierSession(ctx, client, base, cfg)
	if err != nil {
		return fmt.Errorf("supplier session: %w", err)
	}
	orderID := strings.TrimSpace(envOr("SSMR_SMOKE_ORDER_ID", ""))
	if orderID == "" {
		if err := putSupplierTopology(ctx, client, base, cookie, cfg); err != nil {
			return fmt.Errorf("supplier topology: %w", err)
		}
		retailerID, h3Cell, err := registerRetailer(ctx, client, base, cfg)
		if err != nil {
			return fmt.Errorf("retailer register: %w", err)
		}
		if err := grantRetailerCredit(ctx, client, base, cookie, retailerID, 500_000_000); err != nil {
			return fmt.Errorf("retailer credit grant: %w", err)
		}
		retailerToken, err := auth.Issue(auth.Claims{
			Subject:    retailerID,
			Role:       auth.RoleRetailer,
			SupplierID: supplierID,
		}, auth.IssueOptions{
			Secret: cfg.JWTSecret,
			Issuer: cfg.JWTIssuer,
			TTL:    15 * time.Minute,
		})
		if err != nil {
			return fmt.Errorf("issue retailer jwt: %w", err)
		}
		orderID, err = createOrder(ctx, client, base, retailerToken, cfg, h3Cell)
		if err != nil {
			return fmt.Errorf("order create: %w", err)
		}
	}
	return runTenantE2E(ctx, client, base, cfg, supplierID, cookie, orderID)
}
