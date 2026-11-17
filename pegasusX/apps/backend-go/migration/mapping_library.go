package migration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/migration/parser"
)

// ErrIncompleteRequiredMappings is returned when required pack fields lack confirmed >=0.80 mappings.
var ErrIncompleteRequiredMappings = errors.New("incomplete_required_mappings")

// ColumnMappingProposal describes how one raw column maps to a canonical PegasusX pack field.
type ColumnMappingProposal struct {
	SourceColumn string  `json:"source_column"`
	TargetField  string  `json:"target_field"`
	Confidence   float64 `json:"confidence"`
	Reason       string  `json:"reason,omitempty"`
	Source       string  `json:"source,omitempty"` // library | deterministic | gemini | heuristic | manual
	Confirmed    bool    `json:"confirmed"`
	Transform    string  `json:"transform,omitempty"`
}

// SessionMappingDocument stores active/proposed mappings and audit metadata on SupplierImportMapping.
type SessionMappingDocument struct {
	SessionID         string                  `json:"session_id"`
	Pack              Pack                    `json:"pack"`
	HeaderFingerprint string                  `json:"header_fingerprint"`
	AIModel           string                  `json:"ai_model"`
	AIStatus          string                  `json:"ai_status"`
	TokensUsed        int                     `json:"tokens_used,omitempty"`
	FallbackReason    string                  `json:"fallback_reason,omitempty"`
	Mappings          []ColumnMappingProposal `json:"mappings"`
	MissingRequired   []string                `json:"missing_required,omitempty"`
	LowConfidenceCols []string                `json:"low_confidence_columns,omitempty"`
	AllRequiredMapped bool                    `json:"all_required_mapped"`
	MappingHash       string                  `json:"mapping_hash"`
	ConfirmedBy       string                  `json:"confirmed_by,omitempty"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

// MappingLibraryEntry mirrors a row in SupplierImportMappingLibrary.
type MappingLibraryEntry struct {
	SupplierID        string                  `json:"supplier_id"`
	Pack              Pack                    `json:"pack"`
	HeaderFingerprint string                  `json:"header_fingerprint"`
	Mappings          []ColumnMappingProposal `json:"mappings"`
	ConfirmedBy       string                  `json:"confirmed_by"`
	UpdatedAt         time.Time               `json:"updated_at"`
}

// ConfirmMappingRequest represents POST /v1/supplier/migrations/sessions/{sessionId}/mapping.
type ConfirmMappingRequest struct {
	Mappings              []ColumnMappingProposal `json:"mappings,omitempty"`
	AcceptAboveConfidence *float64                `json:"accept_above_confidence,omitempty"`
	SaveToLibrary         *bool                   `json:"save_to_library,omitempty"`
	Strict                bool                    `json:"strict,omitempty"`
}

// ComputeMappingHash computes a stable digest of confirmed+proposed column targets for dry-run freshness checks.
func ComputeMappingHash(doc SessionMappingDocument) string {
	lines := make([]string, 0, len(doc.Mappings))
	for _, m := range doc.Mappings {
		if m.TargetField == "" {
			continue
		}
		lines = append(lines, fmt.Sprintf("%s=%s:%t:%.2f:%s",
			strings.TrimSpace(m.SourceColumn),
			strings.TrimSpace(m.TargetField),
			m.Confirmed,
			m.Confidence,
			strings.TrimSpace(m.Transform),
		))
	}
	sort.Strings(lines)
	sum := sha256.Sum256([]byte(string(doc.Pack) + "|" + strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])
}

// EvaluateRequiredMappings checks whether every required group for `pack` has a proposal >= 0.80.
func EvaluateRequiredMappings(pack Pack, mappings []ColumnMappingProposal) (allOK bool, missing []string, lowConf []string) {
	return evaluateMappingsInternal(pack, mappings, false)
}

// ValidateMappingsReadyForApply enforces that every required group is both Confirmed=true and >= 0.80.
func ValidateMappingsReadyForApply(doc SessionMappingDocument) error {
	allOK, missing, lowConf := evaluateMappingsInternal(doc.Pack, doc.Mappings, true)
	if !allOK {
		return fmt.Errorf("%w: missing required=%v low_confidence=%v", ErrIncompleteRequiredMappings, missing, lowConf)
	}
	return nil
}

func evaluateMappingsInternal(pack Pack, mappings []ColumnMappingProposal, requireConfirmed bool) (bool, []string, []string) {
	bestByTarget := make(map[string]ColumnMappingProposal)
	var lowConf []string

	for _, m := range mappings {
		if m.TargetField == "" {
			continue
		}
		if m.Confidence < MinRequiredMappingConfidence {
			lowConf = append(lowConf, m.SourceColumn)
		}
		if requireConfirmed && !m.Confirmed {
			continue
		}
		prev, exists := bestByTarget[m.TargetField]
		if !exists || m.Confidence > prev.Confidence {
			bestByTarget[m.TargetField] = m
		}
	}

	var missing []string
	for _, g := range PackRequiredGroups(pack) {
		satisfied := false
		for _, field := range g.Fields {
			if m, ok := bestByTarget[field]; ok && m.Confidence >= MinRequiredMappingConfidence {
				satisfied = true
				break
			}
		}
		if !satisfied {
			missing = append(missing, g.Name)
		}
	}

	return len(missing) == 0, missing, lowConf
}

// ConfirmSessionMapping merges user overrides and optional `accept_above_confidence` bulk acceptance,
// updates SupplierImportMapping & SupplierImportMappingLibrary, and advances the session to DISCOVERED when ready.
func ConfirmSessionMapping(
	ctx context.Context,
	repo Repository,
	supplierID, sessionID, actorLabel string,
	req ConfirmMappingRequest,
	now time.Time,
) (Session, SessionMappingDocument, error) {
	if repo == nil {
		return Session{}, SessionMappingDocument{}, fmt.Errorf("migration repository is required")
	}
	sess, err := repo.GetSession(ctx, supplierID, sessionID)
	if err != nil {
		return Session{}, SessionMappingDocument{}, err
	}
	existingDoc, hasDoc, err := repo.GetSessionMapping(ctx, supplierID, sessionID)
	if err != nil {
		return Session{}, SessionMappingDocument{}, err
	}
	if !hasDoc {
		existingDoc = SessionMappingDocument{
			SessionID: sessionID,
			Pack:      sess.Pack,
			AIModel:   "manual",
			AIStatus:  "MANUAL",
		}
	}
	if existingDoc.Pack == "" {
		existingDoc.Pack = sess.Pack
	}

	validTargets := make(map[string]bool)
	for _, f := range PackTaxonomy(existingDoc.Pack) {
		validTargets[f.Key] = true
	}

	byCol := make(map[string]int, len(existingDoc.Mappings))
	merged := append([]ColumnMappingProposal(nil), existingDoc.Mappings...)
	for i, m := range merged {
		byCol[m.SourceColumn] = i
	}

	for _, upd := range req.Mappings {
		target := strings.TrimSpace(upd.TargetField)
		if target != "" && !validTargets[target] {
			return Session{}, SessionMappingDocument{}, fmt.Errorf("%w: unknown target field %q for pack %s", ErrInvalidInput, target, existingDoc.Pack)
		}
		src := strings.TrimSpace(upd.SourceColumn)
		if src == "" {
			continue
		}
		entry := ColumnMappingProposal{
			SourceColumn: src,
			TargetField:  target,
			Confidence:   1.0,
			Reason:       strings.TrimSpace(upd.Reason),
			Source:       "manual",
			Confirmed:    target != "",
			Transform:    strings.TrimSpace(upd.Transform),
		}
		if idx, exists := byCol[src]; exists {
			merged[idx] = entry
		} else {
			byCol[src] = len(merged)
			merged = append(merged, entry)
		}
	}

	if req.AcceptAboveConfidence != nil {
		thresh := *req.AcceptAboveConfidence
		if thresh < 0 {
			thresh = 0
		}
		for i := range merged {
			if merged[i].TargetField != "" && merged[i].Confidence >= thresh {
				merged[i].Confirmed = true
			}
		}
	}

	if existingDoc.HeaderFingerprint == "" {
		headers := make([]string, 0, len(merged))
		for _, m := range merged {
			headers = append(headers, m.SourceColumn)
		}
		existingDoc.HeaderFingerprint = ComputeHeaderFingerprint(existingDoc.Pack, headers)
	}

	existingDoc.Mappings = merged
	existingDoc.ConfirmedBy = actorLabel
	existingDoc.UpdatedAt = now.UTC()
	existingDoc.AllRequiredMapped, existingDoc.MissingRequired, existingDoc.LowConfidenceCols = evaluateMappingsInternal(existingDoc.Pack, merged, true)
	existingDoc.MappingHash = ComputeMappingHash(existingDoc)

	if req.Strict && !existingDoc.AllRequiredMapped {
		return sess, existingDoc, fmt.Errorf("%w: missing required fields %v", ErrIncompleteRequiredMappings, existingDoc.MissingRequired)
	}

	nextStatus := sess.Status
	switch NormalizeSessionStatus(string(sess.Status)) {
	case SessionStatusMappingRequired, SessionStatusMappingProposed, SessionStatusDiscovered:
		if existingDoc.AllRequiredMapped {
			nextStatus = SessionStatusDiscovered
		} else {
			nextStatus = SessionStatusMappingRequired
		}
	}

	if err := repo.SaveSessionMapping(ctx, supplierID, sessionID, nextStatus, existingDoc); err != nil {
		return Session{}, SessionMappingDocument{}, err
	}

	saveLib := req.SaveToLibrary == nil || *req.SaveToLibrary
	if saveLib && existingDoc.AllRequiredMapped && existingDoc.HeaderFingerprint != "" {
		_ = repo.UpsertMappingLibrary(ctx, MappingLibraryEntry{
			SupplierID:        supplierID,
			Pack:              existingDoc.Pack,
			HeaderFingerprint: existingDoc.HeaderFingerprint,
			Mappings:          existingDoc.Mappings,
			ConfirmedBy:       actorLabel,
			UpdatedAt:         now.UTC(),
		})
	}

	updatedSess, err := repo.GetSession(ctx, supplierID, sessionID)
	if err != nil {
		updatedSess = sess
		updatedSess.Status = nextStatus
	}
	return updatedSess, existingDoc, nil
}

// ProposeMappingsForBundle generates and persists initial AI/library/heuristic column mappings
// for every child session in a sorted bundle that is still awaiting mapping.
func ProposeMappingsForBundle(
	ctx context.Context,
	repo Repository,
	mapper *AIMapper,
	supplierID, homeCell, projectID, bundleID string,
) ([]SessionMappingDocument, error) {
	if repo == nil {
		return nil, fmt.Errorf("migration repository is required")
	}
	if mapper == nil {
		mapper = NewAIMapper(GeminiMapperConfigFromEnv())
	}

	sessions, err := repo.ListSessionsByBundle(ctx, supplierID, projectID, bundleID)
	if err != nil {
		return nil, err
	}

	out := make([]SessionMappingDocument, 0, len(sessions))
	for _, sess := range sessions {
		st := NormalizeSessionStatus(string(sess.Status))
		if st != SessionStatusMappingRequired && st != SessionStatusMappingProposed && st != SessionStatusDiscovered {
			continue
		}
		staged, _, listErr := repo.ListStagedRows(ctx, supplierID, sess.SessionID, 20, 0)
		if listErr != nil {
			return nil, listErr
		}
		headers, samples := extractHeadersAndSampleRows(staged)
		if len(headers) == 0 {
			continue
		}
		fp := ComputeHeaderFingerprint(sess.Pack, headers)
		if existingDoc, hasDoc, _ := repo.GetSessionMapping(ctx, supplierID, sess.SessionID); hasDoc &&
			existingDoc.AllRequiredMapped &&
			existingDoc.HeaderFingerprint == fp &&
			allMappingsConfirmed(existingDoc) {
			out = append(out, existingDoc)
			continue
		}
		libEntry, _ := repo.GetMappingLibrary(ctx, supplierID, sess.Pack, fp)

		doc := mapper.ProposeMappings(ctx, supplierID, homeCell, sess.SessionID, sess.Pack, headers, samples, libEntry)
		nextStatus := SessionStatusMappingProposed
		if doc.AIStatus == "LIBRARY_HIT" && doc.AllRequiredMapped {
			nextStatus = SessionStatusDiscovered
		} else if !doc.AllRequiredMapped && len(doc.Mappings) == 0 {
			nextStatus = SessionStatusMappingRequired
		}
		if err := repo.SaveSessionMapping(ctx, supplierID, sess.SessionID, nextStatus, doc); err != nil {
			return nil, err
		}
		out = append(out, doc)
	}
	return out, nil
}

func extractHeadersAndSampleRows(staged []StagedRow) ([]string, []parser.ParsedRow) {
	if len(staged) == 0 {
		return nil, nil
	}
	headerSet := make(map[string]bool)
	var headers []string
	samples := make([]parser.ParsedRow, 0, len(staged))
	for _, sr := range staged {
		if len(sr.RawData) == 0 {
			continue
		}
		var m map[string]any
		if err := json.Unmarshal(sr.RawData, &m); err != nil {
			continue
		}
		for k := range m {
			if !headerSet[k] {
				headerSet[k] = true
				headers = append(headers, k)
			}
		}
		samples = append(samples, parser.ParsedRow{
			RowIndex: sr.RowIndex,
			Fields:   m,
		})
	}
	sort.Strings(headers)
	return headers, samples
}

func allMappingsConfirmed(doc SessionMappingDocument) bool {
	if len(doc.Mappings) == 0 {
		return false
	}
	for _, m := range doc.Mappings {
		if !m.Confirmed {
			return false
		}
	}
	return true
}
