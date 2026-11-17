// ── Wave 4B (P2): Anomaly & Fraud Catalogue, Full E2E What-If Simulator,
//    Cold-Chain HACCP, L2 Soak Thresholds, Custom Roles & SCIM v2 ──

export type AutonomyLevel = "L1" | "L2" | string;

export interface P2BProposalSummary {
  tenant_id?: string;
  proposal_id: string;
  agent_id: string;
  proposal_type: string;
  target_object_type: string;
  target_object_id: string;
  title: string;
  summary: string;
  rationale?: string;
  payload_json?: string;
  payload?: Record<string, unknown>;
  payload_hash: string;
  confidence_bps: number;
  estimated_impact_minor: number;
  currency: string;
  roi_category?: string;
  autonomy_level?: AutonomyLevel;
  status: string;
  approval_request_id?: string;
  roi_event_id?: string;
  created_at?: string;
  updated_at?: string;
}

// ── §7.7 Anomaly & Fraud Detector Catalogue ──

export type AnomalyRuleCategory =
  | "CASH_FRAUD"
  | "COLLUSION_FRAUD"
  | "INVENTORY_LOSS"
  | "PRICING_FRAUD"
  | "GL_CONTROLS"
  | string;

export type AnomalySeverity = "LOW" | "MEDIUM" | "HIGH" | "CRITICAL" | string;

export type AnomalyCaseStatus =
  | "OPEN"
  | "RESOLVED"
  | "DISMISSED"
  | "ESCALATED"
  | string;

export interface AnomalyRuleDefinition {
  rule_code: string;
  name: string;
  category: AnomalyRuleCategory;
  default_severity: AnomalySeverity;
  description: string;
  default_threshold_bps?: number;
  default_threshold_minor?: number;
  playbook_action?: string;
  source_tables?: string[];
}

export interface AnomalyCatalogueResponse {
  tenant_id?: string;
  rules: AnomalyRuleDefinition[];
  count: number;
}

export interface RunAnomalyScanRequest {
  tenant_id?: string;
  entity_id?: string;
  rule_codes?: string[];
  threshold_bps?: number;
  threshold_minor?: number;
}

export interface AnomalyCase {
  tenant_id: string;
  case_id: string;
  rule_code: string;
  category: AnomalyRuleCategory;
  severity: AnomalySeverity;
  status: AnomalyCaseStatus;
  object_type: string;
  object_id: string;
  exposure_minor: number;
  score_bps: number;
  evidence_json?: string;
  evidence?: Record<string, unknown>;
  playbook_action?: string;
  proposal_id?: string;
  assigned_to?: string;
  resolution_note?: string;
  resolved_by?: string;
  resolved_at?: string;
  detected_at: string;
  updated_at: string;
}

export interface RunAnomalyScanResponse {
  tenant_id?: string;
  cases: AnomalyCase[];
  created_count: number;
  deduplicated_count?: number;
  scanned_rules?: string[];
  scan_at?: string;
}

export interface AnomalyCasesListResponse {
  tenant_id?: string;
  cases: AnomalyCase[];
  count: number;
}

export interface ResolveAnomalyCaseRequest {
  tenant_id?: string;
  status: "RESOLVED" | "DISMISSED" | "ESCALATED" | string;
  resolution_note: string;
  playbook_action?: string;
  assigned_to?: string;
  create_proposal?: boolean;
}

export interface ResolveAnomalyCaseResponse {
  case: AnomalyCase;
  proposal?: P2BProposalSummary | null;
  proposal_id?: string;
}

// ── §7.9 Full E2E What-If Simulator ──

export interface WhatIfShocks {
  demand_delta_bps: number;
  factory_downtime_hours: number;
  fx_rate_shock_bps: number;
  credit_limit_delta_bps: number;
  unit_price_delta_bps: number;
  currency?: string;
}

export interface WhatIfStageBreakdown {
  stage: string;
  label?: string;
  delta_minor?: number;
  cost_minor?: number;
  metric_bps?: number;
  units_delta_milli?: number;
  detail?: string;
  [key: string]: unknown;
}

export interface WhatIfRecommendedAction {
  proposal_type: string;
  agent_id: string;
  target_object_type: string;
  target_object_id: string;
  title: string;
  summary: string;
  estimated_impact_minor: number;
  confidence_bps: number;
  payload?: Record<string, unknown>;
}

