export type MigrationPack =
  | "warehouses"
  | "products_prices"
  | "customers"
  | "stock"
  | "lots"
  | "historical_orders"
  | "nasiya_ar"
  | "open_orders";

export type MigrationProjectStatus =
  | "DRAFT"
  | "STAGING"
  | "DRY_RUN_READY"
  | "READY_TO_CUTOVER"
  | "CUTOVER_COMPLETE"
  | "FAILED";

export type MigrationBundleStatus =
  | "CREATED"
  | "UPLOADING"
  | "CLASSIFYING"
  | "SORTED"
  | "NEEDS_REVIEW"
  | "FAILED";

export type MigrationSessionStatus =
  | "INITIALIZED"
  | "UPLOADED"
  | "DISCOVERING"
  | "DISCOVERED"
  | "MAPPING_REQUIRED"
  | "MAPPING_PROPOSED"
  | "VALIDATING"
  | "DRY_RUN_READY"
  | "DRY_RUNNING"
  | "APPROVED"
  | "SNAPSHOTTING"
  | "APPLYING"
  | "APPLIED"
  | "ROLLING_BACK"
  | "ROLLED_BACK"
  | "CUTOVER_COMPLETE"
  | "FAILED";

export interface MigrationProject {
  supplier_id: string;
  project_id: string;
  name: string;
  mode: "CUTOVER" | string;
  status: MigrationProjectStatus | string;
  checklist?: unknown;
  default_warehouse_id?: string;
  created_by: string;
  cutover_at?: string;
  created_at: string;
  updated_at: string;
}

