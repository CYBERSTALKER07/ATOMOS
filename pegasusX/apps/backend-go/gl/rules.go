package gl

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// M0.2 canonical event types evaluated by PostingRules.
const (
	RuleEventOrderDelivered          = "order.delivered"
	RuleEventARPaymentReceived       = "ar.payment.received"
	RuleEventARAdvanceCredited       = "ar.advance.credited"
	RuleEventARAdjustWaive           = "ar.adjust.waive"
	RuleEventPayoutPaid              = "payout.paid"
	RuleEventCreditNoteIssued        = "creditnote.issued"
	RuleEventCashVariance            = "cash.variance"
	RuleEventFeePerOrder             = "fee.per_order"
	RuleEventSplitTender             = "split.tender"
	RuleEventClaimDispute            = "claim.dispute"
	RuleEventDoorstepShortfall       = "doorstep.shortfall"
	RuleEventMigrationOpeningBalance = "migration.opening_balance"
)

// PostingRule defines one data-driven accounting rule stored in PostingRules.
type PostingRule struct {
	CountryPack   string    `json:"country_pack"`
	RuleID        string    `json:"rule_id"`
	Version       int64     `json:"version"`
	EventType     string    `json:"event_type"`
	ConditionExpr string    `json:"condition_expr,omitempty"`
	DebitAccount  string    `json:"debit_account"`
	CreditAccount string    `json:"credit_account"`
	AmountExpr    string    `json:"amount_expr"`
	Description   string    `json:"description,omitempty"`
	Active        bool      `json:"active"`
	UpdatedAt     time.Time `json:"updated_at,omitempty"`
}

// TenderComponent is one payment rail component in a split-tender checkout.
type TenderComponent struct {
	Method      string `json:"method"`       // CASH | CARD | WALLET | CREDIT
	Gateway     string `json:"gateway"`      // PAYME | CLICK | ADYEN | MANUAL | ""
	AmountMinor int64  `json:"amount_minor"` // strictly > 0
	ReferenceID string `json:"reference_id,omitempty"`
}

// RuleEvalContext provides structured integer minor-unit inputs for posting rule evaluation.
type RuleEvalContext struct {
	EntityID        string            `json:"entity_id"`
	CountryPack     string            `json:"country_pack,omitempty"`
	Currency        string            `json:"currency,omitempty"`
	SourceEventID   string            `json:"source_event_id,omitempty"`
	SourceID        string            `json:"source_id"`
	OrderID         string            `json:"order_id,omitempty"`
	RetailerID      string            `json:"retailer_id,omitempty"`
	SupplierID      string            `json:"supplier_id,omitempty"`
	DriverID        string            `json:"driver_id,omitempty"`
	WarehouseID     string            `json:"warehouse_id,omitempty"`
	ProductID       string            `json:"product_id,omitempty"`
	LotID           string            `json:"lot_id,omitempty"`
	PaymentMode     string            `json:"payment_mode,omitempty"`
	Gateway         string            `json:"gateway,omitempty"`
	SettlementMode  string            `json:"settlement_mode,omitempty"`
	Action          string            `json:"action,omitempty"`
	IdempotencyKey  string            `json:"idempotency_key"`
	PostedBy        string            `json:"posted_by,omitempty"`
	Memo            string            `json:"memo,omitempty"`
	OccurredAt      time.Time         `json:"occurred_at,omitempty"`
	GrossMinor      int64             `json:"gross_minor,omitempty"`
	NetMinor        int64             `json:"net_minor,omitempty"`
	VatMinor        int64             `json:"vat_minor,omitempty"`
	CostMinor       int64             `json:"cost_minor,omitempty"`
	FeeMinor        int64             `json:"fee_minor,omitempty"`
	DiscountMinor   int64             `json:"discount_minor,omitempty"`
	ExpectedMinor   int64             `json:"expected_minor,omitempty"`
	ReceivedMinor   int64             `json:"received_minor,omitempty"`
	DifferenceMinor int64             `json:"difference_minor,omitempty"`
	Tenders         []TenderComponent `json:"tenders,omitempty"`
}

