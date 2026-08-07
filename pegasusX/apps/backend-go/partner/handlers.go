package partner

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/idempotency"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

const maxBody = 64 * 1024

// Handlers expose partner + admin key management HTTP.
type Handlers struct {
	Svc        *Service
	Delivery   *DeliveryWorker
	EdiInbound *EdiInboundWorker
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	return io.ReadAll(io.LimitReader(r.Body, maxBody))
}

// HandleCreateOrder POST /partner/v1/orders
// Honors Idempotency-Key: a replay with the same tenant + key + body returns the
// stored response; the same key with a different body is rejected 409. The store
// (Redis in production) is the latency layer; OrderPaymentLegs/PaymentLedgerEntries
// unique indexes are the database guarantee for the money side effects.
func (h *Handlers) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req order.CreateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	retailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	if retailerID == "" {
		var wrap struct {
			RetailerID string `json:"retailer_id"`
		}
		_ = json.Unmarshal(body, &wrap)
		retailerID = strings.TrimSpace(wrap.RetailerID)
	}

	store := h.Svc.IdempotencyStore()
	rawKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	guardKey := ""
	bodyHash := ""
	if rawKey != "" && store != nil {
		sum := sha256.Sum256(body)
		bodyHash = hex.EncodeToString(sum[:])
		guardKey = idempotency.ScopeKey(string(p.TenantType)+":"+p.TenantID, "POST /partner/v1/orders", rawKey)
		rec, hit, gErr := idempotency.Guard(r.Context(), store, guardKey, bodyHash)
		if gErr != nil {
			switch {
			case errors.Is(gErr, idempotency.ErrConflict):
				writePartnerError(w, http.StatusConflict, "idempotency_key_payload_mismatch")
			case errors.Is(gErr, idempotency.ErrInProgress):
				writePartnerError(w, http.StatusConflict, "request_in_progress")
			default:
				writePartnerError(w, http.StatusInternalServerError, "idempotency_guard_failed")
			}
			return
		}
		if hit {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(rec.StatusCode)
			_, _ = w.Write(rec.Response)
			return
		}
	}

	resp, err := h.Svc.CreateOrder(r.Context(), p, retailerID, req)
	if err != nil {
		if guardKey != "" {
			_ = store.Release(r.Context(), guardKey)
		}
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	respBytes, mErr := json.Marshal(resp)
	if mErr != nil {
		if guardKey != "" {
			_ = store.Release(r.Context(), guardKey)
		}
		writePartnerError(w, http.StatusInternalServerError, "encode_error")
		return
	}
	if guardKey != "" {
		if saveErr := store.Save(r.Context(), guardKey, idempotency.Record{
			BodyHash:   bodyHash,
			StatusCode: http.StatusCreated,
			Response:   respBytes,
			StoredAt:   time.Now().UTC(),
		}, 24*time.Hour); saveErr != nil {
			slog.Warn("partner order idempotency save failed", "err", saveErr)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

// HandleGetOrder GET /partner/v1/orders/{orderID}
func (h *Handlers) HandleGetOrder(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	orderID := chi.URLParam(r, "orderID")
	o, err := h.Svc.GetOrder(r.Context(), p, orderID)
	if err != nil {
		var nf notFoundError
		if errors.As(err, &nf) {
			writePartnerError(w, http.StatusNotFound, "order_not_found")
			return
		}
		writePartnerError(w, http.StatusNotFound, "order_not_found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

// HandleCatalog GET /partner/v1/catalog
func (h *Handlers) HandleCatalog(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	q := r.URL.Query()
	products, err := h.Svc.ListCatalog(r.Context(), p, q.Get("supplier_id"), q.Get("retailer_id"), q.Get("category_id"))
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"products": products})
}

// HandleAvailability GET /partner/v1/inventory/availability
func (h *Handlers) HandleAvailability(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	q := r.URL.Query()
	var skus []string
	if raw := strings.TrimSpace(q.Get("product_ids")); raw != "" {
		skus = strings.Split(raw, ",")
	}
	rows, err := h.Svc.Availability(r.Context(), p, q.Get("supplier_id"), q.Get("retailer_id"), skus)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"availability": rows})
}

// HandleCreateWebhook POST /partner/v1/webhooks
func (h *Handlers) HandleCreateWebhook(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		URL        string   `json:"url"`
		EventTypes []string `json:"event_types"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	sub, secret, err := h.Svc.CreateWebhookSubscription(r.Context(), p, req.URL, req.EventTypes)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"subscription_id": sub.SubscriptionID,
		"url":             sub.URL,
		"event_types":     sub.EventTypes,
		"signing_secret":  secret,
	})
}

// HandlePingWebhook POST /partner/v1/webhooks/{subscriptionID}/ping
func (h *Handlers) HandlePingWebhook(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "subscriptionID")
	if h.Delivery == nil {
		writePartnerError(w, http.StatusServiceUnavailable, "delivery_unavailable")
		return
	}
	err := h.Svc.PingWebhook(r.Context(), p, id, h.Delivery.DeliverHTTP)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleListDeadLetter GET /partner/v1/webhooks/dead-letter
func (h *Handlers) HandleListDeadLetter(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	rows, err := h.Svc.webhooks.ListDeadByTenant(r.Context(), p.TenantType, p.TenantID, 50)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "list_failed")
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, a := range rows {
		out = append(out, map[string]any{
			"attempt_id":      a.AttemptID,
			"subscription_id": a.SubscriptionID,
			"event_id":        a.EventID,
			"event_type":      a.EventType,
			"status":          a.Status,
			"last_error":      a.LastError,
			"attempt_count":   a.AttemptCount,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"attempts": out})
}

// HandleListWebhooks GET /partner/v1/webhooks
func (h *Handlers) HandleListWebhooks(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	subs, err := h.Svc.ListWebhooks(r.Context(), p)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "list_failed")
		return
	}
	out := make([]map[string]any, 0, len(subs))
	for _, s := range subs {
		prefix := s.SigningSecret
		if len(prefix) > 12 {
			prefix = prefix[:12] + "…"
		}
		out = append(out, map[string]any{
			"subscription_id": s.SubscriptionID,
			"url":             s.URL,
			"event_types":     s.EventTypes,
			"is_active":       s.IsActive,
			"secret_prefix":   prefix,
			"created_at":      s.CreatedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"subscriptions": out})
}

// HandleDeactivateWebhook DELETE /partner/v1/webhooks/{subscriptionID}
func (h *Handlers) HandleDeactivateWebhook(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "subscriptionID")
	if err := h.Svc.DeactivateWebhook(r.Context(), p, id); err != nil {
		writePartnerError(w, http.StatusNotFound, "subscription_not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleReplayDeadLetter POST .../webhooks/dead-letter/{attemptID}/replay
func (h *Handlers) HandleReplayDeadLetter(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "attemptID")
	att, err := h.Svc.ReplayDeadLetter(r.Context(), p, id)
	if err != nil {
		var nf notFoundError
		if errors.As(err, &nf) {
			writePartnerError(w, http.StatusNotFound, "attempt_not_found")
			return
		}
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"attempt_id": att.AttemptID, "status": att.Status, "ok": true,
	})
}

// HandleCreateExport POST /partner/v1/exports
func (h *Handlers) HandleCreateExport(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Resource string  `json:"resource"`
		Format   string  `json:"format"`
		From     *string `json:"from"`
		To       *string `json:"to"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	var from, to *time.Time
	if req.From != nil && strings.TrimSpace(*req.From) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.From))
		if err != nil {
			t2, err2 := time.Parse("2006-01-02", strings.TrimSpace(*req.From))
			if err2 != nil {
				writePartnerError(w, http.StatusUnprocessableEntity, "invalid_from")
				return
			}
			t = t2
		}
		from = &t
	}
	if req.To != nil && strings.TrimSpace(*req.To) != "" {
		t, err := time.Parse(time.RFC3339, strings.TrimSpace(*req.To))
		if err != nil {
			t2, err2 := time.Parse("2006-01-02", strings.TrimSpace(*req.To))
			if err2 != nil {
				writePartnerError(w, http.StatusUnprocessableEntity, "invalid_to")
				return
			}
			t = t2
		}
		to = &t
	}
	j, err := h.Svc.CreateExportJob(r.Context(), p, req.Resource, req.Format, from, to)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusAccepted, exportJobDTO(j, ""))
}

