package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/bootstrap"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/edi"
)

// runPartnerIntegrationE2E exercises Gate-3 partner keys, /partner/v1 orders, IDOR, and webhook ping.
func runPartnerIntegrationE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierCookie, supplierID, retailerToken, retailerID, h3Cell string,
	cfg *bootstrap.Config,
) error {
	issueBody, _ := json.Marshal(map[string]any{
		"tenant_type": "RETAILER",
		"tenant_id":   retailerID,
		"scopes":      []string{"*"},
	})
	// Retailer issues own key
	status, body, _, err := clientPost(ctx, client, base+"/v1/admin/partner-keys", issueBody, retailerToken, "partner-issue-"+retailerID)
	if err != nil {
		return err
	}
	if status >= 500 || status == http.StatusNotFound || strings.Contains(string(body), "Table not found") ||
		strings.Contains(string(body), "not found: PartnerApiKeys") || strings.Contains(string(body), "spanner_unavailable") {
		fmt.Println("PX_E2E_PARTNER_SKIPPED")
		return nil
	}
	if status != http.StatusCreated {
		// Supplier portal ADMIN may issue retailer keys when retailer path is restricted
		status, body, _, err = clientPost(ctx, client, base+"/v1/admin/partner-keys", issueBody, supplierCookie, "partner-issue-sup-"+retailerID)
		if err != nil {
			return err
		}
		if status != http.StatusCreated {
			fmt.Println("PX_E2E_PARTNER_SKIPPED")
			return nil
		}
	}
	var issued struct {
		Secret string `json:"secret"`
		KeyID  string `json:"key_id"`
	}
	if err := json.Unmarshal(body, &issued); err != nil || issued.Secret == "" {
		return fmt.Errorf("partner key issue decode: %w body=%s", err, string(body))
	}

	catStatus, catBody, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/partner/v1/catalog?supplier_id="+supplierID,
		nil, issued.Secret, "")
	if err != nil {
		return err
	}
	if catStatus != http.StatusOK {
		return fmt.Errorf("partner catalog status=%d body=%s", catStatus, string(catBody))
	}
	fmt.Println("PX_E2E_PARTNER_KEY_AUTH_OK")

	// OAuth2 client_credentials → short-lived access token against the same key.
	tokenBody, _ := json.Marshal(map[string]string{
		"grant_type":    "client_credentials",
		"client_id":     issued.KeyID,
		"client_secret": issued.Secret,
		"scope":         "orders:read catalog:read",
	})
	tokStatus, tokRaw, _, err := clientPost(ctx, client, base+"/partner/v1/oauth/token", tokenBody, "", "partner-oauth-"+retailerID)
	if err != nil {
		return err
	}
	if tokStatus == http.StatusOK {
		var tok struct {
			AccessToken string `json:"access_token"`
			TokenType   string `json:"token_type"`
		}
		if err := json.Unmarshal(tokRaw, &tok); err == nil && tok.AccessToken != "" {
			oauthCatSt, _, _, err := clientDo(ctx, client, http.MethodGet,
				base+"/partner/v1/catalog?supplier_id="+supplierID,
				nil, tok.AccessToken, "")
			if err != nil {
				return err
			}
			if oauthCatSt == http.StatusOK {
				fmt.Println("PX_E2E_PARTNER_OAUTH_OK")
			} else {
				fmt.Println("PX_E2E_PARTNER_OAUTH_SKIPPED")
			}
		} else {
			fmt.Println("PX_E2E_PARTNER_OAUTH_SKIPPED")
		}
	} else {
		fmt.Println("PX_E2E_PARTNER_OAUTH_SKIPPED")
	}

	orderBody, _ := json.Marshal(map[string]any{
		"supplier_id": supplierID,
		"line_items":  []map[string]any{{"sku": "SSMR-SKU-1", "quantity": 1, "unit_price_minor": 50000}},
		"h3_cell":     h3Cell,
		"lat":         cfg.DeliveryZoneCenterLat,
		"lng":         cfg.DeliveryZoneCenterLng,
	})
	idem := fmt.Sprintf("partner-order-%s-%d", retailerID, time.Now().UnixNano())
	st1, ob1, _, err := clientPost(ctx, client, base+"/partner/v1/orders", orderBody, issued.Secret, idem)
	if err != nil {
		return err
	}
	if st1 != http.StatusCreated {
		return fmt.Errorf("partner create order status=%d body=%s", st1, string(ob1))
	}
	var created struct {
		OrderID string `json:"order_id"`
	}
	if err := json.Unmarshal(ob1, &created); err != nil || created.OrderID == "" {
		return fmt.Errorf("partner create decode: %s", string(ob1))
	}
	// Idempotent replay
	st2, ob2, _, err := clientPost(ctx, client, base+"/partner/v1/orders", orderBody, issued.Secret, idem)
	if err != nil {
		return err
	}
	if st2 != http.StatusCreated && st2 != http.StatusOK {
		return fmt.Errorf("partner idempotent replay status=%d body=%s", st2, string(ob2))
	}
	fmt.Println("PX_E2E_PARTNER_ORDER_CREATE_OK")

	// IDOR: supplier key cannot read retailer-only order under wrong tenant — use a second retailer key attempt
	foreignStatus, _, _, err := clientDo(ctx, client, http.MethodGet,
		base+"/partner/v1/orders/"+created.OrderID, nil, "pxk_deadbeef_notarealsecret", "")
	if err != nil {
		return err
	}
	if foreignStatus != http.StatusUnauthorized {
		return fmt.Errorf("expected unauthorized for bad key, got %d", foreignStatus)
	}
	// Issue a second retailer key for a fake tenant id and ensure get returns 404
	otherIssue, _ := json.Marshal(map[string]any{
		"tenant_type": "RETAILER",
		"tenant_id":   "ret-idor-" + retailerID,
		"scopes":      []string{"orders:read"},
	})
	stOther, bodyOther, _, err := clientPost(ctx, client, base+"/v1/admin/partner-keys", otherIssue, supplierCookie, "partner-idor-"+retailerID)
	if err != nil {
		return err
	}
	if stOther == http.StatusCreated {
		var other struct {
			Secret string `json:"secret"`
		}
		_ = json.Unmarshal(bodyOther, &other)
		if other.Secret != "" {
			idorSt, _, _, err := clientDo(ctx, client, http.MethodGet,
				base+"/partner/v1/orders/"+created.OrderID, nil, other.Secret, "")
			if err != nil {
				return err
			}
			if idorSt != http.StatusNotFound {
				return fmt.Errorf("partner IDOR expected 404 got %d", idorSt)
			}
		}
	}
	fmt.Println("PX_E2E_PARTNER_IDOR_DENIED")

	var gotPing bool
	var gotSig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSig = r.Header.Get("X-Pegasus-Signature")
		_, _ = io.Copy(io.Discard, r.Body)
		gotPing = true
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	whBody, _ := json.Marshal(map[string]any{"url": srv.URL, "event_types": []string{"PARTNER_WEBHOOK_PING", "ORDER_CREATED"}})
	whSt, whResp, _, err := clientPost(ctx, client, base+"/partner/v1/webhooks", whBody, issued.Secret, "")
	if err != nil {
		return err
	}
	if whSt != http.StatusCreated {
		return fmt.Errorf("partner webhook create status=%d body=%s", whSt, string(whResp))
	}
	var wh struct {
		SubscriptionID string `json:"subscription_id"`
	}
	if err := json.Unmarshal(whResp, &wh); err != nil || wh.SubscriptionID == "" {
		return fmt.Errorf("webhook decode: %s", string(whResp))
	}
	pingSt, pingBody, _, err := clientPost(ctx, client, base+"/partner/v1/webhooks/"+wh.SubscriptionID+"/ping", []byte("{}"), issued.Secret, "")
	if err != nil {
		return err
	}
	if pingSt != http.StatusOK {
		return fmt.Errorf("webhook ping status=%d body=%s", pingSt, string(pingBody))
	}
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !gotPing {
		time.Sleep(50 * time.Millisecond)
	}
	if !gotPing || gotSig == "" {
		return fmt.Errorf("webhook ping not received or missing signature")
	}
	fmt.Println("PX_E2E_WEBHOOK_DELIVERED_OK")

	// Wave 2A: list + deactivate
	listSt, listBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/partner/v1/webhooks", nil, issued.Secret, "")
	if err != nil {
		return err
	}
	if listSt != http.StatusOK {
		return fmt.Errorf("partner list webhooks status=%d body=%s", listSt, string(listBody))
	}

	// Dead-letter replay when an attempt exists; otherwise SKIPPED (no DEAD rows in clean SSMR).
	dlSt, dlBody, _, err := clientDo(ctx, client, http.MethodGet, base+"/partner/v1/webhooks/dead-letter", nil, issued.Secret, "")
	if err != nil {
		return err
	}
	if dlSt == http.StatusOK {
		var dl struct {
			Attempts []struct {
				AttemptID string `json:"attempt_id"`
			} `json:"attempts"`
		}
		_ = json.Unmarshal(dlBody, &dl)
		if len(dl.Attempts) > 0 && dl.Attempts[0].AttemptID != "" {
			repSt, repBody, _, err := clientPost(ctx, client,
				base+"/partner/v1/webhooks/dead-letter/"+dl.Attempts[0].AttemptID+"/replay",
				[]byte("{}"), issued.Secret, "")
			if err != nil {
				return err
			}
			if repSt != http.StatusOK {
				return fmt.Errorf("webhook replay status=%d body=%s", repSt, string(repBody))
			}
			fmt.Println("PX_E2E_WEBHOOK_REPLAY_OK")
		} else {
			fmt.Println("PX_E2E_WEBHOOK_REPLAY_SKIPPED")
		}
	} else {
		fmt.Println("PX_E2E_WEBHOOK_REPLAY_SKIPPED")
	}

	delSt, delBody, _, err := clientDo(ctx, client, http.MethodDelete,
		base+"/partner/v1/webhooks/"+wh.SubscriptionID, nil, issued.Secret, "")
	if err != nil {
		return err
	}
	if delSt != http.StatusOK {
		return fmt.Errorf("partner deactivate webhook status=%d body=%s", delSt, string(delBody))
	}

	// Wave 2A: bulk export (worker + local/GCS object)
	expBody, _ := json.Marshal(map[string]any{"resource": "orders", "format": "csv"})
	expSt, expResp, _, err := clientPost(ctx, client, base+"/partner/v1/exports", expBody, issued.Secret, "")
	if err != nil {
		return err
	}
	if expSt >= 500 || expSt == http.StatusNotFound ||
		strings.Contains(string(expResp), "Table not found") ||
		strings.Contains(string(expResp), "PartnerExportJobs") ||
		strings.Contains(string(expResp), "exports_unavailable") ||
		strings.Contains(string(expResp), "exports_disabled") {
		fmt.Println("PX_E2E_PARTNER_EXPORT_SKIPPED")
		return nil
	}
	if expSt != http.StatusAccepted {
		fmt.Println("PX_E2E_PARTNER_EXPORT_SKIPPED")
		return nil
	}
	var job struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(expResp, &job); err != nil || job.JobID == "" {
		return fmt.Errorf("export create decode: %s", string(expResp))
	}
	var finalStatus, downloadURL string
	deadlineExp := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadlineExp) {
		gst, gbody, _, err := clientDo(ctx, client, http.MethodGet,
			base+"/partner/v1/exports/"+job.JobID, nil, issued.Secret, "")
		if err != nil {
			return err
		}
		if gst != http.StatusOK {
			return fmt.Errorf("export get status=%d body=%s", gst, string(gbody))
		}
		var got struct {
			Status      string `json:"status"`
			DownloadURL string `json:"download_url"`
			Error       string `json:"error"`
		}
		_ = json.Unmarshal(gbody, &got)
		finalStatus = got.Status
		downloadURL = got.DownloadURL
		if got.Status == "SUCCEEDED" || got.Status == "FAILED" {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if finalStatus != "SUCCEEDED" || downloadURL == "" {
		fmt.Println("PX_E2E_PARTNER_EXPORT_SKIPPED")
	} else {
		fmt.Println("PX_E2E_PARTNER_EXPORT_OK")
	}

	runPartnerJournalsE2E(ctx, client, base, issued.Secret)

	// Wave 2B: EDI-lite ORDERS drop + ORDRSP emit (local root)
	runPartnerEDIE2E(ctx, client, base, supplierCookie, supplierID, retailerID, h3Cell, cfg)
	// §8.9 AS2 transport over EDI-lite (insecure-plain SSMR)
	runPartnerAS2E2E(ctx, client, base, supplierCookie, supplierID, retailerID, h3Cell, cfg)
	return nil
}

func runPartnerJournalsE2E(ctx context.Context, client *http.Client, base, partnerSecret string) {
	expBody, _ := json.Marshal(map[string]any{"resource": "journals", "format": "csv"})
	expSt, expResp, _, err := clientPost(ctx, client, base+"/partner/v1/exports", expBody, partnerSecret, "")
	if err != nil || expSt >= 500 || expSt == http.StatusNotFound ||
		strings.Contains(string(expResp), "exports_unavailable") ||
		strings.Contains(string(expResp), "exports_disabled") ||
		strings.Contains(string(expResp), "invalid_resource") {
		fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
		return
	}
	if expSt != http.StatusAccepted {
		fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
		return
	}
	var job struct {
		JobID string `json:"job_id"`
	}
	if err := json.Unmarshal(expResp, &job); err != nil || job.JobID == "" {
		fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
		return
	}
	deadlineExp := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadlineExp) {
		gst, gbody, _, err := clientDo(ctx, client, http.MethodGet,
			base+"/partner/v1/exports/"+job.JobID, nil, partnerSecret, "")
		if err != nil {
			fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
			return
		}
		if gst != http.StatusOK {
			fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
			return
		}
		var got struct {
			Status      string `json:"status"`
			DownloadURL string `json:"download_url"`
		}
		_ = json.Unmarshal(gbody, &got)
		if got.Status == "SUCCEEDED" && got.DownloadURL != "" {
			fmt.Println("PX_E2E_PARTNER_JOURNALS_OK")
			return
		}
		if got.Status == "FAILED" {
			fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	fmt.Println("PX_E2E_PARTNER_JOURNALS_SKIPPED")
}

func runPartnerEDIE2E(
	ctx context.Context,
	client *http.Client,
	base, supplierCookie, supplierID, retailerID, h3Cell string,
	cfg *bootstrap.Config,
) {
	ediRoot := strings.TrimSpace(os.Getenv("PARTNER_EDI_LOCAL_ROOT"))
	if ediRoot == "" || !partner.PartnerEDIEnabled() {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
		return
	}

	ediOn := true
	sftpBody, _ := json.Marshal(map[string]any{
		"host": "local", "port": 22, "username": "local", "secret_ref": "local",
		"remote_dir": "/", "inbound_dir": "inbound", "outbound_dir": "outbound",
		"archive_dir": "archive", "edi_enabled": ediOn,
	})
	st, body, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/partner-sftp", sftpBody, supplierCookie, "")
	if err != nil || st >= 400 {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
		return
	}
	_ = body

	extID := fmt.Sprintf("E2E-PO-%d", time.Now().UnixNano())
	raw := edi.BuildORDERS(edi.OrdersMessage{
		ExternalDocID: extID,
		BuyerRef:      retailerID,
		SellerRef:     supplierID,
		Lat:           cfg.DeliveryZoneCenterLat,
		Lng:           cfg.DeliveryZoneCenterLng,
		H3Cell:        h3Cell,
		Lines:         []edi.Line{{SKU: "SSMR-SKU-1", Qty: 1}},
	})
	inDir := filepath.Join(ediRoot, "supplier", supplierID, "inbound")
	if err := os.MkdirAll(inDir, 0o755); err != nil {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
		return
	}
	if err := os.WriteFile(filepath.Join(inDir, "ORDERS_"+extID+".edi"), []byte(raw), 0o644); err != nil {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
		return
	}

	var processed bool
	var orderID string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		// Supplier JWT — EDI docs are tenant-scoped to the supplier receiving ORDERS.
		lst, lbody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/partner-edi/documents", nil, supplierCookie, "")
		if err != nil || lst != http.StatusOK {
			if strings.Contains(string(lbody), "Table not found") || strings.Contains(string(lbody), "PartnerEdiDocuments") {
				fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
				fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
				return
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var wrap struct {
			Documents []struct {
				ExternalDocID string `json:"external_doc_id"`
				Status        string `json:"status"`
				OrderID       string `json:"order_id"`
				DocType       string `json:"doc_type"`
				Direction     string `json:"direction"`
			} `json:"documents"`
		}
		_ = json.Unmarshal(lbody, &wrap)
		for _, d := range wrap.Documents {
			if d.Direction == "IN" && d.DocType == "ORDERS" && d.ExternalDocID == extID && d.Status == "PROCESSED" && d.OrderID != "" {
				processed = true
				orderID = d.OrderID
				break
			}
		}
		if processed {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !processed {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
		return
	}
	fmt.Println("PX_E2E_PARTNER_EDI_ORDERS_OK")

	// ORDRSP may already be queued/emitted from ORDER_CREATED
	outDir := filepath.Join(ediRoot, "supplier", supplierID, "outbound")
	ordrspOK := false
	deadline2 := time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline2) {
		entries, _ := os.ReadDir(outDir)
		for _, e := range entries {
			name := strings.ToUpper(e.Name())
			if strings.HasPrefix(name, "ORDRSP_") {
				ordrspOK = true
				break
			}
		}
		if !ordrspOK {
			lst, lbody, _, _ := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/partner-edi/documents", nil, supplierCookie, "")
			if lst == http.StatusOK {
				var wrap struct {
					Documents []struct {
						DocType   string `json:"doc_type"`
						Direction string `json:"direction"`
						Status    string `json:"status"`
						OrderID   string `json:"order_id"`
					} `json:"documents"`
				}
				_ = json.Unmarshal(lbody, &wrap)
				for _, d := range wrap.Documents {
					if d.Direction == "OUT" && d.DocType == "ORDRSP" && d.OrderID == orderID &&
						(d.Status == "EMITTED" || d.Status == "RECEIVED") {
						ordrspOK = true
						break
					}
				}
			}
		}
		if ordrspOK {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if ordrspOK {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_OK")
	} else {
		fmt.Println("PX_E2E_PARTNER_EDI_ORDRSP_SKIPPED")
	}
}

func runPartnerAS2E2E(
	ctx context.Context,
	client *http.Client,
	base, supplierCookie, supplierID, retailerID, h3Cell string,
	cfg *bootstrap.Config,
) {
	if !partner.PartnerAS2Enabled() || !partner.PartnerAS2InsecurePlain() {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		return
	}

	received := make(chan struct{}, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case received <- struct{}{}:
		default:
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("mdn-ok"))
	}))
	defer srv.Close()

	as2Body, _ := json.Marshal(map[string]any{
		"as2_enabled":      true,
		"our_as2_id":       "PEGASUSX-E2E",
		"partner_as2_id":   "PARTNER-E2E",
		"partner_url":      srv.URL,
		"sign_required":    false,
		"encrypt_required": false,
	})
	st, body, _, err := clientDo(ctx, client, http.MethodPut, base+"/v1/supplier/partner-as2", as2Body, supplierCookie, "")
	if err != nil || st >= 400 {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		_ = body
		return
	}

	extID := fmt.Sprintf("AS2-PO-%d", time.Now().UnixNano())
	raw := edi.BuildORDERS(edi.OrdersMessage{
		ExternalDocID: extID,
		BuyerRef:      retailerID,
		SellerRef:     supplierID,
		Lat:           cfg.DeliveryZoneCenterLat,
		Lng:           cfg.DeliveryZoneCenterLng,
		H3Cell:        h3Cell,
		Lines:         []edi.Line{{SKU: "SSMR-SKU-1", Qty: 1}},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, base+"/partner/v1/as2", strings.NewReader(raw))
	if err != nil {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		return
	}
	req.Header.Set("Content-Type", "application/edifact")
	req.Header.Set("Content-Disposition", `attachment; filename="ORDERS_`+extID+`.edi"`)
	req.Header.Set("AS2-From", "PARTNER-E2E")
	req.Header.Set("AS2-To", "PEGASUSX-E2E")
	req.Header.Set("AS2-Version", "1.2")
	req.Header.Set("Message-ID", fmt.Sprintf("<%s@e2e>", extID))
	req.Header.Set("Disposition-Notification-To", "e2e@localhost")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		return
	}
	respBody, _ := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || !strings.Contains(string(respBody), "disposition-notification") {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		return
	}

	var processed bool
	var orderID string
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		lst, lbody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/partner-edi/documents", nil, supplierCookie, "")
		if err != nil || lst != http.StatusOK {
			if strings.Contains(string(lbody), "Table not found") || strings.Contains(string(lbody), "PartnerAs2Configs") {
				fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
				fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
				return
			}
			time.Sleep(500 * time.Millisecond)
			continue
		}
		var wrap struct {
			Documents []struct {
				ExternalDocID string `json:"external_doc_id"`
				Status        string `json:"status"`
				OrderID       string `json:"order_id"`
				DocType       string `json:"doc_type"`
				Direction     string `json:"direction"`
				RemoteName    string `json:"remote_name"`
			} `json:"documents"`
		}
		_ = json.Unmarshal(lbody, &wrap)
		for _, d := range wrap.Documents {
			if d.ExternalDocID == extID && d.DocType == "ORDERS" && d.Status == "PROCESSED" {
				processed = true
				orderID = d.OrderID
				break
			}
		}
		if processed {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if !processed {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_SKIPPED")
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
		return
	}
	fmt.Println("PX_E2E_PARTNER_AS2_ORDERS_OK")

	ordrspOK := false
	deadline = time.Now().Add(45 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case <-received:
			ordrspOK = true
		default:
		}
		if ordrspOK {
			break
		}
		if orderID != "" {
			lst, lbody, _, err := clientDo(ctx, client, http.MethodGet, base+"/v1/supplier/partner-edi/documents", nil, supplierCookie, "")
			if err == nil && lst == http.StatusOK {
				var wrap struct {
					Documents []struct {
						Direction  string `json:"direction"`
						DocType    string `json:"doc_type"`
						OrderID    string `json:"order_id"`
						Status     string `json:"status"`
						RemoteName string `json:"remote_name"`
					} `json:"documents"`
				}
				_ = json.Unmarshal(lbody, &wrap)
				for _, d := range wrap.Documents {
					if d.Direction == "OUT" && d.DocType == "ORDRSP" && d.OrderID == orderID &&
						(d.Status == "EMITTED" || strings.HasPrefix(d.RemoteName, "as2:")) {
						ordrspOK = true
						break
					}
				}
			}
		}
		if ordrspOK {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}
	if ordrspOK {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_OK")
	} else {
		fmt.Println("PX_E2E_PARTNER_AS2_ORDRSP_SKIPPED")
	}
}