var baseNSBURules = []PostingRule{
	// 1. order.delivered (cash, card/PSP, wallet, credit/nasiya, plus optional VAT & COGS)
	{
		RuleID: "order_delivered_cash_net", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode == 'CASH' && net_minor > 0",
		DebitAccount:  AccountCashInTransit, CreditAccount: AccountSalesRevenue, AmountExpr: "net_minor",
		Description: "Order delivered (cash) net sales revenue", Active: true,
	},
	{
		RuleID: "order_delivered_cash_vat", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode == 'CASH' && vat_minor > 0",
		DebitAccount:  AccountCashInTransit, CreditAccount: AccountVatOutput, AmountExpr: "vat_minor",
		Description: "Order delivered (cash) output VAT", Active: true,
	},
	{
		RuleID: "order_delivered_card_net", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode in ('CARD','PAYME','CLICK','ADYEN','GLOBAL_PAY') && net_minor > 0",
		DebitAccount:  AccountCardPSPTransit, CreditAccount: AccountSalesRevenue, AmountExpr: "net_minor",
		Description: "Order delivered (card/PSP) net sales revenue", Active: true,
	},
	{
		RuleID: "order_delivered_card_vat", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode in ('CARD','PAYME','CLICK','ADYEN','GLOBAL_PAY') && vat_minor > 0",
		DebitAccount:  AccountCardPSPTransit, CreditAccount: AccountVatOutput, AmountExpr: "vat_minor",
		Description: "Order delivered (card/PSP) output VAT", Active: true,
	},
	{
		RuleID: "order_delivered_wallet_net", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode == 'WALLET' && net_minor > 0",
		DebitAccount:  AccountRetailerCredit, CreditAccount: AccountSalesRevenue, AmountExpr: "net_minor",
		Description: "Order delivered (retailer wallet) net sales revenue", Active: true,
	},
	{
		RuleID: "order_delivered_wallet_vat", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode == 'WALLET' && vat_minor > 0",
		DebitAccount:  AccountRetailerCredit, CreditAccount: AccountVatOutput, AmountExpr: "vat_minor",
		Description: "Order delivered (retailer wallet) output VAT", Active: true,
	},
	{
		RuleID: "order_delivered_nasiya_net", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode in ('CREDIT','NASIYA') && net_minor > 0",
		DebitAccount:  AccountAccountsReceivable, CreditAccount: AccountSalesRevenue, AmountExpr: "net_minor",
		Description: "Order delivered (nasiya/credit) AR and net revenue", Active: true,
	},
	{
		RuleID: "order_delivered_nasiya_vat", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "payment_mode in ('CREDIT','NASIYA') && vat_minor > 0",
		DebitAccount:  AccountAccountsReceivable, CreditAccount: AccountVatOutput, AmountExpr: "vat_minor",
		Description: "Order delivered (nasiya/credit) output VAT", Active: true,
	},
	{
		RuleID: "order_delivered_cogs", Version: 1, EventType: RuleEventOrderDelivered,
		ConditionExpr: "cost_minor > 0",
		DebitAccount:  AccountCOGS, CreditAccount: AccountInventory, AmountExpr: "cost_minor",
		Description: "Order delivered cost of goods sold", Active: true,
	},

	// 2. ar.payment.received / advance / waive
	{
		RuleID: "ar_payment_bank", Version: 1, EventType: RuleEventARPaymentReceived,
		ConditionExpr: "payment_mode in ('','BANK','TRANSFER') && gross_minor > 0",
		DebitAccount:  AccountBankOperating, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "AR payment received via bank transfer", Active: true,
	},
	{
		RuleID: "ar_payment_cash_office", Version: 1, EventType: RuleEventARPaymentReceived,
		ConditionExpr: "payment_mode == 'CASH_OFFICE' && gross_minor > 0",
		DebitAccount:  AccountCashOffice, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "AR payment received at finance cash vault", Active: true,
	},
	{
		RuleID: "ar_payment_cash_driver", Version: 1, EventType: RuleEventARPaymentReceived,
		ConditionExpr: "payment_mode == 'CASH' && gross_minor > 0",
		DebitAccount:  AccountCashInTransit, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "AR payment collected by driver", Active: true,
	},
	{
		RuleID: "ar_payment_card", Version: 1, EventType: RuleEventARPaymentReceived,
		ConditionExpr: "payment_mode in ('CARD','PAYME','CLICK','ADYEN') && gross_minor > 0",
		DebitAccount:  AccountCardPSPTransit, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "AR payment collected via PSP gateway", Active: true,
	},
	{
		RuleID: "ar_advance_credited", Version: 1, EventType: RuleEventARAdvanceCredited,
		ConditionExpr: "gross_minor > 0",
		DebitAccount:  AccountBankOperating, CreditAccount: AccountRetailerCredit, AmountExpr: "gross_minor",
		Description: "Unallocated retailer overpayment credited to advance wallet", Active: true,
	},
	{
		RuleID: "ar_adjust_waive", Version: 1, EventType: RuleEventARAdjustWaive,
		ConditionExpr: "gross_minor > 0",
		DebitAccount:  AccountCreditNoteAdjustment, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "AR invoice balance waive or adjustment", Active: true,
	},

	// 3. payout.paid (escrow release + platform fee, or bank disbursement)
	{
		RuleID: "payout_escrow_to_supplier", Version: 1, EventType: RuleEventPayoutPaid,
		ConditionExpr: "action != 'BANK_DISBURSE' && net_supplier_minor > 0",
		DebitAccount:  AccountPlatformEscrow, CreditAccount: AccountSupplierPayable, AmountExpr: "net_supplier_minor",
		Description: "Escrow release net payable to supplier", Active: true,
	},
	{
		RuleID: "payout_escrow_platform_fee", Version: 1, EventType: RuleEventPayoutPaid,
		ConditionExpr: "action != 'BANK_DISBURSE' && fee_minor > 0",
		DebitAccount:  AccountPlatformEscrow, CreditAccount: AccountPlatformFeeRevenue, AmountExpr: "fee_minor",
		Description: "Platform commission withheld on payout", Active: true,
	},
	{
		RuleID: "payout_bank_disburse", Version: 1, EventType: RuleEventPayoutPaid,
		ConditionExpr: "action == 'BANK_DISBURSE' && gross_minor > 0",
		DebitAccount:  AccountSupplierPayable, CreditAccount: AccountBankOperating, AmountExpr: "gross_minor",
		Description: "Supplier payout bank disbursement", Active: true,
	},

	// 4. creditnote.issued (net adjustment + output VAT reversal vs AR)
	{
		RuleID: "creditnote_issued_net", Version: 1, EventType: RuleEventCreditNoteIssued,
		ConditionExpr: "net_minor > 0",
		DebitAccount:  AccountCreditNoteAdjustment, CreditAccount: AccountAccountsReceivable, AmountExpr: "net_minor",
		Description: "Credit note net sales adjustment", Active: true,
	},
	{
		RuleID: "creditnote_issued_vat", Version: 1, EventType: RuleEventCreditNoteIssued,
		ConditionExpr: "vat_minor > 0",
		DebitAccount:  AccountVatOutput, CreditAccount: AccountAccountsReceivable, AmountExpr: "vat_minor",
		Description: "Credit note output VAT adjustment", Active: true,
	},

	// 5. cash.variance (open shortfall/overage + accept/write-off resolutions)
	{
		RuleID: "cash_variance_shortfall_open", Version: 1, EventType: RuleEventCashVariance,
		ConditionExpr: "action in ('','OPEN') && difference_minor < 0",
		DebitAccount:  AccountDriverReceivableVariance, CreditAccount: AccountCashInTransit, AmountExpr: "abs_difference_minor",
		Description: "Driver cash shift shortfall receivable", Active: true,
	},
	{
		RuleID: "cash_variance_overage_open", Version: 1, EventType: RuleEventCashVariance,
		ConditionExpr: "action in ('','OPEN') && difference_minor > 0",
		DebitAccount:  AccountCashInTransit, CreditAccount: AccountDriverPayableVariance, AmountExpr: "abs_difference_minor",
		Description: "Driver cash shift overage suspense", Active: true,
	},
	{
		RuleID: "cash_variance_shortfall_accept", Version: 1, EventType: RuleEventCashVariance,
		ConditionExpr: "action == 'ACCEPT' && difference_minor < 0",
		DebitAccount:  AccountCashOffice, CreditAccount: AccountDriverReceivableVariance, AmountExpr: "abs_difference_minor",
		Description: "Driver repaid cash shortfall into office vault", Active: true,
	},
	{
		RuleID: "cash_variance_shortfall_writeoff", Version: 1, EventType: RuleEventCashVariance,
		ConditionExpr: "action == 'WRITE_OFF' && difference_minor < 0",
		DebitAccount:  AccountCashWriteOffExpense, CreditAccount: AccountDriverReceivableVariance, AmountExpr: "abs_difference_minor",
		Description: "Uncollectible driver cash shortfall written off", Active: true,
	},
	{
		RuleID: "cash_variance_overage_resolve", Version: 1, EventType: RuleEventCashVariance,
		ConditionExpr: "action in ('ACCEPT','WRITE_OFF') && difference_minor > 0",
		DebitAccount:  AccountDriverPayableVariance, CreditAccount: AccountCashOverShortRevenue, AmountExpr: "abs_difference_minor",
		Description: "Driver cash overage recognized as revenue", Active: true,
	},

	// 6. fee.per_order
	{
		RuleID: "fee_per_order_accrual", Version: 1, EventType: RuleEventFeePerOrder,
		ConditionExpr: "fee_minor > 0",
		DebitAccount:  AccountSupplierFeesReceivable, CreditAccount: AccountPlatformFeeRevenue, AmountExpr: "fee_minor",
		Description: "Per-order platform billing fee accrual", Active: true,
	},

	// 7. claim.dispute
	{
		RuleID: "claim_dispute_store_credit", Version: 1, EventType: RuleEventClaimDispute,
		ConditionExpr: "settlement_mode == 'STORE_CREDIT' && gross_minor > 0",
		DebitAccount:  AccountClaimsSettlementExpense, CreditAccount: AccountRetailerCredit, AmountExpr: "gross_minor",
		Description: "Claim dispute settled via retailer store credit", Active: true,
	},
	{
		RuleID: "claim_dispute_gateway_refund", Version: 1, EventType: RuleEventClaimDispute,
		ConditionExpr: "settlement_mode == 'GATEWAY_REFUND' && gross_minor > 0",
		DebitAccount:  AccountSupplierPayable, CreditAccount: AccountCardPSPTransit, AmountExpr: "gross_minor",
		Description: "Claim dispute settled via PSP gateway refund", Active: true,
	},
	{
		RuleID: "claim_dispute_ledger_only", Version: 1, EventType: RuleEventClaimDispute,
		ConditionExpr: "settlement_mode not_in ('STORE_CREDIT','GATEWAY_REFUND') && gross_minor > 0",
		DebitAccount:  AccountSupplierPayable, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "Claim dispute supplier chargeback vs retailer AR", Active: true,
	},

	// 8. doorstep.shortfall
	{
		RuleID: "doorstep_shortfall_received", Version: 1, EventType: RuleEventDoorstepShortfall,
		ConditionExpr: "received_minor > 0",
		DebitAccount:  AccountCashInTransit, CreditAccount: AccountPlatformEscrow, AmountExpr: "received_minor",
		Description: "Doorstep cash collected by driver", Active: true,
	},
	{
		RuleID: "doorstep_shortfall_variance", Version: 1, EventType: RuleEventDoorstepShortfall,
		ConditionExpr: "shortfall_minor > 0",
		DebitAccount:  AccountDriverReceivableVariance, CreditAccount: AccountPlatformEscrow, AmountExpr: "shortfall_minor",
		Description: "Doorstep cash shortfall assigned to driver receivable", Active: true,
	},

	// 9. migration.opening_balance
	{
		RuleID: "migration_opening_balance_ar", Version: 1, EventType: RuleEventMigrationOpeningBalance,
		ConditionExpr: "action != 'ROLLBACK' && gross_minor > 0",
		DebitAccount:  AccountAccountsReceivable, CreditAccount: AccountOpeningBalanceEquity, AmountExpr: "gross_minor",
		Description: "Migrated opening nasiya AR balance", Active: true,
	},
	{
		RuleID: "migration_opening_balance_rollback", Version: 1, EventType: RuleEventMigrationOpeningBalance,
		ConditionExpr: "action == 'ROLLBACK' && gross_minor > 0",
		DebitAccount:  AccountOpeningBalanceEquity, CreditAccount: AccountAccountsReceivable, AmountExpr: "gross_minor",
		Description: "Rollback of migrated opening nasiya AR balance", Active: true,
	},
}

