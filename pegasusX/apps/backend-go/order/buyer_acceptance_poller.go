package order

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/creditnote"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

// Logger is a subset of the application logger.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// BuyerAcceptancePoller checks the Soliq system for buyer EHF acceptance status.
// ADR-009 marks orders COMPLETED on OFD submit success; this poller is the
// parallel EHF buyer-clearance track that gates reverse settlement (credit note
// on REJECT). Orders only enter the queue when fiscal success stamps
// BuyerAcceptanceStatus=PENDING (MySoliq path).
type BuyerAcceptancePoller struct {
	soliqClient soliq.SoliqClient
	repo        Repository
	logger      Logger
	pollDelay   time.Duration

	creditnoteSvc               *creditnote.Service
	autoCreditNoteOnBuyerReject bool
}

func NewBuyerAcceptancePoller(sc soliq.SoliqClient, repo Repository, logger Logger, cnSvc *creditnote.Service) *BuyerAcceptancePoller {
	// Default ON: a rejected EHF must reverse settlement (credit note). Opt out
	// with CREDIT_NOTE_AUTO_FROM_BUYER_REJECT=false (P1-6).
	autoCN := true
	if v := strings.TrimSpace(os.Getenv("CREDIT_NOTE_AUTO_FROM_BUYER_REJECT")); strings.EqualFold(v, "false") || v == "0" {
		autoCN = false
	} else if strings.EqualFold(v, "true") || v == "1" {
		autoCN = true
	}
	return &BuyerAcceptancePoller{
		soliqClient:                 sc,
		repo:                        repo,
		logger:                      logger,
		pollDelay:                   1 * time.Minute,
		creditnoteSvc:               cnSvc,
		autoCreditNoteOnBuyerReject: autoCN,
	}
}

func (p *BuyerAcceptancePoller) SetAutoCreditNoteOnBuyerReject(enabled bool) {
	p.autoCreditNoteOnBuyerReject = enabled
}

// Run executes the poller in an infinite loop until ctx is cancelled.
func (p *BuyerAcceptancePoller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.pollDelay)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			p.logger.Info("buyer acceptance poller shutting down")
			return
		case <-ticker.C:
			p.poll(ctx)
		}
	}
}

func (p *BuyerAcceptancePoller) poll(ctx context.Context) {
	orders, err := p.repo.FindPendingBuyerAcceptance(ctx, 50)
	if err != nil {
		p.logger.Error("failed to find pending buyer acceptance", "err", err)
		return
	}

	for _, o := range orders {
		if o == nil {
			continue
		}
		if o.LatestFiscalReceiptID == "" {
			p.logger.Error("order missing fiscal receipt id", "orderId", o.OrderID)
			continue
		}

		docStatus, err := p.soliqClient.CheckStatus(ctx, o.LatestFiscalReceiptID)
		if err != nil {
			p.logger.Error("failed to check soliq document status", "orderId", o.OrderID, "ehfId", o.LatestFiscalReceiptID, "err", err)
			continue
		}

		switch docStatus.Status {
		case "ACCEPTED":
			o.BuyerAcceptanceStatus = BuyerAcceptanceAccepted
			o.UpdatedAt = time.Now().UTC()
			if err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
				return emitBuyerAcceptance(ctx, tx, *o, events.EventBuyerAcceptanceAccepted, BuyerAcceptanceAccepted)
			}); err != nil {
				p.logger.Error("failed to update order to ACCEPTED", "orderId", o.OrderID, "err", err)
			}
		case "REJECTED":
			o.BuyerAcceptanceStatus = BuyerAcceptanceRejected
			o.UpdatedAt = time.Now().UTC()
			o.PendingExceptionTicket = &ExceptionTicket{
				TicketID:     uuid.NewString(),
				Type:         "BUYER_EHF_REJECTION",
				OrderID:      o.OrderID,
				EhfID:        o.LatestFiscalReceiptID,
				Severity:     "HIGH",
				Status:       "OPEN",
				Title:        "Buyer Rejected Invoice",
				Description:  "The buyer rejected the invoice on the Soliq portal.",
				AssignedRole: "SUPPLIER_FINANCE",
				CreatedAt:    time.Now().UTC(),
				CreatedBy:    "SYSTEM_POLLER",
				Payload:      docStatus.Raw,
			}
			if err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
				return emitBuyerAcceptance(ctx, tx, *o, events.EventBuyerAcceptanceRejected, BuyerAcceptanceRejected)
			}); err != nil {
				p.logger.Error("failed to update order to REJECTED and create exception ticket", "orderId", o.OrderID, "err", err)
			} else if p.autoCreditNoteOnBuyerReject && p.creditnoteSvc != nil {
				// Reverse settlement: rejected EHF must not leave COMPLETED money
				// stranded without a credit note (P1-6).
				if _, cnErr := p.creditnoteSvc.CreateFromBuyerReject(ctx, o.OrderID, "system:buyer-accept-poller"); cnErr != nil {
					p.logger.Error("failed to create credit note on buyer reject", "orderId", o.OrderID, "err", cnErr)
				}
			}
		default:
			// Still pending at Soliq — check local deadline expiry.
			if o.BuyerAcceptanceDeadline != nil && time.Now().UTC().After(*o.BuyerAcceptanceDeadline) {
				o.BuyerAcceptanceStatus = BuyerAcceptanceExpired
				o.UpdatedAt = time.Now().UTC()
				if err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
					return emitBuyerAcceptance(ctx, tx, *o, events.EventBuyerAcceptanceExpired, BuyerAcceptanceExpired)
				}); err != nil {
					p.logger.Error("failed to update order to EXPIRED", "orderId", o.OrderID, "err", err)
				}
			}
		}
	}
}
