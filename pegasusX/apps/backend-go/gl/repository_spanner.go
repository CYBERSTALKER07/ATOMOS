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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Repository defines persistence and query capabilities for the General Ledger.
type Repository interface {
	PostEntry(ctx context.Context, input JournalEntryInput) (JournalEntryResult, error)
	PostEvent(ctx context.Context, eventType string, ev RuleEvalContext) (JournalEntryResult, error)
	ReverseEntry(ctx context.Context, entityID, entryID, idempotencyKey, reason, actor string) (JournalEntryResult, error)
	GetEntry(ctx context.Context, entityID, entryID string) (JournalEntryRecord, error)
	GetTrialBalance(ctx context.Context, entityID, periodID string) (TrialBalanceReport, error)
	GetAccountLedger(ctx context.Context, entityID, accountCode, periodID string, limit int) ([]AccountLedgerRow, error)
	ListPostingRules(ctx context.Context, countryPack, eventType string) ([]PostingRule, error)
	SeedDefaultAccounts(ctx context.Context, entityID, countryPack, currency string) (int, error)
	SeedPostingRules(ctx context.Context, countryPack string) (int, error)
	BackfillFromPaymentLedgerEntries(ctx context.Context, entityID string, opts BackfillOptions) (BackfillReport, error)
}

// MultiEntityJournalRepository is an optional extension of Repository supporting
// legal-entity-scoped trial balance queries, journal listing, and ownership checks.
type MultiEntityJournalRepository interface {
	GetTrialBalanceFiltered(ctx context.Context, tenantID, periodID, legalEntityID string) (TrialBalanceReport, error)
	ListJournals(ctx context.Context, filter JournalFilter) ([]JournalEntryRecord, error)
	IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error)
}

// SpannerRepository implements Repository against Cloud Spanner.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository constructs a Spanner-backed GL repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// Client exposes the underlying Spanner client.
func (r *SpannerRepository) Client() *spanner.Client {
	if r == nil {
		return nil
	}
	return r.client
}

// PostEntry validates and appends a single balanced journal entry inside a RW transaction.
func (r *SpannerRepository) PostEntry(ctx context.Context, input JournalEntryInput) (JournalEntryResult, error) {
	if r == nil || r.client == nil {
		return JournalEntryResult{}, fmt.Errorf("gl spanner repository: nil client")
	}
	var out JournalEntryResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		res, err := BufferJournalEntryInTxn(ctx, txn, nil, input)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	if err != nil {
		return JournalEntryResult{}, err
	}
	return out, nil
}

// PostEvent evaluates active PostingRules for eventType and persists the resulting entry.
func (r *SpannerRepository) PostEvent(ctx context.Context, eventType string, ev RuleEvalContext) (JournalEntryResult, error) {
	if r == nil || r.client == nil {
		return JournalEntryResult{}, fmt.Errorf("gl spanner repository: nil client")
	}
	var out JournalEntryResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		res, err := PostEventInTxn(ctx, txn, nil, eventType, ev)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	if err != nil {
		return JournalEntryResult{}, err
	}
	return out, nil
}

