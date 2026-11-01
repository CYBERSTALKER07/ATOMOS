/**
 * PegasusX Palantir Foundry Semantic-Kinetic-Dynamic Triad,
 * Role 7 (Depot Cashier & AML Vault), Role 9 (Fleet Yard & DVIR Red-Tag),
 * and 12 Cross-Role Pre-Execution Guards domain types.
 *
 * Monetary amounts are represented strictly in integer minor units (Tiyin: 1 UZS = 100 Tiyin).
 */

export const MARKING_AML_INVESTIGATION = "MARKING:AML_INVESTIGATION" as const;
export const MARKING_DISCREPANT_QUARANTINE = "MARKING:DISCREPANT_QUARANTINE" as const;
export const MARKING_THERMAL_QUARANTINE_SPOILED = "MARKING:THERMAL_QUARANTINE_SPOILED" as const;
export const MARKING_MICRO_SKIMMING_SUSPECT = "MARKING:MICRO_SKIMMING_SUSPECT" as const;
export const MARKING_LOT_RECALL_QUARANTINE = "MARKING:LOT_RECALL_QUARANTINE" as const;

export type FoundrySecurityMarking =
  | typeof MARKING_AML_INVESTIGATION
  | typeof MARKING_DISCREPANT_QUARANTINE
  | typeof MARKING_THERMAL_QUARANTINE_SPOILED
  | typeof MARKING_MICRO_SKIMMING_SUSPECT
  | typeof MARKING_LOT_RECALL_QUARANTINE;

export const UZS_BANKNOTE_DENOMINATIONS = [
  200000, 100000, 50000, 20000, 10000, 5000, 2000, 1000,
] as const;
export type UZSBanknoteDenomination = (typeof UZS_BANKNOTE_DENOMINATIONS)[number];

export const TIYIN_PER_UZS = 100;
export const DEFAULT_AML_CASH_CEILING_MINOR = 2500000000;
export const STATUTORY_AML_CASH_CEILING_MINOR = DEFAULT_AML_CASH_CEILING_MINOR;
export const UNCAPPED_AML_CEILING_MINOR = 0;

// --- Semantic Layer (Ontology Graph) ---

export interface OntologyObject {
  cell_id: string;
  object_type: string;
  object_id: string;
  supplier_id: string;
  version: number;
  properties: Record<string, unknown>;
  security_markings?: string[];
  valid_from: string;
  updated_at: string;
}

export interface OntologyGraphEdge {
  cell_id: string;
  source_object_type: string;
  source_object_id: string;
  link_type: string;
  target_object_type: string;
  target_object_id: string;
  supplier_id: string;
  created_by_action_id: string;
  metadata?: Record<string, string>;
  valid_from: string;
  valid_to?: string;
}

export interface SearchAroundRequest {
  cell_id: string;
  start_object_type: string;
  start_object_ids: string[];
  allowed_link_types?: string[];
  max_hops?: number;
}

export interface SearchAroundResponse {
  objects: OntologyObject[];
}

export interface LotRecallRequest {
  cell_id: string;
  supplier_id: string;
  raw_material_lot_id: string;
}

export interface RecallBlastRadiusReport {
  cell_id: string;
  supplier_id: string;
  contaminated_raw_material_lot: string;
  affected_work_orders: string[];
  affected_finished_lots: string[];
  quarantined_sscc_pallets: string[];
  quarantined_warehouse_totes: string[];
  in_transit_driver_drop_blocks: string[];
  retailer_reverse_pickup_orders: string[];
  impacted_soliq_facturas: string[];
}

// --- Kinetic Layer (Pre-Execution Action Engine) ---

export interface PreconditionResult {
  rule_name: string;
  passed: boolean;
  reason?: string;
}

export interface ActionExecutionRequest {
  cell_id: string;
  supplier_id: string;
  action_type: string;
  actor_id: string;
  actor_role: string;
  scenario_id?: string;
  idempotency_key: string;
  parameters: Record<string, unknown>;
}

export interface ActionExecutionRecord {
  action_execution_id: string;
  cell_id: string;
  supplier_id: string;
  action_type: string;
  actor_id: string;
  actor_role: string;
  scenario_id?: string;
  status: 'COMMITTED' | 'SIMULATED_IN_SCENARIO' | 'REJECTED_PRECONDITION' | string;
  precondition_report: PreconditionResult[];
  diff_payload: Record<string, unknown>;
  idempotency_key: string;
  executed_at: string;
}

// --- Dynamic Layer (Copy-on-Write Scenario Sandbox) ---

export interface ScenarioPatch {
  cell_id: string;
  scenario_id: string;
  object_type: string;
  object_id: string;
  supplier_id: string;
  operation: 'UPSERT' | 'DELETE' | string;
  base_version: number;
  patch_properties: Record<string, unknown>;
  updated_at: string;
}

export interface ScenarioPatchRequest {
  cell_id: string;
  supplier_id: string;
  object_type: string;
  object_id: string;
  operation: 'UPSERT' | 'DELETE' | string;
  patch_properties: Record<string, unknown>;
}

