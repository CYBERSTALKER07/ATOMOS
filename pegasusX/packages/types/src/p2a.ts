import type { ConnectorSyncRun, ExportPack } from "./connectors";

// ── Wave 4A (P2): Multi-Entity/Consolidation/FX, P2P/3-Way Match, Process Mining,
//    2-Way ERP Connectors & Bank Credit Pack ──

export type LegalEntityStatus = "ACTIVE" | "INACTIVE" | "SUSPENDED" | string;

export interface LegalEntity {
  tenant_id: string;
  entity_id: string;
  name: string;
  tin?: string;
  country_code: string;
  base_currency: string;
  is_parent: boolean;
  parent_entity_id?: string;
  status: LegalEntityStatus;
  created_at: string;
  updated_at: string;
}

export interface CreateLegalEntityRequest {
  tenant_id?: string;
  entity_id?: string;
  name: string;
  tin?: string;
  country_code?: string;
  base_currency?: string;
  is_parent?: boolean;
  parent_entity_id?: string;
  status?: LegalEntityStatus;
}

export interface UpdateLegalEntityRequest {
  tenant_id?: string;
  name?: string;
  tin?: string;
  country_code?: string;
  base_currency?: string;
  is_parent?: boolean;
  parent_entity_id?: string;
  status?: LegalEntityStatus;
}

export type LegalEntityResponse = LegalEntity | { entity: LegalEntity };

export interface LegalEntitiesListResponse {
  tenant_id: string;
  count: number;
  entities: LegalEntity[];
}

export type IntercompanyTransferType =
  | "CASH_POOL"
  | "EXPENSE_RECHARGE"
  | "INVENTORY_TRANSFER"
  | "LOAN"
  | string;

export interface IntercompanyTransfer {
  tenant_id: string;
  transfer_id: string;
  from_entity_id: string;
  to_entity_id: string;
  transfer_type: IntercompanyTransferType;
  amount_minor: number;
  currency: string;
  period_id: string;
  reference_type?: string;
  reference_id?: string;
  debit_journal_id?: string;
  credit_journal_id?: string;
  status: string;
  idempotency_key?: string;
  created_by: string;
  created_at: string;
}

export interface CreateIntercompanyTransferRequest {
  tenant_id?: string;
  from_entity_id: string;
  to_entity_id: string;
  transfer_type: IntercompanyTransferType;
  amount_minor: number;
  currency?: string;
  period_id?: string;
  reference_type?: string;
  reference_id?: string;
  memo?: string;
  idempotency_key?: string;
}

export interface CreateIntercompanyTransferResponse {
  transfer: IntercompanyTransfer;
  already_posted?: boolean;
}

export interface IntercompanyTransfersListResponse {
  tenant_id: string;
  count: number;
  transfers: IntercompanyTransfer[];
}

export interface FxRevaluationPosition {
  source_type: string;
  source_id: string;
  account_code: string;
  direction: "ASSET" | "LIABILITY" | string;
  currency: string;
  foreign_amount_minor: number;
  carrying_functional_minor: number;
  revalued_functional_minor: number;
  unrealized_gain_loss_minor: number;
  rate_id?: string;
  rate_scaled: number;
  scale: number;
}

export interface FxRevaluationRun {
  tenant_id: string;
  run_id: string;
  entity_id?: string;
  period_id: string;
  base_currency: string;
  unrealized_gain_loss_minor: number;
  rounding_delta_minor: number;
  rates_json?: string;
  journal_entry_id?: string;
  idempotency_key?: string;
  created_by: string;
  created_at: string;
}

export interface RunFxRevaluationRequest {
  tenant_id?: string;
  entity_id?: string;
  period_id?: string;
  base_currency?: string;
  as_of?: string;
  idempotency_key?: string;
}

export interface RunFxRevaluationResponse {
  run: FxRevaluationRun;
  already_posted?: boolean;
  position_count?: number;
  positions?: FxRevaluationPosition[];
}

export interface FxRevaluationRunsListResponse {
  tenant_id: string;
  count: number;
  runs: FxRevaluationRun[];
}

export interface ConsolidatedEntitySummary {
  entity_id: string;
  name: string;
  tin?: string;
  country_code: string;
  base_currency: string;
  is_parent: boolean;
  fx_rate_scaled?: number;
  fx_scale?: number;
  fx_inverse?: boolean;
  debit_minor: number;
  credit_minor: number;
  net_minor: number;
  converted_debit_minor: number;
  converted_credit_minor: number;
  converted_net_minor: number;
  balanced: boolean;
}