// DefaultPostingRules returns shipped PostingRules for countryPack ("UZ" NSBU default, or "KZ" extensible pack).
func DefaultPostingRules(countryPack string) []PostingRule {
	pack := auth.NormalizeCountryCode(countryPack)
	if pack == "" {
		pack = "UZ"
	}
	out := make([]PostingRule, 0, len(baseNSBURules))
	for _, r := range baseNSBURules {
		cp := r
		cp.CountryPack = pack
		cp.RuleID = strings.ToLower(pack) + "_" + r.RuleID
		if pack == "KZ" {
			cp.Description = "[KZ] " + r.Description
		}
		out = append(out, cp)
	}
	return out
}

// SeedPostingRules idempotently seeds default PostingRules for countryPack into Spanner.
func SeedPostingRules(ctx context.Context, client *spanner.Client, countryPack string) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("spanner client unavailable")
	}
	pack := auth.NormalizeCountryCode(countryPack)
	if pack == "" {
		pack = "UZ"
	}
	var inserted int
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		n, err := SeedPostingRulesInTxn(ctx, txn, pack)
		if err != nil {
			return err
		}
		inserted = n
		return nil
	})
	return inserted, err
}

// SeedPostingRulesInTxn inserts missing default PostingRules for countryPack inside txn.
func SeedPostingRulesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, countryPack string) (int, error) {
	if txn == nil {
		return 0, ErrNilTransaction
	}
	pack := auth.NormalizeCountryCode(countryPack)
	if pack == "" {
		pack = "UZ"
	}
	existing := make(map[string]struct{})
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT RuleId, Version FROM PostingRules WHERE CountryPack = @pack`,
		Params: map[string]any{"pack": pack},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read PostingRules: %w", err)
		}
		var rid string
		var ver int64
		if err := row.Columns(&rid, &ver); err != nil {
			return 0, fmt.Errorf("scan PostingRules: %w", err)
		}
		existing[fmt.Sprintf("%s:%d", rid, ver)] = struct{}{}
	}

	rules := DefaultPostingRules(pack)
	muts := make([]*spanner.Mutation, 0, len(rules))
	for _, r := range rules {
		key := fmt.Sprintf("%s:%d", r.RuleID, r.Version)
		if _, ok := existing[key]; ok {
			continue
		}
		muts = append(muts, spanner.InsertMap("PostingRules", map[string]any{
			"CountryPack":   r.CountryPack,
			"RuleId":        r.RuleID,
			"Version":       r.Version,
			"EventType":     r.EventType,
			"ConditionExpr": nullIfEmpty(r.ConditionExpr),
			"DebitAccount":  r.DebitAccount,
			"CreditAccount": r.CreditAccount,
			"AmountExpr":    r.AmountExpr,
			"Description":   nullIfEmpty(r.Description),
			"Active":        r.Active,
			"UpdatedAt":     spanner.CommitTimestamp,
		}))
	}
	if len(muts) == 0 {
		return 0, nil
	}
	if err := txn.BufferWrite(muts); err != nil {
		return 0, fmt.Errorf("buffer PostingRules seed: %w", err)
	}
	return len(muts), nil
}

// readActivePostingRulesInTxn loads the latest active version of each rule for (countryPack, eventType).
// Falls back to "UZ" DB rules, then to shipped DefaultPostingRules(countryPack) when none exist.
func readActivePostingRulesInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, countryPack, eventType string) ([]PostingRule, error) {
	pack := auth.NormalizeCountryCode(countryPack)
	if pack == "" {
		pack = "UZ"
	}
	rules, err := queryActivePostingRulesByPack(ctx, txn, pack, eventType)
	if err != nil {
		return nil, err
	}
	if len(rules) > 0 {
		return rules, nil
	}
	if pack != "UZ" {
		uzRules, err := queryActivePostingRulesByPack(ctx, txn, "UZ", eventType)
		if err != nil {
			return nil, err
		}
		if len(uzRules) > 0 {
			return uzRules, nil
		}
	}
	var fallback []PostingRule
	for _, r := range DefaultPostingRules(pack) {
		if r.EventType == eventType && r.Active {
			fallback = append(fallback, r)
		}
	}
	return fallback, nil
}

func queryActivePostingRulesByPack(ctx context.Context, txn *spanner.ReadWriteTransaction, pack, eventType string) ([]PostingRule, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT CountryPack, RuleId, Version, EventType, ConditionExpr, DebitAccount, CreditAccount, AmountExpr, Description, Active
		      FROM PostingRules@{FORCE_INDEX=Idx_PostingRules_ByEvent}
		      WHERE EventType = @eventType AND CountryPack = @pack AND Active = TRUE
		      ORDER BY Version DESC, RuleId ASC`,
		Params: map[string]any{
			"eventType": eventType,
			"pack":      pack,
		},
	})
	defer iter.Stop()

	seenRule := make(map[string]struct{})
	var out []PostingRule
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PostingRules (%s/%s): %w", pack, eventType, err)
		}
		var r PostingRule
		var cond, desc spanner.NullString
		if err := row.Columns(&r.CountryPack, &r.RuleID, &r.Version, &r.EventType, &cond, &r.DebitAccount, &r.CreditAccount, &r.AmountExpr, &desc, &r.Active); err != nil {
			return nil, fmt.Errorf("scan PostingRule: %w", err)
		}
		r.ConditionExpr = nullStrVal(cond)
		r.Description = nullStrVal(desc)
		if _, dup := seenRule[r.RuleID]; dup {
			continue
		}
		seenRule[r.RuleID] = struct{}{}
		out = append(out, r)
	}
	return out, nil
}