// ReverseEntry appends a balanced inverse entry referencing entryID without modifying or deleting the original.
func (r *SpannerRepository) ReverseEntry(ctx context.Context, entityID, entryID, idempotencyKey, reason, actor string) (JournalEntryResult, error) {
	if r == nil || r.client == nil {
		return JournalEntryResult{}, fmt.Errorf("gl spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	entryID = strings.TrimSpace(entryID)
	if entityID == "" || entryID == "" {
		return JournalEntryResult{}, fmt.Errorf("%w: entity_id and entry_id are required", ErrInvalidJournalInput)
	}
	idemKey := strings.TrimSpace(idempotencyKey)
	if idemKey == "" {
		idemKey = "reverse:" + entryID
	}
	actor = strings.TrimSpace(actor)
	if actor == "" {
		actor = "system"
	}

	var out JournalEntryResult
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		orig, err := readJournalEntryInTxn(ctx, txn, entityID, entryID)
		if err != nil {
			return err
		}
		if len(orig.Postings) < 2 {
			return fmt.Errorf("%w: original entry %s has %d postings", ErrInsufficientPostings, entryID, len(orig.Postings))
		}
		revPostings := make([]PostingInput, 0, len(orig.Postings))
		for _, p := range orig.Postings {
			invDir := DirectionCredit
			if p.Direction == DirectionCredit {
				invDir = DirectionDebit
			}
			revPostings = append(revPostings, PostingInput{
				AccountCode:           p.AccountCode,
				Direction:             invDir,
				AmountMinor:           p.AmountMinor,
				Currency:              p.Currency,
				FunctionalAmountMinor: p.FunctionalAmountMinor,
				FxRateID:              p.FxRateID,
				PartyType:             p.PartyType,
				PartyID:               p.PartyID,
				WarehouseID:           p.WarehouseID,
				ProductID:             p.ProductID,
				LotID:                 p.LotID,
				Description:           coalesceNonEmpty(p.Description, "Reversal of "+entryID),
			})
		}
		revIn := JournalEntryInput{
			EntityID:        entityID,
			LegalEntityID:   orig.LegalEntityID,
			SourceType:      "REVERSAL",
			SourceID:        entryID,
			OrderID:         orig.OrderID,
			Memo:            coalesceNonEmpty(strings.TrimSpace(reason), "Reversal of journal entry "+entryID),
			Currency:        orig.Currency,
			CountryPack:     orig.CountryPack,
			ReversesEntryID: entryID,
			IdempotencyKey:  idemKey,
			PostedBy:        actor,
			OccurredAt:      time.Now().UTC(),
			Postings:        revPostings,
		}
		res, err := BufferJournalEntryInTxn(ctx, txn, nil, revIn)
		if err != nil {
			return err
		}
		out = res
		return nil
	})
	if err != nil {
		return JournalEntryResult{}, err
	}
	return out, nil
}

// GetEntry returns one JournalEntries record with its ordered postings inside a consistent ReadOnlyTransaction.
func (r *SpannerRepository) GetEntry(ctx context.Context, entityID, entryID string) (JournalEntryRecord, error) {
	if r == nil || r.client == nil {
		return JournalEntryRecord{}, fmt.Errorf("gl spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	entryID = strings.TrimSpace(entryID)
	if entityID == "" || entryID == "" {
		return JournalEntryRecord{}, fmt.Errorf("%w: entity_id and entry_id are required", ErrInvalidJournalInput)
	}

	var out JournalEntryRecord
	err := spannerutils.RunReadOnlyTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadOnlyTransaction) error {
		rec, err := readJournalEntryInTxn(ctx, txn, entityID, entryID)
		if err != nil {
			return err
		}
		out = rec
		return nil
	})
	if err != nil {
		return JournalEntryRecord{}, err
	}
	return out, nil
}

// GetTrialBalance aggregates sharded GlBalances for (entityID, periodID) using a bounded stale read snapshot.
func (r *SpannerRepository) GetTrialBalance(ctx context.Context, entityID, periodID string) (TrialBalanceReport, error) {
	if r == nil || r.client == nil {
		return TrialBalanceReport{}, fmt.Errorf("gl spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	periodID = strings.TrimSpace(periodID)
	if entityID == "" {
		return TrialBalanceReport{}, fmt.Errorf("%w: entity_id is required", ErrInvalidJournalInput)
	}
	if periodID == "" {
		periodID = ComputePeriodID(ctx, entityID, "", time.Now().UTC())
	}

	iter := r.client.Single().WithTimestampBound(spanner.MaxStaleness(5*time.Second)).Query(ctx, spanner.Statement{
		SQL: `SELECT b.AccountCode, b.Currency, SUM(b.DebitMinor) AS DebitMinor, SUM(b.CreditMinor) AS CreditMinor,
		             a.Class, a.Name
		      FROM GlBalances b
		      LEFT JOIN GlAccounts a ON a.EntityId = b.EntityId AND a.AccountCode = b.AccountCode
		      WHERE b.EntityId = @entityId AND b.PeriodId = @periodId
		      GROUP BY b.AccountCode, b.Currency, a.Class, a.Name
		      ORDER BY b.AccountCode ASC`,
		Params: map[string]any{
			"entityId": entityID,
			"periodId": periodID,
		},
	})
	defer iter.Stop()

	lines := make([]TrialBalanceLine, 0)
	var totalDebits, totalCredits int64
	reportCurrency := ""
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return TrialBalanceReport{}, fmt.Errorf("query GlBalances trial balance: %w", err)
		}
		var line TrialBalanceLine
		var classNull, nameNull spanner.NullString
		if err := row.Columns(&line.AccountCode, &line.Currency, &line.DebitMinor, &line.CreditMinor, &classNull, &nameNull); err != nil {
			return TrialBalanceReport{}, fmt.Errorf("scan GlBalances trial balance: %w", err)
		}
		uc := strings.ToUpper(strings.TrimSpace(line.AccountCode))
		line.Class = strings.ToUpper(strings.TrimSpace(nullStrVal(classNull)))
		line.AccountName = nullStrVal(nameNull)
		if line.Class == "" {
			if def, ok := LookupSystemAccount(uc); ok {
				line.Class = def.Class
				if line.AccountName == "" {
					line.AccountName = def.Name
				}
			}
		}
		if IsDebitNormalClass(line.Class) {
			line.NetMinor = line.DebitMinor - line.CreditMinor
		} else {
			line.NetMinor = line.CreditMinor - line.DebitMinor
		}
		totalDebits += line.DebitMinor
		totalCredits += line.CreditMinor
		if reportCurrency == "" {
			reportCurrency = line.Currency
		}
		lines = append(lines, line)
	}

	if reportCurrency == "" {
		if c, err := auth.CurrencyFromContext(ctx, entityID); err == nil && c != "" {
			reportCurrency = c
		} else {
			reportCurrency = DefaultCurrencyForPack("")
		}
	}

	return TrialBalanceReport{
		EntityID:          entityID,
		PeriodID:          periodID,
		Currency:          reportCurrency,
		TotalDebitsMinor:  totalDebits,
		TotalCreditsMinor: totalCredits,
		Balanced:          totalDebits == totalCredits,
		Accounts:          lines,
	}, nil
}

