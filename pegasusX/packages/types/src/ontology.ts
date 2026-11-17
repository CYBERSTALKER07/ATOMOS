import {
  MARKING_AML_INVESTIGATION,
  MARKING_DISCREPANT_QUARANTINE,
  MARKING_LOT_RECALL_QUARANTINE,
  MARKING_MICRO_SKIMMING_SUSPECT,
  MARKING_THERMAL_QUARANTINE_SPOILED,
} from "./foundry";

export const ONTOLOGY_SCHEMA_VERSION = "1.0.0" as const;

export const MARKING_FINANCE_ONLY = "MARKING:FINANCE_ONLY" as const;
export const MARKING_PII = "MARKING:PII" as const;

export {
  MARKING_AML_INVESTIGATION,
  MARKING_DISCREPANT_QUARANTINE,
  MARKING_LOT_RECALL_QUARANTINE,
  MARKING_MICRO_SKIMMING_SUSPECT,
  MARKING_THERMAL_QUARANTINE_SPOILED,
};

export const ONTOLOGY_SECURITY_MARKINGS = [
  MARKING_FINANCE_ONLY,
  MARKING_PII,
  MARKING_AML_INVESTIGATION,
  MARKING_DISCREPANT_QUARANTINE,
  MARKING_THERMAL_QUARANTINE_SPOILED,
  MARKING_MICRO_SKIMMING_SUSPECT,
  MARKING_LOT_RECALL_QUARANTINE,
] as const;

export type OntologySecurityMarking = (typeof ONTOLOGY_SECURITY_MARKINGS)[number];

export const ONTOLOGY_OBJECT_TYPES = [
  "Supplier",
  "LegalEntity",
  "Warehouse",
  "Factory",
  "SKU",
  "Lot",
  "Shop",
  "Order",
  "OrderLine",
  "Manifest",
  "Vehicle",
  "Driver",
  "NasiyaInvoice",
  "CreditLine",
  "Payment",
  "FiscalDoc",
  "Proposal",
] as const;

export type OntologyObjectType = (typeof ONTOLOGY_OBJECT_TYPES)[number];

export const ONTOLOGY_LINK_TYPES = [
  "supplies",
  "stocks",
  "ordered",
  "shipped_on",
  "owes",
  "substitutes",
  "recalled_with",
] as const;

export type OntologyLinkType = (typeof ONTOLOGY_LINK_TYPES)[number];

// ── Typed Ontology Object Properties (v1, int64 minor units — zero floats) ──

export interface SupplierOntologyProperties {
  supplier_id?: string;
  name?: string;
  country_code?: string;
  currency?: string;
  is_configured?: boolean;
  is_registered?: boolean;
  region_id?: string;
  market_code?: string;
  home_cell?: string;
  tax_id?: string;
  normalized_tax_id?: string;
  gln?: string;
  phone?: string;
  email?: string;
  contact_name?: string;
  fleet_vehicle_count?: number;
  fleet_max_vu?: number;
  factory_count?: number;
  payment_acceptor?: string;
  status?: string;
}

export interface LegalEntityOntologyProperties {
  legal_entity_id?: string;
  tin?: string;
  tax_id?: string;
  normalized_tax_id?: string;
  company_reg_number?: string;
  legal_name?: string;
  entity_kind?: "SUPPLIER" | "RETAILER" | "PAYER" | string;
  supplier_id?: string;
  retailer_id?: string;
  payer_id?: string;
  country_code?: string;
  gln?: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  bank_name?: string;
  account_holder?: string;
  account_number?: string;
  iban?: string;
  swift_bic?: string;
  currency?: string;
  /** Integer minor units (int64). */
  daily_cash_accumulated_minor?: number;
  /** Integer minor units (int64). */
  daily_cash_ceiling_minor?: number;
  dedup_cluster_id?: string;
  dedup_status?: string;
  is_active?: boolean;
}

export interface WarehouseOntologyProperties {
  warehouse_id?: string;
  supplier_id?: string;
  name?: string;
  country_code?: string;
  region_id?: string;
  h3_cell?: string;
  address?: string;
  gln?: string;
  primary_factory_id?: string;
  secondary_factory_id?: string;
  transfer_mode?: string;
  default_out_of_stock_policy?: string;
  preorder_min_lead_days?: number;
  preorder_max_lead_days?: number;
  is_active?: boolean;
  is_on_shift?: boolean;
  auto_dispatch_enabled?: boolean;
  coverage_radius_km?: number;
  lat?: number;
  lng?: number;
}