// PostEventInTxn resolves active PostingRules for eventType and ev.CountryPack inside txn,
// evaluates ev into a balanced JournalEntryInput, and buffers it via BufferJournalEntryInTxn.
func PostEventInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	outboxBuf outbox.TxnBuffer,
	eventType string,
	ev RuleEvalContext,
) (JournalEntryResult, error) {
	if txn == nil {
		return JournalEntryResult{}, ErrNilTransaction
	}
	if ctx == nil {
		ctx = context.Background()
	}
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if err := normalizeRuleEvalContext(ctx, eventType, &ev); err != nil {
		return JournalEntryResult{}, err
	}

	var jeIn JournalEntryInput
	var err error
	if eventType == RuleEventSplitTender {
		jeIn, err = BuildSplitTenderInput(ev)
	} else {
		rules, rErr := readActivePostingRulesInTxn(ctx, txn, ev.CountryPack, eventType)
		if rErr != nil {
			return JournalEntryResult{}, rErr
		}
		jeIn, err = EvaluateEventToJournalInput(ctx, rules, eventType, ev)
	}
	if err != nil {
		return JournalEntryResult{}, err
	}
	return BufferJournalEntryInTxn(ctx, txn, outboxBuf, jeIn)
}

// EvaluateEventToJournalInput evaluates rules against ev to construct a validated JournalEntryInput.
func EvaluateEventToJournalInput(ctx context.Context, rules []PostingRule, eventType string, ev RuleEvalContext) (JournalEntryInput, error) {
	eventType = strings.ToLower(strings.TrimSpace(eventType))
	if err := normalizeRuleEvalContext(ctx, eventType, &ev); err != nil {
		return JournalEntryInput{}, err
	}
	if eventType == RuleEventSplitTender {
		return BuildSplitTenderInput(ev)
	}

	type coalKey struct {
		account   string
		direction Direction
		partyType string
		partyID   string
		wh        string
		prod      string
		lot       string
	}
	var order []coalKey
	coalesced := make(map[coalKey]*PostingInput)

	appendPosting := func(acct string, dir Direction, amt int64, desc string) {
		acct = strings.ToUpper(strings.TrimSpace(acct))
		pt, pid := inferPartyForAccount(acct, ev)
		var wh, prod, lot string
		if acct == AccountCOGS || acct == AccountInventory {
			wh, prod, lot = ev.WarehouseID, ev.ProductID, ev.LotID
		}
		k := coalKey{account: acct, direction: dir, partyType: pt, partyID: pid, wh: wh, prod: prod, lot: lot}
		if existing, ok := coalesced[k]; ok {
			existing.AmountMinor += amt
			existing.FunctionalAmountMinor += amt
			if desc != "" && !strings.Contains(existing.Description, desc) {
				existing.Description = existing.Description + "; " + desc
			}
			return
		}
		coalesced[k] = &PostingInput{
			AccountCode:           acct,
			Direction:             dir,
			AmountMinor:           amt,
			Currency:              ev.Currency,
			FunctionalAmountMinor: amt,
			PartyType:             pt,
			PartyID:               pid,
			WarehouseID:           wh,
			ProductID:             prod,
			LotID:                 lot,
			Description:           desc,
		}
		order = append(order, k)
	}

	matchedAny := false
	for _, r := range rules {
		if !r.Active || !strings.EqualFold(strings.TrimSpace(r.EventType), eventType) {
			continue
		}
		ok, err := evalRuleCondition(r.ConditionExpr, ev)
		if err != nil {
			return JournalEntryInput{}, fmt.Errorf("rule %s condition: %w", r.RuleID, err)
		}
		if !ok {
			continue
		}
		amt, err := evalRuleAmount(r.AmountExpr, ev)
		if err != nil {
			return JournalEntryInput{}, fmt.Errorf("rule %s amount: %w", r.RuleID, err)
		}
		if amt <= 0 {
			continue
		}
		matchedAny = true
		appendPosting(r.DebitAccount, DirectionDebit, amt, r.Description)
		appendPosting(r.CreditAccount, DirectionCredit, amt, r.Description)
	}

	if !matchedAny {
		return JournalEntryInput{}, fmt.Errorf("%w: no posting rule matched event %q (payment_mode=%q action=%q)", ErrInvalidJournalInput, eventType, ev.PaymentMode, ev.Action)
	}

	postings := make([]PostingInput, 0, len(order))
	for _, k := range order {
		postings = append(postings, *coalesced[k])
	}

	// Sort debits before credits for deterministic readable presentation.
	sort.SliceStable(postings, func(i, j int) bool {
		if postings[i].Direction != postings[j].Direction {
			return postings[i].Direction == DirectionDebit
		}
		return false
	})

	in := JournalEntryInput{
		EntityID:       ev.EntityID,
		SourceEventID:  ev.SourceEventID,
		SourceType:     eventToSourceType(eventType),
		SourceID:       ev.SourceID,
		OrderID:        ev.OrderID,
		Memo:           coalesceNonEmpty(ev.Memo, fmt.Sprintf("GL posting for %s (%s)", eventType, ev.SourceID)),
		Currency:       ev.Currency,
		CountryPack:    ev.CountryPack,
		IdempotencyKey: ev.IdempotencyKey,
		PostedBy:       ev.PostedBy,
		OccurredAt:     ev.OccurredAt,
		Postings:       postings,
	}
	if _, _, err := ValidateJournalInput(ctx, &in); err != nil {
		return JournalEntryInput{}, err
	}
	return in, nil
}