// GetAccountLedger returns bounded postings for one account ordered newest-first.
func (r *SpannerRepository) GetAccountLedger(ctx context.Context, entityID, accountCode, periodID string, limit int) ([]AccountLedgerRow, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("gl spanner repository: nil client")
	}
	entityID = strings.TrimSpace(entityID)
	accountCode = strings.ToUpper(strings.TrimSpace(accountCode))
	periodID = strings.TrimSpace(periodID)
	if entityID == "" || accountCode == "" {
		return nil, fmt.Errorf("%w: entity_id and account_code are required", ErrInvalidJournalInput)
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	params := map[string]any{
		"entityId":    entityID,
		"accountCode": accountCode,
		"limit":       int64(limit),
	}
	periodFilter := ""
	if periodID != "" {
		periodFilter = " AND e.PeriodId = @periodId"
		params["periodId"] = periodID
	}

	sql := `SELECT p.EntryId, e.PeriodId, p.Line, p.AccountCode, p.Direction, p.AmountMinor, p.Currency,
	               p.FunctionalAmountMinor, p.FxRateId, p.PartyType, p.PartyId,
	               e.SourceType, e.SourceId, e.OrderId, e.ReversesEntryId, e.Memo, p.Description,
	               e.EntryHash, e.OccurredAt, e.PostedAt
	        FROM JournalPostings@{FORCE_INDEX=Idx_JournalPostings_ByAccount} p
	        JOIN JournalEntries e ON p.EntityId = e.EntityId AND p.EntryId = e.EntryId
	        WHERE p.EntityId = @entityId AND p.AccountCode = @accountCode` + periodFilter + `
	        ORDER BY e.PostedAt DESC, p.EntryId DESC, p.Line ASC
	        LIMIT @limit`

	iter := r.client.Single().WithTimestampBound(spanner.MaxStaleness(5*time.Second)).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	rows := make([]AccountLedgerRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query account ledger: %w", err)
		}
		var item AccountLedgerRow
		var dir string
		var fxRateID, partyType, partyID, orderID, revID, memo, desc spanner.NullString
		if err := row.Columns(
			&item.EntryID, &item.PeriodID, &item.Line, &item.AccountCode, &dir,
			&item.AmountMinor, &item.Currency, &item.FunctionalAmountMinor, &fxRateID,
			&partyType, &partyID, &item.SourceType, &item.SourceID, &orderID,
			&revID, &memo, &desc, &item.EntryHash, &item.OccurredAt, &item.PostedAt,
		); err != nil {
			return nil, fmt.Errorf("scan account ledger: %w", err)
		}
		item.Direction = Direction(dir)
		item.FxRateID = nullStrVal(fxRateID)
		item.PartyType = nullStrVal(partyType)
		item.PartyID = nullStrVal(partyID)
		item.OrderID = nullStrVal(orderID)
		item.ReversesEntryID = nullStrVal(revID)
		item.Memo = nullStrVal(memo)
		item.Description = nullStrVal(desc)
		item.OccurredAt = item.OccurredAt.UTC()
		item.PostedAt = item.PostedAt.UTC()
		rows = append(rows, item)
	}
	return rows, nil
}