// HandleGetExport GET /partner/v1/exports/{jobID}
func (h *Handlers) HandleGetExport(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	j, dl, err := h.Svc.GetExportJob(r.Context(), p, chi.URLParam(r, "jobID"))
	if err != nil {
		writePartnerError(w, http.StatusNotFound, "job_not_found")
		return
	}
	writeJSON(w, http.StatusOK, exportJobDTO(j, dl))
}

// HandleListExports GET /partner/v1/exports
func (h *Handlers) HandleListExports(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	jobs, err := h.Svc.ListExportJobs(r.Context(), p, 50)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "list_failed")
		return
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, j := range jobs {
		out = append(out, exportJobDTO(j, ""))
	}
	writeJSON(w, http.StatusOK, map[string]any{"jobs": out})
}

// HandleGetSftp GET /v1/supplier/partner-sftp
func (h *Handlers) HandleGetSftp(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.jwtTenant(w, r)
	if !ok {
		return
	}
	cfg, found, err := h.Svc.GetSftpConfig(r.Context(), tt, tid)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "get_failed")
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"configured":   true,
		"host":         cfg.Host,
		"port":         cfg.Port,
		"username":     cfg.Username,
		"secret_ref":   cfg.SecretRef,
		"remote_dir":   cfg.RemoteDir,
		"is_active":    cfg.IsActive,
		"inbound_dir":  cfg.InboundDir,
		"outbound_dir": cfg.OutboundDir,
		"archive_dir":  cfg.ArchiveDir,
		"edi_enabled":  cfg.EdiEnabled,
	})
}

