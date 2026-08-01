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

// CapabilityPackRow is a persisted pack toggle.
type CapabilityPackRow struct {
	RetailerID      string
	PackID          string
	Enabled         bool
	EnabledByUserID string
	EnabledAt       *time.Time
	ConfigJSON      string
	UpdatedAt       time.Time
}

// LoadEnabledPacks returns the enabled set for a retailer (memory + Spanner).
func (s *Service) LoadEnabledPacks(ctx context.Context, retailerID string) (EnabledSet, error) {
	set := EnabledSet{}.WithCORE()
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.packsByRetailer != nil {
			for id, on := range s.packsByRetailer[retailerID] {
				if on {
					set[id] = true
				}
			}
		}
		return set.WithCORE(), nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT PackId, Enabled, IFNULL(ConfigJson, '') FROM RetailerCapabilityPacks
			WHERE RetailerId = @rid`,
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
			return set, err
		}
		var packID string
		var enabled bool
		var cfg string
		if err := row.Columns(&packID, &enabled, &cfg); err != nil {
			return set, err
		}
		if enabled {
			set[NormalizePackID(packID)] = true
		}
	}
	return set.WithCORE(), nil
}

// SetPackEnabled persists a pack enable/disable with optional config JSON.
func (s *Service) SetPackEnabled(ctx context.Context, retailerID, packID, actorUserID string, enabled bool, config map[string]any) error {
	packID = NormalizePackID(packID)
	cfgBytes, _ := json.Marshal(config)
	if cfgBytes == nil {
		cfgBytes = []byte("{}")
	}

	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.packsByRetailer == nil {
			s.packsByRetailer = map[string]map[string]bool{}
		}
		if s.packsByRetailer[retailerID] == nil {
			s.packsByRetailer[retailerID] = map[string]bool{}
		}
		if enabled {
			s.packsByRetailer[retailerID][packID] = true
		} else {
			delete(s.packsByRetailer[retailerID], packID)
		}
		return nil
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row := map[string]any{
			"RetailerId": retailerID,
			"PackId":     packID,
			"Enabled":    enabled,
			"ConfigJson": string(cfgBytes),
			"UpdatedAt":  spanner.CommitTimestamp,
		}
		if enabled {
			row["EnabledByUserId"] = actorUserID
			row["EnabledAt"] = spanner.CommitTimestamp
		}
		m := spanner.InsertOrUpdateMap("RetailerCapabilityPacks", row)
		if err := txn.BufferWrite([]*spanner.Mutation{m}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := map[string]any{
			"type":        events.EventRetailerCapabilityChanged,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": retailerID,
			"pack_id":     packID,
			"enabled":     enabled,
			"actor_user":  actorUserID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err != nil {
		return fmt.Errorf("set pack enabled: %w", err)
	}
	if s.cache != nil {
		s.cache.Invalidate(ctx, "retailer:packs:"+retailerID)
	}
	return nil
}

// LoadPackConfig returns config JSON object for a pack (or empty map).
func (s *Service) LoadPackConfig(ctx context.Context, retailerID, packID string) (map[string]any, error) {
	packID = NormalizePackID(packID)
	out := map[string]any{}
	if s.spannerClient == nil {
		return out, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerCapabilityPacks",
		spanner.Key{retailerID, packID}, []string{"ConfigJson"})
	if err != nil {
		if isNotFound(err) {
			return out, nil
		}
		return out, err
	}
	var cfg spanner.NullString
	if err := row.Columns(&cfg); err != nil {
		return out, err
	}
	if cfg.Valid && strings.TrimSpace(cfg.StringVal) != "" {
		_ = json.Unmarshal([]byte(cfg.StringVal), &out)
	}
	return out, nil
}
