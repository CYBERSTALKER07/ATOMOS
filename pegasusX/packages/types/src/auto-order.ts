import { RetailerId, SupplierId } from "./primitives";

// ── §8.3 Inventory-grounded auto-order ──────────────────────────────────────
export type AutoOrderExecutionMode = "off" | "shadow" | "draft" | "place";

export interface AutoOrderShadowStats {
  proposal_count: number;
  matched_orders: number;
  wape: number;
  unmodified_accept_rate: number;
  window_days: number;
}

export interface AutoOrderShadowProposal {
  proposal_id: string;
  retailer_id: string;
  sku: string;
  supplier_id?: string;
  proposed_qty: number;
  ip: number;
  reorder_point: number;
  order_up_to: number;
  confidence?: number;
  reason?: string;
  bucket_date: string;
  status: string;
  run_id?: string;
  created_at?: string;
}

export interface RetailerAutoOrderSettings {
  global_enabled: boolean;
  execution_mode?: AutoOrderExecutionMode | string;
  has_any_history?: boolean;
  analytics_start_date?: string;
  supplier_overrides?: Array<{ supplier_id: string; enabled: boolean }>;
  category_overrides?: Array<{ category_id: string; enabled: boolean }>;
  product_overrides?: Array<{ product_id: string; enabled: boolean }>;
  variant_overrides?: Array<{ variant_id: string; enabled: boolean }>;
  shadow_stats?: AutoOrderShadowStats;
}

/** §8.7 Wave 1A warehouse bin / slot location. */
export interface WarehouseBinLocation {
  warehouse_id: string;
  location_id: string;
  zone?: string;
  aisle?: string;
  rack?: string;
  level?: string;
  bin?: string;
  location_type: "PICK" | "BULK" | "STAGE" | "QUARANTINE" | string;
  pick_sequence: number;
  max_volume_vu?: number;
  is_active: boolean;
  updated_at?: string;
}

export interface WarehouseBinListResponse {
  bins: WarehouseBinLocation[];
  lots_enabled?: boolean;
}

export interface WarehouseBinCreateRequest {
  location_id?: string;
  zone?: string;
  aisle?: string;
  rack?: string;
  level?: string;
  bin?: string;
  location_type?: string;
  pick_sequence?: number;
  max_volume_vu?: number;
}

/** §8.7 Wave 1A stock lot with expiry. */
export interface StockLot {
  lot_id: string;
  supplier_id: string;
  warehouse_id: string;
  product_id: string;
  location_id: string;
  lot_code?: string;
  expiry_date?: string;
  manufactured_date?: string;
  quantity_on_hand: number;
  quantity_reserved: number;
  status: "AVAILABLE" | "QUARANTINE" | "EXPIRED" | "DEPLETED" | string;
  received_at?: string;
}

export interface StockLotListResponse {
  lots: StockLot[];
  lots_enabled?: boolean;
}

export interface StockLotPutawayRequest {
  product_id: string;
  location_id: string;
  quantity: number;
  lot_code?: string;
  expiry_date?: string;
  manufactured_date?: string;
  lot_id?: string;
}

export interface StockLotPutawayResponse {
  lot_id: string;
  product_id: string;
  location_id: string;
  lot_code?: string;
  quantity_on_hand: number;
  quantity_reserved: number;
  status: string;
  expiry_date?: string;
  received_at?: string;
}

export interface QuarantineLotRequest {
  reason_code?: string;
  notes?: string;
}

export interface ReleaseLotRequest {
  reason_code?: string;
  notes?: string;
}

export interface LotQuarantineEventView {
  event_id: string;
  lot_id: string;
  warehouse_id: string;
  supplier_id: string;
  product_id: string;
  from_status: string;
  to_status: string;
  reason_code: string;
  actor: string;
  notes?: string;
  created_at: string;
}

export interface RecallImpactedOrderView {
  campaign_id: string;
  order_id: string;
  retailer_id: string;
  warehouse_id: string;
  lot_id: string;
  sku: string;
  quantity: number;
  order_status: string;
  customer_notified: boolean;
  created_at?: string;
}

export interface RecallCampaignView {
  campaign_id: string;
  supplier_id: string;
  product_id: string;
  lot_code?: string;
  lot_id?: string;
  recall_reason: string;
  severity: "CRITICAL" | "WARNING" | string;
  status: "INITIATED" | "IN_PROGRESS" | "COMPLETED" | "CANCELLED" | string;
  initiated_by: string;
  impacted_lot_count: number;
  impacted_units_count: number;
  impacted_order_count: number;
  created_at: string;
  updated_at: string;
  impacted_orders?: RecallImpactedOrderView[];
}

export interface InitiateRecallRequest {
  supplier_id?: string;
  product_id: string;
  lot_code?: string;
  lot_id?: string;
  recall_reason: string;
  severity?: "CRITICAL" | "WARNING" | string;
  initiated_by?: string;
}

export interface LotGenealogyView {
  lot: StockLot;
  quarantine_events: LotQuarantineEventView[];
  impacted_orders: RecallImpactedOrderView[];
}

