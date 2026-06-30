package promotion

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Repository defines Spanner access for supplier promotions.
type Repository interface {
	ListActiveBySupplier(ctx context.Context, supplierID string, now time.Time) ([]Promotion, error)
	ListBySupplier(ctx context.Context, supplierID string) ([]Promotion, error)
	GetByID(ctx context.Context, supplierID, promotionID string) (Promotion, bool, error)
	Upsert(ctx context.Context, p Promotion, emit func(*spannerTxnBuffer) error) error
	SetActive(ctx context.Context, supplierID, promotionID string, isActive bool, version int64, emit func(*spannerTxnBuffer) error) error
	LookupProductCategories(ctx context.Context, productIDs []string) (map[string]string, error)
	LookupListPrices(ctx context.Context, productIDs []string) (map[string]int64, error)
	LookupActivePriceOverrides(ctx context.Context, supplierID, retailerID string, productIDs []string, now time.Time) (map[string]int64, error)
	RedeemPromotion(ctx context.Context, promotionID string) error
}

// SpannerRepository implements Repository.
type SpannerRepository struct {
	client *spanner.Client
}

// NewSpannerRepository creates a Spanner-backed promotion repository.
func NewSpannerRepository(client *spanner.Client) *SpannerRepository {
	return &SpannerRepository{client: client}
}

// ListActiveBySupplier returns promotions that may apply at now.
func (r *SpannerRepository) ListActiveBySupplier(ctx context.Context, supplierID string, now time.Time) ([]Promotion, error) {
	stmt := spanner.Statement{
		SQL: `SELECT PromotionId, SupplierId, Name, Description, DiscountBps, ScopeType,
			ScopeProductId, ScopeCategoryId, RetailerScope, RetailerIdsJson,
			MinLineQuantity, MinOrderAmountMinor, StartsAt, EndsAt, MaxRedemptions, CurrentRedemptions, IsActive, Priority, Version, CreatedAt, UpdatedAt
			FROM SupplierPromotions
			WHERE SupplierId = @sid AND IsActive = TRUE
			AND (StartsAt IS NULL OR StartsAt <= @now)
			AND (EndsAt IS NULL OR EndsAt > @now)
			AND (MaxRedemptions IS NULL OR MaxRedemptions = 0 OR CurrentRedemptions < MaxRedemptions)
			ORDER BY Priority DESC, UpdatedAt DESC`,
		Params: map[string]any{"sid": supplierID, "now": now},
	}
	return r.queryPromotions(ctx, stmt)
}

// ListBySupplier returns all promotions for supplier admin surfaces.
func (r *SpannerRepository) ListBySupplier(ctx context.Context, supplierID string) ([]Promotion, error) {
	stmt := spanner.Statement{
		SQL: `SELECT PromotionId, SupplierId, Name, Description, DiscountBps, ScopeType,
			ScopeProductId, ScopeCategoryId, RetailerScope, RetailerIdsJson,
			MinLineQuantity, MinOrderAmountMinor, StartsAt, EndsAt, MaxRedemptions, CurrentRedemptions, IsActive, Priority, Version, CreatedAt, UpdatedAt
			FROM SupplierPromotions
			WHERE SupplierId = @sid
			ORDER BY UpdatedAt DESC`,
		Params: map[string]any{"sid": supplierID},
	}
	return r.queryPromotions(ctx, stmt)
}

// GetByID reads one promotion scoped to supplier.
func (r *SpannerRepository) GetByID(ctx context.Context, supplierID, promotionID string) (Promotion, bool, error) {
	row, err := r.client.Single().ReadRow(ctx, "SupplierPromotions", spanner.Key{promotionID},
		[]string{"PromotionId", "SupplierId", "Name", "Description", "DiscountBps", "ScopeType",
			"ScopeProductId", "ScopeCategoryId", "RetailerScope", "RetailerIdsJson",
			"MinLineQuantity", "MinOrderAmountMinor", "StartsAt", "EndsAt", "MaxRedemptions", "CurrentRedemptions", "IsActive", "Priority", "Version", "CreatedAt", "UpdatedAt"})
	if err != nil {
		if errors.Is(err, spanner.ErrRowNotFound) {
			return Promotion{}, false, nil
		}
		return Promotion{}, false, fmt.Errorf("read promotion %s: %w", promotionID, err)
	}
	p, err := scanPromotionRow(row)
	if err != nil {
		return Promotion{}, false, err
	}
	if p.SupplierID != supplierID {
		return Promotion{}, false, nil
	}
	return p, true, nil
}