export interface FactoryOntologyProperties {
  factory_id?: string;
  supplier_id?: string;
  name?: string;
  country_code?: string;
  h3_cell?: string;
  address?: string;
  daily_output_capacity?: number;
  is_active?: boolean;
  lat?: number;
  lng?: number;
}

export interface SKUOntologyProperties {
  product_id?: string;
  sku?: string;
  supplier_id?: string;
  category_id?: string;
  name?: string;
  barcode?: string;
  ikpu_code?: string;
  package_code?: string;
  /** Integer minor units (int64). */
  price_minor?: number;
  currency?: string;
  stock_quantity?: number;
  unit?: string;
  sale_unit?: string;
  units_per_pack?: number;
  handling_class?: string;
  requires_cold_chain?: boolean;
  is_hazardous?: boolean;
  is_perishable?: boolean;
  min_shelf_life_days?: number;
  stackable?: boolean;
  max_stack_height?: number;
  unit_volume_vu?: number;
  is_active?: boolean;
  version?: number;
}

export interface LotOntologyProperties {
  lot_id?: string;
  supplier_id?: string;
  warehouse_id?: string;
  product_id?: string;
  sku?: string;
  location_id?: string;
  lot_code?: string;
  status?: "AVAILABLE" | "QUARANTINED" | "RECALLED" | "DEPLETED" | "EXPIRED" | string;
  quantity_on_hand?: number;
  quantity_reserved?: number;
  quantity_available?: number;
  manufactured_date?: string;
  expiry_date?: string;
  received_at?: string;
  recall_reason?: string;
}

export interface ShopOntologyProperties {
  retailer_id?: string;
  shop_id?: string;
  supplier_id?: string;
  name?: string;
  legal_name?: string;
  phone?: string;
  email?: string;
  tax_id?: string;
  normalized_tax_id?: string;
  country_code?: string;
  region_id?: string;
  h3_cell?: string;
  delivery_address?: string;
  gln?: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
  timezone?: string;
  min_shelf_life_days?: number;
  segment?: string;
  lat?: number;
  lng?: number;
}

export interface OrderOntologyProperties {
  order_id?: string;
  parent_order_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  warehouse_id?: string;
  driver_id?: string;
  vehicle_id?: string;
  route_id?: string;
  manifest_id?: string;
  status?: string;
  order_source?: string;
  confirmation_status?: string;
  fiscal_status?: string;
  buyer_acceptance_status?: string;
  /** Integer minor units (int64). */
  total_minor?: number;
  /** Integer minor units (int64). */
  original_total_minor?: number;
  /** Integer minor units (int64). */
  delivery_fee_minor?: number;
  currency?: string;
  delivery_priority?: string;
  partial_delivery?: boolean;
  h3_cell?: string;
  requested_delivery_date?: string;
  latest_fiscal_receipt_id?: string;
  line_count?: number;
  version?: number;
}

export interface OrderLineOntologyProperties {
  order_line_id?: string;
  order_id?: string;
  supplier_id?: string;
  warehouse_id?: string;
  product_id?: string;
  sku?: string;
  substitute_sku?: string;
  original_sku?: string;
  name?: string;
  quantity?: number;
  delivered_qty?: number;
  remaining_qty?: number;
  /** Integer minor units (int64). */
  unit_price_minor?: number;
  /** Integer minor units (int64). */
  line_total_minor?: number;
  /** Integer minor units (int64). */
  vat_minor?: number;
  /** Integer minor units (int64). */
  price_gap_minor?: number;
  currency?: string;
  ikpu_code?: string;
  package_code?: string;
  allocation_mode?: string;
  offload_status?: string;
}

export interface ManifestOntologyProperties {
  manifest_id?: string;
  supplier_id?: string;
  warehouse_id?: string;
  route_id?: string;
  truck_id?: string;
  vehicle_id?: string;
  driver_id?: string;
  pick_wave_id?: string;
  state?: string;
  status?: string;
  stop_count?: number;
  replan_count?: number;
  replan_reason?: string;
  total_volume_vu?: number;
  max_volume_vu?: number;
  sealed_at?: string;
  dispatched_at?: string;
  completed_at?: string;
}

