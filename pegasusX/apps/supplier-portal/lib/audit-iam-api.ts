import type { ApiClient } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";

function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && String(v).trim() !== "") {
      params.set(k, String(v).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

export type AuditActorType = "human" | "agent" | "connector" | "system";

export interface AuditEvent {
  tenant_id: string;
  day_bucket: string;
  event_id: string;
  seq_num: number;
  actor_id: string;
  actor_type: AuditActorType;
  role?: string;
  action: string;
  object_type?: string;
  object_id?: string;
  before_json?: string;
  after_json?: string;
  diff_gcs_uri?: string;
  reason_code?: string;
  approval_id?: string;
  trace_id?: string;
  ip?: string;
  device_id?: string;
  prev_hash: string;
  entry_hash: string;
  legal_hold: boolean;
  legal_hold_reason?: string;
  retention_until?: string;
  created_at: string;
  source?: string;
  pii_masked?: boolean;
}

export interface AuditEventsResponse {
  tenant_id: string;
  pii_masked: boolean;
  count: number;
  events: AuditEvent[];
}

export interface ObjectHistoryResponse {
  tenant_id: string;
  object_type: string;
  object_id: string;
  pii_masked: boolean;
  count: number;
  events: AuditEvent[];
}

export interface ChainVerifyResult {
  tenant_id: string;
  day_bucket: string;
  valid: boolean;
  event_count: number;
  head_event_id?: string;
  root_hash: string;
  anchored_root_hash?: string;
  anchor_matched?: boolean;
  broken_event_id?: string;
  failure_reason?: string;
}

export interface AuditAnchor {
  tenant_id: string;
  day_bucket: string;
  head_event_id: string;
  event_count: number;
  root_hash: string;
  worm_uri?: string;
  signer_key_id?: string;
  anchored_by: string;
  anchored_at: string;
}

export interface AuditAnchorsResponse {
  tenant_id: string;
  count: number;
  anchors: AuditAnchor[];
}

export interface WriteAnchorInput {
  entity_id?: string;
  date?: string;
  day_bucket?: string;
}

export interface AuditLegalHold {
  tenant_id: string;
  hold_id: string;
  object_type: string;
  object_id: string;
  reason: string;
  status: "ACTIVE" | "RELEASED" | string;
  created_by: string;
  expires_at?: string;
  created_at: string;
}

export interface AuditLegalHoldsResponse {
  tenant_id: string;
  count: number;
  holds: AuditLegalHold[];
}

export interface CreateLegalHoldInput {
  entity_id?: string;
  object_type: string;
  object_id: string;
  reason: string;
  expires_at?: string;
}

export interface IamRoleDefinition {
  role_name: string;
  description: string;
  read_only: boolean;
  agent_role: boolean;
  permissions: string[];
}

export interface IamPermissionDefinition {
  key: string;
  category: string;
  description: string;
}

export interface IamUserRoleAssignment {
  tenant_id: string;
  user_id: string;
  role_name: string;
  assigned_by: string;
  created_at: string;
  updated_at: string;
}

export interface IamRolesResponse {
  tenant_id: string;
  roles: IamRoleDefinition[];
  user_roles: IamUserRoleAssignment[];
}

export interface IamPermissionsResponse {
  permissions: IamPermissionDefinition[];
  count: number;
}

export type RoleAssignmentAction = "ASSIGN" | "REPLACE" | "REVOKE";

export interface AssignUserRoleInput {
  role_name: string;
  action?: RoleAssignmentAction;
  entity_id?: string;
}

export interface AssignUserRoleResponse {
  tenant_id: string;
  user_id: string;
  role_name: string;
  action: RoleAssignmentAction | string;
  roles: string[];
}

export interface SodPolicy {
  tenant_id: string;
  policy_id: string;
  name: string;
  permission_a: string;
  permission_b: string;
  enforce_same_object: boolean;
  enabled: boolean;
  created_at: string;
}

export interface SodPoliciesResponse {
  tenant_id: string;
  policies: SodPolicy[];
  count: number;
}

export interface UpdateSodPolicyInput {
  entity_id?: string;
  name?: string;
  permission_a?: string;
  permission_b?: string;
  enforce_same_object?: boolean;
  enabled?: boolean;
}

export interface SodViolation {
  tenant_id: string;
  violation_id: string;
  actor_id: string;
  action: string;
  object_type?: string;
  object_id?: string;
  maker_id?: string;
  rule_code: string;
  route_path?: string;
  trace_id?: string;
  created_at: string;
}

export interface SodViolationsResponse {
  tenant_id: string;
  violations: SodViolation[];
  count: number;
}

export function currentUtcDayBucket(): string {
  return new Date().toISOString().slice(0, 10);
}

export interface SupplierAuditIamApi {
  listEvents: (params?: {
    entity?: string;
    object?: string;
    object_id?: string;
    actor?: string;
    actor_type?: AuditActorType | "";
    action?: string;
    date?: string;
    from?: string;
    to?: string;
    limit?: number;
  }) => Promise<AuditEventsResponse>;

  getObjectHistory: (
    objectType: string,
    objectId: string,
    params?: { entity?: string; limit?: number },
  ) => Promise<ObjectHistoryResponse>;

  verifyDayChain: (params?: { date?: string; entity?: string }) => Promise<ChainVerifyResult>;

  listAnchors: (params?: {
    entity?: string;
    date?: string;
    limit?: number;
  }) => Promise<AuditAnchorsResponse>;

  writeAnchor: (input?: WriteAnchorInput, idempotencyKey?: string) => Promise<AuditAnchor>;

  listLegalHolds: (params?: {
    entity?: string;
    object?: string;
    object_id?: string;
    status?: string;
    limit?: number;
  }) => Promise<AuditLegalHoldsResponse>;

  createLegalHold: (
    input: CreateLegalHoldInput,
    idempotencyKey?: string,
  ) => Promise<AuditLegalHold>;

  listRoles: (params?: { entity?: string }) => Promise<IamRolesResponse>;

  listPermissions: () => Promise<IamPermissionsResponse>;

  assignUserRole: (
    userId: string,
    input: AssignUserRoleInput,
    idempotencyKey?: string,
  ) => Promise<AssignUserRoleResponse>;

  listSodPolicies: (params?: { entity?: string }) => Promise<SodPoliciesResponse>;

  updateSodPolicy: (
    policyId: string,
    input: UpdateSodPolicyInput,
    idempotencyKey?: string,
  ) => Promise<{ policy: SodPolicy }>;

  listSodViolations: (params?: {
    entity?: string;
    actor?: string;
    limit?: number;
  }) => Promise<SodViolationsResponse>;
}

export function createSupplierAuditIamApi(client?: ApiClient): SupplierAuditIamApi {
  const api = client ?? createSupplierApi();

  return {
    listEvents(params = {}) {
      return api.request<AuditEventsResponse>(
        withQuery("/v1/audit/events", {
          entity: params.entity,
          object: params.object,
          object_id: params.object_id,
          actor: params.actor,
          actor_type: params.actor_type || undefined,
          action: params.action,
          date: params.date,
          from: params.from,
          to: params.to,
          limit: params.limit,
        }),
        "GET",
      );
    },

    getObjectHistory(objectType, objectId, params = {}) {
      const ot = encodeURIComponent(objectType.trim());
      const oid = encodeURIComponent(objectId.trim());
      return api.request<ObjectHistoryResponse>(
        withQuery(`/v1/audit/history/${ot}/${oid}`, {
          entity: params.entity,
          limit: params.limit,
        }),
        "GET",
      );
    },

    verifyDayChain(params = {}) {
      return api.request<ChainVerifyResult>(
        withQuery("/v1/audit/verify", {
          date: params.date,
          entity: params.entity,
        }),
        "GET",
      );
    },

    listAnchors(params = {}) {
      return api.request<AuditAnchorsResponse>(
        withQuery("/v1/audit/anchors", {
          entity: params.entity,
          date: params.date,
          limit: params.limit,
        }),
        "GET",
      );
    },

    writeAnchor(input = {}, idempotencyKey) {
      return api.request<AuditAnchor>("/v1/audit/anchors", "POST", {
        body: input,
        idempotencyKey,
      });
    },

    listLegalHolds(params = {}) {
      return api.request<AuditLegalHoldsResponse>(
        withQuery("/v1/audit/legal-holds", {
          entity: params.entity,
          object: params.object,
          object_id: params.object_id,
          status: params.status,
          limit: params.limit,
        }),
        "GET",
      );
    },

    createLegalHold(input, idempotencyKey) {
      return api.request<AuditLegalHold>("/v1/audit/legal-hold", "POST", {
        body: input,
        idempotencyKey,
      });
    },

    listRoles(params = {}) {
      return api.request<IamRolesResponse>(
        withQuery("/v1/iam/roles", { entity: params.entity }),
        "GET",
      );
    },

    listPermissions() {
      return api.request<IamPermissionsResponse>("/v1/iam/permissions", "GET");
    },

    assignUserRole(userId, input, idempotencyKey) {
      const uid = encodeURIComponent(userId.trim());
      return api.request<AssignUserRoleResponse>(`/v1/iam/users/${uid}/roles`, "POST", {
        body: input,
        idempotencyKey,
      });
    },

    listSodPolicies(params = {}) {
      return api.request<SodPoliciesResponse>(
        withQuery("/v1/iam/sod-policies", { entity: params.entity }),
        "GET",
      );
    },

    updateSodPolicy(policyId, input, idempotencyKey) {
      const pid = encodeURIComponent(policyId.trim());
      return api.request<{ policy: SodPolicy }>(`/v1/iam/sod-policies/${pid}`, "PUT", {
        body: input,
        idempotencyKey,
      });
    },

    listSodViolations(params = {}) {
      return api.request<SodViolationsResponse>(
        withQuery("/v1/iam/sod-violations", {
          entity: params.entity,
          actor: params.actor,
          limit: params.limit,
        }),
        "GET",
      );
    },
  };
}
