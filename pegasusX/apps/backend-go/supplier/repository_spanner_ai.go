package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

type aiRecommendationData struct {
	Action       string                     `json:"action"`
	Confidence   float64                    `json:"confidence"`
	Source       string                     `json:"source"`
	Explanation  string                     `json:"explanation"`
	ReasonCodes  []string                   `json:"reason_codes"`
	Evidence     []AIRecommendationEvidence `json:"evidence"`
	Decision     string                     `json:"decision"`
	DecisionNote string                     `json:"decision_note"`
	DecidedBy    string                     `json:"decided_by"`
	DecidedAt    string                     `json:"decided_at"`
	ExpiresAt    string                     `json:"expires_at"`
}

// ListAIRecommendations returns recent supplier-scoped advisory outputs.
func (r *SpannerRepository) ListAIRecommendations(ctx context.Context, supplierID string, query AIRecommendationQuery) ([]AIRecommendation, error) {
	if r == nil || r.client == nil {
		return nil, fmt.Errorf("spanner supplier repository: nil client")
	}
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return []AIRecommendation{}, nil
	}
	limit := query.Limit
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	status := strings.ToUpper(strings.TrimSpace(query.Status))

	sql := `SELECT PredictionId, AggregateId, AggregateType, SupplierId, PredictionData, Score, Status, CreatedAt, UpdatedAt
	        FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_BySupplierCreated}
	        WHERE SupplierId = @supplier_id
	        ORDER BY CreatedAt DESC
	        LIMIT @limit`
	params := map[string]any{"supplier_id": supplierID, "limit": int64(limit)}
	if status != "" {
		sql = `SELECT PredictionId, AggregateId, AggregateType, SupplierId, PredictionData, Score, Status, CreatedAt, UpdatedAt
	           FROM AIPredictions@{FORCE_INDEX=Idx_AIPredictions_BySupplierStatusCreated}
	           WHERE SupplierId = @supplier_id AND Status = @status
	           ORDER BY CreatedAt DESC
	           LIMIT @limit`
		params["status"] = status
	}

	iter := r.client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	items := make([]AIRecommendation, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			return items, nil
		}
		if err != nil {
			return nil, fmt.Errorf("query ai recommendations: %w", err)
		}
		item, err := decodeAIRecommendationRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
}

// RecordAIRecommendationDecision persists one operator decision without applying operational side effects.
func (r *SpannerRepository) RecordAIRecommendationDecision(ctx context.Context, supplierID string, decision AIRecommendationDecision, emit func(outbox.TxnBuffer, AIRecommendation) error) (AIRecommendation, error) {
	if r == nil || r.client == nil {
		return AIRecommendation{}, fmt.Errorf("spanner supplier repository: nil client")
	}
	recommendationID := strings.TrimSpace(decision.RecommendationID)
	if recommendationID == "" || strings.TrimSpace(supplierID) == "" {
		return AIRecommendation{}, ErrAIRecommendationNotFound
	}
	decidedAt := decision.DecidedAt.UTC()
	if decidedAt.IsZero() {
		decidedAt = time.Now().UTC()
	}

	var updated AIRecommendation
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "AIPredictions", spanner.Key{recommendationID}, []string{
			"PredictionId", "AggregateId", "AggregateType", "SupplierId", "PredictionData", "Score", "Status", "CreatedAt", "UpdatedAt",
		})
		if err != nil {
			if errors.Is(err, spanner.ErrRowNotFound) {
				return ErrAIRecommendationNotFound
			}
			return fmt.Errorf("read ai recommendation %s: %w", recommendationID, err)
		}

		var (
			predictionID   string
			aggregateID    string
			aggregateType  string
			rowSupplierID  string
			predictionData []byte
			score          float64
			status         string
			createdAt      time.Time
			updatedAt      time.Time
		)
		if err := row.Columns(&predictionID, &aggregateID, &aggregateType, &rowSupplierID, &predictionData, &score, &status, &createdAt, &updatedAt); err != nil {
			return fmt.Errorf("scan ai recommendation %s: %w", recommendationID, err)
		}
		current := decodeAIRecommendationRecord(predictionID, aggregateID, aggregateType, rowSupplierID, predictionData, score, status, createdAt, updatedAt)
		if current.SupplierID != strings.TrimSpace(supplierID) {
			return ErrAIRecommendationNotFound
		}

		payload := map[string]any{}
		_ = json.Unmarshal(predictionData, &payload)
		payload = decodeAIRecommendationMap(current, payload)
		payload["decision"] = decision.Decision
		payload["decision_note"] = strings.TrimSpace(decision.Note)
		payload["decided_by"] = strings.TrimSpace(decision.DecidedBy)
		payload["decided_at"] = decidedAt.Format(time.RFC3339Nano)
		payload["previous_status"] = current.Status
		payload["decision_history"] = appendDecisionHistory(payload["decision_history"], decision, current.Status, decidedAt)

		encoded, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("encode ai recommendation decision %s: %w", recommendationID, err)
		}

		newStatus := statusForAIRecommendationDecision(decision.Decision)
		updated = decodeAIRecommendationRecord(
			current.RecommendationID,
			current.AggregateID,
			current.AggregateType,
			current.SupplierID,
			encoded,
			current.Score,
			newStatus,
			createdAt,
			decidedAt,
		)

		buf := &spannerTxnBuffer{}
		if emit != nil {
			if err := emit(buf, updated); err != nil {
				return err
			}
		}

		mutations := []*spanner.Mutation{spanner.InsertOrUpdateMap("AIPredictions", map[string]any{
			"PredictionId":   current.RecommendationID,
			"AggregateId":    current.AggregateID,
			"AggregateType":  current.AggregateType,
			"SupplierId":     current.SupplierID,
			"PredictionData": encoded,
			"Score":          current.Score,
			"Status":         newStatus,
			"CreatedAt":      createdAt,
			"UpdatedAt":      decidedAt,
		})}

		for _, event := range buf.events {
			if event.CreatedAt.IsZero() {
				event.CreatedAt = decidedAt
			}
			mutations = append(mutations, portalOutboxMutation(event))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return AIRecommendation{}, fmt.Errorf("record ai recommendation decision transaction: %w", err)
	}

	return updated, nil
}