export interface VehicleOntologyProperties {
  vehicle_id?: string;
  supplier_id?: string;
  label?: string;
  license_plate?: string;
  home_node_type?: string;
  home_node_id?: string;
  vehicle_class?: string;
  max_volume_vu?: number;
  is_active?: boolean;
  unavailable_reason?: string;
  dvir_status?: string;
  red_tag_applied?: boolean;
  dispatch_authorized?: boolean;
}

export interface DriverOntologyProperties {
  driver_id?: string;
  supplier_id?: string;
  name?: string;
  phone?: string;
  home_node_type?: string;
  home_node_id?: string;
  vehicle_id?: string;
  is_active?: boolean;
  on_shift?: boolean;
  unavailable_reason?: string;
}

export interface NasiyaInvoiceOntologyProperties {
  invoice_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  order_id?: string;
  status?: "OPEN" | "PARTIALLY_PAID" | "SETTLED" | "OVERDUE" | "WRITTEN_OFF" | string;
  /** Integer minor units (int64). */
  principal_minor?: number;
  /** Integer minor units (int64). */
  balance_minor?: number;
  /** Integer minor units (int64). */
  paid_minor?: number;
  /** Integer minor units (int64). */
  amount_minor?: number;
  currency?: string;
  terms_days?: number;
  grace_period_days?: number;
  aging_bucket?: string;
  dunning_step?: number;
  credit_leave_at?: string;
  due_at?: string;
  last_dunned_at?: string;
  version?: number;
}

export interface CreditLineOntologyProperties {
  credit_line_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  status?: string;
  credit_enabled?: boolean;
  /** Integer minor units (int64). */
  credit_limit_minor?: number;
  /** Integer minor units (int64). */
  current_balance_minor?: number;
  /** Integer minor units (int64). */
  reserved_minor?: number;
  /** Integer minor units (int64). */
  available_credit_minor?: number;
  currency?: string;
  terms_days?: number;
  grace_period_days?: number;
  risk_score?: number;
  delinquency_count?: number;
  use_global_defaults?: boolean;
  version?: number;
}

export interface PaymentOntologyProperties {
  payment_id?: string;
  session_id?: string;
  order_id?: string;
  invoice_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  gateway?: string;
  mode?: string;
  status?: string;
  /** Integer minor units (int64). */
  amount_minor?: number;
  /** Integer minor units (int64). */
  cash_minor?: number;
  /** Integer minor units (int64). */
  card_minor?: number;
  /** Integer minor units (int64). */
  refunded_minor?: number;
  currency?: string;
  provider_reference?: string;
  rrn?: string;
  settled_at?: string;
}

export interface FiscalDocOntologyProperties {
  fiscal_doc_id?: string;
  attempt_id?: string;
  order_id?: string;
  credit_note_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  provider?: string;
  doc_type?: "RECEIPT" | "EHF" | "CORRECTIVE_EHF" | string;
  status?: string;
  fiscal_receipt_id?: string;
  ehf_id?: string;
  buyer_acceptance_status?: string;
  /** Integer minor units (int64). */
  amount_minor?: number;
  /** Integer minor units (int64). */
  net_minor?: number;
  /** Integer minor units (int64). */
  vat_minor?: number;
  /** Integer minor units (int64). */
  gross_minor?: number;
  currency?: string;
  payment_method?: string;
  fiscal_qr?: string;
}

export interface ProposalOntologyProperties {
  proposal_id?: string;
  proposal_kind?:
    | "AUTO_ORDER_SHADOW"
    | "NEGOTIATION"
    | "REPLENISHMENT"
    | "FINANCIAL_DRAFT"
    | "CONTROL_TOWER"
    | string;
  supplier_id?: string;
  retailer_id?: string;
  order_id?: string;
  driver_id?: string;
  warehouse_id?: string;
  product_id?: string;
  sku?: string;
  status?: string;
  current_stock?: number;
  suggested_quantity?: number;
  proposed_qty?: number;
  /** Integer minor units (int64). */
  proposed_amount_minor?: number;
  /** Integer minor units (int64). */
  proposed_price_minor?: number;
  urgency_level?: string;
  currency?: string;
  reason_code?: string;
  resolution?: string;
  resolved_by?: string;
  resolved_at?: string;
  expires_at?: string;
}