export interface IntercompanyElimination {
  transfer_id?: string;
  account_code?: string;
  from_entity_id: string;
  to_entity_id: string;
  currency?: string;
  amount_minor: number;
  converted_amount_minor: number;
  transfer_type?: string;
  description?: string;
}

export interface ConsolidatedAccountLine {
  account_code: string;
  account_name?: string;
  class?: string;
  currency: string;
  pre_elimination_debit_minor: number;
  pre_elimination_credit_minor: number;
  debit_minor: number;
  credit_minor: number;
  eliminated_minor: number;
  net_minor: number;
  by_entity?: Record<string, number>;
}

export interface FxRateApplied {
  rate_id?: string;
  base_currency: string;
  quote_currency: string;
  rate_scaled: number;
  scale: number;
  inverse?: boolean;
  effective_at?: string;
  source?: string;
}

export interface ConsolidationReport {
  tenant_id: string;
  period_id: string;
  reporting_currency: string;
  entity_ids: string[];
  entities: ConsolidatedEntitySummary[];
  eliminations: IntercompanyElimination[];
  accounts: ConsolidatedAccountLine[];
  total_debits_minor: number;
  total_credits_minor: number;
  total_eliminations_minor: number;
  ic_variance_minor: number;
  balanced: boolean;
  fx_rates_applied: FxRateApplied[];
}

// ── §6.6 P2P, Process Templates, Instances & Process Mining ──

export type ProcessDomain = "O2C" | "P2P" | "RETURNS" | string;

export interface ProcessStep {
  code: string;
  name: string;
  sequence: number;
  sla_seconds: number;
  required_role?: string;
}