/** §8.7 Wave 1B pick wave (manifest strategy). */
export interface PickWave {
  wave_id: string;
  warehouse_id: string;
  supplier_id: string;
  manifest_id: string;
  strategy: "MANIFEST" | string;
  status: "OPEN" | "PICKING" | "READY_TO_SEAL" | "CANCELLED" | string;
  created_at?: string;
  ready_at?: string;
  tasks?: PickTask[];
}

export interface PickTask {
  task_id: string;
  order_id: string;
  product_id: string;
  lot_id: string;
  location_id: string;
  quantity_requested: number;
  quantity_picked: number;
  picker_id?: string;
  status: "PENDING" | "CONFIRMED" | "SHORT" | "SHORT_WAIVED" | string;
  pick_sequence: number;
}

export interface PickWaveListResponse {
  waves: PickWave[];
  pick_waves_enabled?: boolean;
}

export interface PickWaveCreateRequest {
  manifest_id: string;
}

export interface PickTaskConfirmRequest {
  quantity_picked?: number;
}

/** §8.7 Wave 1C cycle count stub. */
export interface CycleCount {
  count_id: string;
  warehouse_id: string;
  location_id: string;
  product_id: string;
  lot_id?: string;
  expected_qty: number;
  counted_qty?: number;
  variance_qty?: number;
  reason_code?: string;
  status: "OPEN" | "SUBMITTED" | "CANCELLED" | string;
  counted_by?: string;
  counted_at?: string;
  created_at?: string;
}

export interface CycleCountListResponse {
  counts: CycleCount[];
  cycle_counts_enabled?: boolean;
}

export interface CycleCountCreateRequest {
  location_id: string;
  product_id: string;
  lot_id?: string;
  expected_qty?: number;
}

export interface CycleCountSubmitRequest {
  counted_qty: number;
  reason_code?: string;
}

export interface InventoryAdjustment {
  adjustment_id: string;
  warehouse_id: string;
  product_id: string;
  lot_id?: string;
  count_id?: string;
  delta_qty: number;
  reason_code?: string;
  status: "PENDING" | "APPROVED" | "REJECTED" | string;
  actor_id?: string;
  approved_by?: string;
  created_at?: string;
}

export interface InventoryAdjustmentListResponse {
  adjustments: InventoryAdjustment[];
  cycle_counts_enabled?: boolean;
}

// --- Phase 2.3: Evidence Vault ---

export interface EvidenceItem {
  dossier_id: string;
  item_id: string;
  item_type: string;
  storage_uri: string;
  file_hash: string;
  mime_type: string;
  size_bytes: number;
  uploader_user_id: string;
  captured_at?: string;
  latitude?: number;
  longitude?: number;
  created_at: string;
}

export interface EvidenceDossier {
  dossier_id: string;
  target_id: string;
  target_type: string;
  status: 'OPEN' | 'SEALED';
  sealed_at?: string;
  sealed_hash?: string;
  created_at: string;
}

export interface CreateDossierRequest {
  target_id: string;
  target_type: string;
}

export interface AddEvidenceItemRequest {
  item_type: string;
  file_hash: string;
  mime_type: string;
  size_bytes: number;
  captured_at?: string;
  latitude?: number;
  longitude?: number;
  extension: string; // Used to generate the upload ticket
}

export interface AddEvidenceItemResponse {
  item: EvidenceItem;
  upload_url: string; // The signed PUT URL for the client to upload the file
  public_url: string; // The resulting public URL
}

export interface GetDossierResponse {
  dossier: EvidenceDossier;
  items: EvidenceItem[];
}


export interface TaxRegimeVersion {
  id: string;
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rates_bps: number[];
  simplified_rules?: any;
  created_at: string;
  created_by: string;
  updated_at: string;
}

export interface CreateRegimeRequest {
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rates_bps: number[];
  simplified_rules?: any;
}

export interface OrderLineFiscalSnapshot {
  order_id: string;
  order_line_id: string;
  regime_id: string;
  vat_rate_bps: number;
  net_minor: number;
  vat_minor: number;
  gross_minor: number;
  snapshot_at: string;
  created_at: string;
}

export interface SupplierPromotionCampaign {
  campaign_id: string;
  supplier_id: SupplierId;
  name: string;
  budget_limit_minor: number;
  budget_used_minor: number;
  status: "ACTIVE" | "EXHAUSTED" | "PAUSED";
  created_at: string;
}

export interface RetailerPromotionEnrollment {
  enrollment_id: string;
  campaign_id: string;
  retailer_id: RetailerId;
  status: "ENROLLED" | "OPTED_OUT";
  enrolled_at: string;
}

export interface RetailerShelfAlert {
  alert_id: string;
  retailer_id: RetailerId;
  location_id: string;
  global_product_id: string;
  status: "OPEN" | "RESOLVED";
  current_stock: number;
  capacity_threshold: number;
  created_at: string;
  resolved_at?: string;
}