// Typed event builders (for callers building inputs without reading DB rules first,
// or for testing deterministic rule output offline).

func BuildOrderDeliveredInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventOrderDelivered, ev)
}

func BuildARPaymentInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventARPaymentReceived, ev)
}

func BuildPayoutPaidInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventPayoutPaid, ev)
}

func BuildCreditNoteInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventCreditNoteIssued, ev)
}

func BuildCashVarianceInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventCashVariance, ev)
}

func BuildPerOrderFeeInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventFeePerOrder, ev)
}

func BuildClaimDisputeInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventClaimDispute, ev)
}

func BuildDoorstepShortfallInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventDoorstepShortfall, ev)
}

func BuildMigrationOpeningBalanceInput(ev RuleEvalContext) (JournalEntryInput, error) {
	return buildFromDefaultRules(RuleEventMigrationOpeningBalance, ev)
}

// BuildSplitTenderInput builds a validated balanced multi-leg JournalEntryInput for split-tender checkouts.
func BuildSplitTenderInput(ev RuleEvalContext) (JournalEntryInput, error) {
	ctx := context.Background()
	if err := normalizeRuleEvalContext(ctx, RuleEventSplitTender, &ev); err != nil {
		return JournalEntryInput{}, err
	}
	if ev.GrossMinor <= 0 {
		return JournalEntryInput{}, fmt.Errorf("%w: split.tender requires gross_minor > 0", ErrInvalidPostingAmount)
	}
	if ev.DiscountMinor < 0 {
		return JournalEntryInput{}, fmt.Errorf("%w: split.tender discount_minor cannot be negative", ErrInvalidPostingAmount)
	}

	postings := make([]PostingInput, 0, len(ev.Tenders)+2)
	for _, t := range ev.Tenders {
		if t.AmountMinor < 0 {
			return JournalEntryInput{}, fmt.Errorf("%w: tender amount cannot be negative", ErrInvalidPostingAmount)
		}
		if t.AmountMinor == 0 {
			continue
		}
		method := strings.ToUpper(strings.TrimSpace(t.Method))
		gw := strings.ToUpper(strings.TrimSpace(t.Gateway))
		var acct, pt, pid string
		switch method {
		case "CASH":
			acct, pt, pid = AccountCashInTransit, "DRIVER", ev.DriverID
		case "WALLET":
			acct, pt, pid = AccountRetailerCredit, "RETAILER", ev.RetailerID
		case "CREDIT", "NASIYA":
			acct, pt, pid = AccountAccountsReceivable, "RETAILER", ev.RetailerID
		default:
			acct, pt, pid = AccountCardPSPTransit, "GATEWAY", coalesceNonEmpty(gw, method, "PSP")
		}
		postings = append(postings, PostingInput{
			AccountCode:           acct,
			Direction:             DirectionDebit,
			AmountMinor:           t.AmountMinor,
			Currency:              ev.Currency,
			FunctionalAmountMinor: t.AmountMinor,
			PartyType:             pt,
			PartyID:               pid,
			Description:           fmt.Sprintf("Split tender %s %s ref:%s", method, gw, t.ReferenceID),
		})
	}

	if ev.DiscountMinor > 0 {
		postings = append(postings, PostingInput{
			AccountCode:           AccountDiscountsAndAllowances,
			Direction:             DirectionDebit,
			AmountMinor:           ev.DiscountMinor,
			Currency:              ev.Currency,
			FunctionalAmountMinor: ev.DiscountMinor,
			PartyType:             "SUPPLIER",
			PartyID:               coalesceNonEmpty(ev.SupplierID, ev.EntityID),
			Description:           "Promotional discount subsidy",
		})
	}

	postings = append(postings, PostingInput{
		AccountCode:           AccountPlatformEscrow,
		Direction:             DirectionCredit,
		AmountMinor:           ev.GrossMinor,
		Currency:              ev.Currency,
		FunctionalAmountMinor: ev.GrossMinor,
		PartyType:             "SUPPLIER",
		PartyID:               coalesceNonEmpty(ev.SupplierID, ev.EntityID),
		Description:           fmt.Sprintf("Order fulfillment escrow obligation for %s", coalesceNonEmpty(ev.OrderID, ev.SourceID)),
	})

	in := JournalEntryInput{
		EntityID:       ev.EntityID,
		SourceEventID:  ev.SourceEventID,
		SourceType:     eventToSourceType(RuleEventSplitTender),
		SourceID:       ev.SourceID,
		OrderID:        ev.OrderID,
		Memo:           coalesceNonEmpty(ev.Memo, fmt.Sprintf("Split tender order %s", coalesceNonEmpty(ev.OrderID, ev.SourceID))),
		Currency:       ev.Currency,
		CountryPack:    ev.CountryPack,
		IdempotencyKey: ev.IdempotencyKey,
		PostedBy:       ev.PostedBy,
		OccurredAt:     ev.OccurredAt,
		Postings:       postings,
	}
	if _, _, err := ValidateJournalInput(ctx, &in); err != nil {
		return JournalEntryInput{}, err
	}
	return in, nil
}

