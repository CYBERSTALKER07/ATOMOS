// Package payment — CardTokenService manages saved payment card tokens in Spanner.
// Tokens are per-retailer, per-gateway, and support soft-deletion only (never hard delete).
package payment

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"backend-go/hotspot"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// RetailerCardToken represents a saved card stored in RetailerCardTokens.
type RetailerCardToken struct {
	TokenID           string     `json:"token_id"`
	RetailerID        string     `json:"retailer_id"`
	Gateway           string     `json:"gateway"`
	ProviderCardToken string     `json:"-"` // Never expose the raw token to clients
	CardLast4         string     `json:"card_last4"`
	CardType          string     `json:"card_type"`
	IsDefault         bool       `json:"is_default"`
	IsActive          bool       `json:"is_active"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

// CardTokenService handles CRUD operations for saved retailer card tokens.
type CardTokenService struct {
	Spanner *spanner.Client
}

// SaveCard persists a new card token. If this is the retailer's first card for the
// given gateway, it is automatically set as the default.
func (s *CardTokenService) SaveCard(ctx context.Context, retailerID, gateway, providerCardToken, last4, cardType string) (string, error) {
	gateway = strings.ToUpper(gateway)
	tokenID := hotspot.NewOrderID()

	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Check if retailer already has cards for this gateway
		countStmt := spanner.Statement{
			SQL: `SELECT COUNT(*) FROM RetailerCardTokens
			      WHERE RetailerId = @rid AND Gateway = @gw AND IsActive = true`,
			Params: map[string]interface{}{"rid": retailerID, "gw": gateway},
		}
		countIter := txn.Query(ctx, countStmt)
		countRow, err := countIter.Next()
		countIter.Stop()
		if err != nil {
			return fmt.Errorf("count existing cards failed: %w", err)
		}
		var existingCount int64
		if err := countRow.Columns(&existingCount); err != nil {
			return fmt.Errorf("count column scan failed: %w", err)
		}

		isDefault := existingCount == 0 // First card → auto-default

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Insert("RetailerCardTokens",
				[]string{"TokenId", "RetailerId", "Gateway", "ProviderCardToken", "CardLast4", "CardType", "IsDefault", "IsActive", "CreatedAt"},
				[]interface{}{tokenID, retailerID, gateway, providerCardToken, last4, cardType, isDefault, true, spanner.CommitTimestamp},
			),
		})
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("save card failed: %w", err)
	}

	log.Printf("[CARD-TOKEN] Saved card %s for retailer %s (gateway=%s, last4=%s, default=%v)", tokenID, retailerID, gateway, last4, true)
	return tokenID, nil
}

// ListCards returns all active card tokens for a retailer, newest first.
func (s *CardTokenService) ListCards(ctx context.Context, retailerID string) ([]RetailerCardToken, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TokenId, RetailerId, Gateway, CardLast4, CardType, IsDefault, IsActive, ExpiresAt, CreatedAt
		      FROM RetailerCardTokens
		      WHERE RetailerId = @rid AND IsActive = true
		      ORDER BY CreatedAt DESC`,
		Params: map[string]interface{}{"rid": retailerID},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var tokens []RetailerCardToken
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("list cards query failed: %w", err)
		}
		var t RetailerCardToken
		var expiresAt spanner.NullTime
		if err := row.Columns(&t.TokenID, &t.RetailerID, &t.Gateway, &t.CardLast4, &t.CardType, &t.IsDefault, &t.IsActive, &expiresAt, &t.CreatedAt); err != nil {
			return nil, err
		}
		if expiresAt.Valid {
			t.ExpiresAt = &expiresAt.Time
		}
		tokens = append(tokens, t)
	}
	return tokens, nil
}

