package partner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/edi"
)

// PartnerEDIEnabled gates EDI workers/APIs.
func PartnerEDIEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("PARTNER_EDI_ENABLED")))
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func partnerEDILocalRoot() string {
	return strings.TrimSpace(os.Getenv("PARTNER_EDI_LOCAL_ROOT"))
}

// RetailerGeo resolves lat/lng/h3 when ORDERS omits LOC.
type RetailerGeo struct {
	Lat    float64
	Lng    float64
	H3Cell string
}

// EdiAckEnqueuer queues CONTRL/APERAK after inbound processing.
type EdiAckEnqueuer interface {
	EnqueueFunctionalAck(ctx context.Context, tenantType, tenantID, refDocID, orderID string, accepted bool, reason string)
}

// EdiInboundWorker polls SFTP/local inbound for ORDERS/ORDRSP/INVOIC files.
type EdiInboundWorker struct {
	ediDocs  EdiDocumentRepository
	sftp     SftpConfigRepository
	profiles EdiProfileRepository
	svc      *Service
	acks     EdiAckEnqueuer
	log      *slog.Logger
	now      func() time.Time
	// ResolveGeo optional; when nil, ORDERS must include LOC.
	ResolveGeo   func(ctx context.Context, retailerID string) (RetailerGeo, error)
	SecretLoader func(secretRef string) (string, error)
}

func NewEdiInboundWorker(docs EdiDocumentRepository, sftp SftpConfigRepository, svc *Service, log *slog.Logger) *EdiInboundWorker {
	if log == nil {
		log = slog.Default()
	}
	return &EdiInboundWorker{
		ediDocs: docs, sftp: sftp, svc: svc, log: log,
		profiles:     NewMemoryEdiProfiles(),
		now:          func() time.Time { return time.Now().UTC() },
		SecretLoader: LoadSecretRef,
	}
}

// SetEdiProfiles wires G5.A tenant profile packs.
func (w *EdiInboundWorker) SetEdiProfiles(repo EdiProfileRepository) {
	if w != nil && repo != nil {
		w.profiles = repo
	}
}

func (w *EdiInboundWorker) profileAllows(ctx context.Context, tenantType, tenantID, docType string) bool {
	p := ResolveEdiProfile(ctx, w.profiles, tenantType, tenantID)
	return p.DocEnabled(docType)
}

// SetAckEnqueuer wires CONTRL/APERAK emission (typically the outbound worker).
func (w *EdiInboundWorker) SetAckEnqueuer(acks EdiAckEnqueuer) {
	if w != nil {
		w.acks = acks
	}
}

func (w *EdiInboundWorker) Start(ctx context.Context, interval time.Duration) {
	if w == nil || !PartnerEDIEnabled() {
		return
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				w.log.Warn("edi inbound tick failed", "err", err)
			}
		}
	}
}

func (w *EdiInboundWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.ediDocs == nil || w.sftp == nil || !PartnerEDIEnabled() {
		return 0, nil
	}
	cfgs, err := w.sftp.ListEdiEnabled(ctx, 50)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, cfg := range cfgs {
		c, err := w.processTenant(ctx, cfg)
		if err != nil {
			w.log.Warn("edi inbound tenant failed", "tenant", cfg.TenantID, "err", err)
		}
		n += c
	}
	// Local-root tenants without live SFTP: scan PARTNER_EDI_LOCAL_ROOT/{SUPPLIER|tenantId}/inbound
	if root := partnerEDILocalRoot(); root != "" {
		c, err := w.processLocalRoot(ctx, root)
		if err != nil {
			w.log.Warn("edi local inbound failed", "err", err)
		}
		n += c
	}
	return n, nil
}