func decodeAIRecommendationRow(row *spanner.Row) (AIRecommendation, error) {
	var (
		predictionID   string
		aggregateID    string
		aggregateType  string
		supplierID     string
		predictionData []byte
		score          float64
		status         string
		createdAt      time.Time
		updatedAt      time.Time
	)
	if err := row.Columns(&predictionID, &aggregateID, &aggregateType, &supplierID, &predictionData, &score, &status, &createdAt, &updatedAt); err != nil {
		return AIRecommendation{}, fmt.Errorf("scan ai recommendation: %w", err)
	}
	return decodeAIRecommendationRecord(predictionID, aggregateID, aggregateType, supplierID, predictionData, score, status, createdAt, updatedAt), nil
}

func decodeAIRecommendationRecord(predictionID string, aggregateID string, aggregateType string, supplierID string, predictionData []byte, score float64, status string, createdAt time.Time, updatedAt time.Time) AIRecommendation {
	var data aiRecommendationData
	_ = json.Unmarshal(predictionData, &data)
	if data.Action == "" {
		data.Action = "review_operational_signal"
	}
	if data.Source == "" {
		data.Source = "ai_worker"
	}
	if data.Confidence <= 0 {
		data.Confidence = score
	}
	if data.Explanation == "" {
		data.Explanation = "Review the advisory output before taking operational action."
	}
	if len(data.ReasonCodes) == 0 {
		data.ReasonCodes = []string{"advisory_output"}
	}
	if len(data.Evidence) == 0 {
		data.Evidence = []AIRecommendationEvidence{
			{Label: "Aggregate", Value: aggregateType + ":" + aggregateID},
			{Label: "Score", Value: strconvFormatFloat(score)},
		}
	}
	if strings.TrimSpace(status) == "" {
		status = "PENDING"
	}

	return AIRecommendation{
		RecommendationID: predictionID,
		SupplierID:       supplierID,
		AggregateID:      aggregateID,
		AggregateType:    aggregateType,
		Action:           data.Action,
		Status:           strings.ToUpper(strings.TrimSpace(status)),
		Score:            score,
		Confidence:       data.Confidence,
		Source:           data.Source,
		Explanation:      data.Explanation,
		ReasonCodes:      data.ReasonCodes,
		Evidence:         data.Evidence,
		Decision:         data.Decision,
		DecisionNote:     data.DecisionNote,
		DecidedBy:        data.DecidedBy,
		DecidedAt:        data.DecidedAt,
		ExpiresAt:        data.ExpiresAt,
		GeneratedAt:      createdAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt:        updatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func decodeAIRecommendationMap(current AIRecommendation, fallback map[string]any) map[string]any {
	if fallback == nil {
		fallback = map[string]any{}
	}
	fallback["action"] = current.Action
	fallback["confidence"] = current.Confidence
	fallback["source"] = current.Source
	fallback["explanation"] = current.Explanation
	fallback["reason_codes"] = current.ReasonCodes
	fallback["evidence"] = current.Evidence
	if current.ExpiresAt != "" {
		fallback["expires_at"] = current.ExpiresAt
	}
	return fallback
}

func appendDecisionHistory(raw any, decision AIRecommendationDecision, previousStatus string, decidedAt time.Time) []map[string]any {
	history := make([]map[string]any, 0)
	if existing, ok := raw.([]any); ok {
		for _, item := range existing {
			if object, ok := item.(map[string]any); ok {
				history = append(history, object)
			}
		}
	}
	history = append(history, map[string]any{
		"decision":        decision.Decision,
		"note":            strings.TrimSpace(decision.Note),
		"decided_by":      strings.TrimSpace(decision.DecidedBy),
		"decided_at":      decidedAt.Format(time.RFC3339Nano),
		"previous_status": previousStatus,
		"status":          statusForAIRecommendationDecision(decision.Decision),
	})
	return history
}

func strconvFormatFloat(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.4f", value), "0"), ".")
}
