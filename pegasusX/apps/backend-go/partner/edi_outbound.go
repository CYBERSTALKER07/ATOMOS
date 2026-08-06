package partner

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/as2"
	"github.com/pegasusx/pegasusx/apps/backend-go/partner/edi"
	"google.golang.org/api/iterator"
)

// EdiOutboundWorker emits ORDRSP/DESADV/INVOIC files for queued documents.
type EdiOutboundWorker struct {
	ediDocs      EdiDocumentRepository
	sftp         SftpConfigRepository
	as2          As2ConfigRepository
	orders       *order.Service
	spanner      *spanner.Client
	log          *slog.Logger
	now          func() time.Time
	SecretLoader func(secretRef string) (string, error)
	Uploader     func(ctx context.Context, cfg SftpConfig, secret, remoteDir, localPath, remoteName string) error
	AS2Sender    func(ctx context.Context, req as2.SendRequest) (as2.SendResult, error)
}

func NewEdiOutboundWorker(docs EdiDocumentRepository, sftp SftpConfigRepository, orders *order.Service, client *spanner.Client, log *slog.Logger) *EdiOutboundWorker {
	if log == nil {
		log = slog.Default()
	}
	cli := as2.NewClient()
	return &EdiOutboundWorker{
		ediDocs: docs, sftp: sftp, orders: orders, spanner: client, log: log,
		now:          func() time.Time { return time.Now().UTC() },
		SecretLoader: LoadSecretRef,
		Uploader:     UploadSFTPToDir,
		AS2Sender:    cli.Send,
	}
}

// SetAs2Repository wires optional AS2 outbound push.
func (w *EdiOutboundWorker) SetAs2Repository(repo As2ConfigRepository) {
	if w != nil {
		w.as2 = repo
	}
}

func (w *EdiOutboundWorker) Start(ctx context.Context, interval time.Duration) {
	if w == nil || !PartnerEDIEnabled() {
		return
	}
	if interval <= 0 {
		interval = 20 * time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := w.RunOnce(ctx); err != nil {
				w.log.Warn("edi outbound tick failed", "err", err)
			}
		}
	}
}

func (w *EdiOutboundWorker) RunOnce(ctx context.Context) (int, error) {
	if w == nil || w.ediDocs == nil || !PartnerEDIEnabled() {
		return 0, nil
	}
	pending, err := w.ediDocs.ListPendingOutbound(ctx, 20)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, d := range pending {
		if err := w.emit(ctx, d); err != nil {
			w.log.Warn("edi emit failed", "document_id", d.DocumentID, "err", err)
		} else {
			n++
		}
	}
	return n, nil
}

// EnqueueOutbound inserts a RECEIVED outbound doc if not already present.
func (w *EdiOutboundWorker) EnqueueOutbound(ctx context.Context, tenantType, tenantID, docType, externalDocID, orderID string) error {
	if w == nil || w.ediDocs == nil || !PartnerEDIEnabled() {
		return nil
	}
	if tenantType == "" || tenantID == "" || docType == "" || externalDocID == "" {
		return fmt.Errorf("invalid_edi_enqueue")
	}
	_, ok, err := w.ediDocs.GetByExternal(ctx, tenantType, tenantID, EdiDirectionOut, docType, externalDocID)
	if err != nil || ok {
		return err
	}
	cfg, found, err := w.sftp.Get(ctx, tenantType, tenantID)
	if err != nil {
		return err
	}
	if !found || !cfg.EdiEnabled {
		// Still allow enqueue for supplier mirror when retailer events carry supplier_id —
		// caller should pass supplier tenant. Skip when EDI off.
		return nil
	}
	d := EdiDocument{
		DocumentID:    uuid.NewString(),
		TenantType:    tenantType,
		TenantID:      tenantID,
		Direction:     EdiDirectionOut,
		DocType:       docType,
		ExternalDocID: externalDocID,
		OrderID:       orderID,
		Status:        EdiStatusReceived,
		CreatedAt:     w.now(),
	}
	return w.ediDocs.Insert(ctx, d)
}