func (w *EdiInboundWorker) processLocalRoot(ctx context.Context, root string) (int, error) {
	// Layout: {root}/{tenantType}/{tenantId}/inbound/*.edi
	n := 0
	for _, tt := range []string{TenantSupplier, TenantRetailer} {
		base := filepath.Join(root, strings.ToLower(tt))
		entries, err := os.ReadDir(base)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			tenantID := e.Name()
			cfg, ok, _ := w.sftp.Get(ctx, tt, tenantID)
			if !ok || !cfg.EdiEnabled {
				// Allow local-only processing when a stub config exists OR synthesize
				cfg = SftpConfig{TenantType: tt, TenantID: tenantID, EdiEnabled: true, IsActive: true}
				normalizeSftpDirs(&cfg)
			}
			inDir := filepath.Join(base, tenantID, cfg.InboundDir)
			files, err := os.ReadDir(inDir)
			if err != nil {
				continue
			}
			for _, f := range files {
				if f.IsDir() || !isInboundEDIFilename(f.Name()) {
					continue
				}
				path := filepath.Join(inDir, f.Name())
				body, err := os.ReadFile(path)
				if err != nil {
					continue
				}
				if err := w.ingestInboundBytes(ctx, cfg, f.Name(), body); err != nil {
					w.log.Warn("edi local ingest", "file", f.Name(), "err", err)
					continue
				}
				arch := filepath.Join(base, tenantID, cfg.ArchiveDir)
				_ = os.MkdirAll(arch, 0o755)
				_ = os.Rename(path, filepath.Join(arch, f.Name()))
				n++
			}
		}
	}
	return n, nil
}

func (w *EdiInboundWorker) processTenant(ctx context.Context, cfg SftpConfig) (int, error) {
	normalizeSftpDirs(&cfg)
	if !PartnerSFTPEnabled() || strings.TrimSpace(cfg.Host) == "" {
		return 0, nil
	}
	loader := w.SecretLoader
	if loader == nil {
		loader = LoadSecretRef
	}
	secret, err := loader(cfg.SecretRef)
	if err != nil || secret == "" {
		return 0, fmt.Errorf("sftp_secret: %w", err)
	}
	remoteIn := joinRemote(cfg.RemoteDir, cfg.InboundDir)
	files, err := ListSFTP(ctx, cfg, secret, remoteIn)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, f := range files {
		if !isInboundEDIFilename(f.Name) {
			continue
		}
		remotePath := joinRemote(remoteIn, f.Name)
		body, err := DownloadSFTP(ctx, cfg, secret, remotePath)
		if err != nil {
			w.log.Warn("edi download", "file", f.Name, "err", err)
			continue
		}
		if err := w.ingestInboundBytes(ctx, cfg, f.Name, body); err != nil {
			w.log.Warn("edi ingest", "file", f.Name, "err", err)
			continue
		}
		archPath := joinRemote(joinRemote(cfg.RemoteDir, cfg.ArchiveDir), f.Name)
		_ = RenameSFTP(ctx, cfg, secret, remotePath, archPath)
		n++
	}
	return n, nil
}

func isOrdersFilename(name string) bool {
	n := strings.ToUpper(name)
	return strings.HasPrefix(n, "ORDERS_") || strings.HasSuffix(n, ".ORDERS") ||
		(strings.Contains(n, "ORDERS") && (strings.HasSuffix(n, ".EDI") || strings.HasSuffix(n, ".TXT")))
}

func isInboundEDIFilename(name string) bool {
	n := strings.ToUpper(name)
	if isOrdersFilename(name) {
		return true
	}
	for _, tok := range []string{"ORDRSP", "INVOIC", "CONTRL", "APERAK", "PRICAT", "INVRPT", "SLSRPT", "RECADV", "ORDCHG", "DELFOR", "REMADV"} {
		if strings.Contains(n, tok) && (strings.HasSuffix(n, ".EDI") || strings.HasSuffix(n, ".TXT") || strings.HasPrefix(n, tok+"_")) {
			return true
		}
	}
	return strings.HasSuffix(n, ".EDI") || strings.HasSuffix(n, ".TXT")
}

// IngestORDERSBytes is the AS2/SFTP transport boundary into ORDERS ingest (codecs unchanged).
func (w *EdiInboundWorker) IngestORDERSBytes(ctx context.Context, tenantType, tenantID, remoteName string, body []byte) error {
	if w == nil {
		return fmt.Errorf("edi_unavailable")
	}
	cfg := SftpConfig{TenantType: tenantType, TenantID: tenantID, EdiEnabled: true, IsActive: true}
	normalizeSftpDirs(&cfg)
	if remoteName == "" {
		remoteName = "as2:ORDERS.edi"
	}
	return w.ingestInboundBytes(ctx, cfg, remoteName, body)
}

