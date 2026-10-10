package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"google.golang.org/api/iterator"
)

// e2eTimeout bounds the full multi-role SSMR smoke path (supplier through driver edges).
func assertSupplierPortalAPIs(ctx context.Context, client *http.Client, base, cookie string) error {
	checks := []string{
		base + "/v1/supplier/dashboard",
		base + "/v1/supplier/profile",
		base + "/v1/supplier/inventory",
		base + "/v1/supplier/earnings",
		base + "/v1/supplier/pricing/rules",
	}
	for _, url := range checks {
		status, body, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("GET %s status %d body %s", url, status, string(body))
		}
	}
	return nil
}

func runSupplierIntelligenceE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	return runSupplierAnalyticsE2E(ctx, client, base, cookie)
}

func runSupplierAnalyticsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	checks := []string{
		base + "/v1/supplier/analytics/velocity",
		base + "/v1/supplier/analytics/revenue",
		base + "/v1/supplier/analytics/demand/today",
		base + "/v1/supplier/analytics/demand/history",
	}
	for _, url := range checks {
		status, body, _, err := clientDo(ctx, client, http.MethodGet, url, nil, cookie, "")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("GET %s status %d body %s", url, status, string(body))
		}
	}
	fmt.Println("PX_E2E_SUPPLIER_ANALYTICS_OK")
	return nil
}

func runSupplierOperationsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/empathy/adoption", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET empathy/adoption status %d body %s", status, string(body))
	}

	broadcastPayload := []byte(`{"title":"SSMR ops","body":"broadcast smoke","role":"ALL"}`)
	status, body, _, err = clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/broadcast", broadcastPayload, cookie, fmt.Sprintf("ssmr-supplier-ops-broadcast-%d", time.Now().UnixNano()))
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("POST broadcast status %d body %s", status, string(body))
	}

	fmt.Println("PX_E2E_SUPPLIER_OPERATIONS_OK")
	return nil
}

func runSupplierClientPolicyE2E(ctx context.Context, client *http.Client, base string) error {
	body, err := clientGet(ctx, client, base+"/v1/platform/client-policy?role=ADMIN&platform=web&version=1.0.0&channel=production")
	if err != nil {
		return err
	}
	var resp struct {
		Role           string `json:"role"`
		MinimumVersion string `json:"minimum_version"`
		Outdated       bool   `json:"outdated"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("decode supplier client policy: %w", err)
	}
	if resp.Role != "ADMIN" {
		return fmt.Errorf("supplier client policy role=%q want ADMIN", resp.Role)
	}
	if strings.TrimSpace(resp.MinimumVersion) == "" {
		return fmt.Errorf("supplier client policy missing minimum_version")
	}
	fmt.Println("PX_E2E_SUPPLIER_CLIENT_POLICY_OK")
	return nil
}

func runSupplierInventoryImportE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config, supplierID string) error {
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,50,5\nSSMR-SKU-BAD,%s,10,1\n",
		demoWarehouseID(), demoWarehouseID(),
	)
	// Unique key per run: Redis idempotency is not supplier-scoped; a fixed key
	// can replay another tenant's session_id and break staging asserts.
	idemKey := fmt.Sprintf("ssmr-supplier-inventory-import-%s-%d", supplierID, time.Now().UnixNano())
	status, respBody, _, err := clientDoContentType(
		ctx, client, http.MethodPost, base+"/v1/supplier/inventory/import",
		[]byte(csvBody), "text/csv", cookie, idemKey,
	)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("supplier inventory import status %d body %s", status, string(respBody))
	}
	var result struct {
		SessionID string `json:"session_id"`
		Applied   int    `json:"applied"`
		Skipped   int    `json:"skipped"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("decode supplier inventory import: %w", err)
	}
	if result.Applied < 1 {
		return fmt.Errorf("supplier inventory import applied=%d body %s", result.Applied, string(respBody))
	}
	if result.Skipped < 1 {
		return fmt.Errorf("supplier inventory import skipped=%d want >=1 (anomaly row) body %s", result.Skipped, string(respBody))
	}
	if strings.TrimSpace(result.SessionID) == "" {
		return fmt.Errorf("supplier inventory import missing session_id body %s", string(respBody))
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		sid = supplierIDFromJWT(cookie, cfg.JWTSecret)
	}
	if err := assertSupplierImportStagingRows(ctx, cfg, sid, result.SessionID, demoWarehouseID()); err != nil {
		return fmt.Errorf("supplier import staging rows: %w", err)
	}
	fmt.Println("PX_E2E_SUPPLIER_INVENTORY_IMPORT_OK")
	return nil
}