func appendOutboxMutations(buf *spannerTxnBuffer) []*spanner.Mutation {
	if buf == nil || len(buf.events) == 0 {
		return nil
	}
	mutations := make([]*spanner.Mutation, 0, len(buf.events))
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
	return mutations
}

// Upsert inserts or updates a promotion row.
func (r *SpannerRepository) Upsert(ctx context.Context, p Promotion, emit func(*spannerTxnBuffer) error) error {
	retailerJSON, err := json.Marshal(p.RetailerIDs)
	if err != nil {
		return fmt.Errorf("marshal retailer ids: %w", err)
	}
	_, err = r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		m := spanner.InsertOrUpdateMap("SupplierPromotions", map[string]any{
			"PromotionId":         p.PromotionID,
			"SupplierId":          p.SupplierID,
			"Name":                p.Name,
			"Description":         spanner.NullString{StringVal: p.Description, Valid: p.Description != ""},
			"DiscountBps":         p.DiscountBps,
			"ScopeType":           string(p.ScopeType),
			"ScopeProductId":      nullString(p.ScopeProductID),
			"ScopeCategoryId":     nullString(p.ScopeCategoryID),
			"RetailerScope":       string(p.RetailerScope),
			"RetailerIdsJson":     retailerJSON,
			"MinLineQuantity":     nullInt64(p.MinLineQuantity),
			"MinOrderAmountMinor": nullInt64(p.MinOrderAmountMinor),
			"StartsAt":            nullTime(p.StartsAt),
			"EndsAt":              nullTime(p.EndsAt),
			"MaxRedemptions":      p.MaxRedemptions,
			"CurrentRedemptions":  p.CurrentRedemptions,
			"IsActive":            p.IsActive,
			"Priority":            p.Priority,
			"Version":             p.Version,
			"CreatedAt":           spanner.CommitTimestamp,
			"UpdatedAt":           spanner.CommitTimestamp,
		})
		mutations := append([]*spanner.Mutation{m}, appendOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("upsert promotion %s: %w", p.PromotionID, err)
	}
	return nil
}

// SetActive toggles promotion active flag with optimistic version check.
func (r *SpannerRepository) SetActive(ctx context.Context, supplierID, promotionID string, isActive bool, version int64, emit func(*spannerTxnBuffer) error) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierPromotions", spanner.Key{promotionID}, []string{"SupplierId", "Version"})
		if readErr != nil {
			return readErr
		}
		var sid string
		var currentVersion int64
		if err := row.Columns(&sid, &currentVersion); err != nil {
			return err
		}
		if sid != supplierID {
			return fmt.Errorf("promotion scope violation")
		}
		if version > 0 && version != currentVersion {
			return fmt.Errorf("version conflict")
		}
		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf); err != nil {
				return err
			}
		}
		m := spanner.UpdateMap("SupplierPromotions", map[string]any{
			"PromotionId": promotionID,
			"IsActive":    isActive,
			"Version":     currentVersion + 1,
			"UpdatedAt":   spanner.CommitTimestamp,
		})
		mutations := append([]*spanner.Mutation{m}, appendOutboxMutations(buf)...)
		return txn.BufferWrite(mutations)
	})
	return err
}

