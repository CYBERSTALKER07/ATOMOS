package entityresolution

import (
	"context"
	"errors"
	"math"
	"regexp"
	"sort"
	"strings"
)

const (
	EntityTypeAny       = "ANY"
	EntityTypeSupplier  = "SUPPLIER"
	EntityTypeWarehouse = "WAREHOUSE"
	EntityTypeFactory   = "FACTORY"
	EntityTypeDriver    = "DRIVER"
	EntityTypeVehicle   = "VEHICLE"
	EntityTypeRetailer  = "RETAILER"
	EntityTypeOrder     = "ORDER"
	EntityTypeInvoice   = "INVOICE"
	EntityTypeRoute     = "ROUTE"
)

const (
	defaultMaxCandidates = 5
	maxCandidatesCap     = 20
)

var (
	ErrInvalidInput = errors.New("invalid entity resolution input")
	ErrNotFound     = errors.New("entity not found")

	nonAlphaNumPattern = regexp.MustCompile(`[^a-z0-9]+`)
)

var searchableEntityTypes = []string{
	EntityTypeSupplier,
	EntityTypeWarehouse,
	EntityTypeFactory,
	EntityTypeDriver,
	EntityTypeVehicle,
	EntityTypeRetailer,
	EntityTypeOrder,
}

// ResolveInput is the normalized resolve request contract.
type ResolveInput struct {
	SupplierID    string
	EntityType    string
	EntityID      string
	Query         string
	MaxCandidates int
}

// ExplainInput is the normalized explain request contract.
type ExplainInput struct {
	SupplierID string
	EntityType string
	EntityID   string
}

// ResolutionCandidate is one ranked identity candidate.
type ResolutionCandidate struct {
	NodeID          string            `json:"node_id"`
	EntityType      string            `json:"entity_type"`
	EntityID        string            `json:"entity_id"`
	Label           string            `json:"label"`
	Score           float64           `json:"score"`
	ConfidenceClass string            `json:"confidence_class"`
	Deterministic   bool              `json:"deterministic"`
	Reasons         []string          `json:"reasons,omitempty"`
	Metadata        map[string]string `json:"metadata,omitempty"`
}

// ResolveResult is the entity-resolution response model.
type ResolveResult struct {
	ScopeSupplierID string                `json:"scope_supplier_id"`
	RequestedType   string                `json:"requested_type"`
	Query           string                `json:"query,omitempty"`
	Resolved        *ResolutionCandidate  `json:"resolved,omitempty"`
	Candidates      []ResolutionCandidate `json:"candidates"`
}

// GraphNode is one semantic-graph node.
type GraphNode struct {
	NodeID     string `json:"node_id"`
	EntityType string `json:"entity_type"`
	EntityID   string `json:"entity_id"`
	Label      string `json:"label"`
}

// GraphEdge is one semantic-graph relationship.
type GraphEdge struct {
	From       string  `json:"from"`
	To         string  `json:"to"`
	Relation   string  `json:"relation"`
	Confidence float64 `json:"confidence"`
}

// GraphProjection is the lineage projection for a source node.
type GraphProjection struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

// ExplainResult is the explainability response model.
type ExplainResult struct {
	ScopeSupplierID string              `json:"scope_supplier_id"`
	Source          ResolutionCandidate `json:"source"`
	Projection      GraphProjection     `json:"projection"`
}

// EntityRecord is the repository row shape used for ranking.
type EntityRecord struct {
	EntityType string
	EntityID   string
	Label      string
	SearchText string
	Metadata   map[string]string
}

// LineageLink is a direct relation from source to target in the projected graph.
type LineageLink struct {
	TargetType  string
	TargetID    string
	TargetLabel string
	Relation    string
	Confidence  float64
}

// Repository is the read-side contract used by Service.
type Repository interface {
	FindExactByID(ctx context.Context, supplierID, entityType, entityID string) ([]EntityRecord, error)
	ListScopedRecords(ctx context.Context, supplierID, entityType string, perTypeLimit int) ([]EntityRecord, error)
	LoadLineage(ctx context.Context, supplierID, entityType, entityID string) ([]LineageLink, error)
}