func (w *EdiOutboundWorker) emit(ctx context.Context, d EdiDocument) error {
	cfg, ok, err := w.sftp.Get(ctx, d.TenantType, d.TenantID)
	if err != nil || !ok || !cfg.EdiEnabled {
		return w.fail(ctx, d, "edi_not_configured")
	}
	normalizeSftpDirs(&cfg)

	snap, err := w.loadSnapshot(ctx, d.OrderID)
	if err != nil {
		return w.fail(ctx, d, err.Error())
	}

	var body string
	switch d.DocType {
	case EdiDocORDRSP:
		body = edi.BuildORDRSP(snap, d.ExternalDocID)
	case EdiDocDESADV:
		body = edi.BuildDESADV(snap, d.ExternalDocID)
	case EdiDocINVOIC:
		body = edi.BuildINVOIC(snap, nil, d.ExternalDocID)
	default:
		return w.fail(ctx, d, "unknown_doc_type")
	}

	remoteName := fmt.Sprintf("%s_%s_%d.edi", d.DocType, sanitizeName(d.ExternalDocID), w.now().Unix())
	objectPath := fmt.Sprintf("partner-edi/%s/%s/%s", strings.ToLower(d.TenantType), d.TenantID, remoteName)
	localPath, err := w.writeLocal(objectPath, []byte(body))
	if err != nil {
		return w.fail(ctx, d, "write_failed:"+err.Error())
	}

	// Always write under local EDI root / temp; optionally SFTP push.
	if PartnerSFTPEnabled() && strings.TrimSpace(cfg.Host) != "" {
		loader := w.SecretLoader
		if loader == nil {
			loader = LoadSecretRef
		}
		secret, err := loader(cfg.SecretRef)
		if err != nil || secret == "" {
			return w.fail(ctx, d, "sftp_secret_unavailable")
		}
		outDir := joinRemote(cfg.RemoteDir, cfg.OutboundDir)
		up := w.Uploader
		if up == nil {
			up = UploadSFTPToDir
		}
		if err := up(ctx, cfg, secret, outDir, localPath, remoteName); err != nil {
			return w.fail(ctx, d, "sftp_upload:"+err.Error())
		}
	} else if root := partnerEDILocalRoot(); root != "" {
		destDir := filepath.Join(root, strings.ToLower(d.TenantType), d.TenantID, cfg.OutboundDir)
		_ = os.MkdirAll(destDir, 0o755)
		_ = os.WriteFile(filepath.Join(destDir, remoteName), []byte(body), 0o644)
	}

	as2Note := ""
	if PartnerAS2Enabled() && w.as2 != nil {
		if note, err := w.pushAS2(ctx, d, []byte(body), remoteName); err != nil {
			as2Note = "as2_push_failed:" + err.Error()
			w.log.Warn("as2 outbound push failed", "doc", d.DocumentID, "err", err)
		} else if note != "" {
			as2Note = note
			remoteName = note
		}
	}

	now := w.now()
	d.Status = EdiStatusEmitted
	d.ObjectPath = objectPath
	d.RemoteName = remoteName
	d.FinishedAt = &now
	d.Error = as2Note
	return w.ediDocs.Update(ctx, d)
}

func (w *EdiOutboundWorker) pushAS2(ctx context.Context, d EdiDocument, body []byte, filename string) (string, error) {
	cfg, ok, err := w.as2.Get(ctx, d.TenantType, d.TenantID)
	if err != nil {
		return "", err
	}
	if !ok || !cfg.As2Enabled || strings.TrimSpace(cfg.PartnerURL) == "" {
		return "", nil
	}
	loader := w.SecretLoader
	if loader == nil {
		loader = LoadSecretRef
	}
	plain := PartnerAS2InsecurePlain()
	req := as2.SendRequest{
		URL:        cfg.PartnerURL,
		From:       cfg.OurAs2Id,
		To:         cfg.PartnerAs2Id,
		EDI:        body,
		Filename:   filename,
		Plain:      plain,
		RequestMDN: true,
	}
	if !plain {
		certPEM, err1 := loader(cfg.OurCertSecretRef)
		keyPEM, err2 := loader(cfg.OurKeySecretRef)
		partnerPEM, err3 := loader(cfg.PartnerCertSecretRef)
		if err1 != nil || err2 != nil || err3 != nil {
			return "", fmt.Errorf("as2_secret_unavailable")
		}
		mat, err := as2.LoadMaterial([]byte(certPEM), []byte(keyPEM))
		if err != nil {
			return "", err
		}
		req.Signer = mat
		req.RecipientCert = []byte(partnerPEM)
	}
	sender := w.AS2Sender
	if sender == nil {
		sender = as2.NewClient().Send
	}
	res, err := sender(ctx, req)
	if err != nil {
		return "", err
	}
	return "as2:" + res.MessageID, nil
}

func (w *EdiOutboundWorker) fail(ctx context.Context, d EdiDocument, msg string) error {
	now := w.now()
	d.Status = EdiStatusFailed
	d.Error = msg
	d.FinishedAt = &now
	_ = w.ediDocs.Update(ctx, d)
	return fmt.Errorf("%s", msg)
}

