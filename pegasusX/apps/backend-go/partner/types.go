// Package partner implements the Gate-3 Partner Integration Layer:
// machine identity (API keys), /partner/v1 HTTP surfaces, and outbound webhooks.
package partner

import (
	"context"
	"time"
)

const (
	TenantRetailer = "RETAILER"
	TenantSupplier = "SUPPLIER"

	KeyStatusActive  = "ACTIVE"
	KeyStatusRevoked = "REVOKED"

	ScopeOrdersRead     = "orders:read"
	ScopeOrdersWrite    = "orders:write"
	ScopeCatalogRead    = "catalog:read"
	ScopeInventoryRead  = "inventory:read"
	ScopeWebhooksManage = "webhooks:manage"
	ScopeExportsRead    = "exports:read"

	DeliveryPending = "PENDING"
	DeliverySuccess = "SUCCESS"
	DeliveryFailed  = "FAILED"
	DeliveryDead    = "DEAD"

	MaxDeliveryAttempts = 8

	ExportStatusPending   = "PENDING"
	ExportStatusRunning   = "RUNNING"
	ExportStatusSucceeded = "SUCCEEDED"
	ExportStatusFailed    = "FAILED"

	SftpStatusSkipped  = "SKIPPED"
	SftpStatusUploaded = "UPLOADED"
	SftpStatusFailed   = "FAILED"

	ExportResourceOrders    = "orders"
	ExportResourceInvoices  = "invoices"
	ExportResourceInventory = "inventory"
	ExportResourceLedger    = "ledger"
	ExportResourceJournals  = "journals"

	ExportFormatCSV  = "csv"
	ExportFormatJSON = "json"
	ExportFormatXML  = "xml"

	MaxExportRows       = 50000
	MaxExportWindowDays = 90

	EdiDirectionIn  = "IN"
	EdiDirectionOut = "OUT"

	EdiDocORDERS = "ORDERS"
	EdiDocORDRSP = "ORDRSP"
	EdiDocDESADV = "DESADV"
	EdiDocINVOIC = "INVOIC"

	EdiStatusReceived  = "RECEIVED"
	EdiStatusProcessed = "PROCESSED"
	EdiStatusFailed    = "FAILED"
	EdiStatusEmitted   = "EMITTED"
)

// Principal is the authenticated partner key context (request-scoped tenancy).
type Principal struct {
	KeyID      string
	TenantType string
	TenantID   string
	Scopes     []string
}

// ApiKey is the persisted key row (never includes plaintext secret).
type ApiKey struct {
	KeyID          string
	TenantType     string
	TenantID       string
	KeyPrefix      string
	KeyHash        string
	Scopes         []string
	RateLimitClass string
	Status         string
	ExpiresAt      *time.Time
	CreatedBy      string
	CreatedAt      time.Time
	LastUsedAt     *time.Time
}

// IssuedKey is returned once on create.
type IssuedKey struct {
	KeyID      string   `json:"key_id"`
	TenantType string   `json:"tenant_type"`
	TenantID   string   `json:"tenant_id"`
	Scopes     []string `json:"scopes"`
	Prefix     string   `json:"key_prefix"`
	Secret     string   `json:"secret"` // shown once: pxk_<prefix>_<secret>
	ExpiresAt  *string  `json:"expires_at,omitempty"`
}

// WebhookSubscription is a tenant outbound hook.
type WebhookSubscription struct {
	SubscriptionID string
	TenantType     string
	TenantID       string
	URL            string
	SigningSecret  string
	EventTypes     []string
	IsActive       bool
	CreatedAt      time.Time
}

// DeliveryAttempt tracks one signed POST.
type DeliveryAttempt struct {
	AttemptID      string
	SubscriptionID string
	EventID        string
	EventType      string
	PayloadJSON    []byte
	Status         string
	HTTPCode       int64
	NextRetryAt    *time.Time
	AttemptCount   int64
	LastError      string
	CreatedAt      time.Time
}

// KeyRepository persists partner API keys.
type KeyRepository interface {
	Insert(ctx context.Context, k ApiKey) error
	GetByPrefix(ctx context.Context, prefix string) (ApiKey, bool, error)
	GetByID(ctx context.Context, keyID string) (ApiKey, bool, error)
	ListByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]ApiKey, error)
	Revoke(ctx context.Context, keyID, tenantType, tenantID string) error
	TouchLastUsed(ctx context.Context, keyID string) error
}