export interface WhatIfStructuredBreakdown {
  currency?: string;
  scenario_applied?: boolean;
  patch_count?: number;
  demand?: {
    baseline_revenue_minor?: number;
    shocked_revenue_minor?: number;
    forecast_delta_minor?: number;
    baseline_units_milli?: number;
    shocked_units_milli?: number;
    sku_count?: number;
  };
  replenishment?: {
    baseline_cost_minor?: number;
    shocked_cost_minor?: number;
    cost_delta_minor?: number;
    shortfall_units_milli?: number;
    affected_skus?: string[];
  };
  dispatch?: {
    baseline_cost_minor?: number;
    shocked_cost_minor?: number;
    dispatch_cost_delta_minor?: number;
    route_count?: number;
    order_count?: number;
  };
  credit?: {
    baseline_exposure_minor?: number;
    shocked_exposure_minor?: number;
    credit_exposure_delta_minor?: number;
    over_limit_retailers?: number;
    risk_cost_delta_minor?: number;
  };
  pnl?: {
    cogs_delta_minor?: number;
    gross_margin_delta_minor?: number;
    net_pnl_delta_minor?: number;
  };
  recommended_actions?: WhatIfRecommendedAction[];
  commit_idempotency_key?: string;
}

export type WhatIfSimulationStatus = "SIMULATED" | "COMMITTED" | string;

export interface WhatIfSimulation {
  tenant_id: string;
  simulation_id: string;
  scenario_id?: string;
  name: string;
  entity_id?: string;
  horizon_days: number;
  shocks_json: string;
  shocks?: WhatIfShocks;
  forecast_delta_minor: number;
  replenishment_cost_minor: number;
  dispatch_cost_delta_minor: number;
  credit_exposure_delta_minor: number;
  net_pnl_delta_minor: number;
  breakdown_json?: string;
  breakdown?: WhatIfStageBreakdown[] | WhatIfStructuredBreakdown | Record<string, unknown>;
  committed_proposal_ids_json?: string;
  committed_proposal_ids?: string[];
  status: WhatIfSimulationStatus;
  created_by: string;
  created_at: string;
  updated_at: string;
}

export interface RunWhatIfSimulationRequest {
  tenant_id?: string;
  name: string;
  scenario_id?: string;
  entity_id?: string;
  horizon_days?: number;
  demand_delta_bps: number;
  factory_downtime_hours: number;
  fx_rate_shock_bps: number;
  credit_limit_delta_bps: number;
  unit_price_delta_bps: number;
  currency?: string;
  idempotency_key?: string;
}

export type WhatIfSimulationResponse = WhatIfSimulation | { simulation: WhatIfSimulation };

export interface WhatIfSimulationsListResponse {
  tenant_id?: string;
  simulations: WhatIfSimulation[];
  count: number;
}

export interface CommitWhatIfProposalsRequest {
  tenant_id?: string;
  proposal_types?: string[];
  agent_id?: string;
}

export interface CommitWhatIfProposalsResponse {
  simulation: WhatIfSimulation;
  proposals: P2BProposalSummary[];
  committed_proposal_ids: string[];
}

// ── §6.5 Cold-Chain HACCP Excursion & Compliance Report ──

export interface HaccpExcursionRow {
  reading_id: string;
  manifest_id: string;
  warehouse_id?: string;
  sensor_id?: string;
  recorded_at: string;
  temp_milli_c: number;
  min_milli_c: number;
  max_milli_c: number;
  breach_delta_milli_c: number;
  severity: string;
  status: string;
}

export interface HaccpAffectedLot {
  lot_id: string;
  warehouse_id?: string;
  product_id: string;
  product_name?: string;
  location_id?: string;
  lot_code?: string;
  status: string;
  quantity_on_hand: number;
  required_zone?: string;
  location_zone?: string;
  reason?: string;
  manifest_id?: string;
}

export interface HaccpZoneMismatch {
  lot_id: string;
  warehouse_id?: string;
  product_id: string;
  product_name?: string;
  location_id: string;
  required_zone: string;
  location_zone: string;
  location_type?: string;
  warning: string;
}

export interface HaccpConditionReport {
  report_id: string;
  order_id: string;
  retailer_id?: string;
  sku?: string;
  condition_type: string;
  severity: string;
  resolution_status: string;
  description?: string;
  reported_by?: string;
  created_at: string;
}

