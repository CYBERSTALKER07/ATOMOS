package ledger

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

type SpannerRepository struct {
	client *spanner.Client
}

func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

type spannerTxnBuffer struct {
	events []outbox.Event
	audits []outbox.AuditEntry
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}

func (b *spannerTxnBuffer) BufferAudit(_ context.Context, e outbox.AuditEntry) error {
	b.audits = append(b.audits, e)
	return nil
}

func (r *SpannerRepository) writeWithOutbox(ctx context.Context, emit func(outbox.TxnBuffer) error, bases ...*spanner.Mutation) error {
	err := spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := make([]*spanner.Mutation, 0, len(bases)+len(buf.events))
		mutations = append(mutations, bases...)
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			row := map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}
			if e.PublishedAt != nil {
				row["PublishedAt"] = e.PublishedAt.UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", row))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("ledger write transaction: %w", err)
	}
	return nil
}

func (r *SpannerRepository) CreateAccount(ctx context.Context, act Account) error {
	mut := spanner.InsertOrUpdateMap("LedgerAccounts", map[string]any{
		"AccountId": act.AccountID,
		"Name":      act.Name,
		"Type":      string(act.Type),
		"Currency":  act.Currency,
		"CreatedAt": spanner.CommitTimestamp,
	})
	return r.writeWithOutbox(ctx, nil, mut)
}

func (r *SpannerRepository) RecordTransaction(ctx context.Context, txn Transaction, entries []Entry, emit func(outbox.TxnBuffer) error) error {
	muts := make([]*spanner.Mutation, 0, 1+len(entries))
	
	muts = append(muts, spanner.InsertMap("LedgerTransactions", map[string]any{
		"TransactionId": txn.TransactionID,
		"ReferenceId":   txn.ReferenceID,
		"Description":   txn.Description,
		"CreatedAt":     spanner.CommitTimestamp,
	}))

	for _, e := range entries {
		muts = append(muts, spanner.InsertMap("LedgerEntries", map[string]any{
			"EntryId":       e.EntryID,
			"TransactionId": e.TransactionID,
			"AccountId":     e.AccountID,
			"AmountMinor":   e.AmountMinor,
			"Type":          string(e.Type),
			"CreatedAt":     spanner.CommitTimestamp,
		}))
	}

	return r.writeWithOutbox(ctx, emit, muts...)
}

func (r *SpannerRepository) GetAccountBalance(ctx context.Context, accountID string) (AccountBalance, error) {
	var actType, currency string
	stmtAcc := spanner.Statement{
		SQL:    `SELECT Type, Currency FROM LedgerAccounts WHERE AccountId = @act`,
		Params: map[string]any{"act": accountID},
	}
	iterAcc := r.client.Single().Query(ctx, stmtAcc)
	defer iterAcc.Stop()
	row, err := iterAcc.Next()
	if err == iterator.Done {
		return AccountBalance{}, fmt.Errorf("account not found")
	}
	if err != nil {
		return AccountBalance{}, err
	}
	if err := row.Columns(&actType, &currency); err != nil {
		return AccountBalance{}, err
	}

	stmt := spanner.Statement{
		SQL: `SELECT SUM(CASE WHEN Type = 'DEBIT' THEN AmountMinor ELSE 0 END) AS TotalDebit,
		             SUM(CASE WHEN Type = 'CREDIT' THEN AmountMinor ELSE 0 END) AS TotalCredit
		      FROM LedgerEntries
		      WHERE AccountId = @act`,
		Params: map[string]any{"act": accountID},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var totalDebit spanner.NullInt64
	var totalCredit spanner.NullInt64
	row, err = iter.Next()
	if err != nil && err != iterator.Done {
		return AccountBalance{}, err
	}
	if err == nil {
		if err := row.Columns(&totalDebit, &totalCredit); err != nil {
			return AccountBalance{}, err
		}
	}

	deb := int64(0)
	cred := int64(0)
	if totalDebit.Valid {
		deb = totalDebit.Int64
	}
	if totalCredit.Valid {
		cred = totalCredit.Int64
	}

	bal := AccountBalance{
		AccountID: accountID,
		Currency:  currency,
	}
	// Calculate balance based on normal balances.
	switch AccountType(actType) {
	case AccountTypeAsset, AccountTypeExpense:
		bal.Balance = deb - cred
	case AccountTypeLiability, AccountTypeEquity, AccountTypeRevenue:
		bal.Balance = cred - deb
	}

	return bal, nil
}

func (r *SpannerRepository) ListTransactions(ctx context.Context, accountID string, limit int) ([]Transaction, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	stmt := spanner.Statement{
		SQL: `SELECT t.TransactionId, t.ReferenceId, t.Description, t.CreatedAt
		      FROM LedgerTransactions t
		      JOIN LedgerEntries e ON t.TransactionId = e.TransactionId
		      WHERE e.AccountId = @act
		      ORDER BY t.CreatedAt DESC
		      LIMIT @lim`,
		Params: map[string]any{
			"act": accountID,
			"lim": int64(limit),
		},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var txns []Transaction
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var t Transaction
		if err := row.Columns(&t.TransactionID, &t.ReferenceID, &t.Description, &t.CreatedAt); err != nil {
			return nil, err
		}
		txns = append(txns, t)
	}
	return txns, nil
}
