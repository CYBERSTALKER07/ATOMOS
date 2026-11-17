package audit

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// GenesisHash is the PrevHash for SeqNum == 1 within any (TenantId, DayBucket) chain (64 zeros).
const GenesisHash = "0000000000000000000000000000000000000000000000000000000000000000"

// MaxInlineDiffBytes is the threshold above which BeforeJson/AfterJson payloads are offloaded to WORM storage.
const MaxInlineDiffBytes = 32 * 1024

var (
	dayBucketPattern = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])-(0[1-9]|[12]\d|3[01])$`)
	fallbackPhoneRe  = regexp.MustCompile(`(\+?\d[\d\s\-()]{7,16}\d)`)
	fallbackINNRe    = regexp.MustCompile(`\b\d{9}(\d{5})?\b`)
)

// IsValidDayBucket validates YYYY-MM-DD calendar date formatting.
func IsValidDayBucket(dayBucket string) bool {
	trimmed := strings.TrimSpace(dayBucket)
	if !dayBucketPattern.MatchString(trimmed) {
		return false
	}
	_, err := time.Parse("2006-01-02", trimmed)
	return err == nil
}

// CanonicalAuditPayload defines the exact deterministic JSON structure hashed as
// SHA256(PrevHash || canonical_json) for each AuditEvents row.
// CreatedAt is excluded because Spanner assigns CommitTimestamp at commit time;
// SeqNum establishes strict ordering within (TenantId, DayBucket).
type CanonicalAuditPayload struct {
	TenantID        string `json:"tenant_id"`
	DayBucket       string `json:"day_bucket"`
	EventID         string `json:"event_id"`
	SeqNum          int64  `json:"seq_num"`
	ActorID         string `json:"actor_id"`
	ActorType       string `json:"actor_type"`
	Role            string `json:"role,omitempty"`
	Action          string `json:"action"`
	ObjectType      string `json:"object_type,omitempty"`
	ObjectID        string `json:"object_id,omitempty"`
	BeforeJSON      string `json:"before_json,omitempty"`
	AfterJSON       string `json:"after_json,omitempty"`
	DiffGcsURI      string `json:"diff_gcs_uri,omitempty"`
	ReasonCode      string `json:"reason_code,omitempty"`
	ApprovalID      string `json:"approval_id,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
	IP              string `json:"ip,omitempty"`
	DeviceID        string `json:"device_id,omitempty"`
	LegalHold       bool   `json:"legal_hold"`
	LegalHoldReason string `json:"legal_hold_reason,omitempty"`
	RetentionUntil  string `json:"retention_until,omitempty"`
}

// CanonicalAuditBytes serializes ev into deterministic canonical JSON bytes for hashing.
func CanonicalAuditBytes(ev AuditEvent) ([]byte, error) {
	retention := ""
	if ev.RetentionUntil != nil && !ev.RetentionUntil.IsZero() {
		retention = ev.RetentionUntil.UTC().Truncate(time.Microsecond).Format(time.RFC3339Nano)
	}
	payload := CanonicalAuditPayload{
		TenantID:        ev.TenantID,
		DayBucket:       ev.DayBucket,
		EventID:         ev.EventID,
		SeqNum:          ev.SeqNum,
		ActorID:         ev.ActorID,
		ActorType:       string(ev.ActorType),
		Role:            ev.Role,
		Action:          ev.Action,
		ObjectType:      ev.ObjectType,
		ObjectID:        ev.ObjectID,
		BeforeJSON:      ev.BeforeJSON,
		AfterJSON:       ev.AfterJSON,
		DiffGcsURI:      ev.DiffGcsURI,
		ReasonCode:      ev.ReasonCode,
		ApprovalID:      ev.ApprovalID,
		TraceID:         ev.TraceID,
		IP:              ev.IP,
		DeviceID:        ev.DeviceID,
		LegalHold:       ev.LegalHold,
		LegalHoldReason: ev.LegalHoldReason,
		RetentionUntil:  retention,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("audit canonical json: %w", err)
	}
	return raw, nil
}

// ComputeEntryHash computes SHA256(prevHash || canonicalJSON) as lowercase 64-char hex.
func ComputeEntryHash(prevHash string, canonicalJSON []byte) string {
	h := sha256.New()
	_, _ = h.Write([]byte(prevHash))
	_, _ = h.Write(canonicalJSON)
	return hex.EncodeToString(h.Sum(nil))
}

// IsFinanceOrAdminRole reports whether role is authorized to view unmasked PII (phone, INN) in audit trails.
func IsFinanceOrAdminRole(role string) bool {
	return auth.IsFinanceOrAdminRole(role)
}