// ListPostingRules returns active rules stored in PostingRules for (countryPack, eventType),
// falling back to shipped defaults if none are stored yet.
func (r *SpannerRepository) ListPostingRules(ctx context.Context, countryPack, eventType string) ([]PostingRule, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("gl spanner repository: nil client")
	}
	pack := auth.NormalizeCountryCode(countryPack)
	if pack == "" {
		pack = "UZ"
	}
	eventType = strings.ToLower(strings.TrimSpace(eventType))

	params := map[string]any{"pack": pack}
	sql := `SELECT CountryPack, RuleId, Version, EventType, ConditionExpr, DebitAccount, CreditAccount, AmountExpr, Description, Active, UpdatedAt
	        FROM PostingRules
	        WHERE CountryPack = @pack AND Active = TRUE`
	if eventType != "" {
		sql += ` AND EventType = @eventType`
		params["eventType"] = eventType
	}
	sql += ` ORDER BY EventType ASC, RuleId ASC, Version DESC`

	iter := r.client.Single().WithTimestampBound(spanner.MaxStaleness(5*time.Second)).Query(ctx, spanner.Statement{
		SQL:    sql,
		Params: params,
	})
	defer iter.Stop()

	seen := make(map[string]struct{})
	var out []PostingRule
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query PostingRules: %w", err)
		}
		var pr PostingRule
		var cond, desc spanner.NullString
		if err := row.Columns(&pr.CountryPack, &pr.RuleID, &pr.Version, &pr.EventType, &cond, &pr.DebitAccount, &pr.CreditAccount, &pr.AmountExpr, &desc, &pr.Active, &pr.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan PostingRules: %w", err)
		}
		pr.ConditionExpr = nullStrVal(cond)
		pr.Description = nullStrVal(desc)
		pr.UpdatedAt = pr.UpdatedAt.UTC()
		if _, dup := seen[pr.RuleID]; dup {
			continue
		}
		seen[pr.RuleID] = struct{}{}
		out = append(out, pr)
	}
	if len(out) == 0 {
		for _, d := range DefaultPostingRules(pack) {
			if eventType == "" || d.EventType == eventType {
				out = append(out, d)
			}
		}
	}
	return out, nil
}

// SeedDefaultAccounts delegates to SeedDefaultAccounts on the repository client.
func (r *SpannerRepository) SeedDefaultAccounts(ctx context.Context, entityID, countryPack, currency string) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("gl spanner repository: nil client")
	}
	return SeedDefaultAccounts(ctx, r.client, entityID, countryPack, currency)
}

// SeedPostingRules delegates to SeedPostingRules on the repository client.
func (r *SpannerRepository) SeedPostingRules(ctx context.Context, countryPack string) (int, error) {
	if r == nil || r.client == nil {
		return 0, fmt.Errorf("gl spanner repository: nil client")
	}
	return SeedPostingRules(ctx, r.client, countryPack)
}

type spannerReader interface {
	ReadRow(ctx context.Context, table string, key spanner.Key, columns []string) (*spanner.Row, error)
	Query(ctx context.Context, statement spanner.Statement) *spanner.RowIterator
}