// GetDefaultCard returns the default active card for a retailer + gateway, or nil.
func (s *CardTokenService) GetDefaultCard(ctx context.Context, retailerID, gateway string) (*RetailerCardToken, error) {
	stmt := spanner.Statement{
		SQL: `SELECT TokenId, RetailerId, Gateway, ProviderCardToken, CardLast4, CardType, IsDefault, IsActive, ExpiresAt, CreatedAt
		      FROM RetailerCardTokens
		      WHERE RetailerId = @rid AND Gateway = @gw AND IsDefault = true AND IsActive = true
		      LIMIT 1`,
		Params: map[string]interface{}{"rid": retailerID, "gw": gateway},
	}
	iter := s.Spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // No default card — caller should fall back to hosted checkout
	}
	if err != nil {
		return nil, fmt.Errorf("get default card failed: %w", err)
	}

	var t RetailerCardToken
	var expiresAt spanner.NullTime
	if err := row.Columns(&t.TokenID, &t.RetailerID, &t.Gateway, &t.ProviderCardToken, &t.CardLast4, &t.CardType, &t.IsDefault, &t.IsActive, &expiresAt, &t.CreatedAt); err != nil {
		return nil, err
	}
	if expiresAt.Valid {
		t.ExpiresAt = &expiresAt.Time
	}
	return &t, nil
}

// DeactivateCard soft-deletes a card token. Ownership is verified via retailerID.
func (s *CardTokenService) DeactivateCard(ctx context.Context, tokenID, retailerID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Verify ownership
		row, err := txn.ReadRow(ctx, "RetailerCardTokens", spanner.Key{tokenID}, []string{"RetailerId", "IsActive"})
		if err != nil {
			return fmt.Errorf("card token %s not found: %w", tokenID, err)
		}
		var ownerID string
		var isActive bool
		if err := row.Columns(&ownerID, &isActive); err != nil {
			return err
		}
		if ownerID != retailerID {
			return fmt.Errorf("card token %s does not belong to retailer %s", tokenID, retailerID)
		}
		if !isActive {
			return nil // Already deactivated — idempotent
		}

		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("RetailerCardTokens",
				[]string{"TokenId", "IsActive", "IsDefault"},
				[]interface{}{tokenID, false, false},
			),
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("deactivate card failed: %w", err)
	}
	log.Printf("[CARD-TOKEN] Deactivated card %s for retailer %s", tokenID, retailerID)
	return nil
}

// SetDefaultCard marks a card as the default, unsetting any previous default for the
// same retailer + gateway combination. Ownership is verified.
func (s *CardTokenService) SetDefaultCard(ctx context.Context, tokenID, retailerID string) error {
	_, err := s.Spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read the target card
		row, err := txn.ReadRow(ctx, "RetailerCardTokens", spanner.Key{tokenID}, []string{"RetailerId", "Gateway", "IsActive"})
		if err != nil {
			return fmt.Errorf("card token %s not found: %w", tokenID, err)
		}
		var ownerID, gateway string
		var isActive bool
		if err := row.Columns(&ownerID, &gateway, &isActive); err != nil {
			return err
		}
		if ownerID != retailerID {
			return fmt.Errorf("card token %s does not belong to retailer %s", tokenID, retailerID)
		}
		if !isActive {
			return fmt.Errorf("card token %s is deactivated", tokenID)
		}

		// Unset previous defaults for this retailer + gateway
		clearStmt := spanner.Statement{
			SQL: `SELECT TokenId FROM RetailerCardTokens
			      WHERE RetailerId = @rid AND Gateway = @gw AND IsDefault = true AND IsActive = true`,
			Params: map[string]interface{}{"rid": retailerID, "gw": gateway},
		}
		clearIter := txn.Query(ctx, clearStmt)
		defer clearIter.Stop()
		for {
			clearRow, err := clearIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var prevTokenID string
			if err := clearRow.Columns(&prevTokenID); err != nil {
				return err
			}
			txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("RetailerCardTokens",
					[]string{"TokenId", "IsDefault"},
					[]interface{}{prevTokenID, false},
				),
			})
		}

		// Set new default
		txn.BufferWrite([]*spanner.Mutation{
			spanner.Update("RetailerCardTokens",
				[]string{"TokenId", "IsDefault"},
				[]interface{}{tokenID, true},
			),
		})
		return nil
	})
	if err != nil {
		return fmt.Errorf("set default card failed: %w", err)
	}
	log.Printf("[CARD-TOKEN] Set card %s as default for retailer %s", tokenID, retailerID)
	return nil
}
