package migration

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	// DryRunReportTTL enforces preview freshness: approvals older than 24h require a fresh dry-run.
	DryRunReportTTL = 24 * time.Hour

	// SnapshotRetentionTTL governs automatic row deletion on MigrationSnapshots after 30 days.
	SnapshotRetentionTTL = 30 * 24 * time.Hour

	// RollbackWindowTTL restricts instant 1-click session rollback to 72 hours after apply.
	RollbackWindowTTL = 72 * time.Hour

	// ReportKindDryRun, ReportKindProjectDryRun, ReportKindApply classify MigrationReports rows.
	ReportKindDryRun        = "DRY_RUN"
	ReportKindProjectDryRun = "PROJECT_DRY_RUN"
	ReportKindApply         = "APPLY"

	// ConflictSeverityBlocking and ConflictSeverityWarning classify dry-run diagnostics.
	ConflictSeverityBlocking = "BLOCKING"
	ConflictSeverityWarning  = "WARNING"

	// ApplyModeMerge and ApplyModeSetOnHand control stock quantity application semantics.
	ApplyModeMerge     = "MERGE"
	ApplyModeSetOnHand = "SET_ON_HAND"
)

// DryRunConflict describes one row- or pack-level diagnostic surfaced during virtual overlay dry-run.
type DryRunConflict struct {
	SessionID  string `json:"session_id"`
	Pack       Pack   `json:"pack"`
	RowIndex   int64  `json:"row_index"`
	Code       string `json:"code"`
	Severity   string `json:"severity"` // BLOCKING | WARNING
	Field      string `json:"field,omitempty"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// PackDryRunReport summarizes one child session's simulated outcome against Live ∪ Preceding Virtual Packs.
type PackDryRunReport struct {
	SessionID                string `json:"session_id"`
	Pack                     Pack   `json:"pack"`
	StreamSplitKey           string `json:"stream_split_key,omitempty"`
	RowsTotal                int64  `json:"rows_total"`
	RowsValid                int64  `json:"rows_valid"`
	RowsInvalid              int64  `json:"rows_invalid"`
	WouldCreate              int64  `json:"would_create"`
	WouldUpdate              int64  `json:"would_update"`
	WouldSkip                int64  `json:"would_skip"`
	MoneyImpactMinor         int64  `json:"money_impact_minor"`
	Currency                 string `json:"currency"`
	BlockingCount            int64  `json:"blocking_count"`
	WarningCount             int64  `json:"warning_count"`
	ColdLotWithoutLiveCount  int64  `json:"cold_lot_migrated_without_live_baseline,omitempty"`
	OnHandBelowReservedCount int64  `json:"on_hand_below_reserved,omitempty"`
}

// DryRunSummary is embedded in MigrationReports.summary_json and returned to Desktop/Web/Mobile UIs.
type DryRunSummary struct {
	ProjectID               string             `json:"project_id"`
	ContentFingerprint      string             `json:"content_fingerprint"`
	Packs                   []PackDryRunReport `json:"packs"`
	Conflicts               []DryRunConflict   `json:"conflicts,omitempty"`
	BalancedJournalCheck    bool               `json:"balanced_journal_check"`
	JournalDebitMinor       int64              `json:"journal_debit_minor"`
	JournalCreditMinor      int64              `json:"journal_credit_minor"`
	OpeningBalanceAccount   string             `json:"opening_balance_account"`
	ReadyToApprove          bool               `json:"ready_to_approve"`
	ColdLotWarningCount     int64              `json:"cold_lot_migrated_without_live_baseline"`
	OnHandBelowReservedHits int64              `json:"on_hand_below_reserved"`
	EvaluatedAt             time.Time          `json:"evaluated_at"`
	ExpiresAt               time.Time          `json:"expires_at"`
}

// MigrationReport mirrors a row in MigrationReports (all financial totals strictly INT64 minor units).
type MigrationReport struct {
	SupplierID         string          `json:"supplier_id"`
	ReportID           string          `json:"report_id"`
	ProjectID          string          `json:"project_id,omitempty"`
	SessionID          string          `json:"session_id,omitempty"`
	ReportKind         string          `json:"report_kind"`
	ContentFingerprint string          `json:"content_fingerprint"`
	RowsTotal          int64           `json:"rows_total"`
	RowsValid          int64           `json:"rows_valid"`
	RowsInvalid        int64           `json:"rows_invalid"`
	WouldCreate        int64           `json:"would_create"`
	WouldUpdate        int64           `json:"would_update"`
	WouldSkip          int64           `json:"would_skip"`
	MoneyImpactMinor   int64           `json:"money_impact_minor"`
	Currency           string          `json:"currency,omitempty"`
	BlockingCount      int64           `json:"blocking_count"`
	WarningCount       int64           `json:"warning_count"`
	SummaryJSON        json.RawMessage `json:"summary_json"`
	ExpiresAt          time.Time       `json:"expires_at"`
	CreatedAt          time.Time       `json:"created_at"`
}

// MigrationSnapshotRecord captures a pre-apply row before-image for 72h rollback.
type MigrationSnapshotRecord struct {
	SupplierID  string          `json:"supplier_id"`
	SessionID   string          `json:"session_id"`
	SnapshotSeq int64           `json:"snapshot_seq"`
	TargetTable string          `json:"target_table"`
	RowKeyJSON  string          `json:"row_key_json"`
	OpKind      string          `json:"op_kind"` // INSERT | UPDATE
	BeforeJSON  json.RawMessage `json:"before_json,omitempty"`
	ExpiresAt   time.Time       `json:"expires_at"`
	CreatedAt   time.Time       `json:"created_at"`
}

// ComputeSessionDryRunFingerprint hashes artifact content SHA256, mapping hash, and apply mode
// so any underlying file re-upload or column remapping invalidates stale dry-run approvals.
func ComputeSessionDryRunFingerprint(sess Session, doc SessionMappingDocument, applyMode string) string {
	lines := []string{
		strings.TrimSpace(sess.ContentSHA256),
		ComputeMappingHash(doc),
		NormalizeApplyMode(applyMode),
		string(sess.Pack),
		strings.TrimSpace(sess.StreamSplitKey),
	}
	sum := sha256.Sum256([]byte(strings.Join(lines, "|")))
	return hex.EncodeToString(sum[:])
}

// ComputeProjectDryRunFingerprint binds a project-wide dry-run report to every constituent session fingerprint.
func ComputeProjectDryRunFingerprint(sessionFingerprints []string) string {
	cp := append([]string(nil), sessionFingerprints...)
	sort.Strings(cp)
	sum := sha256.Sum256([]byte(strings.Join(cp, "\n")))
	return hex.EncodeToString(sum[:])
}

// NormalizeApplyMode normalizes inventory apply modes, defaulting safely to MERGE.
func NormalizeApplyMode(raw string) string {
	if strings.EqualFold(strings.TrimSpace(raw), ApplyModeSetOnHand) {
		return ApplyModeSetOnHand
	}
	return ApplyModeMerge
}

// ParseMoneyMinor converts decimal major currency strings (or spaced numbers like "12 500.50") into integer minor units (tins/tiyin, x100).
// Uses exact integer arithmetic after string cleanup to avoid float precision drift where possible.
func ParseMoneyMinor(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, fmt.Errorf("empty money value")
	}
	s = strings.NewReplacer(" ", "", "\u00a0", "", ",", ".").Replace(s)
	// Keep only optional leading '-', digits, and decimal point.
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, fmt.Errorf("invalid money format %q", raw)
	}
	wholePart := parts[0]
	if wholePart == "" {
		wholePart = "0"
	}
	whole, err := strconv.ParseInt(wholePart, 10, 64)
	if err != nil {
		// Fall back cleanly if scientific notation or trailing spaces exist.
		f, fErr := strconv.ParseFloat(s, 64)
		if fErr != nil || math.IsNaN(f) || math.IsInf(f, 0) {
			return 0, fmt.Errorf("invalid money value %q", raw)
		}
		minor := int64(math.Round(f * 100.0))
		if neg {
			minor = -minor
		}
		return minor, nil
	}
	var frac int64
	if len(parts) == 2 && parts[1] != "" {
		fracStr := parts[1]
		if len(fracStr) == 1 {
			fracStr += "0"
		} else if len(fracStr) > 2 {
			fracStr = fracStr[:2]
		}
		frac, err = strconv.ParseInt(fracStr, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("invalid money fraction %q", raw)
		}
	}
	minor := whole*100 + frac
	if neg {
		minor = -minor
	}
	return minor, nil
}