func buildFromDefaultRules(eventType string, ev RuleEvalContext) (JournalEntryInput, error) {
	ctx := context.Background()
	pack := auth.NormalizeCountryCode(ev.CountryPack)
	if pack == "" {
		pack = "UZ"
	}
	return EvaluateEventToJournalInput(ctx, DefaultPostingRules(pack), eventType, ev)
}

func normalizeRuleEvalContext(ctx context.Context, eventType string, ev *RuleEvalContext) error {
	ev.EntityID = strings.TrimSpace(ev.EntityID)
	ev.SupplierID = strings.TrimSpace(ev.SupplierID)
	if ev.EntityID == "" {
		ev.EntityID = ev.SupplierID
	}
	if ev.SupplierID == "" {
		ev.SupplierID = ev.EntityID
	}
	if ev.EntityID == "" {
		return fmt.Errorf("%w: entity_id (or supplier_id) is required", ErrInvalidJournalInput)
	}
	ev.CountryPack = resolveCountryPack(ctx, ev.EntityID, ev.CountryPack)
	ev.Currency = strings.ToUpper(strings.TrimSpace(ev.Currency))
	if ev.Currency == "" {
		if c, err := auth.CurrencyFromContext(ctx, ev.EntityID); err == nil && c != "" {
			ev.Currency = c
		} else {
			ev.Currency = DefaultCurrencyForPack(ev.CountryPack)
		}
	}
	ev.SourceID = strings.TrimSpace(ev.SourceID)
	if ev.SourceID == "" {
		ev.SourceID = strings.TrimSpace(ev.OrderID)
	}
	if ev.SourceID == "" {
		return fmt.Errorf("%w: source_id is required", ErrInvalidJournalInput)
	}
	ev.PaymentMode = strings.ToUpper(strings.TrimSpace(ev.PaymentMode))
	ev.Gateway = strings.ToUpper(strings.TrimSpace(ev.Gateway))
	ev.SettlementMode = strings.ToUpper(strings.TrimSpace(ev.SettlementMode))
	ev.Action = strings.ToUpper(strings.TrimSpace(ev.Action))

	switch eventType {
	case RuleEventOrderDelivered:
		if ev.PaymentMode == "" {
			return fmt.Errorf("%w: order.delivered requires payment_mode", ErrInvalidJournalInput)
		}
		if ev.NetMinor == 0 && ev.VatMinor == 0 && ev.GrossMinor > 0 {
			ev.NetMinor = ev.GrossMinor
		}
		if ev.GrossMinor == 0 && (ev.NetMinor > 0 || ev.VatMinor > 0) {
			ev.GrossMinor = ev.NetMinor + ev.VatMinor
		}
		if ev.GrossMinor <= 0 || ev.NetMinor < 0 || ev.VatMinor < 0 || ev.NetMinor+ev.VatMinor != ev.GrossMinor {
			return fmt.Errorf("%w: order.delivered net(%d)+vat(%d) must equal gross(%d) > 0", ErrUnbalancedEntry, ev.NetMinor, ev.VatMinor, ev.GrossMinor)
		}
	case RuleEventCreditNoteIssued:
		if ev.NetMinor == 0 && ev.VatMinor == 0 && ev.GrossMinor > 0 {
			ev.NetMinor = ev.GrossMinor
		}
		if ev.GrossMinor == 0 && (ev.NetMinor > 0 || ev.VatMinor > 0) {
			ev.GrossMinor = ev.NetMinor + ev.VatMinor
		}
		if ev.GrossMinor <= 0 || ev.NetMinor < 0 || ev.VatMinor < 0 || ev.NetMinor+ev.VatMinor != ev.GrossMinor {
			return fmt.Errorf("%w: creditnote.issued net(%d)+vat(%d) must equal gross(%d) > 0", ErrUnbalancedEntry, ev.NetMinor, ev.VatMinor, ev.GrossMinor)
		}
	case RuleEventPayoutPaid:
		if ev.GrossMinor <= 0 {
			return fmt.Errorf("%w: payout.paid gross_minor must be > 0", ErrInvalidPostingAmount)
		}
		if ev.FeeMinor < 0 || ev.FeeMinor > ev.GrossMinor {
			return fmt.Errorf("%w: payout.paid fee_minor(%d) invalid for gross_minor(%d)", ErrInvalidPostingAmount, ev.FeeMinor, ev.GrossMinor)
		}
	case RuleEventCashVariance:
		if ev.DifferenceMinor == 0 && (ev.ExpectedMinor != 0 || ev.ReceivedMinor != 0) {
			ev.DifferenceMinor = ev.ReceivedMinor - ev.ExpectedMinor
		}
		if ev.DifferenceMinor == 0 {
			return fmt.Errorf("%w: cash.variance difference_minor must be non-zero", ErrInvalidPostingAmount)
		}
	case RuleEventDoorstepShortfall:
		if ev.ExpectedMinor <= 0 || ev.ReceivedMinor < 0 || ev.ExpectedMinor-ev.ReceivedMinor <= 0 {
			return fmt.Errorf("%w: doorstep.shortfall expected(%d) > received(%d) >= 0 required", ErrInvalidPostingAmount, ev.ExpectedMinor, ev.ReceivedMinor)
		}
	case RuleEventFeePerOrder:
		if ev.FeeMinor == 0 && ev.GrossMinor > 0 {
			ev.FeeMinor = ev.GrossMinor
		}
		if ev.FeeMinor <= 0 {
			return fmt.Errorf("%w: fee.per_order fee_minor must be > 0", ErrInvalidPostingAmount)
		}
	case RuleEventARPaymentReceived, RuleEventARAdvanceCredited, RuleEventARAdjustWaive, RuleEventClaimDispute, RuleEventMigrationOpeningBalance:
		if ev.GrossMinor <= 0 {
			return fmt.Errorf("%w: %s requires gross_minor > 0", ErrInvalidPostingAmount, eventType)
		}
	}
	return nil
}

