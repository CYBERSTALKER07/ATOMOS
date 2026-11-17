// Package connectors implements the ATOMOS M1.9 Enterprise Connector Spine:
// common Connector contract, fail-closed SecretResolver, Billz read-only HTTP
// client, 1C CommerceML multi-account adapter, and Spanner-vs-external ShadowCompare.
package connectors

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderBillz    = "billz"
	ProviderOnec     = "onec"
	ProviderMoySklad = "moysklad"
	ProviderSmartup  = "smartup"
	ProviderIiko     = "iiko"
)

const (
	AuthTypeBearer = "bearer"
	AuthTypeAPIKey = "api_key"
	AuthTypeBasic  = "basic"
	AuthTypeOAuth2 = "oauth2"
	AuthTypeNone   = "none"
)

const (
	DirectionHealth        = "HEALTH"
	DirectionPull          = "PULL"
	DirectionPush          = "PUSH"
	DirectionShadowCompare = "SHADOW_COMPARE"
)

const (
	RunStatusCompleted     = "COMPLETED"
	RunStatusFailed        = "FAILED"
	RunStatusMismatchFound = "MISMATCH_FOUND"
)

const (
	CredStatusActive   = "ACTIVE"
	CredStatusDisabled = "DISABLED"
)

const (
	StreamAll      = "all"
	StreamProducts = "products"
	StreamStocks   = "stocks"
	StreamShops    = "shops"
	StreamOrders   = "orders"
	StreamClients  = "clients"
)

const (
	MismatchMissingExternal   = "missing_external"
	MismatchMissingInternal   = "missing_internal"
	MismatchStockDelta        = "stock_delta"
	MismatchBalanceDeltaMinor = "balance_delta_minor"
)

const (
	ObjectTypeProduct   = "Product"
	ObjectTypeInventory = "SupplierInventoryV2"
	ObjectTypeShop      = "Shop"
	ObjectTypeOrder     = "Order"
	ObjectTypeInvoice   = "ArInvoice"
	ObjectTypeClient    = "Client"
)

var (
	ErrInvalidInput            = errors.New("connectors: invalid input")
	ErrFloatForbidden          = errors.New("connectors: float values are forbidden on money/counts")
	ErrCredentialNotFound      = errors.New("connectors: credential not found")
	ErrRunNotFound             = errors.New("connectors: sync run not found")
	ErrUnsupportedProvider     = errors.New("connectors: unsupported provider")
	ErrConnectorNotConfigured  = errors.New("connectors: external connector url or secret not configured")
	ErrSecretNotFound          = errors.New("connectors: secret ref could not be resolved")
	ErrConnectorAuthFailed     = errors.New("connectors: upstream authentication failed")
	ErrConnectorTimeout        = errors.New("connectors: upstream request timed out")
	ErrConnectorUpstreamError  = errors.New("connectors: upstream request failed")
	ErrConnectorBadPayload     = errors.New("connectors: malformed upstream payload")
	ErrConnectorReadOnly       = errors.New("connectors: connector is strictly read-only")
	ErrApprovalRequired        = errors.New("connectors: verified human approval_id is required for push write-back")
	ErrApprovalPayloadMismatch = errors.New("connectors: approval payload hash mismatch")
	ErrAuditorReadOnly         = errors.New("connectors: AUDITOR role is strictly read-only")
	ErrAgentForbidden          = errors.New("connectors: AGENT actor is forbidden from connector write-backs")
	ErrRepoUnavailable         = errors.New("connectors: repository unavailable")
)

// Connector is the canonical M1.9 external ERP/POS integration contract.
type Connector interface {
	Provider() string
	Health(ctx context.Context, cred ExternalCredential) (HealthResult, error)
	Pull(ctx context.Context, cred ExternalCredential, req PullRequest) (PullResult, error)
	Push(ctx context.Context, cred ExternalCredential, req PushRequest) (PushResult, error)
	MapSchema(ctx context.Context, stream string, rawRows []map[string]any) ([]CanonicalRecord, error)
}