export interface MigrationBundle {
  supplier_id: string;
  project_id: string;
  bundle_id: string;
  status: MigrationBundleStatus | string;
  sort_summary?: MigrationSortSummary | unknown;
  error_message?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface MigrationArtifact {
  supplier_id: string;
  project_id: string;
  bundle_id: string;
  artifact_id: string;
  file_name: string;
  byte_size: number;
  declared_mime?: string;
  detected_mime?: string;
  content_sha256: string;
  encoding?: string;
  gcs_path: string;
  streams?: unknown;
  created_at: string;
  updated_at: string;
}

export interface MigrationBundleTree {
  bundle: MigrationBundle;
  artifacts: MigrationArtifact[];
}

export interface MigrationUploadTicket {
  artifact_id: string;
  file_name: string;
  gcs_path: string;
  upload_url: string;
  method: "PUT" | "POST" | string;
  content_type: string;
  resumable: boolean;
  required_headers?: Record<string, string>;
  expires_at: string;
}

export interface MigrationSession {
  supplier_id: string;
  session_id: string;
  status: MigrationSessionStatus | string;
  file_name: string;
  total_rows: number;
  error_summary?: unknown;
  project_id?: string;
  bundle_id?: string;
  artifact_id?: string;
  stream_key?: string;
  stream_split_key?: string;
  pack?: MigrationPack | string;
  sheet_name?: string;
  gcs_path?: string;
  content_sha256?: string;
  detected_mime?: string;
  encoding?: string;
  ai_model?: string;
  ai_status?: string;
  dry_run_report_id?: string;
  dry_run_fingerprint?: string;
  apply_mode?: string;
  set_on_hand_confirmed?: boolean;
  resume_row_index?: number;
  last_chunk_id?: string;
  rollback_until?: string;
  created_at: string;
  updated_at: string;
}

export interface MigrationDAGStatus {
  order: MigrationPack[];
  runnable_now: MigrationPack[];
  blocked_by: Record<string, MigrationPack[]>;
  missing_packs?: MigrationPack[];
}

export interface MigrationProjectTree {
  project: MigrationProject;
  bundles: MigrationBundleTree[];
  sessions: MigrationSession[];
  dag: MigrationDAGStatus;
}

export interface MigrationArtifactDescriptorInput {
  artifact_id?: string;
  file_name: string;
  byte_size: number;
  declared_mime?: string;
  content_sha256: string;
}

export interface MigrationCreateBundleResponse {
  bundle: MigrationBundle;
  artifacts: MigrationArtifact[];
  upload_tickets: MigrationUploadTicket[];
}

export interface MigrationUploadedArtifactInput {
  artifact_id: string;
  content_sha256?: string;
  byte_size?: number;
  encoding?: string;
}

export interface MigrationSortedStreamDecision {
  artifact_id: string;
  file_name: string;
  stream_key: string;
  encoding: string;
  total_rows: number;
  headers: string[];
  preview_raw?: string[];
  detected_packs: Array<MigrationPack | string>;
  split_keys: string[];
  confidence: number;
  is_composite: boolean;
  ignored?: boolean;
  needs_review: boolean;
  review_reason?: string;
}

export interface MigrationSortSummary {
  streams: MigrationSortedStreamDecision[];
  conflicting_inventory_mode: boolean;
  needs_review: boolean;
  review_reasons?: string[];
  artifact_encodings?: Record<string, string>;
}

export interface MigrationBundleUploadedResponse {
  bundle: MigrationBundle;
  summary?: MigrationSortSummary;
  sessions?: MigrationSession[];
}

export interface MigrationStreamOverrideItem {
  artifact_id: string;
  stream_key: string;
  packs?: Array<MigrationPack | string>;
  ignore?: boolean;
  encoding?: string;
  confirmed?: boolean;
}

export interface MigrationBundleSortOverrideRequest {
  overrides?: MigrationStreamOverrideItem[];
  resolved_inventory_mode?: "stock" | "lots" | MigrationPack;
  confirm_review?: boolean;
}

export interface MigrationBundleSortOverrideResponse {
  bundle_id: string;
  bundle_status: MigrationBundleStatus | string;
  summary: MigrationSortSummary;
  sessions: MigrationSession[];
}

export interface MigrationStagedRow {
  supplier_id: string;
  session_id: string;
  row_index: number;
  raw_data?: Record<string, unknown>;
  cleaned_data?: Record<string, unknown>;
  validation_errors?: string[];
  is_new_product: boolean;
  external_id?: string;
  stream_split_key?: string;
  apply_action?: string;
  apply_chunk?: number;
  is_new_entity?: boolean;
  created_at: string;
  updated_at: string;
}

export interface MigrationSessionRowsResponse {
  session_id: string;
  total_rows: number;
  rows: MigrationStagedRow[];
}

export interface MigrationCanonicalField {
  key: string;
  label: string;
  required?: boolean;
  required_group?: string;
  data_type?: string;
  aliases?: string[];
}

export interface MigrationRequiredGroup {
  group_id: string;
  label: string;
  any_of: string[];
}

export interface MigrationColumnMappingProposal {
  source_column: string;
  target_field: string;
  confidence: number;
  reason?: string;
  source?: string;
  confirmed: boolean;
  transform?: string;
}

export interface MigrationSessionMappingDocument {
  session_id: string;
  pack: MigrationPack | string;
  header_fingerprint: string;
  ai_model: string;
  ai_status: string;
  tokens_used?: number;
  fallback_reason?: string;
  mappings: MigrationColumnMappingProposal[];
  missing_required?: string[];
  low_confidence_columns?: string[];
  all_required_mapped: boolean;
  mapping_hash: string;
  confirmed_by?: string;
  updated_at: string;
}

export interface MigrationGetMappingResponse {
  session: MigrationSession;
  taxonomy: MigrationCanonicalField[];
  required_groups: MigrationRequiredGroup[];
  mapping: MigrationSessionMappingDocument;
}

export interface MigrationConfirmMappingRequest {
  mappings?: MigrationColumnMappingProposal[];
  accept_above_confidence?: number;
  save_to_library?: boolean;
  strict?: boolean;
}

export interface MigrationConfirmMappingResponse {
  session: MigrationSession;
  mapping: MigrationSessionMappingDocument;
}

export interface MigrationReport {
  supplier_id: string;
  report_id: string;
  project_id?: string;
  session_id?: string;
  report_kind: string;
  content_fingerprint: string;
  rows_total: number;
  rows_valid: number;
  rows_invalid: number;
  would_create: number;
  would_update: number;
  would_skip: number;
  money_impact_minor: number;
  currency?: string;
  blocking_count: number;
  warning_count: number;
  summary_json: unknown;
  expires_at: string;
  created_at: string;
}

export interface MigrationDryRunConflict {
  session_id: string;
  pack: MigrationPack | string;
  row_index: number;
  code: string;
  severity: "BLOCKING" | "WARNING" | string;
  field?: string;
  message: string;
  suggestion?: string;
}

export interface MigrationPackDryRunReport {
  session_id: string;
  pack: MigrationPack | string;
  stream_split_key?: string;
  rows_total: number;
  rows_valid: number;
  rows_invalid: number;
  would_create: number;
  would_update: number;
  would_skip: number;
  money_impact_minor: number;
  currency: string;
  blocking_count: number;
  warning_count: number;
  cold_lot_migrated_without_live_baseline?: number;
  on_hand_below_reserved?: number;
}

export interface MigrationDryRunSummary {
  project_id: string;
  content_fingerprint: string;
  packs: MigrationPackDryRunReport[];
  conflicts?: MigrationDryRunConflict[];
  balanced_journal_check: boolean;
  journal_debit_minor: number;
  journal_credit_minor: number;
  opening_balance_account: string;
  ready_to_approve: boolean;
  cold_lot_migrated_without_live_baseline: number;
  on_hand_below_reserved: number;
  evaluated_at: string;
  expires_at: string;
}

export interface MigrationDryRunResponse {
  report: MigrationReport;
  summary: MigrationDryRunSummary;
}

export interface MigrationApplyResult {
  session: MigrationSession;
  already_applied?: boolean;
  created: number;
  updated: number;
  skipped: number;
  chunks_applied: number;
  last_row_index: number;
  warnings?: Array<{ row_index?: number; code: string; message: string }>;
}

export interface MigrationRollbackResult {
  session: MigrationSession;
  reverted_snapshots: number;
  reversed_invoices: number;
  superseded_orders: number;
  reversed_amount_minor: number;
  reversal_entry_ids?: string[];
}

export interface MigrationOpenOrderReleaseRequest {
  items?: Array<{ order_id: string; target_status?: "PENDING" | "SCHEDULED" }>;
  order_ids?: string[];
  target_status?: "PENDING" | "SCHEDULED";
}

export interface MigrationOpenOrderReleaseResponse {
  released_order_ids: string[];
  skipped_order_ids?: string[];
  errors?: Record<string, string>;
}

export interface MigrationCutoverResult {
  project: MigrationProject;
  sessions: MigrationSession[];
  checklist: Array<{ name: string; passed: boolean; detail: string }>;
  cutover_at: string;
}

// ── INN Lookup, Duplicate-INN Join Requests & Retailer Dual Tax Identity ──

export type INNLookupStatus = "FOUND" | "NOT_FOUND" | "UNAVAILABLE";
export type INNLookupEntityType = "YURIDIK_9D" | "PINFL_14D";

export interface INNLookupRequest {
  tax_id?: string;
  inn?: string;
  home_cell?: string;
  market_code?: string;
  tenant_type?: "SUPPLIER" | "RETAILER";
}

export interface INNLookupResponse {
  status: INNLookupStatus | string;
  normalized_inn: string;
  entity_type?: INNLookupEntityType | string;
  legal_name?: string;
  address?: string;
  registry_state?: string;
  cached: boolean;
  fetched_at?: string;
  expires_at?: string;
  existing_tenant_id?: string;
  existing_tenant_type?: "SUPPLIER" | "RETAILER" | string;
  existing_legal_name?: string;
  existing_join_eligible: boolean;
}

export type JoinRequestStatus = "PENDING" | "APPROVED" | "REJECTED";

export interface JoinRequest {
  request_id: string;
  home_cell: string;
  target_tenant_type: "SUPPLIER" | "RETAILER" | string;
  target_tenant_id: string;
  normalized_inn: string;
  requester_phone: string;
  requester_name?: string;
  requester_email?: string;
  role_sought: string;
  status: JoinRequestStatus | string;
  decided_by?: string;
  decided_at?: string;
  decision_reason?: string;
  created_at: string;
}

export interface JoinRequestsListResponse {
  supplier_id?: string;
  retailer_id?: string;
  requests: JoinRequest[];
}

export interface JoinRequestDecisionResponse {
  supplier_id?: string;
  retailer_id?: string;
  request: JoinRequest;
}

export interface JoinRequestConflictPayload {
  error: "join_request_pending" | string;
  code?: string;
  request_id: string;
  target_tenant_id: string;
  legal_name?: string;
  normalized_inn?: string;
}

export type RetailerLegalEntityStatus = "LEGAL_VERIFIED" | "INFORMAL_PHONE";

export interface RetailerTaxIdentityRequest {
  tax_id?: string;
  inn?: string;
  informal_shop?: boolean;
  legal_entity_status?: RetailerLegalEntityStatus;
  home_cell?: string;
  market_code?: string;
  role_sought?: string;
  legal_name?: string;
}

export interface MigrationOpeningBalanceInvoice {
  invoice_id: string;
  supplier_id: string;
  supplier_name?: string;
  retailer_id: string;
  order_id?: string;
  external_id?: string;
  principal_minor: number;
  balance_minor: number;
  currency: string;
  status: string;
  source: "MIGRATION" | string;
  dunning_frozen: boolean;
  due_at?: string;
  acknowledged_at?: string;
  disputed_at?: string;
  dispute_reason?: string;
  updated_at?: string;
}

export interface MigrationOpeningBalancesResponse {
  retailer_id: string;
  invoices: MigrationOpeningBalanceInvoice[];
}

export interface MigrationInvoiceActionResponse {
  retailer_id: string;
  invoice: MigrationOpeningBalanceInvoice;
}
