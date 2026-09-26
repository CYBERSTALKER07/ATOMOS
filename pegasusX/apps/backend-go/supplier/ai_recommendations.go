package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ErrAIRecommendationNotFound signals that a recommendation id is absent or outside supplier scope.
var ErrAIRecommendationNotFound = errors.New("ai_recommendation_not_found")

// AIRecommendationEvidence is one bounded explanation data point shown to operators.
type AIRecommendationEvidence struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// AIRecommendation is the supplier-facing projection of one advisory output.
type AIRecommendation struct {
	RecommendationID string                     `json:"recommendation_id"`
	SupplierID       string                     `json:"supplier_id"`
	AggregateID      string                     `json:"aggregate_id"`
	AggregateType    string                     `json:"aggregate_type"`
	Action           string                     `json:"action"`
	Status           string                     `json:"status"`
	Score            float64                    `json:"score"`
	Confidence       float64                    `json:"confidence"`
	Source           string                     `json:"source"`
	Explanation      string                     `json:"explanation"`
	ReasonCodes      []string                   `json:"reason_codes"`
	Evidence         []AIRecommendationEvidence `json:"evidence"`
	Decision         string                     `json:"decision,omitempty"`
	DecisionNote     string                     `json:"decision_note,omitempty"`
	DecidedBy        string                     `json:"decided_by,omitempty"`
	DecidedAt        string                     `json:"decided_at,omitempty"`
	ExpiresAt        string                     `json:"expires_at,omitempty"`
	GeneratedAt      string                     `json:"generated_at"`
	UpdatedAt        string                     `json:"updated_at"`
}

// AIRecommendationQuery bounds supplier recommendation reads.
type AIRecommendationQuery struct {
	Status string
	Limit  int
}

// AIRecommendationDecision captures one human review or override action.
type AIRecommendationDecision struct {
	RecommendationID string
	Decision         string
	Note             string
	DecidedBy        string
	DecidedAt        time.Time
}

type aiRecommendationRepository interface {
	ListAIRecommendations(ctx context.Context, supplierID string, query AIRecommendationQuery) ([]AIRecommendation, error)
	RecordAIRecommendationDecision(ctx context.Context, supplierID string, decision AIRecommendationDecision, emit func(outbox.TxnBuffer, AIRecommendation) error) (AIRecommendation, error)
}

type supplierAIRecommendationsResponse struct {
	SupplierID string             `json:"supplier_id"`
	Items      []AIRecommendation `json:"items"`
	Count      int                `json:"count"`
	Limit      int                `json:"limit"`
	Status     string             `json:"status,omitempty"`
	UpdatedAt  string             `json:"updated_at"`
}

type aiRecommendationDecisionRequest struct {
	RecommendationID string `json:"recommendation_id"`
	Decision         string `json:"decision"`
	Note             string `json:"note,omitempty"`
}

type aiRecommendationDecisionResponse struct {
	Status         string           `json:"status"`
	Recommendation AIRecommendation `json:"recommendation"`
}

// HandleAIRecommendations supports supplier review and override authority for advisory outputs.
func (s *Service) HandleAIRecommendations(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.handleAIRecommendationsGet(w, r)
	case http.MethodPost:
		s.handleAIRecommendationsPost(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

func (s *Service) handleAIRecommendationsGet(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.repo.(aiRecommendationRepository)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ai_recommendations_unavailable"})
		return
	}

	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant_required"})
		return
	}

	query := AIRecommendationQuery{
		Status: strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("status"))),
		Limit:  parseRecommendationLimit(r.URL.Query().Get("limit")),
	}
	items, err := repo.ListAIRecommendations(r.Context(), supplierID, query)
	if err != nil {
		s.log.Warn("supplier ai recommendations load failed", "supplier_id", supplierID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_ai_recommendations_failed"})
		return
	}

	writeJSON(w, http.StatusOK, supplierAIRecommendationsResponse{
		SupplierID: supplierID,
		Items:      items,
		Count:      len(items),
		Limit:      query.Limit,
		Status:     query.Status,
		UpdatedAt:  s.now().Format(time.RFC3339Nano),
	})
}

func (s *Service) handleAIRecommendationsPost(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.repo.(aiRecommendationRepository)
	if !ok {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "ai_recommendations_unavailable"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "tenant_required"})
		return
	}
	body, bodyOK := readMutationBody(w, r, 32*1024)
	if !bodyOK {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}

	var req aiRecommendationDecisionRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	decision := AIRecommendationDecision{
		RecommendationID: strings.TrimSpace(req.RecommendationID),
		Decision:         normalizeAIRecommendationDecision(req.Decision),
		Note:             strings.TrimSpace(req.Note),
		DecidedBy:        aiRecommendationActor(r.Context()),
		DecidedAt:        s.now(),
	}
	if decision.RecommendationID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "recommendation_id_required"})
		return
	}
	if decision.Decision == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_ai_recommendation_decision"})
		return
	}

	recommendation, err := repo.RecordAIRecommendationDecision(r.Context(), supplierID, decision, func(txn outbox.TxnBuffer, recommendation AIRecommendation) error {
		return outbox.EmitJSON(r.Context(), txn, events.AggregateAIRecommendation, decision.RecommendationID, events.TopicMain, events.AIRecommendationEvent{
			BaseEvent:        events.BaseEvent{Type: events.EventAIRecommendationDecided, Timestamp: decision.DecidedAt.Format(time.RFC3339Nano)},
			SupplierID:       supplierID,
			RecommendationID: decision.RecommendationID,
			AggregateID:      recommendation.AggregateID,
			AggregateType:    recommendation.AggregateType,
			Decision:         decision.Decision,
			Status:           recommendation.Status,
			DecidedBy:        decision.DecidedBy,
			Note:             decision.Note,
		})
	})
	if err != nil {
		if errors.Is(err, ErrAIRecommendationNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "ai_recommendation_not_found"})
			return
		}
		s.log.Warn("supplier ai recommendation decision failed", "supplier_id", supplierID, "recommendation_id", decision.RecommendationID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "record_ai_recommendation_decision_failed"})
		return
	}

	response := aiRecommendationDecisionResponse{Status: "recorded", Recommendation: recommendation}
	encoded, err := json.Marshal(response)
	if err == nil {
		s.storeMutationReplay(r.Context(), key, body, http.StatusOK, append(encoded, '\n'))
	}
	writeJSON(w, http.StatusOK, response)
}

func parseRecommendationLimit(raw string) int {
	limit := 25
	if strings.TrimSpace(raw) != "" {
		if parsed, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil {
			limit = parsed
		}
	}
	if limit <= 0 {
		return 25
	}
	if limit > 100 {
		return 100
	}
	return limit
}

func normalizeAIRecommendationDecision(value string) string {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "ACKNOWLEDGED", "OVERRIDDEN", "DISMISSED", "REOPENED":
		return strings.ToUpper(strings.TrimSpace(value))
	default:
		return ""
	}
}

func statusForAIRecommendationDecision(decision string) string {
	if strings.EqualFold(decision, "REOPENED") {
		return "PENDING"
	}
	return strings.ToUpper(strings.TrimSpace(decision))
}

func aiRecommendationActor(ctx context.Context) string {
	claims, ok := auth.FromContext(ctx)
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		return "supplier_portal"
	}
	return strings.TrimSpace(claims.Subject)
}