export interface ScenarioWatermarkConflict {
  cell_id: string;
  object_type: string;
  object_id: string;
  base_version: number;
  live_version: number;
}

export interface ScenarioCommitResult {
  scenario_id: string;
  committed_objects: number;
  conflicts?: ScenarioWatermarkConflict[];
}

// --- Role 7 Depot Cashier & Vault Operations ---

export type CashierSessionStatus =
  | 'AWAITING_BLIND_COUNT_1'
  | 'BLIND_RECOUNT_REQUIRED'
  | 'BALANCED'
  | 'TOLERANCE_VARIANCE'
  | 'DISCREPANT_QUARANTINE';

export interface CashierSessionView {
  session_id: string;
  supplier_id: string;
  depot_id: string;
  driver_id: string;
  route_id: string;
  bag_seal_barcode: string;
  attempt_number: number;
  counted_cash_minor: number;
  expected_cash_minor?: number;
  variance_minor?: number;
  status: CashierSessionStatus | string;
  clearance_token_hash?: string;
  micro_skimming_flagged?: boolean;
  created_at: string;
  resolved_at?: string;
}

export interface StartCashierSessionRequest {
  cell_id: string;
  supplier_id: string;
  depot_id: string;
  driver_id: string;
  route_id: string;
  bag_seal_barcode: string;
  expected_cash_minor: number;
}

export interface BlindCountRequest {
  denominations_uzs: Record<number, number>;
  tolerance_threshold_minor: number;
}

export interface RRNTenderReclassificationRequest {
  session_id: string;
  order_id: string;
  disputed_cash_minor: number;
  gateway_rrn: string;
  rrn_amount_minor: number;
  epod_captured_at: string;
  rrn_captured_at: string;
  actor_id: string;
}

export interface CITHandoverRequest {
  depot_id: string;
  supplier_id: string;
  session_ids: string[];
  courier_badge_id: string;
  cit_seal_barcode: string;
  amount_minor: number;
}

export interface DepotVaultLedgerEntry {
  vault_entry_id: string;
  depot_id: string;
  supplier_id: string;
  session_id?: string;
  cit_manifest_id?: string;
  entry_type: 'DRIVER_DROP_INFLOW' | 'CIT_ARMORED_OUTFLOW' | string;
  amount_minor: number;
  running_balance_minor: number;
  courier_badge_id?: string;
  seal_barcode?: string;
  created_at: string;
}

// --- Role 9 Fleet, Yard & Maintenance Operations ---

export interface DVIRChecklist8Point {
  steering_and_horn_ok: boolean;
  service_and_parking_brakes_ok: boolean;
  tires_tread_and_pressure_ok: boolean;
  lights_and_hazards_ok: boolean;
  wipers_and_fluids_ok: boolean;
  emergency_kit_and_extinguisher_ok: boolean;
  cargo_box_sanitation_ok: boolean;
  reefer_temp_calibrated_ok: boolean;
}

export interface FleetDVIRInspection {
  inspection_id?: string;
  cell_id: string;
  supplier_id: string;
  vehicle_id: string;
  driver_id: string;
  inspection_type: 'PRE_TRIP' | 'POST_TRIP' | string;
  requires_cold_chain: boolean;
  checklist: DVIRChecklist8Point;
  failed_points?: string[];
  critical_defect_found?: boolean;
  red_tag_applied?: boolean;
  odometer_km: number;
  reefer_temp_celsius: number;
  status?: 'DISPATCH_AUTHORIZED' | 'RED_TAG_QUARANTINE' | string;
  maintenance_work_order_id?: string;
  preventative_service_due?: boolean;
  created_at?: string;
}

export type YardBaySlot =
  | 'INBOUND_STAGING'
  | 'CROSS_DOCK'
  | 'OUTBOUND_LOADING'
  | 'MECHANICAL_SERVICE';

export interface YardGateCheckEvent {
  gate_event_id?: string;
  depot_id: string;
  supplier_id: string;
  vehicle_id: string;
  license_plate: string;
  direction: 'GATE_IN' | 'GATE_OUT' | string;
  vehicle_purpose: 'INBOUND_FTL' | 'CROSS_DOCK' | 'OUTBOUND_ROUTE' | 'WORKSHOP' | string;
  bolt_seal_barcode?: string;
  seal_intact: boolean;
  assigned_bay_slot?: YardBaySlot | string;
  created_at?: string;
}

export interface PredictiveReeferRequest {
  vehicle_id: string;
  cooling_rate_deg_per_min: number;
  min_required_rate_deg_per_min: number;
}

export interface PredictiveReeferResponse {
  vehicle_id: string;
  pre_failure_maintenance_hold: boolean;
}

export interface DispatchCheckResponse {
  vehicle_id: string;
  dispatch_authorized: boolean;
}

export interface RescueToteItem {
  tote_id: string;
  stop_sequence: number;
  is_cold_chain: boolean;
  expected_cash_minor: number;
  thermal_excursion_minutes: number;
  max_allowed_excursion_mins: number;
}