export interface OntologyPropertiesByType {
  Supplier: SupplierOntologyProperties;
  LegalEntity: LegalEntityOntologyProperties;
  Warehouse: WarehouseOntologyProperties;
  Factory: FactoryOntologyProperties;
  SKU: SKUOntologyProperties;
  Lot: LotOntologyProperties;
  Shop: ShopOntologyProperties;
  Order: OrderOntologyProperties;
  OrderLine: OrderLineOntologyProperties;
  Manifest: ManifestOntologyProperties;
  Vehicle: VehicleOntologyProperties;
  Driver: DriverOntologyProperties;
  NasiyaInvoice: NasiyaInvoiceOntologyProperties;
  CreditLine: CreditLineOntologyProperties;
  Payment: PaymentOntologyProperties;
  FiscalDoc: FiscalDocOntologyProperties;
  Proposal: ProposalOntologyProperties;
}

export interface OntologyObject<
  TType extends OntologyObjectType | string = OntologyObjectType | string,
  TProps extends Record<string, unknown> = Record<string, unknown>,
> {
  cell_id: string;
  object_type: TType;
  object_id: string;
  supplier_id: string;
  version: number;
  properties: TProps;
  security_markings?: (OntologySecurityMarking | string)[];
  valid_from: string;
  updated_at: string;
}

export type TypedOntologyObject = {
  [K in OntologyObjectType]: OntologyObject<K, OntologyPropertiesByType[K] & Record<string, unknown>>;
}[OntologyObjectType];

export interface OntologyGraphEdge {
  cell_id: string;
  source_object_type: OntologyObjectType | string;
  source_object_id: string;
  link_type: OntologyLinkType | string;
  target_object_type: OntologyObjectType | string;
  target_object_id: string;
  supplier_id: string;
  created_by_action_id: string;
  metadata?: Record<string, string>;
  valid_from: string;
  valid_to?: string;
}

export interface OntologyNeighborNode {
  object: OntologyObject;
  hop: number;
  via_link_type?: OntologyLinkType | string;
  direction?: "OUTBOUND" | "INBOUND" | string;
  parent_type?: OntologyObjectType | string;
  parent_id?: string;
}

export interface OntologyNeighborResponse {
  cell_id: string;
  supplier_id: string;
  root: OntologyObject;
  depth: number;
  nodes: OntologyNeighborNode[];
  edges: OntologyGraphEdge[];
  total_nodes: number;
  total_edges: number;
  pii_masked: boolean;
  finance_masked: boolean;
}

export interface OntologyTypeDescriptor {
  object_type: OntologyObjectType | string;
  version: string;
  schema_version?: string;
  schema_file: string;
  primary_key: string[];
  required_properties: string[];
  money_fields_minor?: string[];
  security_markings?: (OntologySecurityMarking | string)[];
  allowed_outgoing_links?: (OntologyLinkType | string)[];
  source_tables?: string[];
  pii_fields?: string[];
  finance_fields?: string[];
  description: string;
  count?: number;
}

export interface OntologyTypesResponse {
  version: string;
  types: OntologyTypeDescriptor[];
  link_types: (OntologyLinkType | string)[];
  security_markings: (OntologySecurityMarking | string)[];
}

export interface OntologyObjectsResponse {
  cell_id: string;
  supplier_id: string;
  objects: OntologyObject[];
  total: number;
  limit: number;
  offset: number;
  pii_masked: boolean;
  finance_masked: boolean;
}

export interface OntologyRebuildRequest {
  cell_id?: string;
  supplier_id?: string;
}

export interface OntologyRebuildResponse {
  cell_id: string;
  supplier_id: string;
  status: string;
  objects_upserted: number;
  edges_upserted: number;
  batches_committed: number;
  duration_ms: number;
  tables_scanned: string[];
  rebuilt_at: string;
}

export interface OntologyObjectChangedPayload {
  type?: "ontology.object.changed" | string;
  version: number;
  timestamp?: string;
  cell_id: string;
  object_type: OntologyObjectType | string;
  object_id: string;
  supplier_id: string;
  change_kind: string;
  source_event_type?: string;
  source_event_id?: string;
  security_markings?: string[];
  updated_at: string;
}