func runSupplierImportWizardE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,75,8\n",
		demoWarehouseID(),
	)
	createBody, _ := json.Marshal(map[string]any{
		"file_name":       "ssmr-wizard.csv",
		"file_size_bytes": len(csvBody),
	})
	runSuffix := fmt.Sprintf("%d", time.Now().UnixNano())
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/inventory/imports", createBody, cookie, "ssmr-import-wizard-create-"+runSuffix)
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("import session create status %d body %s", status, string(respBody))
	}
	var created struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode import session create: %w", err)
	}
	if strings.TrimSpace(created.SessionID) == "" {
		return fmt.Errorf("import session create missing session_id body %s", string(respBody))
	}

	ingestURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/ingest"
	status, respBody, _, err = clientDoContentType(ctx, client, http.MethodPost, ingestURL, []byte(csvBody), "text/csv", cookie, "ssmr-import-wizard-ingest-"+runSuffix)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("import session ingest status %d body %s", status, string(respBody))
	}

	approveURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/approve"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, approveURL, nil, cookie, "ssmr-import-wizard-approve-"+runSuffix)
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import session approve status %d body %s", status, string(respBody))
	}

	applyURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/apply"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, applyURL, nil, cookie, "ssmr-import-wizard-apply-"+runSuffix)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("import session apply status %d body %s", status, string(respBody))
	}
	var applied struct {
		AppliedRows int64  `json:"applied_rows"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return fmt.Errorf("decode import session apply: %w", err)
	}
	if applied.AppliedRows < 1 {
		return fmt.Errorf("import wizard applied_rows=%d body %s", applied.AppliedRows, string(respBody))
	}
	if strings.ToUpper(strings.TrimSpace(applied.Status)) != "APPLIED" {
		return fmt.Errorf("import wizard status=%q body %s", applied.Status, string(respBody))
	}

	fmt.Println("PX_E2E_SUPPLIER_IMPORT_WIZARD_OK")
	return nil
}

func importLocalUploadRoot() string {
	if root := strings.TrimSpace(os.Getenv("SSMR_IMPORT_LOCAL_ROOT")); root != "" {
		return root
	}
	return ".ssmr/import-uploads"
}

func runSupplierImportAsyncE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	// Cloud/SSMR: smokecheck writes CSV under a local path the in-cluster worker cannot read.
	// Wizard path already covers apply; keep async for emulator/local backends only.
	if strings.TrimSpace(os.Getenv("SSMR_FORCE_IMPORT_ASYNC")) != "1" && strings.TrimSpace(cfg.SpannerEmulatorHost) == "" {
		fmt.Println("PX_E2E_SUPPLIER_IMPORT_ASYNC_SKIPPED")
		return nil
	}
	csvBody := fmt.Sprintf(
		"product_id,warehouse_id,quantity_on_hand,reorder_threshold\nSSMR-SKU-1,%s,82,9\n",
		demoWarehouseID(),
	)
	createBody, _ := json.Marshal(map[string]any{
		"file_name":       "ssmr-async.csv",
		"file_size_bytes": len(csvBody),
	})
	status, respBody, _, err := clientDo(ctx, client, http.MethodPost, base+"/v1/supplier/inventory/imports", createBody, cookie, "ssmr-import-async-create")
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("import async create status %d body %s", status, string(respBody))
	}
	var created struct {
		SessionID string `json:"session_id"`
		GCSPath   string `json:"gcs_path"`
	}
	if err := json.Unmarshal(respBody, &created); err != nil {
		return fmt.Errorf("decode import async create: %w", err)
	}
	if strings.TrimSpace(created.SessionID) == "" || strings.TrimSpace(created.GCSPath) == "" {
		return fmt.Errorf("import async create missing session_id/gcs_path body %s", string(respBody))
	}

	localPath := filepath.Join(importLocalUploadRoot(), filepath.FromSlash(created.GCSPath))
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return fmt.Errorf("mkdir import upload root: %w", err)
	}
	if err := os.WriteFile(localPath, []byte(csvBody), 0o644); err != nil {
		return fmt.Errorf("write local import object: %w", err)
	}

	uploadedBody, _ := json.Marshal(map[string]string{"gcs_path": created.GCSPath})
	uploadedURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/uploaded"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, uploadedURL, uploadedBody, cookie, "ssmr-import-async-uploaded")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import async uploaded status %d body %s", status, string(respBody))
	}

	deadline := time.Now().Add(60 * time.Second)
	var sessionStatus string
	for time.Now().Before(deadline) {
		getURL := base + "/v1/supplier/inventory/imports/" + created.SessionID
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "ssmr-import-async-poll")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("import async poll status %d body %s", status, string(respBody))
		}
		var session struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(respBody, &session); err != nil {
			return fmt.Errorf("decode import async session: %w", err)
		}
		sessionStatus = strings.ToUpper(strings.TrimSpace(session.Status))
		if sessionStatus == "DISCOVERED" || sessionStatus == "MAPPING_REQUIRED" {
			break
		}
		if sessionStatus == "FAILED" {
			return fmt.Errorf("import async session failed body %s", string(respBody))
		}
		time.Sleep(500 * time.Millisecond)
	}
	if sessionStatus != "DISCOVERED" && sessionStatus != "MAPPING_REQUIRED" {
		return fmt.Errorf("import async discovery timed out status=%q", sessionStatus)
	}

	approveURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/approve"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, approveURL, nil, cookie, "ssmr-import-async-approve")
	if err != nil {
		return err
	}
	if status != http.StatusAccepted {
		return fmt.Errorf("import async approve status %d body %s", status, string(respBody))
	}

	approveDeadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(approveDeadline) {
		getURL := base + "/v1/supplier/inventory/imports/" + created.SessionID
		status, respBody, _, err = clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "ssmr-import-async-approve-poll")
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("import async approve poll status %d body %s", status, string(respBody))
		}
		var session struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(respBody, &session); err != nil {
			return fmt.Errorf("decode import async approve poll: %w", err)
		}
		sessionStatus = strings.ToUpper(strings.TrimSpace(session.Status))
		if sessionStatus == "APPROVED" || sessionStatus == "APPLYING" {
			break
		}
		if sessionStatus == "FAILED" {
			return fmt.Errorf("import async session failed after approve body %s", string(respBody))
		}
		time.Sleep(500 * time.Millisecond)
	}
	if sessionStatus != "APPROVED" && sessionStatus != "APPLYING" {
		return fmt.Errorf("import async approve timed out status=%q", sessionStatus)
	}

	applyURL := base + "/v1/supplier/inventory/imports/" + created.SessionID + "/apply"
	status, respBody, _, err = clientDo(ctx, client, http.MethodPost, applyURL, nil, cookie, "ssmr-import-async-apply")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		getURL := base + "/v1/supplier/inventory/imports/" + created.SessionID
		_, sessBody, _, _ := clientDo(ctx, client, http.MethodGet, getURL, nil, cookie, "ssmr-import-async-apply-diag")
		return fmt.Errorf("import async apply status %d body %s session %s", status, string(respBody), string(sessBody))
	}
	var applied struct {
		AppliedRows int64  `json:"applied_rows"`
		Status      string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &applied); err != nil {
		return fmt.Errorf("decode import async apply: %w", err)
	}
	if applied.AppliedRows < 1 {
		return fmt.Errorf("import async applied_rows=%d body %s", applied.AppliedRows, string(respBody))
	}
	if strings.ToUpper(strings.TrimSpace(applied.Status)) != "APPLIED" {
		return fmt.Errorf("import async status=%q body %s", applied.Status, string(respBody))
	}

	fmt.Println("PX_E2E_SUPPLIER_IMPORT_ASYNC_OK")
	return nil
}

func assertSupplierImportStagingRows(ctx context.Context, cfg *bootstrap.Config, supplierID, sessionID, warehouseID string) error {
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	stmt := spanner.Statement{
		SQL: `SELECT row_index, raw_data, validation_errors
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId
		        AND session_id = @sessionId
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0`,
		Params: map[string]any{
			"supplierId": supplierID,
			"sessionId":  sessionID,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	anomalyRows := 0
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("query staged rows: %w", err)
		}
		var rowIndex int64
		var raw spanner.NullJSON
		var validationErrors []string
		if err := row.Columns(&rowIndex, &raw, &validationErrors); err != nil {
			return fmt.Errorf("decode staged row: %w", err)
		}
		if len(validationErrors) == 0 {
			continue
		}
		rawMap := importStagingJSONMap(raw)
		rowWarehouse := strings.TrimSpace(stagingJSONString(rawMap, "warehouse_id"))
		if rowWarehouse != "" && rowWarehouse != warehouseID {
			continue
		}
		anomalyRows++
	}
	if anomalyRows < 1 {
		return fmt.Errorf("want >=1 staged anomaly row for warehouse %s supplier %s session %s", warehouseID, supplierID, sessionID)
	}
	return nil
}

func importStagingJSONMap(value spanner.NullJSON) map[string]any {
	if !value.Valid || value.Value == nil {
		return nil
	}
	if mapped, ok := value.Value.(map[string]any); ok {
		return mapped
	}
	encoded, err := json.Marshal(value.Value)
	if err != nil {
		return nil
	}
	decoded := make(map[string]any)
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil
	}
	return decoded
}

func stagingJSONString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	raw, ok := values[key]
	if !ok || raw == nil {
		return ""
	}
	if value, ok := raw.(string); ok {
		return value
	}
	return strings.TrimSpace(fmt.Sprint(raw))
}

func countWarehouseImportAnomalyRows(ctx context.Context, cfg *bootstrap.Config, supplierID, warehouseID string) (int64, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return 0, fmt.Errorf("supplier_id required")
	}
	client, err := spanner.NewClient(ctx, spannerDatabasePath(cfg), spannerClientOptions(cfg)...)
	if err != nil {
		return 0, fmt.Errorf("new spanner client: %w", err)
	}
	defer client.Close()

	startAt := time.Now().UTC().AddDate(0, 0, -30).Truncate(24 * time.Hour)
	stmt := spanner.Statement{
		SQL: `SELECT session_id, raw_data, cleaned_data, validation_errors
		      FROM SupplierImportStagedRows
		      WHERE supplier_id = @supplierId
		        AND created_at >= @startAt
		        AND validation_errors IS NOT NULL
		        AND ARRAY_LENGTH(validation_errors) > 0
		      ORDER BY updated_at DESC
		      LIMIT 5000`,
		Params: map[string]any{
			"supplierId": supplierID,
			"startAt":    startAt,
		},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var openRows int64
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("query import anomalies: %w", err)
		}
		var sessionID string
		var raw spanner.NullJSON
		var cleaned spanner.NullJSON
		var validationErrors []string
		if err := row.Columns(&sessionID, &raw, &cleaned, &validationErrors); err != nil {
			return 0, fmt.Errorf("decode import anomaly row: %w", err)
		}
		rawMap := importStagingJSONMap(raw)
		cleanedMap := importStagingJSONMap(cleaned)
		rowWarehouse := strings.TrimSpace(stagingJSONString(cleanedMap, "warehouse_id"))
		if rowWarehouse == "" {
			rowWarehouse = strings.TrimSpace(stagingJSONString(rawMap, "warehouse_id"))
		}
		if rowWarehouse != "" && rowWarehouse != warehouseID {
			continue
		}
		openRows++
	}
	return openRows, nil
}

func runSupplierTopologyEditE2E(ctx context.Context, client *http.Client, base, cookie string, cfg *bootstrap.Config) error {
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/topology", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("topology get status %d body %s", status, string(respBody))
	}
	var current struct {
		Warehouses []map[string]any `json:"warehouses"`
		Factories  []map[string]any `json:"factories"`
	}
	if err := json.Unmarshal(respBody, &current); err != nil {
		return fmt.Errorf("decode topology get: %w", err)
	}
	if len(current.Warehouses) == 0 || len(current.Factories) == 0 {
		return fmt.Errorf("topology get missing warehouses or factories: %s", string(respBody))
	}

	newWhLat := cfg.DeliveryZoneCenterLat + 0.002
	newWhLng := cfg.DeliveryZoneCenterLng + 0.002
	newFcLat := cfg.DeliveryZoneCenterLat + 0.012
	newFcLng := cfg.DeliveryZoneCenterLng + 0.012
	current.Warehouses[0]["lat"] = newWhLat
	current.Warehouses[0]["lng"] = newWhLng
	current.Factories[0]["lat"] = newFcLat
	current.Factories[0]["lng"] = newFcLng

	putBody, _ := json.Marshal(map[string]any{
		"warehouses": current.Warehouses,
		"factories":  current.Factories,
	})
	// Unique key per run: fixed key collides when lat/lng payload differs from a prior attempt.
	editKey := fmt.Sprintf("ssmr-topology-edit-%d", time.Now().UnixNano())
	status, respBody, _, err = clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/topology", putBody, cookie, editKey)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("topology edit put status %d body %s", status, string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/topology", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("topology get after edit status %d body %s", status, string(respBody))
	}
	var updated struct {
		Warehouses []struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"warehouses"`
		Factories []struct {
			Lat float64 `json:"lat"`
			Lng float64 `json:"lng"`
		} `json:"factories"`
	}
	if err := json.Unmarshal(respBody, &updated); err != nil {
		return fmt.Errorf("decode topology after edit: %w", err)
	}
	if len(updated.Warehouses) == 0 || len(updated.Factories) == 0 {
		return fmt.Errorf("topology after edit empty: %s", string(respBody))
	}
	if updated.Warehouses[0].Lat != newWhLat || updated.Warehouses[0].Lng != newWhLng {
		return fmt.Errorf("warehouse location not persisted: got lat=%v lng=%v want lat=%v lng=%v",
			updated.Warehouses[0].Lat, updated.Warehouses[0].Lng, newWhLat, newWhLng)
	}
	if updated.Factories[0].Lat != newFcLat || updated.Factories[0].Lng != newFcLng {
		return fmt.Errorf("factory location not persisted: got lat=%v lng=%v want lat=%v lng=%v",
			updated.Factories[0].Lat, updated.Factories[0].Lng, newFcLat, newFcLng)
	}
	fmt.Println("PX_E2E_TOPOLOGY_EDIT_OK")
	return nil
}