// MaskPIIForRole masks "phone", "PhoneNumber", "phone_number", "inn", and "INN" JSON fields
// when the viewer holds a non-finance, non-admin role (!IsFinanceOrAdminRole(role)).
func MaskPIIForRole(role string, rawJSON string) string {
	if rawJSON == "" {
		return ""
	}
	if IsFinanceOrAdminRole(role) {
		return rawJSON
	}

	dec := json.NewDecoder(strings.NewReader(rawJSON))
	dec.UseNumber()
	var parsed any
	if err := dec.Decode(&parsed); err != nil {
		return maskUnstructuredPII(rawJSON)
	}
	masked, changed := maskJSONValue(parsed)
	if !changed {
		return rawJSON
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(masked); err != nil {
		return maskUnstructuredPII(rawJSON)
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

func maskJSONValue(v any) (any, bool) {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(val))
		changed := false
		for k, child := range val {
			if isPIIKey(k) {
				out[k] = maskScalarPII(child)
				changed = true
				continue
			}
			mc, c := maskJSONValue(child)
			out[k] = mc
			if c {
				changed = true
			}
		}
		return out, changed
	case []any:
		out := make([]any, len(val))
		changed := false
		for i, child := range val {
			mc, c := maskJSONValue(child)
			out[i] = mc
			if c {
				changed = true
			}
		}
		return out, changed
	default:
		return v, false
	}
}

func isPIIKey(key string) bool {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "phone", "phonenumber", "phone_number", "inn":
		return true
	default:
		return false
	}
}

func maskScalarPII(v any) string {
	var raw string
	switch s := v.(type) {
	case string:
		raw = strings.TrimSpace(s)
	case json.Number:
		raw = s.String()
	case nil:
		return ""
	default:
		raw = strings.TrimSpace(fmt.Sprint(v))
	}
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= 4 {
		return "***"
	}
	return "***" + string(runes[len(runes)-4:])
}

func maskUnstructuredPII(raw string) string {
	s := fallbackINNRe.ReplaceAllString(raw, "***MASKED_INN***")
	s = fallbackPhoneRe.ReplaceAllString(s, "***MASKED_PHONE***")
	return s
}

// DiffWriter offloads oversized audit diff payloads (>32KB) to durable WORM storage.
type DiffWriter interface {
	WriteDiff(ctx context.Context, tenantID, dayBucket, eventID, field string, raw []byte) (uri string, err error)
}

// ApplyLargeDiffOffload checks whether BeforeJSON or AfterJSON exceeds MaxInlineDiffBytes (32KB).
// When oversized, it offloads the full diff via writer (when provided) to populate DiffGcsURI,
// and compacts BeforeJSON/AfterJSON to deterministic SHA-256 summary envelopes.
func ApplyLargeDiffOffload(ctx context.Context, ev *AuditEvent, writer DiffWriter) error {
	if ev == nil {
		return nil
	}
	beforeOversized := len(ev.BeforeJSON) > MaxInlineDiffBytes
	afterOversized := len(ev.AfterJSON) > MaxInlineDiffBytes
	if !beforeOversized && !afterOversized {
		return nil
	}

	beforeSum := sha256HexString(ev.BeforeJSON)
	afterSum := sha256HexString(ev.AfterJSON)
	combined, err := json.Marshal(struct {
		Before string `json:"before,omitempty"`
		After  string `json:"after,omitempty"`
	}{
		Before: ev.BeforeJSON,
		After:  ev.AfterJSON,
	})
	if err != nil {
		return fmt.Errorf("audit marshal large diff: %w", err)
	}

	if ev.DiffGcsURI == "" && writer != nil {
		uri, err := writer.WriteDiff(ctx, ev.TenantID, ev.DayBucket, ev.EventID, "diff", combined)
		if err != nil {
			return fmt.Errorf("audit offload large diff: %w", err)
		}
		ev.DiffGcsURI = uri
	}

	if beforeOversized {
		summary, _ := json.Marshal(struct {
			Offloaded bool   `json:"offloaded"`
			SHA256    string `json:"sha256"`
			ByteLen   int    `json:"byte_len"`
		}{
			Offloaded: true,
			SHA256:    beforeSum,
			ByteLen:   len(ev.BeforeJSON),
		})
		ev.BeforeJSON = string(summary)
	}

	summaryAfter, _ := json.Marshal(struct {
		Offloaded  bool   `json:"offloaded"`
		SHA256     string `json:"sha256"`
		ByteLen    int    `json:"byte_len"`
		DiffGcsURI string `json:"diff_gcs_uri,omitempty"`
		BeforeHash string `json:"before_sha256,omitempty"`
	}{
		Offloaded:  true,
		SHA256:     afterSum,
		ByteLen:    len(ev.AfterJSON),
		DiffGcsURI: ev.DiffGcsURI,
		BeforeHash: beforeSum,
	})
	if afterOversized || ev.AfterJSON == "" {
		ev.AfterJSON = string(summaryAfter)
	}
	return nil
}

func sha256HexString(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}