func (w *EdiOutboundWorker) loadSnapshot(ctx context.Context, orderID string) (edi.OrderSnapshot, error) {
	if w.orders == nil || strings.TrimSpace(orderID) == "" {
		return edi.OrderSnapshot{}, fmt.Errorf("order_unavailable")
	}
	o, ok, err := w.orders.GetOrder(ctx, orderID)
	if err != nil || !ok {
		return edi.OrderSnapshot{}, fmt.Errorf("order_not_found")
	}
	snap := edi.OrderSnapshot{
		OrderID: o.OrderID, RetailerID: o.RetailerID, SupplierID: o.SupplierID,
		ManifestID: o.ManifestID,
		Status:     string(o.Status), Currency: o.Currency, TotalMinor: o.TotalMinor,
	}
	for _, li := range o.LineItems {
		snap.Lines = append(snap.Lines, edi.Line{SKU: li.SKU, Qty: li.Quantity})
	}
	snap.ShipUnits = w.loadShipUnits(ctx, o.OrderID, o.ManifestID)
	return snap, nil
}

// loadShipUnits reads ManifestShipUnits for DESADV SSCC segments (best-effort; empty on miss).
func (w *EdiOutboundWorker) loadShipUnits(ctx context.Context, orderID, manifestID string) []edi.ShipUnit {
	if w == nil || w.spanner == nil || strings.TrimSpace(orderID) == "" {
		return nil
	}
	var stmt spanner.Statement
	if mid := strings.TrimSpace(manifestID); mid != "" {
		stmt = spanner.Statement{
			SQL: `SELECT ManifestId, Sscc, OrderId, Sequence, Gtin
			      FROM ManifestShipUnits
			      WHERE ManifestId = @mid AND OrderId = @oid
			      ORDER BY Sequence`,
			Params: map[string]any{"mid": mid, "oid": orderID},
		}
	} else {
		stmt = spanner.Statement{
			SQL: `SELECT ManifestId, Sscc, OrderId, Sequence, Gtin
			      FROM ManifestShipUnits
			      WHERE OrderId = @oid
			      ORDER BY Sequence`,
			Params: map[string]any{"oid": orderID},
		}
	}
	iter := w.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make([]edi.ShipUnit, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			w.log.Debug("edi ship units load failed", "order_id", orderID, "err", err)
			return out
		}
		var u edi.ShipUnit
		var gtin spanner.NullString
		if err := row.Columns(&u.ManifestID, &u.SSCC, &u.OrderID, &u.Sequence, &gtin); err != nil {
			w.log.Debug("edi ship units row failed", "order_id", orderID, "err", err)
			return out
		}
		u.GTIN = gtin.StringVal
		out = append(out, u)
	}
	return out
}

func (w *EdiOutboundWorker) writeLocal(objectPath string, body []byte) (string, error) {
	var full string
	if root := partnerEDILocalRoot(); root != "" {
		full = filepath.Join(root, "_objects", filepath.FromSlash(objectPath))
	} else {
		full = filepath.Join(os.TempDir(), "pegasusx-partner-edi", filepath.FromSlash(objectPath))
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		return "", err
	}
	return full, nil
}

func sanitizeName(s string) string {
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			return r
		}
		return '_'
	}, s)
	if len(s) > 80 {
		s = s[:80]
	}
	return s
}

// MapEventToOutboundDocs returns (docType, externalDocID) pairs to enqueue for a domain event.
func MapEventToOutboundDocs(eventType string, envelope map[string]any) []struct{ DocType, ExtID, OrderID string } {
	orderID, _ := envelope["order_id"].(string)
	status, _ := envelope["status"].(string)
	if status == "" {
		status, _ = envelope["new_status"].(string)
	}
	out := make([]struct{ DocType, ExtID, OrderID string }, 0)
	switch eventType {
	case "ORDER_CREATED":
		if orderID != "" {
			out = append(out, struct{ DocType, ExtID, OrderID string }{EdiDocORDRSP, orderID + ":CREATED", orderID})
		}
	case "ORDER_STATUS_CHANGED":
		if orderID == "" || status == "" {
			return out
		}
		switch status {
		case "CANCELLED", "CANCEL_REQUESTED", "REJECTED", "BACKORDERED", "SCHEDULED", "PENDING",
			"CONFIRMED", "AUTO_ACCEPTED":
			out = append(out, struct{ DocType, ExtID, OrderID string }{EdiDocORDRSP, orderID + ":" + status, orderID})
		case "LOADED", "IN_TRANSIT":
			out = append(out, struct{ DocType, ExtID, OrderID string }{EdiDocDESADV, orderID + ":" + status, orderID})
		case "DELIVERED_ON_CREDIT":
			out = append(out, struct{ DocType, ExtID, OrderID string }{EdiDocINVOIC, orderID + ":INVOIC", orderID})
		}
	case "PAYMENT_CLEARED":
		if orderID != "" {
			out = append(out, struct{ DocType, ExtID, OrderID string }{EdiDocINVOIC, orderID + ":PAID", orderID})
		}
	}
	return out
}