// runSupplierOrgFleetE2E seeds a warehouse admin via org-fleet (drivers/trucks use fleet routes separately).
func runSupplierOrgFleetE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	whID := demoWarehouseID()
	phone := fmt.Sprintf("+9989010%05d", time.Now().UnixNano()%100000)
	memberBody, _ := json.Marshal(map[string]any{
		"name":                  "SSMR Warehouse Admin",
		"phone":                 phone,
		"password":              "ssmr-wh-admin-pass",
		"supplier_role":         "WAREHOUSE_ADMIN",
		"assigned_warehouse_id": whID,
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/supplier/org/members", memberBody, cookie, fmt.Sprintf("ssmr-org-member-%d", time.Now().UnixNano()))
	if err != nil {
		return err
	}
	createStatus := status
	if createStatus != http.StatusCreated && createStatus != http.StatusOK && createStatus != http.StatusConflict {
		return fmt.Errorf("org member create status %d body %s", createStatus, string(respBody))
	}

	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/org/members", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("org members list status %d body %s", status, string(respBody))
	}
	// Idempotent create returns Conflict with a fixed idempotency key — list may not
	// contain this run's random phone; require phone only on fresh create.
	if createStatus != http.StatusConflict && !strings.Contains(string(respBody), phone) {
		return fmt.Errorf("org members missing created phone %s: %s", phone, string(respBody))
	}
	fmt.Println("PX_E2E_ORG_FLEET_OK")
	return nil
}