// HandlePutSftp PUT /v1/supplier/partner-sftp
func (h *Handlers) HandlePutSftp(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.jwtTenant(w, r)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		Host        string `json:"host"`
		Port        int64  `json:"port"`
		Username    string `json:"username"`
		SecretRef   string `json:"secret_ref"`
		RemoteDir   string `json:"remote_dir"`
		InboundDir  string `json:"inbound_dir"`
		OutboundDir string `json:"outbound_dir"`
		ArchiveDir  string `json:"archive_dir"`
		EdiEnabled  *bool  `json:"edi_enabled"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	cfg := SftpConfig{
		Host: req.Host, Port: req.Port, Username: req.Username,
		SecretRef: req.SecretRef, RemoteDir: req.RemoteDir,
		InboundDir: req.InboundDir, OutboundDir: req.OutboundDir, ArchiveDir: req.ArchiveDir,
	}
	if req.EdiEnabled != nil {
		cfg.EdiEnabled = *req.EdiEnabled
	}
	if err := h.Svc.UpsertSftpConfig(r.Context(), tt, tid, cfg); err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// HandleGetAs2 GET /v1/supplier/partner-as2 or /partner/v1/as2/config
func (h *Handlers) HandleGetAs2(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	cfg, found, err := h.Svc.GetAs2Config(r.Context(), tt, tid)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "get_failed")
		return
	}
	if !found {
		writeJSON(w, http.StatusOK, map[string]any{"configured": false})
		return
	}
	writeJSON(w, http.StatusOK, as2ConfigDTO(cfg))
}

// HandlePutAs2 PUT /v1/supplier/partner-as2 or /partner/v1/as2/config
func (h *Handlers) HandlePutAs2(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		As2Enabled           *bool  `json:"as2_enabled"`
		OurAs2Id             string `json:"our_as2_id"`
		PartnerAs2Id         string `json:"partner_as2_id"`
		PartnerURL           string `json:"partner_url"`
		OurCertSecretRef     string `json:"our_cert_secret_ref"`
		OurKeySecretRef      string `json:"our_key_secret_ref"`
		PartnerCertSecretRef string `json:"partner_cert_secret_ref"`
		SignRequired         *bool  `json:"sign_required"`
		EncryptRequired      *bool  `json:"encrypt_required"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	cfg := As2Config{
		OurAs2Id:             req.OurAs2Id,
		PartnerAs2Id:         req.PartnerAs2Id,
		PartnerURL:           req.PartnerURL,
		OurCertSecretRef:     req.OurCertSecretRef,
		OurKeySecretRef:      req.OurKeySecretRef,
		PartnerCertSecretRef: req.PartnerCertSecretRef,
		SignRequired:         true,
		EncryptRequired:      true,
	}
	if req.As2Enabled != nil {
		cfg.As2Enabled = *req.As2Enabled
	}
	if req.SignRequired != nil {
		cfg.SignRequired = *req.SignRequired
	}
	if req.EncryptRequired != nil {
		cfg.EncryptRequired = *req.EncryptRequired
	}
	if err := h.Svc.UpsertAs2Config(r.Context(), tt, tid, cfg); err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	saved, _, _ := h.Svc.GetAs2Config(r.Context(), tt, tid)
	writeJSON(w, http.StatusOK, as2ConfigDTO(saved))
}

