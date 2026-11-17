// Package bqexport implements the typed BigQuery analytical sink, canonical
// domain-to-analytical table mappers, Spanner offset checkpoints, and the
// Kafka sink consumer (`void-bq-sink`).
package bqexport

import (
	"context"
	"errors"
	"time"
)

// Canonical BigQuery table identifiers matching infra/terraform/modules/analytics.
const (
	TableRawDomainEvents               = "raw.domain_events"
	TableRawOntologyObjects            = "raw.ontology_objects"
	TableCuratedGLEntries              = "curated.gl_entries"
	TableCuratedGLPostings             = "curated.gl_postings"
	TableCuratedReconRuns              = "curated.recon_runs"
	TableFeaturesSKUDemandDaily        = "features.sku_demand_daily"
	TableFeaturesRetailerCreditSignals = "features.retailer_credit_signals"
)

var (
	// ErrBigQueryConfigRequired indicates missing BigQuery project or credentials in fail-closed mode.
	ErrBigQueryConfigRequired = errors.New("bqexport: bigquery project_id and credentials required")
	// ErrLocalSpoolDisallowedInProduction indicates local JSONL spool was requested in production.
	ErrLocalSpoolDisallowedInProduction = errors.New("bqexport: local JSONL spool sink disallowed in production")
	// ErrMalformedEventPayload indicates an invalid or non-JSON Kafka payload that should route to DLQ.
	ErrMalformedEventPayload = errors.New("bqexport: malformed JSON event payload")
	// ErrInvalidRowTable indicates an invalid or unsupported destination table name.
	ErrInvalidRowTable = errors.New("bqexport: invalid destination table (expected dataset.table)")
	// ErrInsertRowsRejected indicates BigQuery streaming insert returned per-row insertErrors.
	ErrInsertRowsRejected = errors.New("bqexport: bigquery rejected one or more rows")
)

// Row represents a single idempotent record destined for a BigQuery dataset.table.
type Row struct {
	Table    string         `json:"table"`
	InsertID string         `json:"insert_id"`
	Data     map[string]any `json:"data"`
}

// HealthStatus describes adapter liveness and target coordinates.
type HealthStatus struct {
	Adapter   string    `json:"adapter"`
	Healthy   bool      `json:"healthy"`
	ProjectID string    `json:"project_id,omitempty"`
	SpoolDir  string    `json:"spool_dir,omitempty"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Sink is the typed analytical sink contract implemented by Cloud BigQuery and local JSONL adapters.
type Sink interface {
	WriteBatch(ctx context.Context, rows []Row) error
	Health(ctx context.Context) (HealthStatus, error)
}

// Checkpoint tracks the highest monotonically exported Kafka offset per (SinkId, Topic, PartitionNum).
type Checkpoint struct {
	SinkID       string    `json:"sink_id"`
	Topic        string    `json:"topic"`
	PartitionNum int64     `json:"partition_num"`
	OffsetVal    int64     `json:"offset_val"`
	LastEventID  string    `json:"last_event_id,omitempty"`
	RowsExported int64     `json:"rows_exported"`
	RowsDelta    int64     `json:"-"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CheckpointStore persists and monotonically advances Kafka sink offsets.
type CheckpointStore interface {
	GetCheckpoint(ctx context.Context, sinkID, topic string, partition int64) (Checkpoint, bool, error)
	SaveCheckpoint(ctx context.Context, cp Checkpoint) error
	ListCheckpoints(ctx context.Context, sinkID string) ([]Checkpoint, error)
}

// DomainEventInput is the canonical typed input for raw.domain_events.
type DomainEventInput struct {
	EventID       string
	EventType     string
	AggregateType string
	AggregateID   string
	CellID        string
	SupplierID    string
	Topic         string
	PartitionNum  int64
	KafkaOffset   int64
	TraceID       string
	SchemaVersion string
	PayloadJSON   string
	OccurredAt    time.Time
	IngestedAt    time.Time
}

// OntologyObjectInput is the canonical typed input for raw.ontology_objects.
type OntologyObjectInput struct {
	CellID           string
	ObjectType       string
	ObjectID         string
	SupplierID       string
	Version          int64
	ChangeKind       string
	SourceEventType  string
	SourceEventID    string
	SecurityMarkings []string
	PropertiesJSON   string
	ValidFrom        *time.Time
	UpdatedAt        time.Time
	IngestedAt       time.Time
}

// GLEntryInput is the canonical typed input for curated.gl_entries (minor units INT64 only).
type GLEntryInput struct {
	EntityID         string
	EntryID          string
	PeriodID         string
	SourceEventID    string
	SourceType       string
	SourceID         string
	OrderID          string
	Currency         string
	CountryPack      string
	TotalDebitMinor  int64
	TotalCreditMinor int64
	ReversesEntryID  string
	PrevHash         string
	EntryHash        string
	PostedBy         string
	OccurredAt       *time.Time
	PostedAt         time.Time
	IngestedAt       time.Time
}

// SKUDemandDailyInput is the canonical typed input for features.sku_demand_daily (minor units INT64).
type SKUDemandDailyInput struct {
	FeatureDate    string // YYYY-MM-DD
	CellID         string
	SupplierID     string
	WarehouseID    string
	ProductID      string
	SKU            string
	Currency       string
	UnitsOrdered   int64
	UnitsDelivered int64
	RevenueMinor   int64
	BaselineQty    *int64
	ConfidencePct  *int64
	ADI            *float64
	CV2            *float64
	DemandCategory string
	StockoutEvents *int64
	UpdatedAt      time.Time
}

// RetailerCreditSignalInput is the canonical typed input for features.retailer_credit_signals (minor units INT64).
type RetailerCreditSignalInput struct {
	SignalDate          string // YYYY-MM-DD
	CellID              string
	SupplierID          string
	RetailerID          string
	Currency            string
	CreditLimitMinor    int64
	CurrentBalanceMinor int64
	OverdueBalanceMinor int64
	OpenInvoiceCount    int64
	LatePayments60d     int64
	PromiseBreaches90d  int64
	RiskTier            string
	Delinquent          bool
	OnHold              bool
	UpdatedAt           time.Time
}
