// Package ontology implements the ATOMOS M0.5 Semantic Ontology Projector,
// typed 17-entity registry, security marking redaction, Spanner persistence,
// Kafka domain-event projector, nightly drift rebuild, and EKG unification.
package ontology

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/foundry"
)

// SchemaVersion is the canonical ontology registry contract version.
const SchemaVersion = "1.0.0"

// DefaultCellID is the default regional Spanner/ontology cell when unset.
const DefaultCellID = "cell-uz"

// Canonical v1 ontology object types (17 total).
const (
	TypeSupplier      = "Supplier"
	TypeLegalEntity   = "LegalEntity"
	TypeWarehouse     = "Warehouse"
	TypeFactory       = "Factory"
	TypeSKU           = "SKU"
	TypeLot           = "Lot"
	TypeShop          = "Shop"
	TypeOrder         = "Order"
	TypeOrderLine     = "OrderLine"
	TypeManifest      = "Manifest"
	TypeVehicle       = "Vehicle"
	TypeDriver        = "Driver"
	TypeNasiyaInvoice = "NasiyaInvoice"
	TypeCreditLine    = "CreditLine"
	TypePayment       = "Payment"
	TypeFiscalDoc     = "FiscalDoc"
	TypeProposal      = "Proposal"
)

// Canonical v1 ontology link types (7 total).
const (
	LinkSupplies     = "supplies"
	LinkStocks       = "stocks"
	LinkOrdered      = "ordered"
	LinkShippedOn    = "shipped_on"
	LinkOwes         = "owes"
	LinkSubstitutes  = "substitutes"
	LinkRecalledWith = "recalled_with"
)

// Security markings enforced on ontology objects and redacted for non-finance/non-admin viewers.
const (
	MarkingFinanceOnly              = "MARKING:FINANCE_ONLY"
	MarkingPII                      = "MARKING:PII"
	MarkingAMLInvestigation         = foundry.MarkingAMLInvestigation
	MarkingDiscrepantQuarantine     = foundry.MarkingDiscrepantQuarantine
	MarkingThermalQuarantineSpoiled = foundry.MarkingThermalQuarantineSpoiled
	MarkingMicroSkimmingSuspect     = foundry.MarkingMicroSkimmingSuspect
	MarkingLotRecallQuarantine      = foundry.MarkingLotRecallQuarantine
)

// Domain errors returned by the ontology service and repository.
var (
	ErrInvalidIdentity   = errors.New("ontology: cell_id, object_type, and object_id are required")
	ErrUnknownObjectType = errors.New("ontology: unknown v1 object_type")
	ErrUnknownLinkType   = errors.New("ontology: unknown v1 link_type")
	ErrSchemaValidation  = errors.New("ontology: schema validation failed")
	ErrFloatForbidden    = errors.New("ontology: float values are forbidden on minor-unit/ontology properties")
	ErrObjectNotFound    = errors.New("ontology: object not found")
	ErrRepoUnavailable   = errors.New("ontology: repository unavailable")
)

// OntologyObject represents one canonical semantic entity persisted in OntologyObjects.
type OntologyObject struct {
	CellID           string         `json:"cell_id"`
	ObjectType       string         `json:"object_type"`
	ObjectID         string         `json:"object_id"`
	SupplierID       string         `json:"supplier_id"`
	Version          int64          `json:"version"`
	Properties       map[string]any `json:"properties"`
	PropertiesJSON   string         `json:"properties_json,omitempty"`
	SecurityMarkings []string       `json:"security_markings,omitempty"`
	ValidFrom        time.Time      `json:"valid_from"`
	UpdatedAt        time.Time      `json:"updated_at"`
}

// PrepareForSpanner synchronizes Properties and PropertiesJSON without float decoding.
func (o *OntologyObject) PrepareForSpanner() error {
	if o == nil {
		return nil
	}
	if o.Properties == nil && strings.TrimSpace(o.PropertiesJSON) != "" {
		m, err := DecodeJSONMapNoFloats([]byte(o.PropertiesJSON))
		if err != nil {
			return fmt.Errorf("ontology: decode properties_json: %w", err)
		}
		o.Properties = m
	}
	if o.Properties == nil {
		o.Properties = map[string]any{}
	}
	b, err := json.Marshal(o.Properties)
	if err != nil {
		return fmt.Errorf("ontology: marshal properties: %w", err)
	}
	o.PropertiesJSON = string(b)
	o.SecurityMarkings = NormalizeMarkings(o.SecurityMarkings)
	return nil
}

// Clone returns a deep copy of the object safe for viewer redaction or mutation.
func (o *OntologyObject) Clone() *OntologyObject {
	if o == nil {
		return nil
	}
	return &OntologyObject{
		CellID:           o.CellID,
		ObjectType:       o.ObjectType,
		ObjectID:         o.ObjectID,
		SupplierID:       o.SupplierID,
		Version:          o.Version,
		Properties:       cloneAnyMap(o.Properties),
		PropertiesJSON:   o.PropertiesJSON,
		SecurityMarkings: slices.Clone(o.SecurityMarkings),
		ValidFrom:        o.ValidFrom,
		UpdatedAt:        o.UpdatedAt,
	}
}

