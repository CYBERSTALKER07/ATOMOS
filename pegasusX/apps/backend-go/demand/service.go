package demand

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

type Service struct {
	spanner *spanner.Client
}

func NewService(spannerClient *spanner.Client) *Service {
	return &Service{
		spanner: spannerClient,
	}
}

func (s *Service) GetAdjustments(ctx context.Context, retailerId string, from, to time.Time) ([]DemandAdjustment, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT RetailerId, Sku, Date, BaseVelocity, Adjustment, AdjustedDemand, FactorsJson, ComputedAt
			FROM DemandAdjustments
			WHERE RetailerId = @RetailerId AND Date >= @From AND Date <= @To
		`,
		Params: map[string]interface{}{
			"RetailerId": retailerId,
			"From":       from.Format("2006-01-02"),
			"To":         to.Format("2006-01-02"),
		},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var adjs []DemandAdjustment
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var adj DemandAdjustment
		var date spanner.NullDate
		var factorsJson spanner.NullJSON
		if err := row.Columns(
			&adj.RetailerId,
			&adj.Sku,
			&date,
			&adj.BaseVelocity,
			&adj.Adjustment,
			&adj.AdjustedDemand,
			&factorsJson,
			&adj.ComputedAt,
		); err != nil {
			return nil, err
		}

		if date.Valid {
			adj.Date = time.Date(date.Date.Year, date.Date.Month, date.Date.Day, 0, 0, 0, 0, time.UTC)
		}
		if factorsJson.Valid {
			_ = json.Unmarshal([]byte(factorsJson.Value.(string)), &adj.Factors)
		}
		adjs = append(adjs, adj)
	}

	return adjs, nil
}

type SignalFilter struct {
	Type   *SignalType
	Scope  *string
	Active *bool
}

func (s *Service) ListSignals(ctx context.Context, filter SignalFilter) ([]DemandSignal, error) {
	query := `
		SELECT SignalId, Type, Scope, Sku, StartAt, EndAt, Multiplier, Meta, CreatedAt, CreatedBy
		FROM DemandSignals
		WHERE 1=1
	`
	params := map[string]interface{}{}

	if filter.Type != nil && *filter.Type != "" {
		query += " AND Type = @Type"
		params["Type"] = string(*filter.Type)
	}
	if filter.Scope != nil && *filter.Scope != "" {
		query += " AND Scope = @Scope"
		params["Scope"] = *filter.Scope
	}
	if filter.Active != nil && *filter.Active {
		query += " AND StartAt <= CURRENT_TIMESTAMP() AND EndAt >= CURRENT_TIMESTAMP()"
	}

	query += " ORDER BY CreatedAt DESC LIMIT 100"

	stmt := spanner.Statement{
		SQL:    query,
		Params: params,
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var signals []DemandSignal
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var sig DemandSignal
		var sku spanner.NullString
		var meta spanner.NullJSON
		if err := row.Columns(
			&sig.SignalId,
			&sig.Type,
			&sig.Scope,
			&sku,
			&sig.StartAt,
			&sig.EndAt,
			&sig.Multiplier,
			&meta,
			&sig.CreatedAt,
			&sig.CreatedBy,
		); err != nil {
			return nil, err
		}

		if sku.Valid {
			sig.Sku = &sku.StringVal
		}
		if meta.Valid {
			sig.Meta = []byte(meta.Value.(string))
		}
		signals = append(signals, sig)
	}
	return signals, nil
}

func (s *Service) GetSignal(ctx context.Context, id string) (*DemandSignal, error) {
	stmt := spanner.Statement{
		SQL: `
			SELECT SignalId, Type, Scope, Sku, StartAt, EndAt, Multiplier, Meta, CreatedAt, CreatedBy
			FROM DemandSignals
			WHERE SignalId = @Id
		`,
		Params: map[string]interface{}{"Id": id},
	}
	iter := s.spanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // Not found
	}
	if err != nil {
		return nil, err
	}

	var sig DemandSignal
	var sku spanner.NullString
	var meta spanner.NullJSON
	if err := row.Columns(
		&sig.SignalId,
		&sig.Type,
		&sig.Scope,
		&sku,
		&sig.StartAt,
		&sig.EndAt,
		&sig.Multiplier,
		&meta,
		&sig.CreatedAt,
		&sig.CreatedBy,
	); err != nil {
		return nil, err
	}

	if sku.Valid {
		sig.Sku = &sku.StringVal
	}
	if meta.Valid {
		sig.Meta = []byte(meta.Value.(string))
	}
	return &sig, nil
}

type CreateSignalRequest struct {
	Type       SignalType      `json:"type"`
	Scope      string          `json:"scope"`
	Sku        *string         `json:"sku,omitempty"`
	StartAt    time.Time       `json:"startAt"`
	EndAt      time.Time       `json:"endAt"`
	Multiplier float64         `json:"multiplier"`
	Meta       json.RawMessage `json:"meta,omitempty"`
}

func (s *Service) CreateSignal(ctx context.Context, req CreateSignalRequest, createdBy string) (*DemandSignal, error) {
	sig := DemandSignal{
		SignalId:   uuid.New().String(),
		Type:       req.Type,
		Scope:      req.Scope,
		Sku:        req.Sku,
		StartAt:    req.StartAt,
		EndAt:      req.EndAt,
		Multiplier: req.Multiplier,
		Meta:       req.Meta,
		CreatedAt:  time.Now().UTC(),
		CreatedBy:  createdBy,
	}

	var sku spanner.NullString
	if sig.Sku != nil {
		sku = spanner.NullString{StringVal: *sig.Sku, Valid: true}
	}
	var meta spanner.NullJSON
	if len(sig.Meta) > 0 {
		meta = spanner.NullJSON{Value: string(sig.Meta), Valid: true}
	}

	m := spanner.Insert("DemandSignals",
		[]string{"SignalId", "Type", "Scope", "Sku", "StartAt", "EndAt", "Multiplier", "Meta", "CreatedAt", "CreatedBy"},
		[]interface{}{sig.SignalId, string(sig.Type), sig.Scope, sku, sig.StartAt, sig.EndAt, sig.Multiplier, meta, sig.CreatedAt, sig.CreatedBy},
	)

	_, err := s.spanner.Apply(ctx, []*spanner.Mutation{m})
	if err != nil {
		return nil, err
	}
	return &sig, nil
}

func (s *Service) UpdateSignal(ctx context.Context, sig *DemandSignal) error {
	var sku spanner.NullString
	if sig.Sku != nil {
		sku = spanner.NullString{StringVal: *sig.Sku, Valid: true}
	}
	var meta spanner.NullJSON
	if len(sig.Meta) > 0 {
		meta = spanner.NullJSON{Value: string(sig.Meta), Valid: true}
	}

	m := spanner.Update("DemandSignals",
		[]string{"SignalId", "Type", "Scope", "Sku", "StartAt", "EndAt", "Multiplier", "Meta"},
		[]interface{}{sig.SignalId, string(sig.Type), sig.Scope, sku, sig.StartAt, sig.EndAt, sig.Multiplier, meta},
	)

	_, err := s.spanner.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (s *Service) DeactivateSignal(ctx context.Context, id, actor string) error {
	// Set EndAt to now
	m := spanner.Update("DemandSignals",
		[]string{"SignalId", "EndAt"},
		[]interface{}{id, time.Now().UTC()},
	)
	_, err := s.spanner.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// Helpers for reading HTTP requests
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