// AdapterDescriptor describes a registered connector adapter without exposing secrets.
type AdapterDescriptor struct {
	Provider     string   `json:"provider"`
	DisplayName  string   `json:"display_name"`
	Capabilities []string `json:"capabilities"`
	ReadOnly     bool     `json:"read_only"`
}

// ExternalCredential mirrors the ExternalCredentials Spanner table.
// SecretRef is tagged json:"-" so raw references are never serialized directly.
type ExternalCredential struct {
	TenantID     string         `json:"tenant_id"`
	CredentialID string         `json:"credential_id"`
	Provider     string         `json:"provider"`
	AccountLabel string         `json:"account_label"`
	AuthType     string         `json:"auth_type"`
	SecretRef    string         `json:"-"`
	BaseURL      string         `json:"base_url,omitempty"`
	MetadataJSON string         `json:"-"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	Status       string         `json:"status"`
	CreatedBy    string         `json:"created_by"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

// ConnectorCredentialSummary is the safe read projection returned by HTTP APIs.
type ConnectorCredentialSummary struct {
	CredentialID    string         `json:"credential_id"`
	TenantID        string         `json:"tenant_id"`
	Provider        string         `json:"provider"`
	AccountLabel    string         `json:"account_label"`
	AuthType        string         `json:"auth_type"`
	SecretRefMasked string         `json:"secret_ref_masked"`
	BaseURL         string         `json:"base_url,omitempty"`
	Metadata        map[string]any `json:"metadata,omitempty"`
	Status          string         `json:"status"`
	CreatedBy       string         `json:"created_by"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// Summary converts ExternalCredential into a masked read model with zero raw secrets.
func (c ExternalCredential) Summary() ConnectorCredentialSummary {
	return ConnectorCredentialSummary{
		CredentialID:    c.CredentialID,
		TenantID:        c.TenantID,
		Provider:        c.Provider,
		AccountLabel:    c.AccountLabel,
		AuthType:        c.AuthType,
		SecretRefMasked: MaskSecretRef(c.SecretRef),
		BaseURL:         c.BaseURL,
		Metadata:        c.Metadata,
		Status:          c.Status,
		CreatedBy:       c.CreatedBy,
		CreatedAt:       c.CreatedAt,
		UpdatedAt:       c.UpdatedAt,
	}
}

// HealthResult summarizes a live connectivity check against an external connector.
type HealthResult struct {
	Healthy      bool              `json:"healthy"`
	Provider     string            `json:"provider"`
	AccountLabel string            `json:"account_label"`
	LatencyMs    int64             `json:"latency_ms"`
	StatusCode   int               `json:"status_code"`
	Message      string            `json:"message"`
	CheckedAt    time.Time         `json:"checked_at"`
	Run          *ConnectorSyncRun `json:"run,omitempty"`
}

// PullRequest configures an inbound external data pull.
type PullRequest struct {
	Stream     string `json:"stream,omitempty"`
	RawPayload []byte `json:"-"`
	Limit      int64  `json:"limit,omitempty"`
}

// PullResult holds the raw external stream rows and normalized canonical records.
type PullResult struct {
	Provider     string                      `json:"provider"`
	AccountLabel string                      `json:"account_label"`
	Stream       string                      `json:"stream"`
	Streams      map[string][]map[string]any `json:"streams,omitempty"`
	Canonical    []CanonicalRecord           `json:"canonical,omitempty"`
	RecordsCount int64                       `json:"records_count"`
	RawCount     int64                       `json:"raw_count,omitempty"`
	StreamCounts map[string]int64            `json:"stream_counts"`
}

// PushRequest configures an approval-gated outbound write-back push.
type PushRequest struct {
	Stream     string            `json:"stream,omitempty"`
	ApprovalID string            `json:"approval_id,omitempty"`
	Records    []CanonicalRecord `json:"records,omitempty"`
	RawPayload []byte            `json:"-"`
}

// PushApprovalInput passes push metadata and canonical payload to approvals.VerifyApprovedRequest.
type PushApprovalInput struct {
	TenantID   string
	Provider   string
	ApprovalID string
	Stream     string
	Payload    map[string]any
	ActorID    string
	ActorType  string
	ActorRole  string
}

// PushApprovalChecker verifies that an approval_id is APPROVED by an authorized human approver
// and matches the canonical push payload.
type PushApprovalChecker interface {
	VerifyPushApproval(ctx context.Context, in PushApprovalInput) error
}

// PushResponse is returned by Service.Push and POST /v1/connectors/{provider}/push.
type PushResponse struct {
	Run        ConnectorSyncRun `json:"run"`
	PushResult PushResult       `json:"push_result"`
	ApprovalID string           `json:"approval_id"`
}

// PushResult summarizes an outbound connector push.
type PushResult struct {
	Provider      string `json:"provider"`
	AccountLabel  string `json:"account_label"`
	Stream        string `json:"stream"`
	RecordsPushed int64  `json:"records_pushed"`
	ExternalDocID string `json:"external_doc_id,omitempty"`
	Status        string `json:"status"`
	Message       string `json:"message"`
}

// CanonicalRecord is the integer-only canonical object extracted from external rows.
type CanonicalRecord struct {
	ObjectType     string         `json:"object_type"`
	ExternalID     string         `json:"external_id"`
	SKU            string         `json:"sku,omitempty"`
	Barcode        string         `json:"barcode,omitempty"`
	Name           string         `json:"name,omitempty"`
	WarehouseID    string         `json:"warehouse_id,omitempty"`
	OrderID        string         `json:"order_id,omitempty"`
	RetailerID     string         `json:"retailer_id,omitempty"`
	Currency       string         `json:"currency,omitempty"`
	Status         string         `json:"status,omitempty"`
	QuantityOnHand int64          `json:"quantity_on_hand"`
	PriceMinor     int64          `json:"price_minor"`
	TotalMinor     int64          `json:"total_minor"`
	BalanceMinor   int64          `json:"balance_minor"`
	ContentHash    string         `json:"content_hash"`
	Extra          map[string]any `json:"extra,omitempty"`
}

// ConnectorSyncRun mirrors the ConnectorSyncRuns Spanner table.
type ConnectorSyncRun struct {
	TenantID            string                  `json:"tenant_id"`
	RunID               string                  `json:"run_id"`
	Provider            string                  `json:"provider"`
	AccountLabel        string                  `json:"account_label"`
	Direction           string                  `json:"direction"`
	StreamName          string                  `json:"stream_name"`
	Status              string                  `json:"status"`
	RecordsProcessed    int64                   `json:"records_processed"`
	MismatchCount       int64                   `json:"mismatch_count"`
	MismatchSummaryJSON string                  `json:"mismatch_summary_json,omitempty"`
	Summary             *ShadowCompareSummary   `json:"summary,omitempty"`
	Mismatches          []ShadowCompareMismatch `json:"mismatches,omitempty"`
	ErrorMessage        string                  `json:"error_message,omitempty"`
	IdempotencyKey      string                  `json:"idempotency_key,omitempty"`
	StartedAt           time.Time               `json:"started_at"`
	CompletedAt         *time.Time              `json:"completed_at,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
}

// ConnectorEntityMap mirrors the ConnectorEntityMaps Spanner table.
type ConnectorEntityMap struct {
	TenantID     string    `json:"tenant_id"`
	Provider     string    `json:"provider"`
	ObjectType   string    `json:"object_type"`
	ExternalID   string    `json:"external_id"`
	InternalID   string    `json:"internal_id"`
	ContentHash  string    `json:"content_hash,omitempty"`
	LastRunID    string    `json:"last_run_id,omitempty"`
	LastSyncedAt time.Time `json:"last_synced_at"`
}

// ShadowCompareSummary holds deterministic integer counts and deltas comparing external vs Spanner state.
type ShadowCompareSummary struct {
	Provider             string `json:"provider"`
	AccountLabel         string `json:"account_label"`
	RunID                string `json:"run_id"`
	ExternalProducts     int64  `json:"external_products"`
	InternalProducts     int64  `json:"internal_products"`
	ExternalStocks       int64  `json:"external_stocks"`
	InternalStocks       int64  `json:"internal_stocks"`
	ExternalOrders       int64  `json:"external_orders"`
	InternalOrders       int64  `json:"internal_orders"`
	ExternalClients      int64  `json:"external_clients"`
	InternalInvoices     int64  `json:"internal_invoices"`
	MatchedCount         int64  `json:"matched_count"`
	MissingExternal      int64  `json:"missing_external"`
	MissingInternal      int64  `json:"missing_internal"`
	StockDeltaCount      int64  `json:"stock_delta_count"`
	BalanceDeltaCount    int64  `json:"balance_delta_count"`
	TotalMismatches      int64  `json:"total_mismatches"`
	NetStockDelta        int64  `json:"net_stock_delta"`
	NetBalanceDeltaMinor int64  `json:"net_balance_delta_minor"`
	Currency             string `json:"currency"`
}

// ShadowCompareMismatch represents one deterministic discrepancy between external and Spanner state.
type ShadowCompareMismatch struct {
	MismatchType        string `json:"mismatch_type"`
	ObjectType          string `json:"object_type"`
	ExternalID          string `json:"external_id,omitempty"`
	InternalID          string `json:"internal_id,omitempty"`
	WarehouseID         string `json:"warehouse_id,omitempty"`
	Name                string `json:"name,omitempty"`
	ExternalStock       int64  `json:"external_stock,omitempty"`
	InternalStock       int64  `json:"internal_stock,omitempty"`
	StockDelta          int64  `json:"stock_delta,omitempty"`
	ExternalAmountMinor int64  `json:"external_amount_minor,omitempty"`
	InternalAmountMinor int64  `json:"internal_amount_minor,omitempty"`
	BalanceDeltaMinor   int64  `json:"balance_delta_minor,omitempty"`
	Currency            string `json:"currency,omitempty"`
	Detail              string `json:"detail"`
}

// ShadowCompareEnvelope is persisted in ConnectorSyncRuns.MismatchSummaryJson.
type ShadowCompareEnvelope struct {
	Summary    ShadowCompareSummary    `json:"summary"`
	Mismatches []ShadowCompareMismatch `json:"mismatches"`
}

// InternalProduct is the Spanner Products projection used by ShadowCompare.
type InternalProduct struct {
	ProductID     string
	Name          string
	Barcode       string
	PriceMinor    int64
	StockQuantity int64
	Currency      string
	IsActive      bool
}

// InternalStock is the Spanner SupplierInventoryV2 projection used by ShadowCompare.
type InternalStock struct {
	WarehouseID    string
	ProductID      string
	QuantityOnHand int64
}

// InternalOrder is the Spanner Orders projection used by ShadowCompare.
type InternalOrder struct {
	OrderID    string
	RetailerID string
	TotalMinor int64
	Currency   string
	Status     string
}

// InternalInvoice is the Spanner ArInvoices projection used by ShadowCompare.
type InternalInvoice struct {
	InvoiceID      string
	OrderID        string
	RetailerID     string
	PrincipalMinor int64
	BalanceMinor   int64
	Currency       string
	Status         string
}

// InternalSnapshot holds tenant Spanner state read inside one ReadOnlyTransaction.
type InternalSnapshot struct {
	Products   []InternalProduct
	Stocks     []InternalStock
	Orders     []InternalOrder
	Invoices   []InternalInvoice
	EntityMaps []ConnectorEntityMap
}

// AuditRecordInput mirrors audit.EventInput without creating a package cycle.
type AuditRecordInput struct {
	TenantID   string
	ActorID    string
	ActorType  string
	Role       string
	Action     string
	ObjectType string
	ObjectID   string
	BeforeJSON string
	AfterJSON  string
	ReasonCode string
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder persists immutable AuditEvents records via a bootstrap-wired adapter.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
}

// OntologyLineageInput describes external source_system lineage updates for matched entities.
type OntologyLineageInput struct {
	TenantID   string
	Provider   string
	RunID      string
	SyncedAt   time.Time
	EntityMaps []ConnectorEntityMap
}

// OntologyLineageHook propagates connector source_system lineage to OntologyObjects.
type OntologyLineageHook func(ctx context.Context, in OntologyLineageInput) error

// UpsertCredentialInput defines fields to create or update a tenant ExternalCredentials row.
type UpsertCredentialInput struct {
	TenantID     string         `json:"tenant_id,omitempty"`
	CredentialID string         `json:"credential_id,omitempty"`
	Provider     string         `json:"provider"`
	AccountLabel string         `json:"account_label,omitempty"`
	AuthType     string         `json:"auth_type"`
	SecretRef    string         `json:"secret_ref"`
	BaseURL      string         `json:"base_url,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
	MetadataJSON string         `json:"metadata_json,omitempty"`
	Status       string         `json:"status,omitempty"`
	CreatedBy    string         `json:"-"`
	ActorType    string         `json:"-"`
	ActorRole    string         `json:"-"`
	TraceID      string         `json:"-"`
	IP           string         `json:"-"`
	DeviceID     string         `json:"-"`
}

// TriggerConnectorInput defines fields for health, pull, push, and shadow-compare triggers.
type TriggerConnectorInput struct {
	TenantID        string            `json:"tenant_id,omitempty"`
	Provider        string            `json:"provider"`
	AccountLabel    string            `json:"account_label,omitempty"`
	CredentialID    string            `json:"credential_id,omitempty"`
	Stream          string            `json:"stream,omitempty"`
	ApprovalID      string            `json:"approval_id,omitempty"`
	Records         []CanonicalRecord `json:"records,omitempty"`
	ApprovalPayload map[string]any    `json:"-"`
	RawPayload      []byte            `json:"-"`
	IdempotencyKey  string            `json:"idempotency_key,omitempty"`
	ActorID         string            `json:"-"`
	ActorType       string            `json:"-"`
	ActorRole       string            `json:"-"`
	TraceID         string            `json:"-"`
	IP              string            `json:"-"`
	DeviceID        string            `json:"-"`
}

// ShadowCompareResult is returned by ShadowCompare and POST /v1/connectors/{provider}/shadow-compare.
type ShadowCompareResult struct {
	Run        ConnectorSyncRun        `json:"run"`
	Summary    ShadowCompareSummary    `json:"summary"`
	Mismatches []ShadowCompareMismatch `json:"mismatches"`
}

// PullResponse is returned by POST /v1/connectors/{provider}/pull.
type PullResponse struct {
	Run          ConnectorSyncRun `json:"run"`
	RecordsCount int64            `json:"records_count"`
	StreamCounts map[string]int64 `json:"stream_counts"`
}

// OutboxEmission carries a typed outbox event emitted atomically with ConnectorSyncRuns.
type OutboxEmission struct {
	AggregateType string
	AggregateID   string
	Topic         string
	Event         any
}

func normalizeProvider(raw string) string {
	return strings.ToLower(strings.TrimSpace(raw))
}

func normalizeAccountLabel(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "default"
	}
	return trimmed
}

func normalizeAuthType(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", AuthTypeBearer:
		return AuthTypeBearer
	case AuthTypeAPIKey, "apikey":
		return AuthTypeAPIKey
	case AuthTypeBasic:
		return AuthTypeBasic
	case AuthTypeOAuth2:
		return AuthTypeOAuth2
	case AuthTypeNone:
		return AuthTypeNone
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if t := strings.TrimSpace(v); t != "" {
			return t
		}
	}
	return ""
}

func validateNoFloats(v any) error {
	switch tv := v.(type) {
	case float32, float64:
		return fmt.Errorf("%w: float type %T is not allowed", ErrFloatForbidden, tv)
	case json.Number:
		s := tv.String()
		if strings.ContainsAny(s, ".eE") {
			return fmt.Errorf("%w: decimal/exponent literal %q is not allowed", ErrFloatForbidden, s)
		}
	case map[string]any:
		for _, elem := range tv {
			if err := validateNoFloats(elem); err != nil {
				return err
			}
		}
	case []any:
		for _, elem := range tv {
			if err := validateNoFloats(elem); err != nil {
				return err
			}
		}
	}
	return nil
}

func decodeJSONWithNumbers(raw []byte, dst any) error {
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber()
	return dec.Decode(dst)
}

func parseStrictInt64(v any) (int64, bool) {
	switch tv := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(tv), true
	case int32:
		return int64(tv), true
	case int64:
		return tv, true
	case uint:
		return int64(tv), true
	case uint64:
		return int64(tv), true
	case json.Number:
		s := strings.TrimSpace(tv.String())
		if s == "" {
			return 0, false
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
		if scaled, ok := parseDecimalStringToScaledInt64(s, 0); ok {
			return scaled, true
		}
		return 0, false
	case string:
		s := strings.ReplaceAll(strings.TrimSpace(tv), ",", "")
		s = strings.ReplaceAll(s, " ", "")
		if s == "" {
			return 0, false
		}
		if n, err := strconv.ParseInt(s, 10, 64); err == nil {
			return n, true
		}
		if scaled, ok := parseDecimalStringToScaledInt64(s, 0); ok {
			return scaled, true
		}
		return 0, false
	default:
		return 0, false
	}
}

func parseDecimalToMinorUnits(v any, scale int) (int64, bool) {
	switch tv := v.(type) {
	case nil:
		return 0, false
	case int:
		return int64(tv) * pow10Int64(scale), true
	case int32:
		return int64(tv) * pow10Int64(scale), true
	case int64:
		return tv * pow10Int64(scale), true
	case json.Number:
		return parseDecimalStringToScaledInt64(tv.String(), scale)
	case string:
		return parseDecimalStringToScaledInt64(tv, scale)
	default:
		return parseDecimalStringToScaledInt64(fmt.Sprint(v), scale)
	}
}

func pow10Int64(n int) int64 {
	res := int64(1)
	for i := 0; i < n; i++ {
		res *= 10
	}
	return res
}

// parseDecimalStringToScaledInt64 parses decimal numeric strings without float64 arithmetic.
func parseDecimalStringToScaledInt64(raw string, scale int) (int64, bool) {
	s := strings.ReplaceAll(strings.TrimSpace(raw), ",", "")
	s = strings.ReplaceAll(s, " ", "")
	if s == "" {
		return 0, false
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if s == "" {
		return 0, false
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, false
	}
	wholeStr := parts[0]
	if wholeStr == "" {
		wholeStr = "0"
	}
	fracStr := ""
	if len(parts) == 2 {
		fracStr = parts[1]
	}
	for _, ch := range wholeStr {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	for _, ch := range fracStr {
		if ch < '0' || ch > '9' {
			return 0, false
		}
	}
	for len(fracStr) < scale {
		fracStr += "0"
	}
	if len(fracStr) > scale {
		fracStr = fracStr[:scale]
	}
	combined := wholeStr + fracStr
	val, ok := new(big.Int).SetString(combined, 10)
	if !ok || !val.IsInt64() {
		return 0, false
	}
	out := val.Int64()
	if neg {
		out = -out
	}
	return out, true
}

func sha256HexString(parts ...string) string {
	h := sha256.New()
	for idx, p := range parts {
		if idx > 0 {
			_, _ = h.Write([]byte{'|'})
		}
		_, _ = h.Write([]byte(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}
