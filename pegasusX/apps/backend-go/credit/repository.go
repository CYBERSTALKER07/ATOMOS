package credit

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

// Repository persists retailer credit profiles.
type Repository interface {
	GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error)
	UpsertProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error
	AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error
}

// SpannerRepository is a Spanner-backed credit profile repository.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository builds a Spanner-backed credit profile repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

const profileSelectColumns = `RetailerId, SupplierId, CreditLimitMinor, CurrentBalanceMinor, AvailableCreditMinor, RiskScore, DelinquencyCount, Status, LastEvaluatedAt, Version, CreatedAt, UpdatedAt`

// GetProfile reads one credit profile by retailer + supplier.
func (r *SpannerRepository) GetProfile(ctx context.Context, retailerID, supplierID string) (Profile, bool, error) {
	if r == nil || r.client == nil {
		return Profile{}, false, fmt.Errorf("spanner credit repository: nil client")
	}
	stmt := spanner.Statement{
		SQL:    "SELECT " + profileSelectColumns + " FROM RetailerCreditProfiles WHERE RetailerId = @rid AND SupplierId = @sid",
		Params: map[string]any{"rid": retailerID, "sid": supplierID},
	}
	row, err := r.client.Single().Query(ctx, stmt).Next()
	if err != nil {
		if err == iterator.Done {
			return Profile{}, false, nil
		}
		return Profile{}, false, fmt.Errorf("get credit profile %s/%s: %w", retailerID, supplierID, err)
	}
	p, err := scanProfileRow(row)
	if err != nil {
		return Profile{}, false, err
	}
	return p, true, nil
}

// UpsertProfile creates or updates a credit profile atomically with an optional outbox event.
func (r *SpannerRepository) UpsertProfile(ctx context.Context, p Profile, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	p.AvailableCreditMinor = p.CreditLimitMinor - p.CurrentBalanceMinor
	if p.AvailableCreditMinor < 0 {
		p.AvailableCreditMinor = 0
	}
	if p.Status == "" {
		p.Status = StatusActive
	}
	if !p.Status.Valid() {
		return fmt.Errorf("invalid credit profile status: %s", p.Status)
	}
	if !p.RiskTier.Valid() {
		p.RiskTier = RiskTierMedium
	}

	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{p.RetailerID, p.SupplierID}, []string{"Version"})
		var expectedVersion int64
		if err == nil {
			if err := row.Columns(&expectedVersion); err != nil {
				return err
			}
			if p.Version != 0 && expectedVersion != p.Version {
				return fmt.Errorf("optimistic concurrency conflict: expected %d, got %d", expectedVersion, p.Version)
			}
			p.Version = expectedVersion + 1
		} else if spanner.ErrCode(err) != 5 {
			return err
		} else {
			p.Version = 1
			p.CreatedAt = p.UpdatedAt
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           p.RetailerID,
				"SupplierId":           p.SupplierID,
				"CreditLimitMinor":     p.CreditLimitMinor,
				"CurrentBalanceMinor":  p.CurrentBalanceMinor,
				"AvailableCreditMinor": p.AvailableCreditMinor,
				"RiskScore":            p.RiskScore,
				"DelinquencyCount":     p.DelinquencyCount,
				"Status":               string(p.Status),
				"LastEvaluatedAt":      p.LastEvaluatedAt,
				"Version":              p.Version,
				"CreatedAt":            p.CreatedAt,
				"UpdatedAt":            p.UpdatedAt,
			}),
		}
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}))
		}
		return txn.BufferWrite(mutations)
	})
}

// AdjustBalance atomically adds deltaMinor to the current balance and recomputes available credit.
func (r *SpannerRepository) AdjustBalance(ctx context.Context, retailerID, supplierID string, deltaMinor int64, emit func(outbox.TxnBuffer) error) error {
	if r == nil || r.client == nil {
		return fmt.Errorf("spanner credit repository: nil client")
	}
	return spannerutils.RunReadWriteTransaction(ctx, r.client, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "RetailerCreditProfiles", spanner.Key{retailerID, supplierID}, []string{"CreditLimitMinor", "CurrentBalanceMinor", "Version"})
		if err != nil {
			if spanner.ErrCode(err) == 5 {
				return ErrProfileNotFound
			}
			return err
		}
		var limit, balance, version int64
		if err := row.Columns(&limit, &balance, &version); err != nil {
			return err
		}
		newBalance := balance + deltaMinor
		if newBalance < 0 {
			newBalance = 0
		}
		available := limit - newBalance
		if available < 0 {
			available = 0
		}

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("RetailerCreditProfiles", map[string]any{
				"RetailerId":           retailerID,
				"SupplierId":           supplierID,
				"CurrentBalanceMinor":  newBalance,
				"AvailableCreditMinor": available,
				"Version":              version + 1,
				"UpdatedAt":            spanner.CommitTimestamp,
			}),
		}
		for _, e := range buf.events {
			createdAt := e.CreatedAt.UTC()
			if createdAt.IsZero() {
				createdAt = time.Now().UTC()
			}
			mutations = append(mutations, spanner.InsertOrUpdateMap("OutboxEvents", map[string]any{
				"EventId":       e.EventID,
				"AggregateType": e.AggregateType,
				"AggregateId":   e.AggregateID,
				"TopicName":     e.TopicName,
				"Payload":       e.Payload,
				"CreatedAt":     createdAt,
				"PublishedAt":   nil,
			}))
		}
		return txn.BufferWrite(mutations)
	})
}

func scanProfileRow(row *spanner.Row) (Profile, error) {
	var p Profile
	var status, riskTier spanner.NullString
	var lastEvaluated spanner.NullTime
	if err := row.Columns(&p.RetailerID, &p.SupplierID, &p.CreditLimitMinor, &p.CurrentBalanceMinor, &p.AvailableCreditMinor,
		&p.RiskScore, &p.DelinquencyCount, &status, &lastEvaluated, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return Profile{}, fmt.Errorf("scan credit profile row: %w", err)
	}
	p.Status = Status(status.StringVal)
	p.RiskTier = RiskTier(riskTier.StringVal)
	if lastEvaluated.Valid {
		p.LastEvaluatedAt = lastEvaluated.Time
	}
	return p, nil
}

type spannerTxnBuffer struct {
	events []outbox.Event
}

func (b *spannerTxnBuffer) BufferOutbox(_ context.Context, e outbox.Event) error {
	b.events = append(b.events, e)
	return nil
}