// Service contains entity-resolution orchestration.
type Service struct {
	repo Repository
}

// NewService builds a Service over a repository seam.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// Resolve returns ranked deterministic and probabilistic candidates.
func (s *Service) Resolve(ctx context.Context, in ResolveInput) (ResolveResult, error) {
	if strings.TrimSpace(in.SupplierID) == "" {
		return ResolveResult{}, ErrInvalidInput
	}

	entityType := normalizeEntityType(in.EntityType)
	if !isSearchableEntityType(entityType) {
		return ResolveResult{}, ErrInvalidInput
	}

	query := strings.TrimSpace(in.Query)
	entityID := strings.TrimSpace(in.EntityID)
	if query == "" && entityID == "" {
		return ResolveResult{}, ErrInvalidInput
	}

	maxCandidates := in.MaxCandidates
	if maxCandidates <= 0 {
		maxCandidates = defaultMaxCandidates
	}
	if maxCandidates > maxCandidatesCap {
		maxCandidates = maxCandidatesCap
	}

	candidateMap := make(map[string]ResolutionCandidate)

	if entityID != "" {
		exactRows, err := s.repo.FindExactByID(ctx, in.SupplierID, entityType, entityID)
		if err != nil {
			return ResolveResult{}, err
		}
		for _, row := range exactRows {
			candidate := newCandidateFromRow(row, 0.99, true, []string{"exact_id_match"})
			mergeCandidate(candidateMap, candidate)
		}
	}

	if query != "" {
		rows, err := s.repo.ListScopedRecords(ctx, in.SupplierID, entityType, maxCandidates)
		if err != nil {
			return ResolveResult{}, err
		}
		for _, row := range rows {
			score, deterministic, reasons, ok := scoreRow(query, row)
			if !ok {
				continue
			}
			candidate := newCandidateFromRow(row, score, deterministic, reasons)
			mergeCandidate(candidateMap, candidate)
		}
	}

	candidates := make([]ResolutionCandidate, 0, len(candidateMap))
	for _, c := range candidateMap {
		candidates = append(candidates, c)
	}

	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			if candidates[i].Deterministic != candidates[j].Deterministic {
				return candidates[i].Deterministic
			}
			if candidates[i].EntityType == candidates[j].EntityType {
				return candidates[i].EntityID < candidates[j].EntityID
			}
			return candidates[i].EntityType < candidates[j].EntityType
		}
		return candidates[i].Score > candidates[j].Score
	})

	if len(candidates) > maxCandidates {
		candidates = candidates[:maxCandidates]
	}

	result := ResolveResult{
		ScopeSupplierID: in.SupplierID,
		RequestedType:   entityType,
		Query:           query,
		Candidates:      candidates,
	}
	if len(candidates) > 0 {
		resolved := candidates[0]
		result.Resolved = &resolved
	}
	return result, nil
}

