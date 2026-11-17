// ── Wave 3C (M1.9 + M1.10): Enterprise Connectors, Shadow Compare, Export Packs & Auditor Access ──

export type ConnectorProvider = "billz" | "onec" | string;
export type ConnectorAuthType = "bearer" | "api_key" | "basic" | "oauth2" | "none" | string;
export type ConnectorCredentialStatus = "ACTIVE" | "DISABLED" | "INACTIVE" | "REVOKED" | string;
export type ConnectorSyncDirection = "PULL" | "PUSH" | "SHADOW_COMPARE" | "HEALTH" | string;
export type ConnectorSyncStatus =
  | "RUNNING"
  | "COMPLETED"
  | "SUCCEEDED"
  | "MISMATCH_FOUND"
  | "FAILED"
  | string;

export type ShadowCompareMismatchType =
  | "missing_external"
  | "missing_internal"
  | "stock_delta"
  | "balance_delta_minor"
  | string;

export interface ConnectorAdapterInfo {
  provider: ConnectorProvider;
  display_name: string;
  capabilities: string[];
  read_only?: boolean;
}

export interface ConnectorCredentialSummary {
  credential_id: string;
  tenant_id: string;
  provider: ConnectorProvider;
  account_label: string;
  auth_type: ConnectorAuthType;
  secret_ref_masked: string;
  base_url?: string;
  metadata?: Record<string, unknown>;
  status: ConnectorCredentialStatus;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface UpsertConnectorCredentialRequest {
  credential_id?: string;
  provider: ConnectorProvider;
  account_label?: string;
  auth_type: ConnectorAuthType;
  secret_ref: string;
  base_url?: string;
  metadata?: Record<string, unknown>;
  status?: ConnectorCredentialStatus;
}

export interface UpsertConnectorCredentialResponse {
  credential: ConnectorCredentialSummary;
}

export interface ConnectorSyncRun {
  run_id: string;
  tenant_id: string;
  provider: ConnectorProvider;
  account_label: string;
  direction: ConnectorSyncDirection;
  stream_name: string;
  status: ConnectorSyncStatus;
  records_processed: number;
  mismatch_count: number;
  mismatch_summary_json?: string;
  error_message?: string;
  idempotency_key?: string;
  started_at: string;
  completed_at?: string;
  created_at?: string;
}

export interface ShadowCompareMismatch {
  mismatch_type: ShadowCompareMismatchType;
  object_type: string;
  external_id?: string;
  internal_id?: string;
  warehouse_id?: string;
  name?: string;
  external_stock?: number;
  internal_stock?: number;
  stock_delta?: number;
  external_amount_minor?: number;
  internal_amount_minor?: number;
  balance_delta_minor?: number;
  currency?: string;
  detail?: string;
}

export interface ShadowCompareSummary {
  provider?: string;
  account_label?: string;
  run_id?: string;
  external_products?: number;
  internal_products?: number;
  external_stocks?: number;
  internal_stocks?: number;
  external_orders?: number;
  internal_orders?: number;
  external_clients?: number;
  internal_invoices?: number;
  matched_count?: number;
  missing_external?: number;
  missing_internal?: number;
  missing_external_count?: number;
  missing_internal_count?: number;
  stock_delta_count?: number;
  balance_delta_count?: number;
  total_mismatches?: number;
  mismatch_count?: number;
  net_stock_delta?: number;
  net_balance_delta_minor?: number;
  currency?: string;
  records_compared?: number;
  [key: string]: unknown;
}

export interface ConnectorsOverviewResponse {
  tenant_id: string;
  adapters: ConnectorAdapterInfo[];
  credentials: ConnectorCredentialSummary[];
  latest_runs: ConnectorSyncRun[];
}

export interface ConnectorHealthRequest {
  account_label?: string;
  credential_id?: string;
}

export interface ConnectorHealthResponse {
  healthy: boolean;
  provider: ConnectorProvider;
  account_label: string;
  latency_ms: number;
  status_code?: number;
  message?: string;
  checked_at: string;
  run?: ConnectorSyncRun;
}

export interface ConnectorPullRequest {
  account_label?: string;
  credential_id?: string;
  stream?: string;
  raw_payload?: unknown;
  idempotency_key?: string;
}

export interface ConnectorPullResponse {
  run: ConnectorSyncRun;
  records_count: number;
  stream_counts?: Record<string, number>;
}

export interface ConnectorShadowCompareRequest {
  account_label?: string;
  credential_id?: string;
  raw_payload?: unknown;
  idempotency_key?: string;
}

export interface ConnectorShadowCompareResponse {
  run: ConnectorSyncRun;
  summary?: ShadowCompareSummary;
  mismatches: ShadowCompareMismatch[];
}

export interface ConnectorRunsListResponse {
  tenant_id: string;
  runs: ConnectorSyncRun[];
  count: number;
}

export interface ConnectorRunDetailResponse {
  run: ConnectorSyncRun;
  summary?: ShadowCompareSummary;
  mismatches?: ShadowCompareMismatch[];
}

// ── M1.10 Unlimited Partner Exports, ExportPacks & Time-Boxed Auditor Grants ──

export type ExportPackType = "AUDIT" | "BANK" | "TAX";
export type ExportPackFormat = "zip" | "xlsx" | "csv" | "xml" | "json";

export interface ExportPackArtifact {
  name?: string;
  filename?: string;
  format: ExportPackFormat | string;
  content_type?: string;
  path?: string;
  storage_uri?: string;
  size_bytes?: number;
  sha256_hex?: string;
  row_count?: number;
  download_url?: string;
}

export interface ExportPack {
  pack_id: string;
  tenant_id: string;
  pack_type: ExportPackType | string;
  entity_id?: string;
  period_id?: string;
  from_date: string;
  to_date: string;
  format: ExportPackFormat | string;
  status: string;
  watermark_text: string;
  row_count: number;
  sha256_hex: string;
  download_token?: string;
  download_url?: string;
  artifacts?: ExportPackArtifact[];
  artifacts_json?: string;
  manifest?: Record<string, unknown>;
  manifest_json?: string;
  created_by: string;
  expires_at: string;
  created_at: string;
  expired?: boolean;
}

export interface CreateExportPackRequest {
  pack_type: ExportPackType | string;
  entity_id?: string;
  period_id?: string;
  from_date?: string;
  to_date?: string;
  format?: ExportPackFormat | string;
  ttl_hours?: number;
}

export type ExportPackResponse = ExportPack | { pack: ExportPack };

export interface ExportPacksListResponse {
  tenant_id: string;
  packs: ExportPack[];
  count: number;
}

export interface AuditorGrant {
  tenant_id: string;
  user_id: string;
  role_name: string;
  assigned_by?: string;
  granted_by?: string;
  status: "ACTIVE" | "EXPIRED" | "REVOKED" | string;
  expires_at: string;
  created_at: string;
  updated_at?: string;
  reason?: string;
}

export interface CreateAuditorGrantRequest {
  entity_id?: string;
  user_id: string;
  expires_at?: string;
  ttl_hours?: number;
  reason?: string;
}

export interface CreateAuditorGrantResponse {
  grant: AuditorGrant;
  token: string;
  expires_at: string;
}

export interface AuditorGrantsListResponse {
  tenant_id: string;
  grants: AuditorGrant[];
  count: number;
}

export interface RevokeAuditorGrantResponse {
  ok?: boolean;
  revoked?: boolean;
  tenant_id: string;
  user_id: string;
  status?: string;
}

export interface UnlimitedPartnerExportRequest {
  resource: string;
  format?: string;
  from?: string;
  to?: string;
  unlimited?: boolean;
}