func as2ConfigDTO(cfg As2Config) map[string]any {
	return map[string]any{
		"configured":              true,
		"as2_enabled":             cfg.As2Enabled,
		"our_as2_id":              cfg.OurAs2Id,
		"partner_as2_id":          cfg.PartnerAs2Id,
		"partner_url":             cfg.PartnerURL,
		"our_cert_secret_ref":     cfg.OurCertSecretRef,
		"our_key_secret_ref":      cfg.OurKeySecretRef,
		"partner_cert_secret_ref": cfg.PartnerCertSecretRef,
		"sign_required":           cfg.SignRequired,
		"encrypt_required":        cfg.EncryptRequired,
	}
}

// HandleGetCoa GET /partner/v1/coa or /v1/supplier/partner-coa
func (h *Handlers) HandleGetCoa(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	m, err := h.Svc.GetCoa(r.Context(), tt, tid)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "coa_load_failed")
		return
	}
	writeJSON(w, http.StatusOK, coaDTO(m))
}

// HandlePutCoa PUT /partner/v1/coa or /v1/supplier/partner-coa
func (h *Handlers) HandlePutCoa(w http.ResponseWriter, r *http.Request) {
	tt, tid, ok := h.tenantFromPartnerOrJWT(w, r)
	if !ok {
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		AccountAR       string `json:"account_ar"`
		AccountRevenue  string `json:"account_revenue"`
		AccountBankCash string `json:"account_bank_cash"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	updatedBy := ""
	if p, pok := PrincipalFromContext(r.Context()); pok {
		updatedBy = p.KeyID
	} else if c, cok := auth.FromContext(r.Context()); cok {
		updatedBy = c.Subject
	}
	m, err := h.Svc.UpsertCoa(r.Context(), tt, tid, updatedBy, CoaMap{
		AccountAR: req.AccountAR, AccountRevenue: req.AccountRevenue, AccountBankCash: req.AccountBankCash,
	})
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, coaDTO(m))
}

// tenantFromPartnerOrJWT resolves tenant from partner key principal or JWT claims.
func (h *Handlers) tenantFromPartnerOrJWT(w http.ResponseWriter, r *http.Request) (tenantType, tenantID string, ok bool) {
	if p, pok := PrincipalFromContext(r.Context()); pok {
		return p.TenantType, p.TenantID, true
	}
	return h.jwtTenant(w, r)
}

// HandleListEdiDocuments GET /partner/v1/edi/documents
func (h *Handlers) HandleListEdiDocuments(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	docs, err := h.Svc.ListEdiDocuments(r.Context(), p, 50)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "list_failed")
		return
	}
	out := make([]map[string]any, 0, len(docs))
	for _, d := range docs {
		out = append(out, ediDocDTO(d))
	}
	writeJSON(w, http.StatusOK, map[string]any{"documents": out})
}

// HandleGetEdiDocument GET /partner/v1/edi/documents/{documentID}
func (h *Handlers) HandleGetEdiDocument(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.GetEdiDocument(r.Context(), p, chi.URLParam(r, "documentID"))
	if err != nil {
		writePartnerError(w, http.StatusNotFound, "document_not_found")
		return
	}
	writeJSON(w, http.StatusOK, ediDocDTO(d))
}

// HandleReplayEdiDocument POST /partner/v1/edi/documents/{documentID}/replay
func (h *Handlers) HandleReplayEdiDocument(w http.ResponseWriter, r *http.Request) {
	p, ok := h.principalOrClaims(w, r)
	if !ok {
		return
	}
	d, err := h.Svc.ReplayEdiDocument(r.Context(), p, chi.URLParam(r, "documentID"))
	if err != nil {
		var nf notFoundError
		if errors.As(err, &nf) {
			writePartnerError(w, http.StatusNotFound, "document_not_found")
			return
		}
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, ediDocDTO(d))
}

func ediDocDTO(d EdiDocument) map[string]any {
	m := map[string]any{
		"document_id": d.DocumentID, "tenant_type": d.TenantType, "tenant_id": d.TenantID,
		"direction": d.Direction, "doc_type": d.DocType, "external_doc_id": d.ExternalDocID,
		"order_id": d.OrderID, "status": d.Status, "remote_name": d.RemoteName,
		"created_at": d.CreatedAt.UTC().Format(time.RFC3339),
	}
	if d.Error != "" {
		m["error"] = d.Error
	}
	if d.ObjectPath != "" {
		m["object_path"] = d.ObjectPath
	}
	if d.FinishedAt != nil {
		m["finished_at"] = d.FinishedAt.UTC().Format(time.RFC3339)
	}
	return m
}

func exportJobDTO(j ExportJob, downloadURL string) map[string]any {
	m := map[string]any{
		"job_id": j.JobID, "tenant_type": j.TenantType, "tenant_id": j.TenantID,
		"resource": j.Resource, "format": j.Format, "status": j.Status,
		"row_count": j.RowCount, "sftp_status": j.SftpStatus,
		"created_at": j.CreatedAt.UTC().Format(time.RFC3339),
	}
	if j.Error != "" {
		m["error"] = j.Error
	}
	if j.ObjectPath != "" {
		m["object_path"] = j.ObjectPath
	}
	if downloadURL != "" {
		m["download_url"] = downloadURL
	}
	if j.FinishedAt != nil {
		m["finished_at"] = j.FinishedAt.UTC().Format(time.RFC3339)
	}
	return m
}

// principalOrClaims uses partner key principal, or supplier/admin JWT as SUPPLIER tenant.
func (h *Handlers) principalOrClaims(w http.ResponseWriter, r *http.Request) (Principal, bool) {
	if p, ok := PrincipalFromContext(r.Context()); ok {
		return p, true
	}
	tt, tid, ok := h.jwtTenant(w, r)
	if !ok {
		return Principal{}, false
	}
	return Principal{
		TenantType: tt, TenantID: tid,
		Scopes: []string{"*", ScopeWebhooksManage, ScopeExportsRead},
	}, true
}

func (h *Handlers) jwtTenant(w http.ResponseWriter, r *http.Request) (tenantType, tenantID string, ok bool) {
	claims, has := auth.FromContext(r.Context())
	if !has {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return "", "", false
	}
	if claims.Role == auth.RoleRetailer {
		return TenantRetailer, auth.ResolveRetailerOrgID(claims), true
	}
	if claims.SupplierID == "" && claims.Role != auth.RoleAdmin {
		writePartnerError(w, http.StatusForbidden, "forbidden")
		return "", "", false
	}
	tid := claims.SupplierID
	return TenantSupplier, tid, true
}

// HandleIssueKey POST /v1/admin/partner-keys or supplier/retailer issue
func (h *Handlers) HandleIssueKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}
	var req struct {
		TenantType string   `json:"tenant_type"`
		TenantID   string   `json:"tenant_id"`
		Scopes     []string `json:"scopes"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}
	tenantType := strings.ToUpper(strings.TrimSpace(req.TenantType))
	tenantID := strings.TrimSpace(req.TenantID)

	switch claims.Role {
	case auth.RoleAdmin:
		if tenantType == "" {
			tenantType = TenantSupplier
		}
		if tenantID == "" {
			tenantID = claims.SupplierID
		}
	case auth.RoleRetailer:
		tenantType = TenantRetailer
		tenantID = auth.ResolveRetailerOrgID(claims)
	case auth.RoleWarehouseAdmin, auth.RoleWarehouse:
		writePartnerError(w, http.StatusForbidden, "forbidden")
		return
	default:
		if claims.Role == auth.RoleAdmin || strings.EqualFold(string(claims.SupplierRole), string(auth.RoleAdmin)) {
			// already handled
		} else if claims.SupplierID != "" && (tenantType == "" || tenantType == TenantSupplier) {
			tenantType = TenantSupplier
			if tenantID == "" {
				tenantID = claims.SupplierID
			}
			if tenantID != claims.SupplierID {
				writePartnerError(w, http.StatusForbidden, "forbidden")
				return
			}
		} else {
			writePartnerError(w, http.StatusForbidden, "forbidden")
			return
		}
	}

	issued, err := h.Svc.IssueKey(r.Context(), tenantType, tenantID, claims.Subject, req.Scopes)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, issued)
}