// Explain projects a one-hop semantic graph for a supplier-scoped entity.
func (s *Service) Explain(ctx context.Context, in ExplainInput) (ExplainResult, error) {
	if strings.TrimSpace(in.SupplierID) == "" {
		return ExplainResult{}, ErrInvalidInput
	}
	entityType := normalizeEntityType(in.EntityType)
	if !isSearchableEntityType(entityType) || entityType == EntityTypeAny {
		return ExplainResult{}, ErrInvalidInput
	}
	entityID := strings.TrimSpace(in.EntityID)
	if entityID == "" {
		return ExplainResult{}, ErrInvalidInput
	}

	exactRows, err := s.repo.FindExactByID(ctx, in.SupplierID, entityType, entityID)
	if err != nil {
		return ExplainResult{}, err
	}
	if len(exactRows) == 0 {
		return ExplainResult{}, ErrNotFound
	}

	source := newCandidateFromRow(exactRows[0], 0.99, true, []string{"exact_id_match"})

	links, err := s.repo.LoadLineage(ctx, in.SupplierID, entityType, entityID)
	if err != nil {
		return ExplainResult{}, err
	}

	nodesByID := map[string]GraphNode{
		source.NodeID: {
			NodeID:     source.NodeID,
			EntityType: source.EntityType,
			EntityID:   source.EntityID,
			Label:      source.Label,
		},
	}

	edges := make([]GraphEdge, 0, len(links))
	for _, link := range links {
		typeName := normalizeEntityType(link.TargetType)
		nodeID := canonicalNodeID(typeName, link.TargetID)
		if _, exists := nodesByID[nodeID]; !exists {
			label := strings.TrimSpace(link.TargetLabel)
			if label == "" {
				label = link.TargetID
			}
			nodesByID[nodeID] = GraphNode{
				NodeID:     nodeID,
				EntityType: typeName,
				EntityID:   link.TargetID,
				Label:      label,
			}
		}

		edges = append(edges, GraphEdge{
			From:       source.NodeID,
			To:         nodeID,
			Relation:   link.Relation,
			Confidence: round3(link.Confidence),
		})
	}

	nodes := make([]GraphNode, 0, len(nodesByID))
	for _, node := range nodesByID {
		nodes = append(nodes, node)
	}
	sort.Slice(nodes, func(i, j int) bool { return nodes[i].NodeID < nodes[j].NodeID })
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].From == edges[j].From {
			if edges[i].To == edges[j].To {
				return edges[i].Relation < edges[j].Relation
			}
			return edges[i].To < edges[j].To
		}
		return edges[i].From < edges[j].From
	})

	return ExplainResult{
		ScopeSupplierID: in.SupplierID,
		Source:          source,
		Projection: GraphProjection{
			Nodes: nodes,
			Edges: edges,
		},
	}, nil
}

func newCandidateFromRow(row EntityRecord, score float64, deterministic bool, reasons []string) ResolutionCandidate {
	typeName := normalizeEntityType(row.EntityType)
	metadata := cloneMetadata(row.Metadata)
	label := strings.TrimSpace(row.Label)
	if label == "" {
		label = row.EntityID
	}
	return ResolutionCandidate{
		NodeID:          canonicalNodeID(typeName, row.EntityID),
		EntityType:      typeName,
		EntityID:        row.EntityID,
		Label:           label,
		Score:           round3(score),
		ConfidenceClass: confidenceClass(score, deterministic),
		Deterministic:   deterministic,
		Reasons:         normalizeReasons(reasons),
		Metadata:        metadata,
	}
}

func mergeCandidate(current map[string]ResolutionCandidate, next ResolutionCandidate) {
	existing, found := current[next.NodeID]
	if !found {
		current[next.NodeID] = next
		return
	}

	if next.Score > existing.Score {
		existing.Score = next.Score
		existing.ConfidenceClass = next.ConfidenceClass
		existing.Deterministic = next.Deterministic
	}
	if existing.Label == "" || existing.Label == existing.EntityID {
		existing.Label = next.Label
	}
	if len(existing.Metadata) == 0 && len(next.Metadata) > 0 {
		existing.Metadata = next.Metadata
	}
	existing.Reasons = normalizeReasons(append(existing.Reasons, next.Reasons...))
	current[next.NodeID] = existing
}