func inferPartyForAccount(accountCode string, ev RuleEvalContext) (partyType, partyID string) {
	switch accountCode {
	case AccountAccountsReceivable, AccountRetailerCredit:
		return "RETAILER", ev.RetailerID
	case AccountCashInTransit, AccountDriverReceivableVariance, AccountDriverPayableVariance:
		return "DRIVER", ev.DriverID
	case AccountCardPSPTransit:
		return "GATEWAY", coalesceNonEmpty(ev.Gateway, ev.PaymentMode)
	case AccountSupplierPayable, AccountSupplierFeesReceivable, AccountPlatformEscrow,
		AccountCreditNoteAdjustment, AccountClaimsSettlementExpense,
		AccountDiscountsAndAllowances, AccountOpeningBalanceEquity:
		return "SUPPLIER", coalesceNonEmpty(ev.SupplierID, ev.EntityID)
	default:
		return "", ""
	}
}

func eventToSourceType(eventType string) string {
	switch eventType {
	case RuleEventOrderDelivered:
		return "ORDER_DELIVERED"
	case RuleEventARPaymentReceived:
		return "AR_PAYMENT"
	case RuleEventARAdvanceCredited:
		return "AR_ADVANCE"
	case RuleEventARAdjustWaive:
		return "AR_ADJUSTMENT"
	case RuleEventPayoutPaid:
		return "PAYOUT_PAID"
	case RuleEventCreditNoteIssued:
		return "CREDIT_NOTE"
	case RuleEventCashVariance:
		return "CASH_RECON"
	case RuleEventFeePerOrder:
		return "FEE_PER_ORDER"
	case RuleEventSplitTender:
		return "SPLIT_TENDER"
	case RuleEventClaimDispute:
		return "CLAIM_DISPUTE"
	case RuleEventDoorstepShortfall:
		return "DOORSTEP_SHORTFALL"
	case RuleEventMigrationOpeningBalance:
		return "MIGRATION_OPENING"
	default:
		return strings.ToUpper(strings.ReplaceAll(eventType, ".", "_"))
	}
}

