package p2p

import (
	"context"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// TxnHooks groups optional transactional side effects executed inside Repository writes.
type TxnHooks struct {
	GLPoster    GLPoster
	GLInput     *GLPostInput
	Audit       AuditRecorder
	AuditEvent  *AuditRecordInput
	OutboxEmit  func(ctx context.Context, buf outbox.TxnBuffer, journalEntryID string) error
	ProcessStep string
	ProcessDone bool
	ActorID     string
	Reason      string
}

// Repository defines durable storage operations for §6.6 P2P, process templates/instances, and process mining.
type Repository interface {
	ListTemplates(ctx context.Context, tenantID, domain string) ([]ProcessTemplate, error)
	SaveTemplate(ctx context.Context, tpl ProcessTemplate) (ProcessTemplate, error)

	ListInstances(ctx context.Context, f ListFilter) ([]ProcessInstance, error)
	UpsertInstance(ctx context.Context, in StartOrAdvanceInstanceInput, tpl ProcessTemplate, now time.Time) (ProcessInstance, error)

	CreatePurchaseOrder(ctx context.Context, po PurchaseOrder, tpl ProcessTemplate, hooks TxnHooks) (PurchaseOrder, bool, error)
	FindPurchaseOrderByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (PurchaseOrder, bool, error)
	ApprovePurchaseOrder(ctx context.Context, tenantID, poID, actorID, approvalID string, now time.Time, hooks TxnHooks) (PurchaseOrder, error)
	GetPurchaseOrder(ctx context.Context, tenantID, poID string) (PurchaseOrder, bool, error)
	ListPurchaseOrders(ctx context.Context, f ListFilter) ([]PurchaseOrder, error)

	PostGoodsReceipt(ctx context.Context, receipt GoodsReceipt, hooks TxnHooks) (GoodsReceipt, PurchaseOrder, bool, error)
	FindGoodsReceiptByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (GoodsReceipt, bool, error)
	ListGoodsReceipts(ctx context.Context, f ListFilter) ([]GoodsReceipt, error)

	CreateVendorInvoice(ctx context.Context, inv VendorInvoice, hooks TxnHooks) (VendorInvoice, bool, error)
	FindVendorInvoiceByIdempotency(ctx context.Context, tenantID, idempotencyKey string) (VendorInvoice, bool, error)
	GetVendorInvoice(ctx context.Context, tenantID, vendorInvoiceID string) (VendorInvoice, bool, error)
	ListVendorInvoices(ctx context.Context, f ListFilter) ([]VendorInvoice, error)

	GetInboundESF(ctx context.Context, tenantID, esfDocID string) (InboundESFDoc, bool, error)
	ApplyThreeWayMatch(ctx context.Context, tenantID, vendorInvoiceID string, update VendorInvoice, matchOK bool, now time.Time, hooks TxnHooks) (VendorInvoice, *ProcessInstance, error)

	CollectMiningSamples(ctx context.Context, f MiningFilter, templates []ProcessTemplate) ([]MiningSample, error)
}

func runNonSpannerGLAndAudit(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, hooks TxnHooks) (string, error) {
	var journalEntryID string
	if hooks.GLInput != nil {
		if hooks.GLPoster == nil {
			return "", ErrGLPosterUnavailable
		}
		if tPoster, ok := hooks.GLPoster.(TransactionalGLPoster); ok && txn != nil {
			res, err := tPoster.PostJournalInTxn(ctx, txn, buf, *hooks.GLInput)
			if err != nil {
				return "", err
			}
			journalEntryID = res.EntryID
		} else {
			res, err := hooks.GLPoster.PostJournal(ctx, *hooks.GLInput)
			if err != nil {
				return "", err
			}
			journalEntryID = res.EntryID
		}
	}
	if hooks.Audit != nil && hooks.AuditEvent != nil {
		if txn != nil {
			if err := hooks.Audit.RecordEventInTxn(ctx, txn, *hooks.AuditEvent); err != nil {
				return "", err
			}
		} else {
			if err := hooks.Audit.RecordEvent(ctx, *hooks.AuditEvent); err != nil {
				return "", err
			}
		}
	}
	return journalEntryID, nil
}
