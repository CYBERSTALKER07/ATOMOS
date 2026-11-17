// Package gl implements the ATOMOS append-only, hash-chained General Ledger
// core: Chart of Accounts, data-driven posting rules, double-entry validation,
// sharded period balances, and transactional Spanner + Outbox persistence.
package gl

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// Chart of accounts standard classifications matching Spanner CHK_GlAccounts_Class.
const (
	ClassAsset     = "ASSET"
	ClassLiability = "LIABILITY"
	ClassEquity    = "EQUITY"
	ClassRevenue   = "REVENUE"
	ClassExpense   = "EXPENSE"
)

// Canonical system Chart of Accounts codes.
// Includes all 17 system accounts from payment/double_entry.go plus §6.1 GL accounts.
const (
	// Assets
	AccountAccountsReceivable       = "AR:RETAILER"
	AccountCashInTransit            = "CASH:DRIVER"
	AccountDriverReceivableVariance = "AR:DRIVER:VARIANCE"
	AccountCashOffice               = "CASH:OFFICE"
	AccountCardPSPTransit           = "PSP:GATEWAY"
	AccountBankOperating            = "BANK:OPERATING"
	AccountVatInput                 = "TAX:VAT:INPUT"
	AccountInventory                = "INVENTORY"
	AccountSupplierFeesReceivable   = "AR:SUPPLIER:FEES"
	AccountIntercompanyDueFrom      = "IC:DUE_FROM"

	// Liabilities
	AccountSupplierPayable       = "AP:SUPPLIER"
	AccountPlatformEscrow        = "ESCROW:ORDER"
	AccountRetailerCredit        = "WALLET:RETAILER"
	AccountDriverPayableVariance = "AP:DRIVER:VARIANCE"
	AccountVatOutput             = "TAX:VAT:OUTPUT"
	AccountVendorPayable         = "AP:VENDOR"
	AccountGRNI                  = "GRNI"
	AccountIntercompanyDueTo     = "IC:DUE_TO"

	// Equity
	AccountOpeningBalanceEquity = "EQUITY:OPENING_BALANCE"

	// Revenue
	AccountPlatformFeeRevenue   = "REV:PLATFORM:FEE"
	AccountCashOverShortRevenue = "REV:CASH:OVERAGE"
	AccountSalesRevenue         = "REV:SALES"

	// Expense / Adjustments
	AccountDiscountsAndAllowances  = "EXP:PROMO:DISCOUNT"
	AccountCashWriteOffExpense     = "EXP:CASH:WRITEOFF"
	AccountClaimsSettlementExpense = "EXP:CLAIMS:SETTLEMENT"
	AccountCreditNoteAdjustment    = "ADJ:CREDITNOTE"
	AccountCOGS                    = "COGS"
	AccountShrinkExpense           = "EXP:SHRINK"
	AccountFxRounding              = "FX:ROUNDING"
	AccountFxUnrealizedGainLoss    = "FX:UNREALIZED_GAIN_LOSS"
)

// AccountDef describes a General Ledger account definition in GlAccounts.
type AccountDef struct {
	AccountCode string `json:"account_code"`
	Class       string `json:"class"`
	Name        string `json:"name"`
	ParentCode  string `json:"parent_code,omitempty"`
	Currency    string `json:"currency"`
	IsControl   bool   `json:"is_control"`
	CountryPack string `json:"country_pack"`
}