export interface MidRouteRescueRequest {
  cell_id: string;
  supplier_id: string;
  original_route_id: string;
  broken_vehicle_id: string;
  original_driver_id: string;
  rescue_vehicle_id: string;
  rescue_driver_id: string;
  last_completed_stop_seq: number;
  new_rescue_bolt_seal: string;
  totes: RescueToteItem[];
}

export interface MidRouteRescueResult {
  rescue_route_id: string;
  original_driver_liability_minor: number;
  rescue_driver_liability_minor: number;
  transferred_tote_ids: string[];
  quarantined_spoiled_tote_ids: string[];
  new_rescue_bolt_seal: string;
}

// --- 12 Cross-Role Pre-Execution Edge Guards ---

export interface VolumeBreakTier {
  min_cases: number;
  discount_bps: number;
}

export interface SplitTenderRejectionInput {
  order_id: string;
  ordered_cases: number;
  accepted_cases: number;
  base_case_price_minor: number;
  rejection_reason: 'DAMAGED_IN_TRANSIT' | 'SHORT_SHIPPED' | 'BUYER_REMORSE_REFUSAL' | string;
  tiers: VolumeBreakTier[];
  original_credit_reserved_minor: number;
  original_cash_planned_minor: number;
}

export interface SplitTenderReconciliationResult {
  order_id: string;
  qualification_cases: number;
  billable_cases: number;
  applied_discount_bps: number;
  effective_case_minor: number;
  final_total_minor: number;
  final_credit_booked_minor: number;
  final_cash_due_minor: number;
  credit_released_minor: number;
  tier_protected: boolean;
}

export interface AMLCheckRequest {
  legal_entity_tin: string;
  supplier_id: string;
  calendar_date: string;
  cash_minor: number;
}

export interface AMLCheckResponse {
  legal_entity_tin: string;
  accumulated_cash_minor: number;
  ceiling_minor: number;
}

export interface OfflineLeaseIssueRequest {
  order_id: string;
  driver_id: string;
  supplier_id: string;
  legal_entity_tin: string;
  calendar_date: string;
  requested_cash_minor: number;
  requested_credit_minor: number;
  ttl_minutes: number;
}

export interface OfflineQuotaLease {
  lease_id: string;
  order_id: string;
  driver_id: string;
  supplier_id: string;
  legal_entity_tin: string;
  calendar_date: string;
  max_cash_allowed_minor: number;
  max_credit_allowed_minor: number;
  expires_at: string;
  status: string;
}

export interface OfflineLeaseValidateRequest {
  lease_id: string;
  collected_cash_minor: number;
  collected_credit_minor: number;
}

export interface OfflineLeaseValidateResponse {
  lease_valid: boolean;
}

export interface FiscalLineItemCheck {
  sku: string;
  ikpu_code: string;
  package_code: string;
}

export interface FiscalPreflightRequest {
  lines: FiscalLineItemCheck[];
  cert_expires_at: string;
}

export interface FiscalPreflightResponse {
  fiscal_preflight_ok: boolean;
}

export interface SlicePayoutEligibilityRequest {
  slice_id: string;
  fiscal_state: string;
}

export interface SlicePayoutEligibilityResponse {
  payout_eligible: boolean;
}

export interface SettlementSliceRecord {
  slice_id: string;
  parent_slice_id?: string;
  order_id: string;
  supplier_id: string;
  net_payout_minor: number;
  status: 'UNSETTLED' | 'BATCHED' | string;
  reason: string;
}

export interface ClawbackSliceRequest {
  original_slice: SettlementSliceRecord;
  approved_damage_claim_minor: number;
}

export interface LoadStackLayer {
  sku: string;
  layer_index: number;
  gross_weight_kg: number;
  max_stack_crush_kg: number;
  volume_dm3: number;
}

export interface ReturnableContainerBalance {
  account_id: string;
  container_sku: string;
  supplier_id: string;
  balance_qty: number;
  deposit_locked_minor: number;
}

export interface Load3DRTIRequest {
  stack_layers: LoadStackLayer[];
  vehicle_capacity_dm3: number;
  account_id: string;
  supplier_id: string;
  container_sku: string;
  delivered_full_containers: number;
  picked_up_empty_containers: number;
  unit_container_volume_dm3: number;
  unit_deposit_minor: number;
}

export interface FXCorridorRequest {
  fx_policy: string;
  checkout_price_minor: number;
  checkout_fx_rate_micros: number;
  current_fx_rate_micros: number;
  max_corridor_bps: number;
  wave_already_released: boolean;
}

export interface FXCorridorResponse {
  effective_price_minor: number;
  error?: string;
}

export interface PickToteState {
  tote_id: string;
  wave_id: string;
  active_picker_id: string;
  status: 'PICKER_IN_HAND' | 'MARSHALING_STAGED' | string;
  completed_line_count: number;
  total_line_count: number;
  picked_lot_ids: string[];
  last_scan_heartbeat_at: string;
}

export interface ToteTakeoverRequest {
  tote_id: string;
  new_picker_id: string;
}