export interface HaccpComplianceReport {
  tenant_id: string;
  from?: string;
  to?: string;
  generated_at: string;
  total_readings: number;
  excursion_count: number;
  compliance_rate_bps: number;
  open_breaches: number;
  affected_lots_count: number;
  affected_lots: HaccpAffectedLot[];
  excursions: HaccpExcursionRow[];
  cold_zone_mismatches?: HaccpZoneMismatch[];
  condition_reports?: HaccpConditionReport[];
}

// ── §7.5 L2 Soak Thresholds & Auto-Execution ──

export interface L2BlockedProposalItem {
  proposal_id: string;
  proposal_type?: string;
  agent_id?: string;
  reason_code: string;
  reason: string;
}

export interface L2EvaluateResponse {
  tenant_id?: string;
  soak_days: number;
  min_soak_days: number;
  executed_count: number;
  min_executed_count: number;
  rollback_count?: number;
  rollback_rate_bps: number;
  max_rollback_rate_bps: number;
  max_l2_amount_minor: number;
  eligible: boolean;
  blockers: string[];
  eligible_proposal_ids?: string[];
  eligible_count?: number;
  auto_executed_count?: number;
  auto_executed_proposal_ids?: string[];
  executed_proposals?: P2BProposalSummary[];
  blocked_proposals?: L2BlockedProposalItem[];
}

export interface RunL2EvaluateRequest {
  tenant_id?: string;
  auto_execute: boolean;
  max_l2_amount_minor?: number;
  proposal_ids?: string[];
}

// ── Enterprise Custom Roles & SCIM 2.0 ──

export interface CustomRole {
  tenant_id: string;
  role_name: string;
  description?: string;
  permissions: string[];
  permissions_json?: string;
  read_only: boolean;
  created_by: string;
  status: string;
  created_at: string;
  updated_at: string;
}

export interface CustomRolesListResponse {
  tenant_id?: string;
  custom_roles: CustomRole[];
  count: number;
}

export interface CreateCustomRoleRequest {
  tenant_id?: string;
  role_name: string;
  description?: string;
  permissions: string[];
  read_only?: boolean;
}

export type CustomRoleResponse = CustomRole | { custom_role: CustomRole };

export interface DeleteCustomRoleResponse {
  deleted: boolean;
  role_name: string;
  status?: string;
}

export interface ScimEmail {
  value: string;
  primary?: boolean;
  type?: string;
}

export interface ScimRoleRef {
  value: string;
  display?: string;
  type?: string;
  primary?: boolean;
}

export interface ScimGroupMemberRef {
  value: string;
  display?: string;
  type?: string;
  $ref?: string;
}

export interface ScimMeta {
  resourceType?: string;
  created?: string;
  lastModified?: string;
  location?: string;
}

export interface ScimUser {
  schemas?: string[];
  id: string;
  externalId?: string;
  userName: string;
  displayName?: string;
  emails?: ScimEmail[];
  active: boolean;
  roles?: ScimRoleRef[];
  meta?: ScimMeta;
}

export interface ScimGroup {
  schemas?: string[];
  id: string;
  externalId?: string;
  displayName: string;
  members?: ScimGroupMemberRef[];
  meta?: ScimMeta;
}

export interface ScimListResponse<T> {
  schemas?: string[];
  totalResults: number;
  startIndex: number;
  itemsPerPage: number;
  Resources: T[];
}

export interface CreateScimUserRequest {
  schemas?: string[];
  externalId?: string;
  userName: string;
  displayName?: string;
  emails?: ScimEmail[];
  active?: boolean;
  roles?: ScimRoleRef[];
}

export interface UpdateScimUserRequest {
  schemas?: string[];
  externalId?: string;
  userName?: string;
  displayName?: string;
  emails?: ScimEmail[];
  active?: boolean;
  roles?: ScimRoleRef[];
}

export interface ScimPatchOperation {
  op: "add" | "replace" | "remove" | string;
  path?: string;
  value?: unknown;
}

export interface ScimPatchRequest {
  schemas?: string[];
  Operations: ScimPatchOperation[];
}

export interface CreateScimGroupRequest {
  schemas?: string[];
  externalId?: string;
  displayName: string;
  members?: ScimGroupMemberRef[];
}

export interface UpdateScimGroupRequest {
  schemas?: string[];
  externalId?: string;
  displayName?: string;
  members?: ScimGroupMemberRef[];
}
