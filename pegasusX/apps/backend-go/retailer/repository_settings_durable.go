package retailer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// loadAutoOrderDurable reads Spanner settings; falls back to memory on miss/nil client.
func (s *Service) loadAutoOrderDurable(ctx context.Context, retailerID string) AutoOrderSettings {
	if s.spannerClient != nil {
		row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerAutoOrderSettings",
			spanner.Key{retailerID}, []string{"SettingsJson"})
		if err == nil {
			var raw string
			if err := row.Columns(&raw); err == nil && strings.TrimSpace(raw) != "" {
				var settings AutoOrderSettings
				if json.Unmarshal([]byte(raw), &settings) == nil {
					// keep memory warm
					s.saveAutoOrderSettings(retailerID, settings)
					return cloneAutoOrderSettings(settings)
				}
			}
		} else if !isNotFound(err) {
			s.log.Warn("auto-order spanner read failed", "retailer_id", retailerID, "err", err)
		}
	}
	return s.getAutoOrderSettings(retailerID)
}

func (s *Service) saveAutoOrderDurable(ctx context.Context, retailerID, actorUserID string, settings AutoOrderSettings) error {
	// Always update memory cache for process-local hot path.
	s.saveAutoOrderSettings(retailerID, settings)

	if s.spannerClient == nil {
		return nil
	}
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("RetailerAutoOrderSettings", map[string]any{
			"RetailerId":      retailerID,
			"SettingsJson":    string(raw),
			"UpdatedByUserId": actorUserID,
			"UpdatedAt":       spanner.CommitTimestamp,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":        events.EventRetailerAutoOrderUpdated,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": retailerID,
			"actor_user":  actorUserID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err != nil {
		return fmt.Errorf("persist auto-order: %w", err)
	}
	return nil
}

func (s *Service) loadFavoriteSuppliersDurable(ctx context.Context, retailerID string) (map[string]bool, error) {
	prefs := map[string]bool{}
	if s.spannerClient != nil {
		stmt := spanner.Statement{
			SQL:    `SELECT SupplierId, IsFavorite FROM RetailerFavoriteSuppliers WHERE RetailerId = @rid`,
			Params: map[string]any{"rid": retailerID},
		}
		iter := s.spannerClient.Single().Query(ctx, stmt)
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return prefs, err
			}
			var sid string
			var fav bool
			if err := row.Columns(&sid, &fav); err != nil {
				return prefs, err
			}
			if fav {
				prefs[sid] = true
			}
		}
		// seed memory
		s.mu.Lock()
		if s.favoriteSuppliers == nil {
			s.favoriteSuppliers = map[string]map[string]bool{}
		}
		s.favoriteSuppliers[retailerID] = prefs
		s.mu.Unlock()
		return prefs, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	src := s.favoriteSuppliers[retailerID]
	for k, v := range src {
		prefs[k] = v
	}
	return prefs, nil
}

func (s *Service) setFavoriteSupplierDurable(ctx context.Context, retailerID, supplierID string, favorite bool) error {
	// memory
	s.mu.Lock()
	if s.favoriteSuppliers == nil {
		s.favoriteSuppliers = map[string]map[string]bool{}
	}
	if s.favoriteSuppliers[retailerID] == nil {
		s.favoriteSuppliers[retailerID] = map[string]bool{}
	}
	if favorite {
		s.favoriteSuppliers[retailerID][supplierID] = true
	} else {
		delete(s.favoriteSuppliers[retailerID], supplierID)
	}
	s.mu.Unlock()

	if s.spannerClient == nil {
		return nil
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		m := spanner.InsertOrUpdateMap("RetailerFavoriteSuppliers", map[string]any{
			"RetailerId": retailerID,
			"SupplierId": supplierID,
			"IsFavorite": favorite,
			"UpdatedAt":  spanner.CommitTimestamp,
		})
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}