func (w *EdiInboundWorker) ingestInboundBytes(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	docType, err := edi.DetectDocType(string(body))
	if err != nil {
		// Filename heuristic when UNA parse fails early.
		n := strings.ToUpper(remoteName)
		switch {
		case strings.Contains(n, "ORDRSP"):
			docType = EdiDocORDRSP
		case strings.Contains(n, "INVOIC"):
			docType = EdiDocINVOIC
		default:
			docType = EdiDocORDERS
		}
	}
	// G5.A: tenant profile may disable doc types.
	if !w.profileAllows(ctx, cfg.TenantType, cfg.TenantID, docType) {
		w.log.Info("edi inbound skipped by profile",
			"tenant_type", cfg.TenantType, "tenant_id", cfg.TenantID, "doc_type", docType)
		if w.acks != nil {
			w.acks.EnqueueFunctionalAck(ctx, cfg.TenantType, cfg.TenantID, remoteName, "", false, "profile_doc_disabled")
		}
		return fmt.Errorf("profile_doc_disabled:%s", docType)
	}
	switch strings.ToUpper(docType) {
	case EdiDocORDERS:
		return w.ingestORDERS(ctx, cfg, remoteName, body)
	case EdiDocORDRSP:
		return w.ingestORDRSP(ctx, cfg, remoteName, body)
	case EdiDocINVOIC:
		return w.ingestINVOIC(ctx, cfg, remoteName, body)
	case EdiDocPRICAT:
		return w.ingestPRICAT(ctx, cfg, remoteName, body)
	case EdiDocINVRPT:
		return w.ingestINVRPT(ctx, cfg, remoteName, body)
	case EdiDocSLSRPT, EdiDocRECADV, EdiDocORDCHG, EdiDocDELFOR, EdiDocREMADV:
		// Parse for validity, then ledger-record (application side-effects follow-on).
		return w.ingestLedgerOnly(ctx, cfg, remoteName, body, strings.ToUpper(docType))
	case EdiDocCONTRL, EdiDocAPERAK:
		// Partner ACKs of our outbound — record only.
		return w.ingestAckRecord(ctx, cfg, remoteName, body, strings.ToUpper(docType))
	default:
		return fmt.Errorf("unsupported_inbound_doc_type:%s", docType)
	}
}

func (w *EdiInboundWorker) emitAcks(ctx context.Context, cfg SftpConfig, refDocID, orderID string, accepted bool, reason string) {
	if w == nil || w.acks == nil {
		return
	}
	w.acks.EnqueueFunctionalAck(ctx, cfg.TenantType, cfg.TenantID, refDocID, orderID, accepted, reason)
}

func (w *EdiInboundWorker) ingestORDERS(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	if w.svc == nil || w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	msg, err := edi.ParseORDERS(string(body))
	if err != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, err.Error())
		return err
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])

	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocORDERS, msg.ExternalDocID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil // idempotent
	}

	doc := EdiDocument{
		DocumentID:    uuid.NewString(),
		TenantType:    cfg.TenantType,
		TenantID:      cfg.TenantID,
		Direction:     EdiDirectionIn,
		DocType:       EdiDocORDERS,
		ExternalDocID: msg.ExternalDocID,
		Status:        EdiStatusReceived,
		RemoteName:    remoteName,
		PayloadHash:   hashHex,
		CreatedAt:     w.now(),
	}
	if ok {
		doc = existing
		doc.Status = EdiStatusReceived
		doc.RemoteName = remoteName
		doc.PayloadHash = hashHex
		doc.Error = ""
		_ = w.ediDocs.Update(ctx, doc)
	} else if err := w.ediDocs.Insert(ctx, doc); err != nil {
		// race: reload
		existing, ok, _ = w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocORDERS, msg.ExternalDocID)
		if ok && existing.Status == EdiStatusProcessed {
			return nil
		}
		if !ok {
			return err
		}
		doc = existing
	}

	p, retailerID, req, err := w.mapOrdersToCreate(ctx, cfg, msg)
	if err != nil {
		w.emitAcks(ctx, cfg, msg.ExternalDocID, "", false, err.Error())
		return w.failDoc(ctx, doc, err.Error())
	}
	resp, err := w.svc.CreateOrder(ctx, p, retailerID, req)
	if err != nil {
		w.emitAcks(ctx, cfg, msg.ExternalDocID, "", false, err.Error())
		return w.failDoc(ctx, doc, err.Error())
	}
	now := w.now()
	doc.Status = EdiStatusProcessed
	doc.OrderID = resp.OrderID
	doc.FinishedAt = &now
	doc.Error = ""
	if err := w.ediDocs.Update(ctx, doc); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, msg.ExternalDocID, resp.OrderID, true, "")
	return nil
}