// AccountRecord represents a persisted GlAccounts row.
type AccountRecord struct {
	EntityID    string     `json:"entity_id"`
	AccountCode string     `json:"account_code"`
	Class       string     `json:"class"`
	Name        string     `json:"name"`
	ParentCode  string     `json:"parent_code,omitempty"`
	Currency    string     `json:"currency"`
	IsControl   bool       `json:"is_control"`
	ActiveFrom  *time.Time `json:"active_from,omitempty"`
	ActiveTo    *time.Time `json:"active_to,omitempty"`
	CountryPack string     `json:"country_pack"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type systemAccountSpec struct {
	code      string
	class     string
	name      string
	parent    string
	isControl bool
}

var systemAccountsList = []systemAccountSpec{
	// Assets
	{AccountAccountsReceivable, ClassAsset, "Accounts Receivable - Retailers", "", true},
	{AccountCashInTransit, ClassAsset, "Cash in Transit - Drivers", "", true},
	{AccountDriverReceivableVariance, ClassAsset, "Driver Shortage Receivable", AccountAccountsReceivable, true},
	{AccountCashOffice, ClassAsset, "Cash Vault - Finance Office", "", false},
	{AccountCardPSPTransit, ClassAsset, "Card / PSP Gateway Transit", "", true},
	{AccountBankOperating, ClassAsset, "Operating Bank Account", "", false},
	{AccountVatInput, ClassAsset, "Input VAT Recoverable", "", false},
	{AccountInventory, ClassAsset, "Merchandise Inventory", "", false},
	{AccountSupplierFeesReceivable, ClassAsset, "Platform Fees Receivable - Suppliers", "", true},
	{AccountIntercompanyDueFrom, ClassAsset, "Intercompany Due From", "", true},

	// Liabilities
	{AccountSupplierPayable, ClassLiability, "Accounts Payable - Suppliers", "", true},
	{AccountPlatformEscrow, ClassLiability, "Order Fulfillment Escrow Liability", "", true},
	{AccountRetailerCredit, ClassLiability, "Retailer Deposit / Credit Wallet", "", true},
	{AccountDriverPayableVariance, ClassLiability, "Driver Cash Overage Suspense", "", true},
	{AccountVatOutput, ClassLiability, "Output VAT Payable", "", false},
	{AccountVendorPayable, ClassLiability, "Accounts Payable - Vendors", "", true},
	{AccountGRNI, ClassLiability, "Goods Received Not Invoiced", "", false},
	{AccountIntercompanyDueTo, ClassLiability, "Intercompany Due To", "", true},

	// Equity
	{AccountOpeningBalanceEquity, ClassEquity, "Opening Balance Migration Offset", "", false},

	// Revenue
	{AccountPlatformFeeRevenue, ClassRevenue, "Platform Marketplace Fee Revenue", "", false},
	{AccountCashOverShortRevenue, ClassRevenue, "Cash Overage Surplus Revenue", "", false},
	{AccountSalesRevenue, ClassRevenue, "Merchandise Sales Revenue", "", false},

	// Expenses / Contra
	{AccountDiscountsAndAllowances, ClassExpense, "Promotional Discounts & Allowances", "", false},
	{AccountCashWriteOffExpense, ClassExpense, "Cash Variance Write-Off Expense", "", false},
	{AccountClaimsSettlementExpense, ClassExpense, "Claims Dispute Settlement Expense", "", false},
	{AccountCreditNoteAdjustment, ClassExpense, "Credit Note Sales Adjustment", "", false},
	{AccountCOGS, ClassExpense, "Cost of Goods Sold", "", false},
	{AccountShrinkExpense, ClassExpense, "Inventory Shrinkage & Spoilage Expense", "", false},
	{AccountFxRounding, ClassExpense, "FX Conversion & Rounding Difference", "", false},
	{AccountFxUnrealizedGainLoss, ClassExpense, "Unrealized FX Gain / Loss", "", false},
}

var systemAccountMap = func() map[string]systemAccountSpec {
	m := make(map[string]systemAccountSpec, len(systemAccountsList))
	for _, s := range systemAccountsList {
		m[s.code] = s
	}
	return m
}()

// DefaultCurrencyForPack returns the canonical ISO currency for a country pack.
func DefaultCurrencyForPack(countryPack string) string {
	packCode := auth.NormalizeCountryCode(countryPack)
	if packCode == "" {
		packCode = auth.DefaultMarketCodeFromEnv()
	}
	if p, ok := auth.ResolveMarketPack(packCode); ok && strings.TrimSpace(p.CurrencyCode) != "" {
		return strings.ToUpper(strings.TrimSpace(p.CurrencyCode))
	}
	return "UZS"
}

// DefaultChartOfAccounts returns the shipped 30-account Chart of Accounts for countryPack and currency.
func DefaultChartOfAccounts(countryPack, currency string) []AccountDef {
	packCode := auth.NormalizeCountryCode(countryPack)
	if packCode == "" {
		packCode = auth.DefaultMarketCodeFromEnv()
	}
	cur := strings.ToUpper(strings.TrimSpace(currency))
	if cur == "" {
		cur = DefaultCurrencyForPack(packCode)
	}
	out := make([]AccountDef, 0, len(systemAccountsList))
	for _, s := range systemAccountsList {
		out = append(out, AccountDef{
			AccountCode: s.code,
			Class:       s.class,
			Name:        s.name,
			ParentCode:  s.parent,
			Currency:    cur,
			IsControl:   s.isControl,
			CountryPack: packCode,
		})
	}
	return out
}

// LookupSystemAccount returns the system AccountDef for code if it is one of the 30 shipped accounts.
func LookupSystemAccount(code string) (AccountDef, bool) {
	c := strings.ToUpper(strings.TrimSpace(code))
	s, ok := systemAccountMap[c]
	if !ok {
		return AccountDef{}, false
	}
	return AccountDef{
		AccountCode: s.code,
		Class:       s.class,
		Name:        s.name,
		ParentCode:  s.parent,
		IsControl:   s.isControl,
	}, true
}

// IsDebitNormalClass reports whether the account class increases on DEBIT (ASSET, EXPENSE).
func IsDebitNormalClass(class string) bool {
	switch strings.ToUpper(strings.TrimSpace(class)) {
	case ClassAsset, ClassExpense:
		return true
	default:
		return false
	}
}

// NormalizeAccountCode maps legacy sub-ledger-suffixed account codes (e.g., "AR:RETAILER:ret_1",
// "ESCROW:ORDER:ord_1", "AR:sup_1") onto canonical COA AccountCode plus inferred PartyType/PartyID.
func NormalizeAccountCode(raw string) (canonical, partyType, partyID string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", "", ""
	}
	upper := strings.ToUpper(trimmed)
	if upper == "VAT:INPUT" || strings.HasPrefix(upper, "VAT:INPUT:") {
		return AccountVatInput, "", ""
	}
	if _, ok := systemAccountMap[upper]; ok {
		return upper, "", ""
	}

	type prefixRule struct {
		prefix    string
		canonical string
		partyType string
	}
	rules := []prefixRule{
		{"AR:DRIVER:VARIANCE:", AccountDriverReceivableVariance, "DRIVER"},
		{"AP:DRIVER:VARIANCE:", AccountDriverPayableVariance, "DRIVER"},
		{"AR:SUPPLIER:FEES:", AccountSupplierFeesReceivable, "SUPPLIER"},
		{"AR:RETAILER:", AccountAccountsReceivable, "RETAILER"},
		{"CASH:DRIVER:", AccountCashInTransit, "DRIVER"},
		{"PSP:GATEWAY:", AccountCardPSPTransit, "GATEWAY"},
		{"AP:SUPPLIER:", AccountSupplierPayable, "SUPPLIER"},
		{"ESCROW:ORDER:", AccountPlatformEscrow, "ORDER"},
		{"WALLET:RETAILER:", AccountRetailerCredit, "RETAILER"},
		{"ADJ:CREDITNOTE:", AccountCreditNoteAdjustment, "SUPPLIER"},
		{"AP:VENDOR:", AccountVendorPayable, "VENDOR"},
		{"IC:DUE_FROM:", AccountIntercompanyDueFrom, "ENTITY"},
		{"IC:DUE_TO:", AccountIntercompanyDueTo, "ENTITY"},
	}
	for _, r := range rules {
		if strings.HasPrefix(upper, r.prefix) {
			pid := strings.TrimSpace(trimmed[len(r.prefix):])
			return r.canonical, r.partyType, pid
		}
	}
	// Legacy BuildMigrationOpeningBalanceJournalEntry used "AR:<supplierId>"
	if strings.HasPrefix(upper, "AR:") && len(trimmed) > 3 {
		rest := strings.TrimSpace(trimmed[3:])
		if !strings.Contains(rest, ":") {
			return AccountAccountsReceivable, "SUPPLIER", rest
		}
	}
	return upper, "", ""
}

// SeedDefaultAccountsInTxn inserts any missing canonical COA accounts for entityID inside txn.
func SeedDefaultAccountsInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, entityID, countryPack, currency string) (int, error) {
	if txn == nil {
		return 0, ErrNilTransaction
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return 0, fmt.Errorf("%w: entity_id is required", ErrInvalidJournalInput)
	}
	existing := make(map[string]struct{}, len(systemAccountsList))
	iter := txn.Query(ctx, spanner.Statement{
		SQL:    `SELECT AccountCode FROM GlAccounts WHERE EntityId = @entityId`,
		Params: map[string]any{"entityId": entityID},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("read GlAccounts: %w", err)
		}
		var code string
		if err := row.Column(0, &code); err != nil {
			return 0, fmt.Errorf("scan GlAccounts: %w", err)
		}
		existing[strings.ToUpper(strings.TrimSpace(code))] = struct{}{}
	}

	defs := DefaultChartOfAccounts(countryPack, currency)
	muts := make([]*spanner.Mutation, 0, len(defs))
	for _, d := range defs {
		if _, ok := existing[d.AccountCode]; ok {
			continue
		}
		muts = append(muts, spanner.InsertMap("GlAccounts", map[string]any{
			"EntityId":    entityID,
			"AccountCode": d.AccountCode,
			"Class":       d.Class,
			"Name":        d.Name,
			"ParentCode":  nullIfEmpty(d.ParentCode),
			"Currency":    d.Currency,
			"IsControl":   d.IsControl,
			"ActiveFrom":  nil,
			"ActiveTo":    nil,
			"CountryPack": d.CountryPack,
			"CreatedAt":   spanner.CommitTimestamp,
			"UpdatedAt":   spanner.CommitTimestamp,
		}))
	}
	if len(muts) == 0 {
		return 0, nil
	}
	if err := txn.BufferWrite(muts); err != nil {
		return 0, fmt.Errorf("buffer GlAccounts seed: %w", err)
	}
	return len(muts), nil
}

// SeedDefaultAccounts idempotently seeds all 30 system COA accounts for entityID.
func SeedDefaultAccounts(ctx context.Context, client *spanner.Client, entityID, countryPack, currency string) (int, error) {
	if client == nil {
		return 0, fmt.Errorf("spanner client unavailable")
	}
	var inserted int
	err := spannerutils.RunReadWriteTransaction(ctx, client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		n, err := SeedDefaultAccountsInTxn(ctx, txn, entityID, countryPack, currency)
		if err != nil {
			return err
		}
		inserted = n
		return nil
	})
	return inserted, err
}
