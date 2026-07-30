package order

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/soliq"
)

// Logger is a subset of the application logger.
type Logger interface {
	Info(msg string, args ...any)
	Error(msg string, args ...any)
}

// BuyerAcceptancePoller checks the Soliq system for buyer EHF acceptance status.
type BuyerAcceptancePoller struct {
	soliqClient soliq.SoliqClient
	repo        Repository
	logger      Logger
	pollDelay   time.Duration
}

func NewBuyerAcceptancePoller(sc soliq.SoliqClient, repo Repository, logger Logger) *BuyerAcceptancePoller {
	return &BuyerAcceptancePoller{
		soliqClient: sc,
		repo:        repo,
		logger:      logger,
		pollDelay:   1 * time.Minute,
	}
}

// Run executes the poller in an infinite loop.
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
		if o.LatestFiscalReceiptID == "" {
			p.logger.Error("order missing fiscal receipt id", "orderId", o.OrderID)
			continue
		}

		docStatus, err := p.soliqClient.CheckStatus(ctx, o.LatestFiscalReceiptID)
		if err != nil {
			p.logger.Error("failed to check soliq document status", "orderId", o.OrderID, "ehfId", o.LatestFiscalReceiptID, "err", err)
			continue
		}

		// Document is accepted by the buyer
		if docStatus.Status == "ACCEPTED" {
			o.BuyerAcceptanceStatus = "ACCEPTED"
			err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
				return nil
			})
			if err != nil {
				p.logger.Error("failed to update order to ACCEPTED", "orderId", o.OrderID, "err", err)
			}
			continue
		}

		// Document is rejected by the buyer
		if docStatus.Status == "REJECTED" {
			o.BuyerAcceptanceStatus = "REJECTED"
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
			err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
				return nil
			})
			if err != nil {
				p.logger.Error("failed to update order to REJECTED and create exception ticket", "orderId", o.OrderID, "err", err)
			}
			continue
		}

		// Check if it's expired
		if o.BuyerAcceptanceDeadline != nil && time.Now().UTC().After(*o.BuyerAcceptanceDeadline) {
			o.BuyerAcceptanceStatus = "EXPIRED"
			err := p.repo.UpdateOrder(ctx, *o, nil, func(tx outbox.TxnBuffer) error {
				return nil
			})
			if err != nil {
				p.logger.Error("failed to update order to EXPIRED", "orderId", o.OrderID, "err", err)
			}
			continue
		}
	}
}