func (w *EdiInboundWorker) ingestORDRSP(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	msg, err := edi.ParseORDRSP(string(body))
	if err != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, err.Error())
		return err
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocORDRSP, msg.ExternalDocID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil
	}
	doc := EdiDocument{
		DocumentID:    uuid.NewString(),
		TenantType:    cfg.TenantType,
		TenantID:      cfg.TenantID,
		Direction:     EdiDirectionIn,
		DocType:       EdiDocORDRSP,
		ExternalDocID: msg.ExternalDocID,
		OrderID:       msg.RefOrderID,
		Status:        EdiStatusProcessed,
		RemoteName:    remoteName,
		PayloadHash:   hashHex,
		CreatedAt:     w.now(),
	}
	now := w.now()
	doc.FinishedAt = &now
	if !msg.Accepted {
		doc.Status = EdiStatusFailed
		doc.Error = "ordrsp_rejected:" + msg.ResponseCode
	}
	if ok {
		doc.DocumentID = existing.DocumentID
		doc.CreatedAt = existing.CreatedAt
		if err := w.ediDocs.Update(ctx, doc); err != nil {
			return err
		}
	} else if err := w.ediDocs.Insert(ctx, doc); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, msg.ExternalDocID, msg.RefOrderID, msg.Accepted, doc.Error)
	return nil
}

func (w *EdiInboundWorker) ingestINVOIC(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	msg, err := edi.ParseINVOIC(string(body))
	if err != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, err.Error())
		return err
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocINVOIC, msg.ExternalDocID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil
	}
	now := w.now()
	doc := EdiDocument{
		DocumentID:    uuid.NewString(),
		TenantType:    cfg.TenantType,
		TenantID:      cfg.TenantID,
		Direction:     EdiDirectionIn,
		DocType:       EdiDocINVOIC,
		ExternalDocID: msg.ExternalDocID,
		OrderID:       msg.RefOrderID,
		Status:        EdiStatusProcessed,
		RemoteName:    remoteName,
		PayloadHash:   hashHex,
		CreatedAt:     now,
		FinishedAt:    &now,
	}
	if ok {
		doc.DocumentID = existing.DocumentID
		doc.CreatedAt = existing.CreatedAt
		if err := w.ediDocs.Update(ctx, doc); err != nil {
			return err
		}
	} else if err := w.ediDocs.Insert(ctx, doc); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, msg.ExternalDocID, msg.RefOrderID, true, "")
	return nil
}

func (w *EdiInboundWorker) ingestPRICAT(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	msg, err := edi.ParsePRICAT(string(body))
	if err != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, err.Error())
		return err
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocPRICAT, msg.ExternalDocID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil
	}
	now := w.now()
	doc := EdiDocument{
		DocumentID: uuid.NewString(), TenantType: cfg.TenantType, TenantID: cfg.TenantID,
		Direction: EdiDirectionIn, DocType: EdiDocPRICAT, ExternalDocID: msg.ExternalDocID,
		Status: EdiStatusReceived, RemoteName: remoteName, PayloadHash: hashHex, CreatedAt: now,
	}
	if ok {
		doc.DocumentID = existing.DocumentID
		doc.CreatedAt = existing.CreatedAt
	}
	if w.svc != nil && cfg.TenantType == TenantSupplier {
		items := make([]PriceUpsertItem, 0, len(msg.Lines))
		for _, ln := range msg.Lines {
			cur := ln.Currency
			if cur == "" {
				cur = "UZS"
			}
			items = append(items, PriceUpsertItem{
				ExternalID: ln.SKU, PriceMinor: ln.PriceMinor, Currency: cur,
			})
		}
		p := Principal{TenantType: cfg.TenantType, TenantID: cfg.TenantID, Scopes: []string{"*"}}
		if _, uErr := w.svc.UpsertPrices(ctx, p, items); uErr != nil {
			doc.Status = EdiStatusFailed
			doc.Error = uErr.Error()
			doc.FinishedAt = &now
			_ = w.upsertDoc(ctx, doc, ok)
			w.emitAcks(ctx, cfg, msg.ExternalDocID, "", false, uErr.Error())
			return uErr
		}
	}
	doc.Status = EdiStatusProcessed
	doc.FinishedAt = &now
	if err := w.upsertDoc(ctx, doc, ok); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, msg.ExternalDocID, "", true, "")
	return nil
}