func evalRuleCondition(expr string, ev RuleEvalContext) (bool, error) {
	expr = strings.TrimSpace(expr)
	if expr == "" {
		return true, nil
	}
	clauses := strings.Split(expr, "&&")
	for _, rawClause := range clauses {
		clause := strings.TrimSpace(rawClause)
		if clause == "" {
			continue
		}
		ok, err := evalSingleClause(clause, ev)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func evalSingleClause(clause string, ev RuleEvalContext) (bool, error) {
	if idx := strings.Index(clause, " not_in "); idx > 0 {
		lhs := strings.TrimSpace(clause[:idx])
		rhs := strings.TrimSpace(clause[idx+len(" not_in "):])
		val, err := resolveStringField(lhs, ev)
		if err != nil {
			return false, err
		}
		list := parseQuotedList(rhs)
		for _, item := range list {
			if strings.EqualFold(val, item) {
				return false, nil
			}
		}
		return true, nil
	}
	if idx := strings.Index(clause, " in "); idx > 0 {
		lhs := strings.TrimSpace(clause[:idx])
		rhs := strings.TrimSpace(clause[idx+len(" in "):])
		val, err := resolveStringField(lhs, ev)
		if err != nil {
			return false, err
		}
		list := parseQuotedList(rhs)
		for _, item := range list {
			if strings.EqualFold(val, item) {
				return true, nil
			}
		}
		return false, nil
	}
	for _, op := range []string{"==", "!=", ">=", "<=", ">", "<"} {
		if idx := strings.Index(clause, op); idx > 0 {
			lhs := strings.TrimSpace(clause[:idx])
			rhs := strings.TrimSpace(clause[idx+len(op):])
			if strings.HasPrefix(rhs, "'") && strings.HasSuffix(rhs, "'") {
				val, err := resolveStringField(lhs, ev)
				if err != nil {
					return false, err
				}
				target := strings.Trim(rhs, "'")
				if op == "==" {
					return strings.EqualFold(val, target), nil
				}
				if op == "!=" {
					return !strings.EqualFold(val, target), nil
				}
				return false, fmt.Errorf("unsupported string op %q in %q", op, clause)
			}
			lhsNum, err := evalRuleAmount(lhs, ev)
			if err != nil {
				return false, err
			}
			var rhsNum int64
			if _, scanErr := fmt.Sscanf(rhs, "%d", &rhsNum); scanErr != nil {
				rhsNum, err = evalRuleAmount(rhs, ev)
				if err != nil {
					return false, fmt.Errorf("invalid rhs %q in %q", rhs, clause)
				}
			}
			switch op {
			case "==":
				return lhsNum == rhsNum, nil
			case "!=":
				return lhsNum != rhsNum, nil
			case ">":
				return lhsNum > rhsNum, nil
			case ">=":
				return lhsNum >= rhsNum, nil
			case "<":
				return lhsNum < rhsNum, nil
			case "<=":
				return lhsNum <= rhsNum, nil
			}
		}
	}
	return false, fmt.Errorf("unrecognized condition clause %q", clause)
}

func resolveStringField(name string, ev RuleEvalContext) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "payment_mode":
		return ev.PaymentMode, nil
	case "gateway":
		return ev.Gateway, nil
	case "settlement_mode":
		return ev.SettlementMode, nil
	case "action":
		return ev.Action, nil
	case "country_pack":
		return ev.CountryPack, nil
	case "currency":
		return ev.Currency, nil
	default:
		return "", fmt.Errorf("unknown rule string field %q", name)
	}
}

func parseQuotedList(raw string) []string {
	raw = strings.TrimSpace(raw)
	raw = strings.TrimPrefix(raw, "(")
	raw = strings.TrimSuffix(raw, ")")
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"")
		out = append(out, p)
	}
	return out
}

func evalRuleAmount(expr string, ev RuleEvalContext) (int64, error) {
	switch strings.ToLower(strings.TrimSpace(expr)) {
	case "gross_minor":
		return ev.GrossMinor, nil
	case "net_minor":
		return ev.NetMinor, nil
	case "vat_minor":
		return ev.VatMinor, nil
	case "cost_minor":
		return ev.CostMinor, nil
	case "fee_minor":
		return ev.FeeMinor, nil
	case "discount_minor":
		return ev.DiscountMinor, nil
	case "expected_minor":
		return ev.ExpectedMinor, nil
	case "received_minor":
		return ev.ReceivedMinor, nil
	case "difference_minor":
		return ev.DifferenceMinor, nil
	case "abs_difference_minor":
		if ev.DifferenceMinor < 0 {
			return -ev.DifferenceMinor, nil
		}
		return ev.DifferenceMinor, nil
	case "shortfall_minor":
		return ev.ExpectedMinor - ev.ReceivedMinor, nil
	case "net_supplier_minor":
		return ev.GrossMinor - ev.FeeMinor, nil
	default:
		return 0, fmt.Errorf("unknown amount expression %q", expr)
	}
}

func coalesceNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}