func scoreRow(query string, row EntityRecord) (score float64, deterministic bool, reasons []string, ok bool) {
	normalizedQuery := normalizeFreeform(query)
	if normalizedQuery == "" {
		return 0, false, nil, false
	}

	if normalizedQuery == normalizeFreeform(row.EntityID) {
		return 0.99, true, []string{"exact_id_match"}, true
	}

	for key, value := range row.Metadata {
		if strings.TrimSpace(value) == "" {
			continue
		}
		if normalizedQuery == normalizeFreeform(value) {
			return 0.96, true, []string{"exact_" + strings.ToLower(key) + "_match"}, true
		}
	}

	combinedText := strings.TrimSpace(row.SearchText + " " + row.Label + " " + flattenMetadata(row.Metadata))
	normalizedCombined := normalizeFreeform(combinedText)
	if normalizedCombined == "" {
		return 0, false, nil, false
	}

	queryTokens := tokenize(normalizedQuery)
	candidateTokens := tokenize(normalizedCombined)
	jaccardScore := jaccard(queryTokens, candidateTokens)
	substring := strings.Contains(normalizedCombined, normalizedQuery)

	if jaccardScore < 0.12 && !substring {
		return 0, false, nil, false
	}

	score = 0.42 + (0.50 * jaccardScore)
	if substring {
		score += 0.10
		reasons = append(reasons, "substring_match")
	}
	if jaccardScore >= 0.20 {
		reasons = append(reasons, "token_overlap")
	}
	if score > 0.93 {
		score = 0.93
	}
	return round3(score), false, normalizeReasons(reasons), true
}

func confidenceClass(score float64, deterministic bool) string {
	if deterministic {
		return "DETERMINISTIC"
	}
	switch {
	case score >= 0.80:
		return "HIGH_PROBABILITY"
	case score >= 0.60:
		return "MEDIUM_PROBABILITY"
	default:
		return "LOW_PROBABILITY"
	}
}

func normalizeEntityType(entityType string) string {
	normalized := strings.ToUpper(strings.TrimSpace(entityType))
	if normalized == "" {
		return EntityTypeAny
	}
	switch normalized {
	case EntityTypeAny,
		EntityTypeSupplier,
		EntityTypeWarehouse,
		EntityTypeFactory,
		EntityTypeDriver,
		EntityTypeVehicle,
		EntityTypeRetailer,
		EntityTypeOrder,
		EntityTypeInvoice,
		EntityTypeRoute:
		return normalized
	default:
		return normalized
	}
}

func isSearchableEntityType(entityType string) bool {
	switch entityType {
	case EntityTypeAny,
		EntityTypeSupplier,
		EntityTypeWarehouse,
		EntityTypeFactory,
		EntityTypeDriver,
		EntityTypeVehicle,
		EntityTypeRetailer,
		EntityTypeOrder:
		return true
	default:
		return false
	}
}

func canonicalNodeID(entityType, entityID string) string {
	return normalizeEntityType(entityType) + ":" + strings.TrimSpace(entityID)
}

func normalizeFreeform(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return ""
	}
	v = nonAlphaNumPattern.ReplaceAllString(v, " ")
	v = strings.TrimSpace(v)
	return strings.Join(strings.Fields(v), " ")
}

func tokenize(v string) map[string]struct{} {
	set := map[string]struct{}{}
	for _, token := range strings.Fields(normalizeFreeform(v)) {
		if token == "" {
			continue
		}
		set[token] = struct{}{}
	}
	return set
}

func jaccard(a, b map[string]struct{}) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	intersect := 0
	for key := range a {
		if _, ok := b[key]; ok {
			intersect++
		}
	}
	union := len(a) + len(b) - intersect
	if union == 0 {
		return 0
	}
	return float64(intersect) / float64(union)
}

func flattenMetadata(metadata map[string]string) string {
	if len(metadata) == 0 {
		return ""
	}
	parts := make([]string, 0, len(metadata))
	for _, value := range metadata {
		if strings.TrimSpace(value) != "" {
			parts = append(parts, value)
		}
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

func cloneMetadata(metadata map[string]string) map[string]string {
	if len(metadata) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(metadata))
	for key, value := range metadata {
		if strings.TrimSpace(value) != "" {
			cloned[key] = strings.TrimSpace(value)
		}
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func normalizeReasons(reasons []string) []string {
	if len(reasons) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(reasons))
	out := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		r := strings.ToLower(strings.TrimSpace(reason))
		if r == "" {
			continue
		}
		if _, ok := seen[r]; ok {
			continue
		}
		seen[r] = struct{}{}
		out = append(out, r)
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil
	}
	return out
}

func round3(v float64) float64 {
	return math.Round(v*1000) / 1000
}