// OntologyGraphEdge represents one directed relationship in OntologyGraphEdges.
type OntologyGraphEdge struct {
	CellID            string            `json:"cell_id"`
	SourceObjectType  string            `json:"source_object_type"`
	SourceObjectID    string            `json:"source_object_id"`
	LinkType          string            `json:"link_type"`
	TargetObjectType  string            `json:"target_object_type"`
	TargetObjectID    string            `json:"target_object_id"`
	SupplierID        string            `json:"supplier_id"`
	CreatedByActionID string            `json:"created_by_action_id"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	MetadataJSON      string            `json:"metadata_json,omitempty"`
	ValidFrom         time.Time         `json:"valid_from"`
	ValidTo           *time.Time        `json:"valid_to,omitempty"`
}

// PrepareForSpanner synchronizes Metadata and MetadataJSON.
func (e *OntologyGraphEdge) PrepareForSpanner() error {
	if e == nil {
		return nil
	}
	if e.Metadata == nil && strings.TrimSpace(e.MetadataJSON) != "" {
		var m map[string]string
		if err := json.Unmarshal([]byte(e.MetadataJSON), &m); err != nil {
			return fmt.Errorf("ontology: decode metadata_json: %w", err)
		}
		e.Metadata = m
	}
	if e.MetadataJSON == "" && len(e.Metadata) > 0 {
		b, err := json.Marshal(e.Metadata)
		if err != nil {
			return fmt.Errorf("ontology: marshal metadata: %w", err)
		}
		e.MetadataJSON = string(b)
	}
	return nil
}

// TypeDescriptor describes one registered v1 ontology entity schema.
type TypeDescriptor struct {
	ObjectType           string   `json:"object_type"`
	Version              string   `json:"version"`
	SchemaFile           string   `json:"schema_file"`
	PrimaryKey           []string `json:"primary_key"`
	RequiredProperties   []string `json:"required_properties"`
	MoneyFieldsMinor     []string `json:"money_fields_minor,omitempty"`
	SecurityMarkings     []string `json:"security_markings,omitempty"`
	AllowedOutgoingLinks []string `json:"allowed_outgoing_links,omitempty"`
	Description          string   `json:"description"`
}

// TypesCatalog is returned by GET /v1/ontology/types.
type TypesCatalog struct {
	Version          string           `json:"version"`
	Types            []TypeDescriptor `json:"types"`
	LinkTypes        []string         `json:"link_types"`
	SecurityMarkings []string         `json:"security_markings"`
}

// ListObjectsFilter defines query filters for ListObjects.
type ListObjectsFilter struct {
	CellID     string `json:"cell_id,omitempty"`
	SupplierID string `json:"supplier_id,omitempty"`
	ObjectType string `json:"object_type,omitempty"`
	Search     string `json:"search,omitempty"`
	Marking    string `json:"marking,omitempty"`
	Limit      int    `json:"limit,omitempty"`
	Offset     int    `json:"offset,omitempty"`
	ViewerRole string `json:"viewer_role,omitempty"`
}

// ListObjectsResult is returned by ListObjects and GET /v1/ontology/objects.
type ListObjectsResult struct {
	CellID        string            `json:"cell_id"`
	SupplierID    string            `json:"supplier_id"`
	Objects       []*OntologyObject `json:"objects"`
	Total         int               `json:"total"`
	Limit         int               `json:"limit"`
	Offset        int               `json:"offset"`
	PIIMasked     bool              `json:"pii_masked"`
	FinanceMasked bool              `json:"finance_masked"`
}

// NeighborNode pairs a discovered neighbor OntologyObject with BFS hop metadata.
type NeighborNode struct {
	Object      *OntologyObject `json:"object"`
	Hop         int             `json:"hop"`
	ViaLinkType string          `json:"via_link_type,omitempty"`
	Direction   string          `json:"direction,omitempty"` // OUTBOUND | INBOUND
	ParentType  string          `json:"parent_type,omitempty"`
	ParentID    string          `json:"parent_id,omitempty"`
}

// NeighborResult is returned by GetNeighbors and GET /v1/ontology/objects/{type}/{id}/neighbors.
type NeighborResult struct {
	CellID        string               `json:"cell_id"`
	SupplierID    string               `json:"supplier_id"`
	Root          *OntologyObject      `json:"root"`
	Depth         int                  `json:"depth"`
	Nodes         []NeighborNode       `json:"nodes"`
	Edges         []*OntologyGraphEdge `json:"edges"`
	TotalNodes    int                  `json:"total_nodes"`
	TotalEdges    int                  `json:"total_edges"`
	PIIMasked     bool                 `json:"pii_masked"`
	FinanceMasked bool                 `json:"finance_masked"`
}

// UpsertStats summarizes monotonic upsert outcomes.
type UpsertStats struct {
	ObjectsUpserted  int `json:"objects_upserted"`
	ObjectsSkipped   int `json:"objects_skipped"`
	EdgesUpserted    int `json:"edges_upserted"`
	BatchesCommitted int `json:"batches_committed"`
}

// RebuildResult is returned by RebuildFromSpanner and POST /v1/ontology/rebuild.
type RebuildResult struct {
	CellID           string    `json:"cell_id"`
	SupplierID       string    `json:"supplier_id"`
	Status           string    `json:"status"`
	ObjectsUpserted  int       `json:"objects_upserted"`
	EdgesUpserted    int       `json:"edges_upserted"`
	BatchesCommitted int       `json:"batches_committed"`
	DurationMs       int64     `json:"duration_ms"`
	TablesScanned    []string  `json:"tables_scanned"`
	RebuiltAt        time.Time `json:"rebuilt_at"`
}

// KGNode mirrors planning.KGNode for backward-compatible EKG unification.
type KGNode struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Name string `json:"name,omitempty"`
}

// KGEdge mirrors planning.KGEdge for backward-compatible EKG unification.
type KGEdge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Relation string `json:"relation"`
}

// UnifiedKnowledgeGraph mirrors planning.KnowledgeGraph with optional ontology counts.
type UnifiedKnowledgeGraph struct {
	SupplierID           string   `json:"supplier_id"`
	Nodes                []KGNode `json:"nodes"`
	Edges                []KGEdge `json:"edges"`
	Source               string   `json:"source,omitempty"`
	OntologyObjectsCount int      `json:"ontology_objects_count,omitempty"`
	OntologyEdgesCount   int      `json:"ontology_edges_count,omitempty"`
}

// NormalizeCellID returns trimmed cellID or DefaultCellID when empty.
func NormalizeCellID(cellID string) string {
	trimmed := strings.TrimSpace(cellID)
	if trimmed == "" {
		return DefaultCellID
	}
	return trimmed
}

// NormalizeMarkings deduplicates, trims, and sorts security markings deterministically.
func NormalizeMarkings(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, raw := range in {
		m := strings.TrimSpace(raw)
		if m == "" {
			continue
		}
		if _, exists := seen[m]; exists {
			continue
		}
		seen[m] = struct{}{}
		out = append(out, m)
	}
	slices.Sort(out)
	return out
}

// MergeMarkings merges two marking slices cleanly.
func MergeMarkings(a, b []string) []string {
	combined := make([]string, 0, len(a)+len(b))
	combined = append(combined, a...)
	combined = append(combined, b...)
	return NormalizeMarkings(combined)
}

// DecodeJSONMapNoFloats decodes JSON object bytes into map[string]any, converting integer json.Numbers
// to int64 and rejecting floating-point numbers so monetary minor units never decay to float64.
func DecodeJSONMapNoFloats(raw []byte) (map[string]any, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return map[string]any{}, nil
	}
	dec := json.NewDecoder(bytes.NewReader(trimmed))
	dec.UseNumber()
	var m map[string]any
	if err := dec.Decode(&m); err != nil {
		return nil, err
	}
	normalized, err := normalizeJSONValue(m)
	if err != nil {
		return nil, err
	}
	out, ok := normalized.(map[string]any)
	if !ok {
		return map[string]any{}, nil
	}
	return out, nil
}

func normalizeJSONValue(v any) (any, error) {
	switch tv := v.(type) {
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return nil, fmt.Errorf("%w: %s", ErrFloatForbidden, s)
		}
		n, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid integer number %q: %w", s, err)
		}
		return n, nil
	case float32, float64:
		return nil, ErrFloatForbidden
	case map[string]any:
		out := make(map[string]any, len(tv))
		for k, val := range tv {
			nv, err := normalizeJSONValue(val)
			if err != nil {
				return nil, err
			}
			out[k] = nv
		}
		return out, nil
	case []any:
		out := make([]any, len(tv))
		for i, val := range tv {
			nv, err := normalizeJSONValue(val)
			if err != nil {
				return nil, err
			}
			out[i] = nv
		}
		return out, nil
	default:
		return v, nil
	}
}

func cloneAnyMap(src map[string]any) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = cloneAnyValue(v)
	}
	return out
}

func cloneAnyValue(v any) any {
	switch tv := v.(type) {
	case map[string]any:
		return cloneAnyMap(tv)
	case []any:
		cp := make([]any, len(tv))
		for i, item := range tv {
			cp[i] = cloneAnyValue(item)
		}
		return cp
	case []string:
		return slices.Clone(tv)
	default:
		return v
	}
}