func (w *EdiInboundWorker) ingestINVRPT(ctx context.Context, cfg SftpConfig, remoteName string, body []byte) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	msg, err := edi.ParseINVRPT(string(body))
	if err != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, err.Error())
		return err
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, EdiDocINVRPT, msg.ExternalDocID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil
	}
	now := w.now()
	doc := EdiDocument{
		DocumentID: uuid.NewString(), TenantType: cfg.TenantType, TenantID: cfg.TenantID,
		Direction: EdiDirectionIn, DocType: EdiDocINVRPT, ExternalDocID: msg.ExternalDocID,
		Status: EdiStatusReceived, RemoteName: remoteName, PayloadHash: hashHex, CreatedAt: now,
	}
	if ok {
		doc.DocumentID = existing.DocumentID
		doc.CreatedAt = existing.CreatedAt
	}
	if w.svc != nil && cfg.TenantType == TenantSupplier {
		items := make([]StockUpsertItem, 0, len(msg.Lines))
		for _, ln := range msg.Lines {
			items = append(items, StockUpsertItem{
				ExternalID: ln.SKU, WarehouseID: ln.Warehouse, QuantityOnHand: ln.QtyOnHand,
			})
		}
		p := Principal{TenantType: cfg.TenantType, TenantID: cfg.TenantID, Scopes: []string{"*"}}
		if _, uErr := w.svc.UpsertStock(ctx, p, items); uErr != nil {
			doc.Status = EdiStatusFailed
			doc.Error = uErr.Error()
			doc.FinishedAt = &now
			_ = w.upsertDoc(ctx, doc, ok)
			w.emitAcks(ctx, cfg, msg.ExternalDocID, "", false, uErr.Error())
			return uErr
		}
	}
	doc.Status = EdiStatusProcessed
	doc.FinishedAt = &now
	if err := w.upsertDoc(ctx, doc, ok); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, msg.ExternalDocID, "", true, "")
	return nil
}

func (w *EdiInboundWorker) ingestLedgerOnly(ctx context.Context, cfg SftpConfig, remoteName string, body []byte, docType string) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	var extID string
	var parseErr error
	switch docType {
	case EdiDocSLSRPT:
		var m edi.SlsrptMessage
		m, parseErr = edi.ParseSLSRPT(string(body))
		extID = m.ExternalDocID
	case EdiDocRECADV:
		var m edi.RecadvMessage
		m, parseErr = edi.ParseRECADV(string(body))
		extID = m.ExternalDocID
	case EdiDocORDCHG:
		var m edi.OrdchgMessage
		m, parseErr = edi.ParseORDCHG(string(body))
		extID = m.ExternalDocID
	case EdiDocDELFOR:
		var m edi.DelforMessage
		m, parseErr = edi.ParseDELFOR(string(body))
		extID = m.ExternalDocID
	case EdiDocREMADV:
		var m edi.RemadvMessage
		m, parseErr = edi.ParseREMADV(string(body))
		extID = m.ExternalDocID
	default:
		parseErr = fmt.Errorf("unsupported_inbound_doc_type:%s", docType)
	}
	if parseErr != nil {
		w.emitAcks(ctx, cfg, remoteName, "", false, parseErr.Error())
		return parseErr
	}
	if extID == "" {
		extID = remoteName
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	existing, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, docType, extID)
	if err != nil {
		return err
	}
	if ok && existing.Status == EdiStatusProcessed {
		return nil
	}
	now := w.now()
	doc := EdiDocument{
		DocumentID: uuid.NewString(), TenantType: cfg.TenantType, TenantID: cfg.TenantID,
		Direction: EdiDirectionIn, DocType: docType, ExternalDocID: extID,
		Status: EdiStatusProcessed, RemoteName: remoteName, PayloadHash: hashHex,
		CreatedAt: now, FinishedAt: &now,
	}
	if ok {
		doc.DocumentID = existing.DocumentID
		doc.CreatedAt = existing.CreatedAt
	}
	if err := w.upsertDoc(ctx, doc, ok); err != nil {
		return err
	}
	w.emitAcks(ctx, cfg, extID, "", true, "")
	return nil
}