export interface ProcessTemplate {
  tenant_id: string;
  template_id: string;
  code: string;
  name: string;
  domain: ProcessDomain;
  steps: ProcessStep[];
  is_default: boolean;
  built_in?: boolean;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CreateProcessTemplateRequest {
  tenant_id?: string;
  template_id?: string;
  code: string;
  name: string;
  domain: ProcessDomain;
  steps: ProcessStep[];
  is_default?: boolean;
  status?: string;
}

export type ProcessTemplateResponse = ProcessTemplate | { template: ProcessTemplate };

export interface ProcessTemplatesListResponse {
  templates: ProcessTemplate[];
  count: number;
}

export interface ProcessTransition {
  from_step: string;
  to_step: string;
  actor_id?: string;
  reason?: string;
  occurred_at: string;
}

export interface ProcessInstance {
  tenant_id: string;
  instance_id: string;
  template_id: string;
  template_code?: string;
  entity_id?: string;
  object_type: string;
  object_id: string;
  current_step: string;
  status: string;
  transitions: ProcessTransition[];
  started_at: string;
  completed_at?: string;
  updated_at: string;
}

export interface StartOrAdvanceProcessInstanceRequest {
  tenant_id?: string;
  instance_id?: string;
  template_id?: string;
  template_code?: string;
  entity_id?: string;
  object_type: string;
  object_id: string;
  next_step?: string;
  current_step?: string;
  reason?: string;
  complete?: boolean;
}

export type ProcessInstanceResponse = ProcessInstance | { instance: ProcessInstance };

export interface ProcessInstancesListResponse {
  instances: ProcessInstance[];
  count: number;
}

export interface MiningEdge {
  from_step: string;
  to_step: string;
  domain: string;
  case_count: number;
  transition_count: number;
  avg_duration_sec: number;
  median_duration_sec: number;
  p95_duration_sec: number;
  sla_target_sec: number;
  sla_breach_count: number;
}

export interface MiningBottleneck {
  from_step: string;
  to_step: string;
  domain: string;
  case_count: number;
  transition_count: number;
  median_duration_sec: number;
  p95_duration_sec: number;
  sla_target_sec: number;
  sla_breach_count: number;
}

export interface ProcessMiningReport {
  tenant_id: string;
  entity_id?: string;
  domain?: string;
  total_cases: number;
  total_transitions: number;
  sla_breach_count: number;
  bottleneck_stage?: MiningBottleneck | null;
  empty: boolean;
  edges: MiningEdge[];
}

export type PurchaseOrderStatus =
  | "PENDING_APPROVAL"
  | "APPROVED"
  | "PARTIAL_RECEIVED"
  | "RECEIVED"
  | "CLOSED"
  | "CANCELLED"
  | string;

export interface PurchaseOrderLine {
  tenant_id: string;
  po_id: string;
  line_num: number;
  sku: string;
  product_name?: string;
  qty_ordered: number;
  qty_received: number;
  unit_cost_minor: number;
  vat_bps: number;
  line_total_minor: number;
}

export interface PurchaseOrder {
  tenant_id: string;
  po_id: string;
  entity_id?: string;
  vendor_id: string;
  vendor_name?: string;
  vendor_tin?: string;
  currency: string;
  subtotal_minor: number;
  vat_minor: number;
  total_minor: number;
  status: PurchaseOrderStatus;
  approval_request_id?: string;
  idempotency_key?: string;
  expected_at?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
  lines: PurchaseOrderLine[];
}

export interface PurchaseOrderLineInput {
  sku: string;
  product_name?: string;
  qty_ordered: number;
  unit_cost_minor: number;
  vat_bps: number;
}

export interface CreatePurchaseOrderRequest {
  tenant_id?: string;
  po_id?: string;
  entity_id?: string;
  vendor_id: string;
  vendor_name?: string;
  vendor_tin?: string;
  currency?: string;
  expected_at?: string;
  approval_id?: string;
  idempotency_key?: string;
  template_code?: string;
  lines: PurchaseOrderLineInput[];
}

export interface CreatePurchaseOrderResponse {
  purchase_order: PurchaseOrder;
  approval_required?: boolean;
  approval_request_id?: string;
}

export interface PurchaseOrdersListResponse {
  purchase_orders: PurchaseOrder[];
  count: number;
}

export interface ApprovePurchaseOrderRequest {
  tenant_id?: string;
  approval_id?: string;
}

export type PurchaseOrderDetailResponse = PurchaseOrder | { purchase_order: PurchaseOrder };

export interface GoodsReceiptLine {
  line_num: number;
  sku: string;
  qty_received: number;
  unit_cost_minor: number;
  vat_bps: number;
  line_net_minor: number;
}

export interface GoodsReceipt {
  tenant_id: string;
  receipt_id: string;
  po_id: string;
  entity_id?: string;
  warehouse_id: string;
  status: string;
  lines: GoodsReceiptLine[];
  total_received_minor: number;
  journal_entry_id?: string;
  idempotency_key?: string;
  received_by: string;
  received_at: string;
  created_at: string;
}

export interface GoodsReceiptLineInput {
  line_num?: number;
  sku: string;
  qty_received: number;
}

export interface PostGoodsReceiptRequest {
  tenant_id?: string;
  receipt_id?: string;
  po_id: string;
  entity_id?: string;
  warehouse_id: string;
  period_id?: string;
  idempotency_key?: string;
  received_at?: string;
  lines: GoodsReceiptLineInput[];
}

export interface PostGoodsReceiptResponse {
  receipt: GoodsReceipt;
  purchase_order: PurchaseOrder;
  journal_entry_id: string;
}

export interface GoodsReceiptsListResponse {
  receipts: GoodsReceipt[];
  count: number;
}

export type VendorInvoiceMatchStatus = "PENDING" | "MATCHED_3WAY" | "DISCREPANCY" | string;

export interface VendorInvoice {
  tenant_id: string;
  vendor_invoice_id: string;
  po_id: string;
  receipt_id?: string;
  esf_doc_id?: string;
  entity_id?: string;
  vendor_id: string;
  currency: string;
  subtotal_minor: number;
  vat_minor: number;
  total_minor: number;
  match_status: VendorInvoiceMatchStatus;
  variance_minor: number;
  variance_reason?: string;
  journal_entry_id?: string;
  idempotency_key?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface CreateVendorInvoiceRequest {
  tenant_id?: string;
  vendor_invoice_id?: string;
  po_id: string;
  receipt_id?: string;
  esf_doc_id?: string;
  entity_id?: string;
  vendor_id?: string;
  currency?: string;
  subtotal_minor: number;
  vat_minor: number;
  total_minor: number;
  idempotency_key?: string;
}

export type VendorInvoiceResponse = VendorInvoice | { vendor_invoice: VendorInvoice };

export interface VendorInvoicesListResponse {
  vendor_invoices: VendorInvoice[];
  count: number;
}

export interface ThreeWayMatchRequest {
  tenant_id?: string;
  vendor_invoice_id: string;
  po_id?: string;
  receipt_id?: string;
  esf_doc_id?: string;
  period_id?: string;
  tolerance_bps?: number;
  tolerance_minor?: number;
  override_discrepancy?: boolean;
  approval_id?: string;
}

export interface ThreeWayMatchResult {
  vendor_invoice: VendorInvoice;
  matched: boolean;
  match_status: VendorInvoiceMatchStatus;
  variance_minor: number;
  variance_reason?: string;
  journal_entry_id?: string;
  verified_by?: string;
  process_instance?: ProcessInstance;
}

// ── §7.10 2-Way ERP Connectors & Approval-Gated Push ──

export type P2AConnectorProvider =
  | "billz"
  | "onec"
  | "moysklad"
  | "smartup"
  | "iiko";

export interface ConnectorCanonicalRecord {
  object_type: string;
  external_id: string;
  sku?: string;
  barcode?: string;
  name?: string;
  warehouse_id?: string;
  order_id?: string;
  retailer_id?: string;
  currency?: string;
  status?: string;
  quantity_on_hand?: number;
  price_minor?: number;
  total_minor?: number;
  balance_minor?: number;
  content_hash?: string;
  extra?: Record<string, unknown>;
}

export interface ConnectorPushRequest {
  tenant_id?: string;
  account_label?: string;
  credential_id?: string;
  stream?: string;
  approval_id: string;
  records?: ConnectorCanonicalRecord[];
  raw_payload?: unknown;
  raw_xml?: string;
  idempotency_key?: string;
}

export interface ConnectorPushResult {
  provider: string;
  account_label: string;
  stream: string;
  records_pushed: number;
  external_doc_id?: string;
  status: string;
  message: string;
}

export interface ConnectorPushResponse {
  run: ConnectorSyncRun;
  push_result: ConnectorPushResult;
  approval_id: string;
}

export interface SubmitConnectorPushApprovalRequest {
  tenant_id?: string;
  object_type: "ConnectorPush" | string;
  object_id: string;
  action: "push" | string;
  amount_minor?: number;
  context?: Record<string, unknown>;
  payload?: Record<string, unknown>;
  force_create?: boolean;
  min_approvers?: number;
  idempotency_key?: string;
}

export interface SubmittedApprovalRequestSummary {
  request_id: string;
  tenant_id?: string;
  object_type: string;
  object_id: string;
  action: string;
  status: string;
  payload_hash: string;
  current_step?: number;
  required_approvers?: number;
  requested_by?: string;
  created_at?: string;
}

export interface SubmitConnectorPushApprovalResponse {
  required?: boolean;
  status?: string;
  request?: SubmittedApprovalRequestSummary;
}

// ── §6.8 Bank Credit Pack Metadata ──

export interface BankDPDBucketRow {
  bucket: string;
  invoice_count: number;
  retailer_count: number;
  principal_minor: number;
  exposure_minor: number;
  share_bps: number;
  currency: string;
}

export interface BankDebtorConcentrationRow {
  rank: number;
  retailer_id: string;
  invoice_count: number;
  exposure_minor: number;
  principal_minor: number;
  credit_limit_minor: number;
  concentration_bps: number;
  cumulative_bps: number;
  max_dpd_days: number;
  consent_verified: boolean;
  relationship_score: number;
  network_score: number;
  currency: string;
}

export interface BankLossRecoveryCurveRow {
  vintage_month: string;
  invoice_count: number;
  origination_minor: number;
  recovered_minor: number;
  written_off_minor: number;
  outstanding_minor: number;
  default_exposure_minor: number;
  recovery_rate_bps: number;
  loss_rate_bps: number;
  default_rate_bps: number;
  currency: string;
}

export interface BankCreditSummary {
  as_of: string;
  currency: string;
  total_invoices: number;
  total_retailers: number;
  consent_verified_retailers: number;
  total_principal_minor: number;
  total_exposure_minor: number;
  total_recovered_minor: number;
  total_written_off_minor: number;
  top_1_concentration_bps: number;
  top_5_concentration_bps: number;
  top_10_concentration_bps: number;
  avg_relationship_score: number;
  avg_network_score: number;
  dpd_buckets: BankDPDBucketRow[];
  debtor_concentration: BankDebtorConcentrationRow[];
  loss_recovery_curves: BankLossRecoveryCurveRow[];
}

export interface BankPackManifest {
  schema_version?: string;
  pack_id?: string;
  pack_type?: string;
  total_rows?: number;
  datasets?: Record<string, number>;
  bank_credit_summary?: BankCreditSummary;
  [key: string]: unknown;
}

export type BankExportPack = ExportPack & {
  manifest?: BankPackManifest;
};
