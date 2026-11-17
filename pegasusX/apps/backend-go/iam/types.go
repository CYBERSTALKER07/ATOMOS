// Package iam implements supplier RBAC role templates, user role assignments,
// Segregation of Duties (SoD) policies, four-eyes runtime enforcement, and
// violation/audit recording for ATOMOS M0.4.
package iam

import (
	"context"
	"errors"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

// Canonical SoD violation / RBAC error sentinels.
var (
	ErrSodViolation           = errors.New("sod_violation")
	ErrSodRoleConflict        = errors.New("sod_role_conflict")
	ErrAuditorReadOnly        = errors.New("auditor_read_only")
	ErrAgentApprovalForbidden = errors.New("agent_approval_forbidden")
	ErrMissingPermission      = errors.New("missing_permission")
	ErrPolicyNotFound         = errors.New("sod_policy_not_found")
	ErrInvalidIAMInput        = errors.New("invalid_iam_input")
	ErrIAMUnavailable         = errors.New("iam_unavailable")
	ErrAuditorGrantNotFound   = errors.New("auditor_grant_not_found")
	ErrCustomRoleNotFound     = errors.New("custom_role_not_found")
	ErrCustomRoleExists       = errors.New("custom_role_exists")
	ErrCannotModifyBuiltin    = errors.New("cannot_modify_builtin_role")
	ErrPrivilegeEscalation    = errors.New("privilege_escalation_forbidden")
	ErrScimResourceNotFound   = errors.New("scim_resource_not_found")
	ErrScimConflict           = errors.New("scim_conflict")
	ErrInvalidScimRequest     = errors.New("invalid_scim_request")
)

// Role assignment action values.
const (
	RoleActionAssign  = "ASSIGN"
	RoleActionReplace = "REPLACE"
	RoleActionRevoke  = "REVOKE"
)

// SoD RuleCodes.
const (
	RuleCodeFourEyesSelfApproval    = "FOUR_EYES_SELF_APPROVAL"
	RuleCodeAuditorReadOnly         = "AUDITOR_READ_ONLY"
	RuleCodeAgentApprovalForbidden  = "AGENT_APPROVAL_FORBIDDEN"
	RuleCodeStaticRoleConflict      = "SOD_STATIC_ROLE_CONFLICT"
	RuleCodePermissionPairViolation = "SOD_PERMISSION_PAIR"
)

// RoleDefinition describes one supplier RBAC role template and its effective permissions for a tenant.
type RoleDefinition struct {
	RoleName    string   `json:"role_name"`
	Description string   `json:"description"`
	ReadOnly    bool     `json:"read_only"`
	AgentRole   bool     `json:"agent_role"`
	Permissions []string `json:"permissions"`
}

// PermissionDefinition describes one canonical supplier permission key.
type PermissionDefinition struct {
	Key         string `json:"key"`
	Category    string `json:"category"`
	Description string `json:"description"`
}

// UserRoleAssignment mirrors SupplierUserRoles rows.
type UserRoleAssignment struct {
	TenantID   string     `json:"tenant_id"`
	UserID     string     `json:"user_id"`
	RoleName   string     `json:"role_name"`
	AssignedBy string     `json:"assigned_by"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// AuditorGrant represents a time-boxed read-only AUDITOR portal access assignment.
type AuditorGrant struct {
	TenantID   string    `json:"tenant_id"`
	UserID     string    `json:"user_id"`
	RoleName   string    `json:"role_name"`
	AssignedBy string    `json:"assigned_by"`
	GrantedBy  string    `json:"granted_by"`
	Status     string    `json:"status"`
	ExpiresAt  time.Time `json:"expires_at"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// GrantAuditorAccessInput defines parameters for issuing time-boxed read-only AUDITOR access.
type GrantAuditorAccessInput struct {
	TenantID        string
	UserID          string
	TargetActorType string
	ExpiresAt       *time.Time
	TTLHours        int64
	Reason          string
	GrantedBy       string
	ActorRole       string
	ActorType       string
	TraceID         string
	IP              string
	DeviceID        string
	JWTSecret       string
	JWTIssuer       string
}

// AuditorGrantResult returns the persisted grant metadata and scoped short-lived bearer JWT.
type AuditorGrantResult struct {
	Grant     AuditorGrant `json:"grant"`
	Token     string       `json:"token"`
	ExpiresAt time.Time    `json:"expires_at"`
}

// RevokeAuditorGrantInput defines parameters for revoking a time-boxed AUDITOR grant.
type RevokeAuditorGrantInput struct {
	TenantID  string
	UserID    string
	RevokedBy string
	ActorRole string
	TraceID   string
	IP        string
	DeviceID  string
}

// SodPolicy mirrors SodPolicies rows.
type SodPolicy struct {
	TenantID          string    `json:"tenant_id"`
	PolicyID          string    `json:"policy_id"`
	Name              string    `json:"name"`
	PermissionA       string    `json:"permission_a"`
	PermissionB       string    `json:"permission_b"`
	EnforceSameObject bool      `json:"enforce_same_object"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"created_at"`
}

// SodViolation mirrors append-only SodViolations rows.
type SodViolation struct {
	TenantID    string    `json:"tenant_id"`
	ViolationID string    `json:"violation_id"`
	ActorID     string    `json:"actor_id"`
	Action      string    `json:"action"`
	ObjectType  string    `json:"object_type,omitempty"`
	ObjectID    string    `json:"object_id,omitempty"`
	MakerID     string    `json:"maker_id,omitempty"`
	RuleCode    string    `json:"rule_code"`
	RoutePath   string    `json:"route_path,omitempty"`
	TraceID     string    `json:"trace_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// CheckFourEyesInput carries context for runtime SoD and actor-constraint validation.
type CheckFourEyesInput struct {
	TenantID    string
	ActorID     string
	ActorType   string
	Role        string
	Claims      *auth.Claims
	Action      string
	ObjectType  string
	ObjectID    string
	MakerID     string
	MakerAction string
	RuleCode    string
	RoutePath   string
	TraceID     string
	IP          string
	DeviceID    string
	IsApproval  bool
}

// AssignRoleInput defines a role grant/replace/revoke mutation.
type AssignRoleInput struct {
	TenantID   string
	UserID     string
	RoleName   string
	Action     string // ASSIGN (default) | REPLACE | REVOKE
	AssignedBy string
	ActorRole  string
	TraceID    string
	IP         string
	DeviceID   string
}

// UpdatePolicyInput defines a mutable update to an SoD policy.
type UpdatePolicyInput struct {
	TenantID          string
	PolicyID          string
	Name              *string
	PermissionA       *string
	PermissionB       *string
	EnforceSameObject *bool
	Enabled           *bool
	ActorID           string
	ActorRole         string
	TraceID           string
	IP                string
	DeviceID          string
}

// AuditRecordInput is a package-local DTO passed to an optional AuditRecorder.
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
	ApprovalID string
	TraceID    string
	IP         string
	DeviceID   string
}

// AuditRecorder bridges IAM mutations and SoD violations into hash-chained AuditEvents.
type AuditRecorder interface {
	RecordEvent(ctx context.Context, ev AuditRecordInput) error
	RecordEventInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, ev AuditRecordInput) error
}

// CustomRole mirrors CustomRoles rows and synced RolePermissions grants.
type CustomRole struct {
	TenantID    string    `json:"tenant_id"`
	RoleName    string    `json:"role_name"`
	Description string    `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	ReadOnly    bool      `json:"read_only"`
	CreatedBy   string    `json:"created_by"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateCustomRoleInput defines parameters for creating or updating an Enterprise Custom Role.
type CreateCustomRoleInput struct {
	TenantID     string
	RoleName     string
	Description  string
	Permissions  []string
	ReadOnly     bool
	CreatedBy    string
	ActorRole    string
	ActorType    string
	CallerClaims *auth.Claims
	TraceID      string
	IP           string
	DeviceID     string
}

// DeleteCustomRoleInput defines parameters for deleting an Enterprise Custom Role.
type DeleteCustomRoleInput struct {
	TenantID  string
	RoleName  string
	DeletedBy string
	ActorRole string
	ActorType string
	TraceID   string
	IP        string
	DeviceID  string
}

// SCIM 2.0 canonical schema URNs (RFC 7643 / RFC 7644).
const (
	ScimSchemaUser         = "urn:ietf:params:scim:schemas:core:2.0:User"
	ScimSchemaGroup        = "urn:ietf:params:scim:schemas:core:2.0:Group"
	ScimSchemaListResponse = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	ScimSchemaPatchOp      = "urn:ietf:params:scim:api:messages:2.0:PatchOp"
	ScimSchemaError        = "urn:ietf:params:scim:api:messages:2.0:Error"

	ScimResourceTypeUser  = "User"
	ScimResourceTypeGroup = "Group"
)

// ScimEmail represents a SCIM 2.0 multi-valued email entry.
type ScimEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// ScimRoleRef represents a SCIM 2.0 role reference on a User resource.
type ScimRoleRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

// ScimMemberRef represents a SCIM 2.0 member reference on a Group resource.
type ScimMemberRef struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Type    string `json:"type,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// ScimMeta captures SCIM 2.0 resource metadata.
type ScimMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

// ScimUser represents a SCIM 2.0 Core User payload.
type ScimUser struct {
	Schemas     []string      `json:"schemas"`
	ID          string        `json:"id"`
	ExternalID  string        `json:"externalId,omitempty"`
	UserName    string        `json:"userName"`
	DisplayName string        `json:"displayName,omitempty"`
	Emails      []ScimEmail   `json:"emails,omitempty"`
	Active      bool          `json:"active"`
	Roles       []ScimRoleRef `json:"roles,omitempty"`
	Meta        ScimMeta      `json:"meta"`
}

// ScimGroup represents a SCIM 2.0 Core Group payload.
type ScimGroup struct {
	Schemas     []string        `json:"schemas"`
	ID          string          `json:"id"`
	ExternalID  string          `json:"externalId,omitempty"`
	DisplayName string          `json:"displayName"`
	Members     []ScimMemberRef `json:"members,omitempty"`
	Roles       []ScimRoleRef   `json:"roles,omitempty"`
	Meta        ScimMeta        `json:"meta"`
}

// ScimListResponse represents a SCIM 2.0 ListResponse envelope.
type ScimListResponse struct {
	Schemas      []string `json:"schemas"`
	TotalResults int      `json:"totalResults"`
	StartIndex   int      `json:"startIndex"`
	ItemsPerPage int      `json:"itemsPerPage"`
	Resources    []any    `json:"Resources"`
}

// ScimPatchOperation represents a single RFC 7644 PATCH operation.
type ScimPatchOperation struct {
	Op    string `json:"op"`
	Path  string `json:"path,omitempty"`
	Value any    `json:"value,omitempty"`
}

// ScimPatchRequest represents a SCIM 2.0 PatchOp request payload.
type ScimPatchRequest struct {
	Schemas    []string             `json:"schemas,omitempty"`
	Operations []ScimPatchOperation `json:"Operations"`
}

// ScimResourceRecord is the persisted row shape for ScimIdentities.
type ScimResourceRecord struct {
	TenantID     string
	ResourceType string
	ResourceID   string
	ExternalID   string
	UserName     string
	DisplayName  string
	Email        string
	Active       bool
	Roles        []ScimRoleRef
	Members      []ScimMemberRef
	Meta         ScimMeta
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