// WebhookRepository persists subscriptions and delivery attempts.
type WebhookRepository interface {
	InsertSubscription(ctx context.Context, s WebhookSubscription) error
	ListSubscriptions(ctx context.Context, tenantType, tenantID string) ([]WebhookSubscription, error)
	ListActiveByEvent(ctx context.Context, eventType string) ([]WebhookSubscription, error)
	GetSubscription(ctx context.Context, id string) (WebhookSubscription, bool, error)
	DeactivateSubscription(ctx context.Context, id, tenantType, tenantID string) error

	InsertAttempt(ctx context.Context, a DeliveryAttempt) error
	GetAttemptBySubEvent(ctx context.Context, subID, eventID string) (DeliveryAttempt, bool, error)
	GetAttempt(ctx context.Context, attemptID string) (DeliveryAttempt, bool, error)
	ListDueAttempts(ctx context.Context, now time.Time, limit int) ([]DeliveryAttempt, error)
	UpdateAttempt(ctx context.Context, a DeliveryAttempt) error
	ListDeadByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]DeliveryAttempt, error)
}

// ExportJob is one async partner bulk export.
type ExportJob struct {
	JobID      string
	TenantType string
	TenantID   string
	Resource   string
	Format     string
	Status     string
	FromDate   *time.Time
	ToDate     *time.Time
	ObjectPath string
	RowCount   int64
	Error      string
	SftpStatus string
	CreatedAt  time.Time
	FinishedAt *time.Time
}

// SftpConfig is tenant SFTP destination (secret never stored in plaintext).
type SftpConfig struct {
	TenantType  string
	TenantID    string
	Host        string
	Port        int64
	Username    string
	SecretRef   string
	RemoteDir   string
	IsActive    bool
	InboundDir  string
	OutboundDir string
	ArchiveDir  string
	EdiEnabled  bool
	UpdatedAt   time.Time
}

// EdiDocument is one inbound or outbound EDI-lite file ledger row.
type EdiDocument struct {
	DocumentID    string
	TenantType    string
	TenantID      string
	Direction     string
	DocType       string
	ExternalDocID string
	OrderID       string
	Status        string
	ObjectPath    string
	RemoteName    string
	Error         string
	PayloadHash   string
	CreatedAt     time.Time
	FinishedAt    *time.Time
}

// ExportRepository persists export jobs.
type ExportRepository interface {
	InsertJob(ctx context.Context, j ExportJob) error
	GetJob(ctx context.Context, jobID string) (ExportJob, bool, error)
	ListJobs(ctx context.Context, tenantType, tenantID string, limit int) ([]ExportJob, error)
	ListPending(ctx context.Context, limit int) ([]ExportJob, error)
	UpdateJob(ctx context.Context, j ExportJob) error
}

// SftpConfigRepository persists SFTP destinations.
type SftpConfigRepository interface {
	Upsert(ctx context.Context, c SftpConfig) error
	Get(ctx context.Context, tenantType, tenantID string) (SftpConfig, bool, error)
	ListEdiEnabled(ctx context.Context, limit int) ([]SftpConfig, error)
}

// As2Config is per-tenant AS2 station metadata (cert PEMs via SecretRef only).
type As2Config struct {
	TenantType           string
	TenantID             string
	As2Enabled           bool
	OurAs2Id             string
	PartnerAs2Id         string
	PartnerURL           string
	OurCertSecretRef     string
	OurKeySecretRef      string
	PartnerCertSecretRef string
	SignRequired         bool
	EncryptRequired      bool
	UpdatedAt            time.Time
}

// As2ConfigRepository persists AS2 station configs.
type As2ConfigRepository interface {
	Upsert(ctx context.Context, c As2Config) error
	Get(ctx context.Context, tenantType, tenantID string) (As2Config, bool, error)
	GetByOurAs2Id(ctx context.Context, ourAs2Id string) (As2Config, bool, error)
}

// EdiDocumentRepository persists EDI-lite document ledger rows.
type EdiDocumentRepository interface {
	Insert(ctx context.Context, d EdiDocument) error
	Get(ctx context.Context, documentID string) (EdiDocument, bool, error)
	GetByExternal(ctx context.Context, tenantType, tenantID, direction, docType, externalDocID string) (EdiDocument, bool, error)
	Update(ctx context.Context, d EdiDocument) error
	ListByTenant(ctx context.Context, tenantType, tenantID string, limit int) ([]EdiDocument, error)
	ListPendingOutbound(ctx context.Context, limit int) ([]EdiDocument, error)
}
