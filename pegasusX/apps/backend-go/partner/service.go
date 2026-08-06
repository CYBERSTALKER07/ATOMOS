package partner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/catalog"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// Service owns partner key issuance, partner HTTP business logic, and webhook enqueue.
type Service struct {
	keys           KeyRepository
	webhooks       WebhookRepository
	exports        ExportRepository
	sftp           SftpConfigRepository
	coa            CoaRepository
	ediDocs        EdiDocumentRepository
	ediOut         *EdiOutboundWorker
	orders         *order.Service
	catalog        *catalog.Service
	log            *slog.Logger
	now            func() time.Time
	oauthJWTSecret string
	oauthJWTIssuer string
	oauthTTL       time.Duration
}

// NewService constructs the partner service.
func NewService(keys KeyRepository, webhooks WebhookRepository, orders *order.Service, cat *catalog.Service, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		keys: keys, webhooks: webhooks, orders: orders, catalog: cat, log: log,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// SetExportRepos wire Wave-2A export + SFTP repos.
func (s *Service) SetExportRepos(exports ExportRepository, sftp SftpConfigRepository) {
	if s == nil {
		return
	}
	s.exports = exports
	s.sftp = sftp
}

// SetCoaRepository wires configurable journals chart-of-accounts.
func (s *Service) SetCoaRepository(coa CoaRepository) {
	if s == nil {
		return
	}
	s.coa = coa
}

// SetOAuthJWT configures partner client_credentials access-token signing.
func (s *Service) SetOAuthJWT(secret, issuer string, ttl time.Duration) {
	if s == nil {
		return
	}
	s.oauthJWTSecret = strings.TrimSpace(secret)
	s.oauthJWTIssuer = strings.TrimSpace(issuer)
	if ttl > 0 {
		s.oauthTTL = ttl
	}
}

// SetEdiRepos wire Wave-2B EDI document ledger + outbound worker.
func (s *Service) SetEdiRepos(docs EdiDocumentRepository, out *EdiOutboundWorker) {
	if s == nil {
		return
	}
	s.ediDocs = docs
	s.ediOut = out
}

// IssueKey creates a partner API key for a tenant (secret returned once).
func (s *Service) IssueKey(ctx context.Context, tenantType, tenantID, createdBy string, scopes []string) (IssuedKey, error) {
	tenantType = strings.ToUpper(strings.TrimSpace(tenantType))
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || (tenantType != TenantRetailer && tenantType != TenantSupplier) {
		return IssuedKey{}, fmt.Errorf("invalid_tenant")
	}
	if len(scopes) == 0 {
		scopes = DefaultScopesForTenant(tenantType)
	}
	plain, prefix, hash, err := GenerateAPIKey()
	if err != nil {
		return IssuedKey{}, err
	}
	id := uuid.NewString()
	k := ApiKey{
		KeyID:          id,
		TenantType:     tenantType,
		TenantID:       tenantID,
		KeyPrefix:      prefix,
		KeyHash:        hash,
		Scopes:         scopes,
		RateLimitClass: "partner_default",
		Status:         KeyStatusActive,
		CreatedBy:      createdBy,
		CreatedAt:      s.now(),
	}
	if err := s.keys.Insert(ctx, k); err != nil {
		return IssuedKey{}, err
	}
	return IssuedKey{
		KeyID: id, TenantType: tenantType, TenantID: tenantID,
		Scopes: scopes, Prefix: prefix, Secret: plain,
	}, nil
}

// ListKeys returns metadata (no secrets).
func (s *Service) ListKeys(ctx context.Context, tenantType, tenantID string) ([]ApiKey, error) {
	return s.keys.ListByTenant(ctx, tenantType, tenantID, 100)
}

// RevokeKey revokes a key owned by the tenant.
func (s *Service) RevokeKey(ctx context.Context, keyID, tenantType, tenantID string) error {
	return s.keys.Revoke(ctx, keyID, tenantType, tenantID)
}

// CreateOrder places an order via order.Service.Create using partner tenancy.
func (s *Service) CreateOrder(ctx context.Context, p Principal, retailerID string, req order.CreateRequest) (order.CreateResponse, error) {
	if s.orders == nil {
		return order.CreateResponse{}, fmt.Errorf("orders_unavailable")
	}
	switch p.TenantType {
	case TenantRetailer:
		retailerID = p.TenantID
		// SupplierID must come from request body for multi-supplier readiness.
		if strings.TrimSpace(req.SupplierID) == "" {
			return order.CreateResponse{}, fmt.Errorf("supplier_id_required")
		}
	case TenantSupplier:
		if strings.TrimSpace(retailerID) == "" {
			return order.CreateResponse{}, fmt.Errorf("retailer_id_required")
		}
		req.SupplierID = p.TenantID
	default:
		return order.CreateResponse{}, fmt.Errorf("invalid_tenant")
	}
	if req.Source == "" {
		req.Source = order.OrderSourceManual
	}
	return s.orders.Create(ctx, retailerID, req)
}

// GetOrder returns an order if the partner tenant may see it.
func (s *Service) GetOrder(ctx context.Context, p Principal, orderID string) (order.Order, error) {
	if s.orders == nil {
		return order.Order{}, fmt.Errorf("orders_unavailable")
	}
	o, ok, err := s.orders.GetOrder(ctx, orderID)
	if err != nil {
		return order.Order{}, err
	}
	if !ok {
		return order.Order{}, errNotFound("order")
	}
	if !s.canAccessOrder(p, o) {
		return order.Order{}, errNotFound("order") // IDOR fail-closed as not found
	}
	return o, nil
}

func (s *Service) canAccessOrder(p Principal, o order.Order) bool {
	switch p.TenantType {
	case TenantRetailer:
		return o.RetailerID == p.TenantID
	case TenantSupplier:
		return o.SupplierID == p.TenantID
	default:
		return false
	}
}

// ListCatalog returns retailer-enriched catalog for the partner relationship.
func (s *Service) ListCatalog(ctx context.Context, p Principal, supplierID, retailerID, categoryID string) ([]catalog.RetailerProduct, error) {
	if s.catalog == nil {
		return nil, fmt.Errorf("catalog_unavailable")
	}
	switch p.TenantType {
	case TenantRetailer:
		retailerID = p.TenantID
		if strings.TrimSpace(supplierID) == "" {
			return nil, fmt.Errorf("supplier_id_required")
		}
	case TenantSupplier:
		supplierID = p.TenantID
		if strings.TrimSpace(retailerID) == "" {
			return nil, fmt.Errorf("retailer_id_required")
		}
	}
	return s.catalog.ListProductsForRetailer(ctx, supplierID, retailerID, categoryID)
}

// Availability returns stock snapshots for SKUs.
func (s *Service) Availability(ctx context.Context, p Principal, supplierID, retailerID string, productIDs []string) ([]map[string]any, error) {
	products, err := s.ListCatalog(ctx, p, supplierID, retailerID, "")
	if err != nil {
		return nil, err
	}
	want := map[string]bool{}
	for _, id := range productIDs {
		want[strings.TrimSpace(id)] = true
	}
	out := make([]map[string]any, 0)
	for _, rp := range products {
		pid := strings.TrimSpace(rp.ProductID)
		if len(want) > 0 && !want[pid] {
			continue
		}
		row := map[string]any{
			"product_id":        pid,
			"supplier_id":       rp.SupplierID,
			"is_out_of_stock":   rp.IsOutOfStock,
			"accepts_backorder": rp.AcceptsBackorder,
			"show_stock_counts": rp.ShowStockCounts,
		}
		if rp.AvailableStock != nil {
			row["available_stock"] = *rp.AvailableStock
		}
		out = append(out, row)
	}
	return out, nil
}

// CreateWebhookSubscription registers an outbound webhook (secret returned once).
func (s *Service) CreateWebhookSubscription(ctx context.Context, p Principal, url string, eventTypes []string) (WebhookSubscription, string, error) {
	if !HasScope(p.Scopes, ScopeWebhooksManage) && !HasScope(p.Scopes, "*") {
		return WebhookSubscription{}, "", fmt.Errorf("insufficient_scope")
	}
	url = strings.TrimSpace(url)
	if url == "" || (!strings.HasPrefix(url, "https://") && !strings.HasPrefix(url, "http://")) {
		return WebhookSubscription{}, "", fmt.Errorf("invalid_url")
	}
	secret, err := GenerateWebhookSecret()
	if err != nil {
		return WebhookSubscription{}, "", err
	}
	if len(eventTypes) == 0 {
		eventTypes = []string{"ORDER_CREATED", "ORDER_STATUS_CHANGED", "CLAIM_FILED"}
	}
	sub := WebhookSubscription{
		SubscriptionID: uuid.NewString(),
		TenantType:     p.TenantType,
		TenantID:       p.TenantID,
		URL:            url,
		SigningSecret:  secret,
		EventTypes:     eventTypes,
		IsActive:       true,
		CreatedAt:      s.now(),
	}
	if err := s.webhooks.InsertSubscription(ctx, sub); err != nil {
		return WebhookSubscription{}, "", err
	}
	return sub, secret, nil
}

// EnqueueEvent fan-outs an event to matching active subscriptions (idempotent per sub+event).
func (s *Service) EnqueueEvent(ctx context.Context, eventID, eventType string, payload map[string]any) error {
	if s.webhooks == nil {
		return nil
	}
	eventID = strings.TrimSpace(eventID)
	eventType = strings.TrimSpace(eventType)
	if eventID == "" || eventType == "" {
		return nil
	}
	subs, err := s.webhooks.ListActiveByEvent(ctx, eventType)
	if err != nil {
		return err
	}
	body, _ := json.Marshal(payload)
	supplierID, _ := payload["supplier_id"].(string)
	retailerID, _ := payload["retailer_id"].(string)
	for _, sub := range subs {
		if !tenantMatchesEvent(sub, supplierID, retailerID) {
			continue
		}
		if _, found, _ := s.webhooks.GetAttemptBySubEvent(ctx, sub.SubscriptionID, eventID); found {
			continue
		}
		att := DeliveryAttempt{
			AttemptID:      uuid.NewString(),
			SubscriptionID: sub.SubscriptionID,
			EventID:        eventID,
			EventType:      eventType,
			PayloadJSON:    body,
			Status:         DeliveryPending,
			AttemptCount:   0,
			CreatedAt:      s.now(),
		}
		if err := s.webhooks.InsertAttempt(ctx, att); err != nil {
			s.log.Warn("webhook enqueue failed", "err", err, "subscription_id", sub.SubscriptionID)
		}
	}
	return nil
}

func tenantMatchesEvent(sub WebhookSubscription, supplierID, retailerID string) bool {
	switch sub.TenantType {
	case TenantSupplier:
		return strings.TrimSpace(supplierID) == "" || supplierID == sub.TenantID
	case TenantRetailer:
		return strings.TrimSpace(retailerID) == "" || retailerID == sub.TenantID
	default:
		return false
	}
}

// PingWebhook sends a signed test payload immediately (for e2e / operator verify).
func (s *Service) PingWebhook(ctx context.Context, p Principal, subscriptionID string, deliver func(ctx context.Context, sub WebhookSubscription, eventID, eventType string, body []byte) error) error {
	sub, ok, err := s.webhooks.GetSubscription(ctx, subscriptionID)
	if err != nil || !ok || sub.TenantType != p.TenantType || sub.TenantID != p.TenantID {
		return errNotFound("subscription")
	}
	payload := map[string]any{
		"type":       "PARTNER_WEBHOOK_PING",
		"tenant_type": p.TenantType,
		"tenant_id":  p.TenantID,
		"timestamp":  s.now().Format(time.RFC3339),
	}
	body, _ := json.Marshal(payload)
	eventID := "ping-" + uuid.NewString()
	return deliver(ctx, sub, eventID, "PARTNER_WEBHOOK_PING", body)
}

// ListWebhooks returns subscriptions for the tenant (signing secret omitted by callers).
func (s *Service) ListWebhooks(ctx context.Context, p Principal) ([]WebhookSubscription, error) {
	return s.webhooks.ListSubscriptions(ctx, p.TenantType, p.TenantID)
}

// DeactivateWebhook soft-disables a subscription.
func (s *Service) DeactivateWebhook(ctx context.Context, p Principal, subscriptionID string) error {
	return s.webhooks.DeactivateSubscription(ctx, subscriptionID, p.TenantType, p.TenantID)
}

// ReplayDeadLetter requeues a DEAD attempt for the tenant (full retry budget).
func (s *Service) ReplayDeadLetter(ctx context.Context, p Principal, attemptID string) (DeliveryAttempt, error) {
	att, ok, err := s.webhooks.GetAttempt(ctx, attemptID)
	if err != nil || !ok {
		return DeliveryAttempt{}, errNotFound("attempt")
	}
	sub, ok, err := s.webhooks.GetSubscription(ctx, att.SubscriptionID)
	if err != nil || !ok || sub.TenantType != p.TenantType || sub.TenantID != p.TenantID {
		return DeliveryAttempt{}, errNotFound("attempt")
	}
	if att.Status != DeliveryDead {
		return DeliveryAttempt{}, fmt.Errorf("not_dead")
	}
	now := s.now()
	att.Status = DeliveryPending
	att.AttemptCount = 0
	att.LastError = ""
	att.NextRetryAt = &now
	if err := s.webhooks.UpdateAttempt(ctx, att); err != nil {
		return DeliveryAttempt{}, err
	}
	return att, nil
}

// CreateExportJob enqueues an async partner export.
func (s *Service) CreateExportJob(ctx context.Context, p Principal, resource, format string, from, to *time.Time) (ExportJob, error) {
	if !PartnerExportsEnabled() {
		return ExportJob{}, fmt.Errorf("exports_disabled")
	}
	if s.exports == nil {
		return ExportJob{}, fmt.Errorf("exports_unavailable")
	}
	resource = strings.ToLower(strings.TrimSpace(resource))
	format = strings.ToLower(strings.TrimSpace(format))
	switch resource {
	case ExportResourceOrders, ExportResourceInvoices, ExportResourceInventory, ExportResourceLedger, ExportResourceJournals:
	default:
		return ExportJob{}, fmt.Errorf("invalid_resource")
	}
	if format == "" {
		format = ExportFormatCSV
	}
	if format != ExportFormatCSV && format != ExportFormatJSON && format != ExportFormatXML {
		return ExportJob{}, fmt.Errorf("invalid_format")
	}
	j := ExportJob{
		JobID:      uuid.NewString(),
		TenantType: p.TenantType,
		TenantID:   p.TenantID,
		Resource:   resource,
		Format:     format,
		Status:     ExportStatusPending,
		FromDate:   from,
		ToDate:     to,
		CreatedAt:  s.now(),
	}
	if err := s.exports.InsertJob(ctx, j); err != nil {
		return ExportJob{}, err
	}
	return j, nil
}

// GetExportJob returns a tenant-scoped job (+ optional download URL).
func (s *Service) GetExportJob(ctx context.Context, p Principal, jobID string) (ExportJob, string, error) {
	if s.exports == nil {
		return ExportJob{}, "", fmt.Errorf("exports_unavailable")
	}
	j, ok, err := s.exports.GetJob(ctx, jobID)
	if err != nil || !ok || j.TenantType != p.TenantType || j.TenantID != p.TenantID {
		return ExportJob{}, "", errNotFound("job")
	}
	var dl string
	if j.Status == ExportStatusSucceeded && j.ObjectPath != "" {
		dl, _ = SignDownloadURL(j.ObjectPath)
	}
	return j, dl, nil
}

// ListExportJobs lists recent jobs for the tenant.
func (s *Service) ListExportJobs(ctx context.Context, p Principal, limit int) ([]ExportJob, error) {
	if s.exports == nil {
		return nil, fmt.Errorf("exports_unavailable")
	}
	return s.exports.ListJobs(ctx, p.TenantType, p.TenantID, limit)
}

// UpsertSftpConfig stores SFTP destination metadata (secret via SecretRef only).
func (s *Service) UpsertSftpConfig(ctx context.Context, tenantType, tenantID string, cfg SftpConfig) error {
	if s.sftp == nil {
		return fmt.Errorf("sftp_unavailable")
	}
	cfg.TenantType = tenantType
	cfg.TenantID = tenantID
	if cfg.Port <= 0 {
		cfg.Port = 22
	}
	if strings.TrimSpace(cfg.RemoteDir) == "" {
		cfg.RemoteDir = "/"
	}
	// Local EDI-only configs may omit host when PARTNER_EDI_LOCAL_ROOT is set.
	if strings.TrimSpace(cfg.Host) == "" && partnerEDILocalRoot() == "" {
		return fmt.Errorf("invalid_sftp_config")
	}
	if strings.TrimSpace(cfg.Host) != "" {
		if strings.TrimSpace(cfg.Username) == "" || strings.TrimSpace(cfg.SecretRef) == "" {
			return fmt.Errorf("invalid_sftp_config")
		}
	} else {
		// Placeholder for local-root EDI
		if cfg.Username == "" {
			cfg.Username = "local"
		}
		if cfg.SecretRef == "" {
			cfg.SecretRef = "local"
		}
		cfg.Host = "local"
	}
	normalizeSftpDirs(&cfg)
	cfg.IsActive = true
	return s.sftp.Upsert(ctx, cfg)
}

// GetSftpConfig returns config without secret material.
func (s *Service) GetSftpConfig(ctx context.Context, tenantType, tenantID string) (SftpConfig, bool, error) {
	if s.sftp == nil {
		return SftpConfig{}, false, nil
	}
	return s.sftp.Get(ctx, tenantType, tenantID)
}

// GetCoa returns the resolved chart of accounts for the tenant (defaults if unset).
func (s *Service) GetCoa(ctx context.Context, tenantType, tenantID string) (CoaMap, error) {
	if s.coa == nil {
		return DefaultCoa(), nil
	}
	stored, found, err := s.coa.Get(ctx, tenantType, tenantID)
	if err != nil {
		return CoaMap{}, err
	}
	out := ResolveCoa(stored, found)
	out.TenantType = tenantType
	out.TenantID = tenantID
	return out, nil
}

// UpsertCoa stores tenant CoA overrides (empty fields keep previous or become defaults on resolve).
func (s *Service) UpsertCoa(ctx context.Context, tenantType, tenantID, updatedBy string, m CoaMap) (CoaMap, error) {
	if s.coa == nil {
		return CoaMap{}, fmt.Errorf("coa_unavailable")
	}
	m.TenantType = tenantType
	m.TenantID = tenantID
	m.UpdatedBy = updatedBy
	NormalizeCoa(&m)
	if err := ValidateCoaAccounts(m); err != nil {
		return CoaMap{}, err
	}
	// Fill blanks with defaults so Spanner NOT NULL columns are satisfied.
	def := DefaultCoa()
	if m.AccountAR == "" {
		m.AccountAR = def.AccountAR
	}
	if m.AccountRevenue == "" {
		m.AccountRevenue = def.AccountRevenue
	}
	if m.AccountBankCash == "" {
		m.AccountBankCash = def.AccountBankCash
	}
	m.UpdatedAt = s.now()
	if err := s.coa.Upsert(ctx, m); err != nil {
		return CoaMap{}, err
	}
	return s.GetCoa(ctx, tenantType, tenantID)
}

// ListEdiDocuments returns recent EDI ledger rows for the tenant.
func (s *Service) ListEdiDocuments(ctx context.Context, p Principal, limit int) ([]EdiDocument, error) {
	if s.ediDocs == nil {
		return nil, fmt.Errorf("edi_unavailable")
	}
	return s.ediDocs.ListByTenant(ctx, p.TenantType, p.TenantID, limit)
}

// GetEdiDocument returns a tenant-scoped document.
func (s *Service) GetEdiDocument(ctx context.Context, p Principal, documentID string) (EdiDocument, error) {
	if s.ediDocs == nil {
		return EdiDocument{}, fmt.Errorf("edi_unavailable")
	}
	d, ok, err := s.ediDocs.Get(ctx, documentID)
	if err != nil || !ok || d.TenantType != p.TenantType || d.TenantID != p.TenantID {
		return EdiDocument{}, errNotFound("document")
	}
	return d, nil
}

// ReplayEdiDocument requeues FAILED inbound (RECEIVED) or outbound (RECEIVED) for retry.
func (s *Service) ReplayEdiDocument(ctx context.Context, p Principal, documentID string) (EdiDocument, error) {
	d, err := s.GetEdiDocument(ctx, p, documentID)
	if err != nil {
		return EdiDocument{}, err
	}
	if d.Status != EdiStatusFailed && d.Status != EdiStatusEmitted {
		return EdiDocument{}, fmt.Errorf("not_replayable")
	}
	d.Status = EdiStatusReceived
	d.Error = ""
	d.FinishedAt = nil
	if err := s.ediDocs.Update(ctx, d); err != nil {
		return EdiDocument{}, err
	}
	return d, nil
}

// EnqueueEdiFromEvent maps Kafka envelopes to outbound EDI documents (supplier tenant).
func (s *Service) EnqueueEdiFromEvent(ctx context.Context, eventType string, envelope map[string]any) {
	if s == nil || s.ediOut == nil || !PartnerEDIEnabled() {
		return
	}
	supplierID, _ := envelope["supplier_id"].(string)
	if supplierID == "" {
		return
	}
	for _, m := range MapEventToOutboundDocs(eventType, envelope) {
		if err := s.ediOut.EnqueueOutbound(ctx, TenantSupplier, supplierID, m.DocType, m.ExtID, m.OrderID); err != nil {
			s.log.Warn("edi enqueue", "err", err, "doc_type", m.DocType)
		}
	}
}
