package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
)

// runGapClosureE2E exercises finance gap-closure APIs (cash recon, credit notes, reverse logistics, prefs).
func runGapClosureE2E(
	ctx context.Context,
	client *http.Client,
	base, cookie, supplierID, retailerToken string,
	cfg *bootstrap.Config,
	orderID string,
) error {
	if err := runGapClosureNotificationPrefsE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("notification prefs: %w", err)
	}
	if err := runGapClosureRoutePerformanceE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("route performance: %w", err)
	}
	if err := runGapClosureCreditProfilesE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("credit profiles: %w", err)
	}
	reconID, err := runGapClosureCashReconE2E(ctx, client, base, cookie, supplierID, cfg)
	if err != nil {
		return fmt.Errorf("cash recon: %w", err)
	}
	if err := runGapClosureExceptionResolveCashE2E(ctx, client, base, cookie, reconID); err != nil {
		return fmt.Errorf("exception resolve cash: %w", err)
	}
	cnID, sku, err := runGapClosureCreditNoteE2E(ctx, client, base, cookie, orderID)
	if err != nil {
		return fmt.Errorf("credit note: %w", err)
	}
	if err := runGapClosureExceptionResolveCreditNoteE2E(ctx, client, base, cookie, cnID); err != nil {
		return fmt.Errorf("exception resolve credit note: %w", err)
	}
	if err := runGapClosureReverseLogisticsE2E(ctx, client, base, supplierID, cfg, cnID, sku); err != nil {
		return fmt.Errorf("reverse logistics: %w", err)
	}
	if err := runGapClosureReorderE2E(ctx, client, base, cookie); err != nil {
		return fmt.Errorf("reorder: %w", err)
	}
	fmt.Println("PX_E2E_GAP_CLOSURE_OK")
	return nil
}

func runGapClosureNotificationPrefsE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/user/notification-preferences", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET prefs status %d body %s", status, string(body))
	}
	patchBody, _ := json.Marshal(map[string]any{
		"preferences": []map[string]any{
			{"event_type": "cash_reconciliation.created", "channel": "inbox", "enabled": true},
		},
	})
	status, body, _, err = clientDo(ctx, client, http.MethodPatch, base+"/v1/user/notification-preferences", patchBody, cookie, "ssmr-gap-prefs-patch")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("PATCH prefs status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_NOTIFICATION_PREFS_OK")
	return nil
}

func runGapClosureRoutePerformanceE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/route-performance", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET route-performance status %d body %s", status, string(body))
	}
	fmt.Println("PX_E2E_ROUTE_PERFORMANCE_OK")
	return nil
}

