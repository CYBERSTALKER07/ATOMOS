import { OrderId, RetailerId, SupplierId } from "./primitives";
import { RouteGeometryWire } from "./compliance";
import { WarehouseSupplyRequest } from "./event-payloads";

// ── Warehouse portal / desktop (fleet, supply detail, live WS) ──

export type WarehouseStaffRole = "WAREHOUSE_STAFF" | "PAYLOADER";

export interface WarehouseStaffMember {
  worker_id: string;
  name: string;
  phone: string;
  role: WarehouseStaffRole | string;
  is_active: boolean;
  created_at?: string;
}

export interface WarehouseStaffListResponse {
  staff: WarehouseStaffMember[];
}

export interface CreateWarehouseStaffRequest {
  name: string;
  phone: string;
  role: WarehouseStaffRole;
  pin?: string;
}

export interface CreateWarehouseStaffResponse {
  worker_id: string;
  name?: string;
  role?: WarehouseStaffRole | string;
  pin: string;
}

export type WarehouseVehicleUnavailableReason =
  | "MAINTENANCE"
  | "TRUCK_DAMAGED"
  | "REGULATORY_HOLD"
  | "MANUAL_HOLD"
  | "OTHER"
  | string;

export interface WarehouseFleetDriver {
  driver_id: string;
  name: string;
  phone: string;
  driver_type?: string;
  vehicle_type?: string;
  license_plate?: string;
  is_active: boolean;
  on_shift?: boolean;
  truck_status: string;
  unavailable_reason?: string;
  unavailable_note?: string;
  created_at?: string;
  vehicle_id?: string;
  vehicle_class?: string;
  max_volume_vu?: number;
  vehicle_is_active?: boolean;
  vehicle_unavailable_reason?: WarehouseVehicleUnavailableReason;
}

export interface WarehouseFleetDriverListResponse {
  drivers: WarehouseFleetDriver[];
}

export interface WarehouseAssignVehicleRequest {
  vehicle_id?: string;
}

export interface WarehouseAssignVehicleResponse {
  status: "ASSIGNED" | "UNASSIGNED" | string;
  driver_id: string;
  vehicle_id?: string;
  previously_assigned_driver?: string;
}

export interface WarehouseFleetVehicle {
  vehicle_id: string;
  vehicle_class: string;
  class_label?: string;
  label: string;
  license_plate: string;
  max_volume_vu?: number;
  capacity_vu: number;
  is_active: boolean;
  status: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
  created_at?: string;
  assigned_driver_id?: string;
  assigned_driver_name?: string;
  driver_truck_status?: string;
}

export interface WarehouseFleetVehicleListResponse {
  vehicles: WarehouseFleetVehicle[];
  total?: number;
}

export interface WarehouseFleetVehicleDetailResponse {
  vehicle: WarehouseFleetVehicle;
}