// HandleListKeys GET /v1/admin/partner-keys
func (h *Handlers) HandleListKeys(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	tenantType := TenantSupplier
	tenantID := claims.SupplierID
	if claims.Role == auth.RoleRetailer {
		tenantType = TenantRetailer
		tenantID = auth.ResolveRetailerOrgID(claims)
	}
	keys, err := h.Svc.ListKeys(r.Context(), tenantType, tenantID)
	if err != nil {
		writePartnerError(w, http.StatusInternalServerError, "list_failed")
		return
	}
	out := make([]map[string]any, 0, len(keys))
	for _, k := range keys {
		out = append(out, map[string]any{
			"key_id": k.KeyID, "tenant_type": k.TenantType, "tenant_id": k.TenantID,
			"key_prefix": k.KeyPrefix, "scopes": k.Scopes, "status": k.Status,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"keys": out})
}

// HandleRevokeKey POST /v1/admin/partner-keys/{keyID}/revoke
func (h *Handlers) HandleRevokeKey(w http.ResponseWriter, r *http.Request) {
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writePartnerError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	keyID := chi.URLParam(r, "keyID")
	tenantType := TenantSupplier
	tenantID := claims.SupplierID
	if claims.Role == auth.RoleRetailer {
		tenantType = TenantRetailer
		tenantID = auth.ResolveRetailerOrgID(claims)
	}
	if err := h.Svc.RevokeKey(r.Context(), keyID, tenantType, tenantID); err != nil {
		writePartnerError(w, http.StatusNotFound, "key_not_found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