// RedeemPromotion atomically increments CurrentRedemptions if MaxRedemptions allows it.
func (r *SpannerRepository) RedeemPromotion(ctx context.Context, promotionID string) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, readErr := txn.ReadRow(ctx, "SupplierPromotions", spanner.Key{promotionID}, []string{"MaxRedemptions", "CurrentRedemptions"})
		if readErr != nil {
			return readErr
		}
		var maxRedemptions, currentRedemptions spanner.NullInt64
		if err := row.Columns(&maxRedemptions, &currentRedemptions); err != nil {
			return err
		}
		if maxRedemptions.Valid && maxRedemptions.Int64 > 0 {
			if currentRedemptions.Int64 >= maxRedemptions.Int64 {
				return fmt.Errorf("promotion redemption limit reached")
			}
		}
		m := spanner.UpdateMap("SupplierPromotions", map[string]any{
			"PromotionId":        promotionID,
			"CurrentRedemptions": currentRedemptions.Int64 + 1,
			"UpdatedAt":          spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

func (r *SpannerRepository) queryPromotions(ctx context.Context, stmt spanner.Statement) ([]Promotion, error) {
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	var out []Promotion
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		p, err := scanPromotionRow(row)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, nil
}

func scanPromotionRow(row *spanner.Row) (Promotion, error) {
	var p Promotion
	var desc spanner.NullString
	var scopeProduct, scopeCategory spanner.NullString
	var retailerScope string
	var retailerJSON []byte
	var minQty, minOrder spanner.NullInt64
	var startsAt, endsAt spanner.NullTime
	var maxRedemptions, currentRedemptions spanner.NullInt64
	if err := row.Columns(
		&p.PromotionID, &p.SupplierID, &p.Name, &desc, &p.DiscountBps, &p.ScopeType,
		&scopeProduct, &scopeCategory, &retailerScope, &retailerJSON,
		&minQty, &minOrder, &startsAt, &endsAt, &maxRedemptions, &currentRedemptions, &p.IsActive, &p.Priority, &p.Version, &p.CreatedAt, &p.UpdatedAt,
	); err != nil {
		return Promotion{}, err
	}
	p.Description = desc.StringVal
	p.ScopeProductID = scopeProduct.StringVal
	p.ScopeCategoryID = scopeCategory.StringVal
	p.RetailerScope = RetailerScope(retailerScope)
	if len(retailerJSON) > 0 {
		_ = json.Unmarshal(retailerJSON, &p.RetailerIDs)
	}
	if minQty.Valid {
		p.MinLineQuantity = minQty.Int64
	}
	if minOrder.Valid {
		p.MinOrderAmountMinor = minOrder.Int64
	}
	if startsAt.Valid {
		t := startsAt.Time
		p.StartsAt = &t
	}
	if endsAt.Valid {
		t := endsAt.Time
		p.EndsAt = &t
	}
	p.MaxRedemptions = maxRedemptions.Int64
	p.CurrentRedemptions = currentRedemptions.Int64
	p.ScopeType = ScopeType(strings.TrimSpace(string(p.ScopeType)))
	return p, nil
}

func nullString(v string) spanner.NullString {
	v = strings.TrimSpace(v)
	return spanner.NullString{StringVal: v, Valid: v != ""}
}

func nullInt64(v int64) spanner.NullInt64 {
	return spanner.NullInt64{Int64: v, Valid: v > 0}
}

func nullTime(v *time.Time) spanner.NullTime {
	if v == nil || v.IsZero() {
		return spanner.NullTime{}
	}
	return spanner.NullTime{Time: v.UTC(), Valid: true}
}

// LookupProductCategories resolves category ids for product ids.
func (r *SpannerRepository) LookupProductCategories(ctx context.Context, productIDs []string) (map[string]string, error) {
	if len(productIDs) == 0 {
		return map[string]string{}, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT ProductId, CategoryId FROM Products WHERE ProductId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": productIDs},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	out := make(map[string]string, len(productIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID, categoryID string
		if err := row.Columns(&productID, &categoryID); err != nil {
			return nil, err
		}
		out[productID] = categoryID
	}
	return out, nil
}

// LookupListPrices resolves canonical list prices for product ids.
func (r *SpannerRepository) LookupListPrices(ctx context.Context, productIDs []string) (map[string]int64, error) {
	if len(productIDs) == 0 {
		return map[string]int64{}, nil
	}
	stmt := spanner.Statement{
		SQL:    `SELECT ProductId, PriceMinor FROM Products WHERE ProductId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": productIDs},
	}
	iter := r.client.Single().WithTimestampBound(spanner.ExactStaleness(15 * time.Second)).Query(ctx, stmt)
	defer iter.Stop()
	out := make(map[string]int64, len(productIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var productID string
		var price int64
		if err := row.Columns(&productID, &price); err != nil {
			return nil, err
		}
		out[productID] = price
	}
	return out, nil
}

// LookupActivePriceOverrides returns effective override prices keyed by product id.
func (r *SpannerRepository) LookupActivePriceOverrides(
	ctx context.Context,
	supplierID, retailerID string,
	productIDs []string,
	now time.Time,
) (map[string]int64, error) {
	if r == nil || r.client == nil || supplierID == "" || retailerID == "" || len(productIDs) == 0 {
		return map[string]int64{}, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT ProductId, OverridePrice, ExpiresAt
		      FROM RetailerPricingOverrides
		      WHERE SupplierId = @sid AND RetailerId = @rid AND ProductId IN UNNEST(@ids)
		        AND IsActive = true`,
		Params: map[string]any{"sid": supplierID, "rid": retailerID, "ids": productIDs},
	}
	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()
	out := make(map[string]int64, len(productIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("lookup retailer price overrides: %w", err)
		}
		var productID string
		var price int64
		var expiresAt spanner.NullTime
		if err := row.Columns(&productID, &price, &expiresAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid && !expiresAt.Time.After(now) {
			continue
		}
		out[productID] = price
	}
	return out, nil
}