func readJournalEntryInTxn(ctx context.Context, r spannerReader, entityID, entryID string) (JournalEntryRecord, error) {
	row, err := r.ReadRow(ctx, "JournalEntries", spanner.Key{entityID, entryID}, []string{
		"EntityId", "EntryId", "LegalEntityId", "PeriodId", "OriginalPeriodId", "SourceEventId", "SourceType", "SourceId", "OrderId",
		"Memo", "Currency", "CountryPack", "ReversesEntryId", "IdempotencyKey", "PrevHash", "EntryHash",
		"PostedBy", "OccurredAt", "PostedAt",
	})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return JournalEntryRecord{}, fmt.Errorf("%w: %s", ErrEntryNotFound, entryID)
		}
		return JournalEntryRecord{}, fmt.Errorf("read journal entry %s: %w", entryID, err)
	}
	rec, err := scanJournalEntryRow(row)
	if err != nil {
		return JournalEntryRecord{}, err
	}

	pIter := r.Query(ctx, spanner.Statement{
		SQL: `SELECT EntityId, EntryId, Line, AccountCode, Direction, AmountMinor, Currency,
		             FunctionalAmountMinor, FxRateId, PartyType, PartyId, WarehouseId,
		             ProductId, LotId, Description
		      FROM JournalPostings
		      WHERE EntityId = @entityId AND EntryId = @entryId
		      ORDER BY Line ASC`,
		Params: map[string]any{
			"entityId": entityID,
			"entryId":  entryID,
		},
	})
	defer pIter.Stop()

	postings := make([]JournalPostingRecord, 0, 4)
	for {
		pRow, err := pIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return JournalEntryRecord{}, fmt.Errorf("query journal postings %s: %w", entryID, err)
		}
		var p JournalPostingRecord
		var dir string
		var fxRateID, partyType, partyID, warehouseID, productID, lotID, desc spanner.NullString
		if err := pRow.Columns(
			&p.EntityID, &p.EntryID, &p.Line, &p.AccountCode, &dir, &p.AmountMinor, &p.Currency,
			&p.FunctionalAmountMinor, &fxRateID, &partyType, &partyID, &warehouseID,
			&productID, &lotID, &desc,
		); err != nil {
			return JournalEntryRecord{}, fmt.Errorf("scan journal posting %s: %w", entryID, err)
		}
		p.Direction = Direction(dir)
		p.FxRateID = nullStrVal(fxRateID)
		p.PartyType = nullStrVal(partyType)
		p.PartyID = nullStrVal(partyID)
		p.WarehouseID = nullStrVal(warehouseID)
		p.ProductID = nullStrVal(productID)
		p.LotID = nullStrVal(lotID)
		p.Description = nullStrVal(desc)
		postings = append(postings, p)
	}
	rec.Postings = postings
	return rec, nil
}

// IsLegalEntityOwned reports whether entityID exists in LegalEntities under tenantID.
func (r *SpannerRepository) IsLegalEntityOwned(ctx context.Context, tenantID, entityID string) (bool, error) {
	if r == nil || r.client == nil {
		return false, fmt.Errorf("gl spanner repository: nil client")
	}
	tenantID = strings.TrimSpace(tenantID)
	entityID = strings.TrimSpace(entityID)
	if tenantID == "" || entityID == "" {
		return false, nil
	}
	if tenantID == entityID {
		return true, nil
	}
	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()
	row, err := ro.ReadRow(ctx, "LegalEntities", spanner.Key{tenantID, entityID}, []string{"EntityId", "Status"})
	if err != nil {
		if status.Code(err) == codes.NotFound || spanner.ErrCode(err) == codes.NotFound {
			return false, nil
		}
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "NotFound") {
			return false, nil
		}
		return false, fmt.Errorf("read LegalEntities(%s, %s): %w", tenantID, entityID, err)
	}
	return row != nil, nil
}