func (w *EdiInboundWorker) upsertDoc(ctx context.Context, doc EdiDocument, existed bool) error {
	if existed {
		return w.ediDocs.Update(ctx, doc)
	}
	return w.ediDocs.Insert(ctx, doc)
}

func (w *EdiInboundWorker) ingestAckRecord(ctx context.Context, cfg SftpConfig, remoteName string, body []byte, docType string) error {
	if w.ediDocs == nil {
		return fmt.Errorf("edi_unavailable")
	}
	hash := sha256.Sum256(body)
	hashHex := hex.EncodeToString(hash[:])
	ext := remoteName + ":" + hashHex[:12]
	_, ok, err := w.ediDocs.GetByExternal(ctx, cfg.TenantType, cfg.TenantID, EdiDirectionIn, docType, ext)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	now := w.now()
	return w.ediDocs.Insert(ctx, EdiDocument{
		DocumentID:    uuid.NewString(),
		TenantType:    cfg.TenantType,
		TenantID:      cfg.TenantID,
		Direction:     EdiDirectionIn,
		DocType:       docType,
		ExternalDocID: ext,
		Status:        EdiStatusProcessed,
		RemoteName:    remoteName,
		PayloadHash:   hashHex,
		CreatedAt:     now,
		FinishedAt:    &now,
	})
}

func (w *EdiInboundWorker) failDoc(ctx context.Context, doc EdiDocument, msg string) error {
	now := w.now()
	doc.Status = EdiStatusFailed
	doc.Error = msg
	doc.FinishedAt = &now
	_ = w.ediDocs.Update(ctx, doc)
	return fmt.Errorf("%s", msg)
}

func (w *EdiInboundWorker) mapOrdersToCreate(ctx context.Context, cfg SftpConfig, msg edi.OrdersMessage) (Principal, string, order.CreateRequest, error) {
	p := Principal{TenantType: cfg.TenantType, TenantID: cfg.TenantID, Scopes: []string{"*"}}
	var retailerID string
	req := order.CreateRequest{Source: order.OrderSourcePartnerEDI}
	for _, ln := range msg.Lines {
		req.LineItems = append(req.LineItems, order.LineItem{SKU: ln.SKU, Quantity: ln.Qty})
	}
	req.RequestedDeliveryDate = msg.DeliveryDate
	req.Lat = msg.Lat
	req.Lng = msg.Lng
	req.H3Cell = msg.H3Cell

	switch cfg.TenantType {
	case TenantSupplier:
		retailerID = strings.TrimSpace(msg.BuyerRef)
		if retailerID == "" {
			return p, "", req, fmt.Errorf("buyer_ref_required")
		}
		req.SupplierID = cfg.TenantID
		if msg.SellerRef != "" && msg.SellerRef != cfg.TenantID {
			return p, "", req, fmt.Errorf("seller_mismatch")
		}
	case TenantRetailer:
		retailerID = cfg.TenantID
		req.SupplierID = strings.TrimSpace(msg.SellerRef)
		if req.SupplierID == "" {
			return p, "", req, fmt.Errorf("seller_ref_required")
		}
		if msg.BuyerRef != "" && msg.BuyerRef != cfg.TenantID {
			return p, "", req, fmt.Errorf("buyer_mismatch")
		}
	default:
		return p, "", req, fmt.Errorf("invalid_tenant")
	}

	if (req.Lat == 0 && req.Lng == 0) || req.H3Cell == "" {
		if w.ResolveGeo == nil {
			return p, "", req, fmt.Errorf("geo_required")
		}
		geo, err := w.ResolveGeo(ctx, retailerID)
		if err != nil {
			return p, "", req, fmt.Errorf("geo_resolve: %w", err)
		}
		if req.Lat == 0 && req.Lng == 0 {
			req.Lat, req.Lng = geo.Lat, geo.Lng
		}
		if req.H3Cell == "" {
			req.H3Cell = geo.H3Cell
		}
	}
	if req.H3Cell == "" || (req.Lat == 0 && req.Lng == 0) {
		return p, "", req, fmt.Errorf("geo_incomplete")
	}
	return p, retailerID, req, nil
}