func runGapClosureCreditProfilesE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, body, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/credit-profiles?limit=10", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("GET credit-profiles status %d body %s", status, string(body))
	}
	var resp struct {
		Profiles []json.RawMessage `json:"profiles"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		// tolerate alternate envelope
		if !strings.Contains(string(body), "retailer_id") {
			return fmt.Errorf("decode credit-profiles: %w", err)
		}
	}
	fmt.Println("PX_E2E_CREDIT_SCORE_NOTIFY_OK")
	return nil
}

func runGapClosureCashReconE2E(ctx context.Context, client *http.Client, base, cookie, supplierID string, cfg *bootstrap.Config) (string, error) {
	driverID := envOr("SSMR_SMOKE_DRIVER_ID", "ssmr-driver-1")
	driverToken, err := issueRoleJWT(cfg, auth.Claims{
		Subject:    driverID,
		Role:       auth.RoleDriver,
		SupplierID: supplierID,
		HomeNodeID: demoWarehouseID(),
	})
	if err != nil {
		return "", fmt.Errorf("issue driver jwt: %w", err)
	}
	submitBody, _ := json.Marshal(map[string]any{
		"declared_cash_minor": 1000,
		"driver_note":         "ssmr gap closure mismatch",
	})
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/driver/cash-reconciliations", submitBody, driverToken, "ssmr-gap-cash-recon-submit")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", fmt.Errorf("driver submit status %d body %s", status, string(respBody))
	}
	var submitResp struct {
		ReconciliationID string `json:"reconciliation_id"`
	}
	_ = json.Unmarshal(respBody, &submitResp)
	reconID := strings.TrimSpace(submitResp.ReconciliationID)
	if reconID == "" {
		var wrap struct {
			Reconciliation struct {
				ReconciliationID string `json:"reconciliation_id"`
			} `json:"reconciliation"`
		}
		_ = json.Unmarshal(respBody, &wrap)
		reconID = wrap.Reconciliation.ReconciliationID
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/cash-reconciliations?limit=20", nil, cookie, "")
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", fmt.Errorf("supplier list status %d body %s", status, string(respBody))
	}
	if reconID == "" {
		var list struct {
			Reconciliations []struct {
				ReconciliationID string `json:"reconciliation_id"`
			} `json:"reconciliations"`
		}
		if err := json.Unmarshal(respBody, &list); err == nil && len(list.Reconciliations) > 0 {
			reconID = list.Reconciliations[0].ReconciliationID
		}
	}
	if reconID == "" {
		return "", fmt.Errorf("no reconciliation_id from submit or list")
	}
	fmt.Println("PX_E2E_CASH_RECON_OK")
	return reconID, nil
}

func runGapClosureExceptionResolveCashE2E(ctx context.Context, client *http.Client, base, cookie, reconID string) error {
	resolveBody, _ := json.Marshal(map[string]string{"note": "ssmr accept"})
	path := base + "/v1/supplier/exceptions/CASH_DISCREPANCY/" + reconID + "/resolve"
	status, respBody, _, err := clientPost(ctx, client, path, resolveBody, cookie, "ssmr-gap-resolve-cash-"+reconID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("resolve cash status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_EXCEPTION_RESOLVE_OK")
	return nil
}

func runGapClosureCreditNoteE2E(ctx context.Context, client *http.Client, base, cookie, orderID string) (cnID, sku string, err error) {
	if strings.TrimSpace(orderID) == "" {
		return "", "", fmt.Errorf("order_id required for credit note e2e")
	}
	status, respBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/credit-notes/order-lines?order_id="+orderID, nil, cookie, "")
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", fmt.Errorf("order-lines status %d body %s", status, string(respBody))
	}
	var linesResp struct {
		Lines []struct {
			OrderLineID string `json:"order_line_id"`
			Sku         string `json:"sku"`
			Qty         int64  `json:"qty"`
		} `json:"lines"`
	}
	if err := json.Unmarshal(respBody, &linesResp); err != nil {
		return "", "", fmt.Errorf("decode order-lines: %w", err)
	}
	if len(linesResp.Lines) == 0 {
		return "", "", fmt.Errorf("no order lines for credit note")
	}
	line := linesResp.Lines[0]
	sku = line.Sku
	createBody, _ := json.Marshal(map[string]any{
		"order_id":    orderID,
		"reason_code": "MISTAKE",
		"reason_text": "ssmr gap closure credit note",
		"lines": []map[string]any{
			{"order_line_id": line.OrderLineID, "qty": 1},
		},
	})
	status, respBody, _, err = clientPost(ctx, client, base+"/v1/supplier/credit-notes", createBody, cookie, "ssmr-gap-cn-create-"+orderID)
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK && status != http.StatusCreated {
		return "", "", fmt.Errorf("create credit note status %d body %s", status, string(respBody))
	}
	var cnResp struct {
		CreditNoteID string `json:"credit_note_id"`
	}
	_ = json.Unmarshal(respBody, &cnResp)
	cnID = strings.TrimSpace(cnResp.CreditNoteID)
	if cnID == "" {
		return "", "", fmt.Errorf("create credit note missing credit_note_id")
	}
	fmt.Println("PX_E2E_CREDIT_NOTE_DRAFT_OK")
	return cnID, sku, nil
}

func runGapClosureExceptionResolveCreditNoteE2E(ctx context.Context, client *http.Client, base, cookie, cnID string) error {
	path := base + "/v1/supplier/exceptions/CREDIT_NOTE_DRAFT/" + cnID + "/resolve"
	status, respBody, _, err := clientPost(ctx, client, path, []byte("{}"), cookie, "ssmr-gap-resolve-cn-"+cnID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("issue via resolve status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_CREDIT_NOTE_OK")
	return nil
}

func runGapClosureReverseLogisticsE2E(ctx context.Context, client *http.Client, base, supplierID string, cfg *bootstrap.Config, cnID, sku string) error {
	whID := demoWarehouseID()
	whToken, err := issueRoleJWT(cfg, auth.Claims{
		Subject:      "ssmr-wh-gap",
		Role:         auth.RoleWarehouse,
		SupplierID:   supplierID,
		SupplierRole: auth.RoleWarehouseAdmin,
		HomeNodeID:   whID,
	})
	if err != nil {
		return fmt.Errorf("issue warehouse jwt: %w", err)
	}
	deadline := time.Now().Add(20 * time.Second)
	var taskID string
	for time.Now().Before(deadline) {
		status, respBody, _, listErr := clientDo(ctx, client, http.MethodGet, base+"/v1/warehouse/reverse-logistics?warehouse_id="+whID+"&status=OPEN", nil, whToken, "")
		if listErr != nil {
			return listErr
		}
		if status != http.StatusOK {
			return fmt.Errorf("list reverse tasks status %d body %s", status, string(respBody))
		}
		var list struct {
			Tasks []struct {
				TaskID string `json:"task_id"`
			} `json:"tasks"`
		}
		if err := json.Unmarshal(respBody, &list); err != nil {
			return fmt.Errorf("decode reverse tasks: %w", err)
		}
		for _, t := range list.Tasks {
			if strings.TrimSpace(t.TaskID) != "" {
				taskID = t.TaskID
				break
			}
		}
		if taskID != "" {
			break
		}
		time.Sleep(400 * time.Millisecond)
	}
	if taskID == "" {
		return fmt.Errorf("no reverse logistics task after credit note issue")
	}
	qtyKey := sku
	if qtyKey == "" {
		qtyKey = "SSMR-SKU-1"
	}
	receiveBody, _ := json.Marshal(map[string]any{
		"warehouse_id": whID,
		"received_qty": map[string]int64{qtyKey: 1},
	})
	path := base + "/v1/warehouse/reverse-logistics/" + taskID + "/receive"
	status, respBody, _, err := clientPost(ctx, client, path, receiveBody, whToken, "ssmr-gap-reverse-receive-"+taskID)
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("receive reverse status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_REVERSE_LOGISTICS_OK")
	return nil
}

func runGapClosureReorderE2E(ctx context.Context, client *http.Client, base, cookie string) error {
	status, respBody, _, err := clientPost(ctx, client, base+"/v1/supplier/replenishment/trigger", []byte("{}"), cookie, "ssmr-gap-replen-trigger")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("replenishment trigger status %d body %s", status, string(respBody))
	}
	status, respBody, _, err = clientDo(ctx, client, http.MethodGet, base+"/v1/replenishment/suggestions?limit=20", nil, cookie, "")
	if err != nil {
		return err
	}
	if status != http.StatusOK {
		return fmt.Errorf("suggestions list status %d body %s", status, string(respBody))
	}
	fmt.Println("PX_E2E_REORDER_SUGGESTION_OK")
	return nil
}