// GetTrialBalanceFiltered returns the trial balance for tenantID and periodID, optionally
// filtered to one LegalEntityId inside a consistent ReadOnlyTransaction.
func (r *SpannerRepository) GetTrialBalanceFiltered(ctx context.Context, tenantID, periodID, legalEntityID string) (TrialBalanceReport, error) {
	tenantID = strings.TrimSpace(tenantID)
	legalEntityID = strings.TrimSpace(legalEntityID)
	periodID = strings.TrimSpace(periodID)
	if legalEntityID == "" || legalEntityID == tenantID {
		return r.GetTrialBalance(ctx, tenantID, periodID)
	}
	if r == nil || r.client == nil {
		return TrialBalanceReport{}, fmt.Errorf("gl spanner repository: nil client")
	}
	if tenantID == "" {
		return TrialBalanceReport{}, fmt.Errorf("%w: tenant_id is required", ErrInvalidJournalInput)
	}
	if periodID == "" {
		periodID = ComputePeriodID(ctx, tenantID, "", time.Now().UTC())
	}

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	entityBaseCurrency := ""
	if leRow, leErr := ro.ReadRow(ctx, "LegalEntities", spanner.Key{tenantID, legalEntityID}, []string{"BaseCurrency"}); leErr == nil && leRow != nil {
		var bc string
		if cErr := leRow.Columns(&bc); cErr == nil {
			entityBaseCurrency = strings.ToUpper(strings.TrimSpace(bc))
		}
	}

	iter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT p.AccountCode, p.Currency, COALESCE(p.FxRateId, '') AS FxRateId,
		             p.Direction, p.AmountMinor, p.FunctionalAmountMinor,
		             a.Class, a.Name
		      FROM JournalPostings p
		      JOIN JournalEntries e ON p.EntityId = e.EntityId AND p.EntryId = e.EntryId
		      LEFT JOIN GlAccounts a ON a.EntityId = p.EntityId AND a.AccountCode = p.AccountCode
		      WHERE e.EntityId = @tenantId AND e.LegalEntityId = @legalEntityId AND e.PeriodId = @periodId
		      ORDER BY p.AccountCode ASC, p.EntryId ASC, p.Line ASC`,
		Params: map[string]any{
			"tenantId":      tenantID,
			"legalEntityId": legalEntityID,
			"periodId":      periodID,
		},
	})
	defer iter.Stop()

	type tbKey struct {
		code string
		curr string
	}
	grouped := make(map[tbKey]*TrialBalanceLine)
	order := make([]tbKey, 0)
	var totalDebits, totalCredits int64

	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return TrialBalanceReport{}, fmt.Errorf("query filtered trial balance: %w", err)
		}
		var code, curr, fxRateID, dir string
		var amtMinor, funcAmtMinor int64
		var classNull, nameNull spanner.NullString
		if err := row.Columns(&code, &curr, &fxRateID, &dir, &amtMinor, &funcAmtMinor, &classNull, &nameNull); err != nil {
			return TrialBalanceReport{}, fmt.Errorf("scan filtered trial balance: %w", err)
		}
		uc := strings.ToUpper(strings.TrimSpace(code))
		effCurr := strings.ToUpper(strings.TrimSpace(curr))
		useAmt := amtMinor
		if strings.TrimSpace(fxRateID) != "" {
			useAmt = funcAmtMinor
			if entityBaseCurrency != "" {
				effCurr = entityBaseCurrency
			}
		} else if funcAmtMinor != 0 && funcAmtMinor == amtMinor {
			useAmt = funcAmtMinor
		}

		k := tbKey{code: uc, curr: effCurr}
		line, ok := grouped[k]
		if !ok {
			cls := strings.ToUpper(strings.TrimSpace(nullStrVal(classNull)))
			nm := nullStrVal(nameNull)
			if cls == "" {
				if def, found := LookupSystemAccount(uc); found {
					cls = def.Class
					if nm == "" {
						nm = def.Name
					}
				}
			}
			line = &TrialBalanceLine{
				AccountCode: uc,
				AccountName: nm,
				Class:       cls,
				Currency:    effCurr,
			}
			grouped[k] = line
			order = append(order, k)
		}
		if strings.EqualFold(dir, string(DirectionDebit)) {
			line.DebitMinor += useAmt
			totalDebits += useAmt
		} else {
			line.CreditMinor += useAmt
			totalCredits += useAmt
		}
	}

	lines := make([]TrialBalanceLine, 0, len(order))
	reportCurrency := entityBaseCurrency
	for _, k := range order {
		line := grouped[k]
		if IsDebitNormalClass(line.Class) {
			line.NetMinor = line.DebitMinor - line.CreditMinor
		} else {
			line.NetMinor = line.CreditMinor - line.DebitMinor
		}
		if reportCurrency == "" {
			reportCurrency = line.Currency
		}
		lines = append(lines, *line)
	}
	if reportCurrency == "" {
		if c, err := auth.CurrencyFromContext(ctx, tenantID); err == nil && c != "" {
			reportCurrency = c
		} else {
			reportCurrency = DefaultCurrencyForPack("")
		}
	}

	return TrialBalanceReport{
		EntityID:          legalEntityID,
		TenantID:          tenantID,
		LegalEntityID:     legalEntityID,
		PeriodID:          periodID,
		Currency:          reportCurrency,
		TotalDebitsMinor:  totalDebits,
		TotalCreditsMinor: totalCredits,
		Balanced:          totalDebits == totalCredits,
		Accounts:          lines,
	}, nil
}

// ListJournals returns journal entries with postings matching filter inside a ReadOnlyTransaction.
func (r *SpannerRepository) ListJournals(ctx context.Context, filter JournalFilter) ([]JournalEntryRecord, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("gl spanner repository: nil client")
	}
	tenantID := strings.TrimSpace(filter.TenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id is required", ErrInvalidJournalInput)
	}
	legalEntityID := strings.TrimSpace(filter.LegalEntityID)
	periodID := strings.TrimSpace(filter.PeriodID)
	sourceType := strings.ToUpper(strings.TrimSpace(filter.SourceType))
	accountCode := strings.ToUpper(strings.TrimSpace(filter.AccountCode))
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	params := map[string]any{
		"tenantId": tenantID,
		"limit":    int64(limit),
	}
	sql := `SELECT e.EntityId, e.EntryId, e.LegalEntityId, e.PeriodId, e.OriginalPeriodId, e.SourceEventId,
	               e.SourceType, e.SourceId, e.OrderId, e.Memo, e.Currency, e.CountryPack,
	               e.ReversesEntryId, e.IdempotencyKey, e.PrevHash, e.EntryHash,
	               e.PostedBy, e.OccurredAt, e.PostedAt
	        FROM JournalEntries e
	        WHERE e.EntityId = @tenantId`
	if legalEntityID != "" {
		sql += ` AND e.LegalEntityId = @legalEntityId`
		params["legalEntityId"] = legalEntityID
	}
	if periodID != "" {
		sql += ` AND e.PeriodId = @periodId`
		params["periodId"] = periodID
	}
	if sourceType != "" {
		sql += ` AND e.SourceType = @sourceType`
		params["sourceType"] = sourceType
	}
	if accountCode != "" {
		sql += ` AND EXISTS (SELECT 1 FROM JournalPostings p WHERE p.EntityId = e.EntityId AND p.EntryId = e.EntryId AND p.AccountCode = @accountCode)`
		params["accountCode"] = accountCode
	}
	sql += ` ORDER BY e.PostedAt DESC, e.EntryId DESC LIMIT @limit`

	ro := r.client.ReadOnlyTransaction()
	defer ro.Close()

	iter := ro.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	entries := make([]JournalEntryRecord, 0)
	entryIDs := make([]string, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ListJournals: %w", err)
		}
		rec, err := scanJournalEntryRow(row)
		if err != nil {
			return nil, err
		}
		entries = append(entries, rec)
		entryIDs = append(entryIDs, rec.EntryID)
	}
	if len(entryIDs) == 0 {
		return entries, nil
	}

	pIter := ro.Query(ctx, spanner.Statement{
		SQL: `SELECT EntityId, EntryId, Line, AccountCode, Direction, AmountMinor, Currency,
		             FunctionalAmountMinor, FxRateId, PartyType, PartyId, WarehouseId,
		             ProductId, LotId, Description
		      FROM JournalPostings
		      WHERE EntityId = @tenantId AND EntryId IN UNNEST(@entryIds)
		      ORDER BY EntryId ASC, Line ASC`,
		Params: map[string]any{
			"tenantId": tenantID,
			"entryIds": entryIDs,
		},
	})
	defer pIter.Stop()

	byEntry := make(map[string][]JournalPostingRecord, len(entryIDs))
	for {
		row, err := pIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query ListJournals postings: %w", err)
		}
		var p JournalPostingRecord
		var dir string
		var fxRateID, partyType, partyID, warehouseID, productID, lotID, desc spanner.NullString
		if err := row.Columns(
			&p.EntityID, &p.EntryID, &p.Line, &p.AccountCode, &dir, &p.AmountMinor, &p.Currency,
			&p.FunctionalAmountMinor, &fxRateID, &partyType, &partyID, &warehouseID,
			&productID, &lotID, &desc,
		); err != nil {
			return nil, fmt.Errorf("scan ListJournals posting: %w", err)
		}
		p.Direction = Direction(dir)
		p.FxRateID = nullStrVal(fxRateID)
		p.PartyType = nullStrVal(partyType)
		p.PartyID = nullStrVal(partyID)
		p.WarehouseID = nullStrVal(warehouseID)
		p.ProductID = nullStrVal(productID)
		p.LotID = nullStrVal(lotID)
		p.Description = nullStrVal(desc)
		byEntry[p.EntryID] = append(byEntry[p.EntryID], p)
	}
	for i := range entries {
		entries[i].Postings = byEntry[entries[i].EntryID]
	}
	return entries, nil
}