export interface WarehouseUpdateVehicleRequest {
  label?: string;
  license_plate?: string;
  is_active?: boolean;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseVehicleMutationResponse {
  status: string;
  vehicle_id: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseDispatchOrder {
  order_id: string;
  retailer_name: string;
  total_uzs: number;
  item_count: number;
  volume_vu?: number;
  created_at?: string;
}

export interface WarehouseDispatchDriver {
  driver_id: string;
  name: string;
  phone?: string;
  truck_status: string;
  vehicle_id?: string;
  vehicle_class?: string;
  max_volume_vu?: number;
  used_volume_vu?: number;
  free_volume_vu?: number;
  active_manifest_id?: string;
  vehicle_label?: string;
}

export interface WarehouseUnavailableDispatchDriver extends WarehouseDispatchDriver {
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseDispatchProposedStop {
  order_id: string;
  retailer_id?: string;
  retailer_name?: string;
  volume_vu?: number;
  sequence?: number;
}

export interface WarehouseDispatchProposedRoute {
  driver_id?: string;
  driver_name?: string;
  order_ids?: string[];
  stops?: WarehouseDispatchProposedStop[];
  volume_vu?: number;
  loaded_volume?: number;
  max_volume_vu?: number;
  max_volume?: number;
  util_pct?: number;
  stop_count?: number;
  route_geometry?: RouteGeometryWire;
}

export interface WarehouseDispatchPreview {
  orders?: WarehouseDispatchOrder[];
  undispatched_orders: WarehouseDispatchOrder[];
  drivers?: WarehouseDispatchDriver[];
  available_drivers: WarehouseDispatchDriver[];
  unavailable_drivers?: WarehouseUnavailableDispatchDriver[];
  pending_count?: number;
  available_driver_count?: number;
  preview_ready?: boolean;
  proposed_routes?: WarehouseDispatchProposedRoute[];
  optimizer_source?: string;
  optimizer_warnings?: string[];
  window_constrained_count?: number;
  fleet_effective_capacity_vu?: number;
  selected_orders_volume_vu?: number;
  plan_fingerprint?: string;
  plan_computed_at?: string;
  plan_stale?: boolean;
}

export interface WarehouseDispatchCapacityWarning {
  driver_id: string;
  loaded_vu: number;
  max_volume_vu: number;
  effective_max_vu: number;
  excess_vu?: number;
  suggested_unselect_order_ids?: string[];
  suggested_defer_order_ids?: string[];
  fleet_effective_capacity_vu?: number;
  requested_volume_vu?: number;
}

export interface WarehouseDispatchExecuteRoute {
  manifest_id?: string;
  route_id?: string;
  driver_id: string;
  vehicle_id?: string;
  order_ids: string[];
  volume_vu?: number;
  max_volume_vu?: number;
}

export interface WarehouseDispatchExecuteRequest {
  mode: "MANUAL" | "AUTO" | string;
  routes?: WarehouseDispatchExecuteRoute[];
  order_ids?: string[];
  force_capacity?: boolean;
  accept_partial?: boolean;
  plan_fingerprint?: string;
}

export interface WarehouseDispatchExecuteResponse {
  status: string;
  supplier_id: string;
  warehouse_id?: string;
  manifests_created?: number;
  orders_assigned?: number;
  optimizer_source?: string;
  warnings?: string[];
  capacity_warnings?: WarehouseDispatchCapacityWarning[];
  manifests?: WarehouseDispatchExecuteRoute[];
  orphan_order_ids?: string[];
}

export interface ProductHandlingUpdatedEvent {
  type: "PRODUCT_HANDLING_UPDATED";
  product_id: string;
  supplier_id: SupplierId;
  handling_class: "GENERAL" | "COLD_CHAIN" | "HAZARDOUS" | "PERISHABLE";
  requires_cold_chain: boolean;
  is_hazardous: boolean;
  is_perishable: boolean;
  storage_temp_min_c?: number | null;
  storage_temp_max_c?: number | null;
}

export interface OrderConditionReportedEvent {
  type: "ORDER_CONDITION_REPORTED";
  report_id: string;
  order_id: OrderId;
  reporter_id: string;
  reporter_role: string;
  condition_type: "DAMAGED" | "EXPIRED" | "TEMPERATURE_BREACH" | "MISSING" | "QUALITY_REJECT" | "OTHER";
  sku?: string;
  quantity?: number;
  gcs_paths?: string[];
  notes?: string;
}

export interface RetailerCreditProfileChangedEvent {
  type: "RETAILER_CREDIT_PROFILE_CHANGED";
  profile_id: string;
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  credit_limit_minor: number;
  current_balance: number;
  risk_tier: "LOW" | "MEDIUM" | "HIGH" | "BLOCK";
  delinquent: boolean;
  reason?: string;
}

export interface RetailerCreditLimitBreachedEvent {
  type: "RETAILER_CREDIT_LIMIT_BREACHED";
  order_id: OrderId;
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  requested_amount: number;
  credit_limit_minor: number;
  current_balance: number;
}

export interface CatalogProduct {
  product_id: string;
  supplier_id: string;
  category_id: string;
  name: string;
  description?: string;
  image_url?: string;
  price_minor: number;
  currency: string;
  stock_quantity: number;
  unit: string;
  unit_volume_vu?: number;
  sale_unit?: string;
  units_per_pack?: number | null;
  is_active: boolean;
  version: number;
  barcode?: string;
  handling_class?: "GENERAL" | "COLD_CHAIN" | "HAZARDOUS" | "PERISHABLE";
  requires_cold_chain?: boolean;
  is_hazardous?: boolean;
  is_perishable?: boolean;
  storage_temp_min_c?: number | null;
  storage_temp_max_c?: number | null;
  created_at?: string;
  updated_at?: string;
}

export interface WarehouseSupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  shipped_quantity?: number;
  received_quantity?: number;
  recommended_qty: number;
  unit_volume_vu: number;
}

export interface WarehouseSupplyRequestDetail extends WarehouseSupplyRequest {
  demand_breakdown?: unknown;
  items: WarehouseSupplyRequestItem[];
  factory_id?: string;
  priority?: string;
  notes?: string;
  transfer_order_id?: string;
  linked_transfer_id?: string;
  created_by?: string;
}

export interface CreateWarehouseSupplyRequestResponse {
  request_id: string;
  state: string;
  priority: string;
  total_volume_vu: number;
  items_count: number;
}

export interface CreateWarehouseDispatchLockResponse {
  lock_id: string;
  lock_type: string;
  status: "LOCKED" | string;
}

export interface WarehouseCRMRetailer {
  retailer_id: string;
  business_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date?: string;
}

export interface WarehouseCRMListResponse {
  retailers: WarehouseCRMRetailer[];
}

export interface WarehouseReturnItem {
  line_item_id: string;
  order_id: string;
  product_name: string;
  quantity: number;
  status: string;
  updated_at?: string;
}

export interface WarehouseReturnListResponse {
  items: WarehouseReturnItem[];
}

export interface WarehouseTreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
}

export interface WarehouseSupplyRequestUpdateEvent {
  type: "SUPPLY_REQUEST_UPDATE";
  warehouse_id: string;
  request_id: string;
  state: string;
  timestamp: string;
}

export interface WarehouseDispatchLockChangeEvent {
  type: "DISPATCH_LOCK_CHANGE";
  warehouse_id: string;
  lock_id: string;
  action: "ACQUIRED" | "RELEASED" | string;
  timestamp: string;
}

export interface WarehouseOutboxFailureEvent {
  type: "OUTBOX_FAILED";
  event_id: string;
  aggregate_id: string;
  topic: string;
  reason: string;
  timestamp: string;
}

export interface WarehouseInventorySyncCompleteEvent {
  type: "INVENTORY_SYNC_COMPLETE";
  supplier_id: string;
  warehouse_id?: string;
  session_id: string;
  rows_affected: number;
  affected_warehouses?: number;
  product_ids?: string[];
  source?: string;
  timestamp: string;
}

export type WarehouseLiveEvent =
  | WarehouseSupplyRequestUpdateEvent
  | WarehouseDispatchLockChangeEvent
  | WarehouseOutboxFailureEvent
  | WarehouseInventorySyncCompleteEvent;

