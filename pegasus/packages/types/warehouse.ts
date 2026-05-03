export type WarehouseStaffRole = 'WAREHOUSE_STAFF' | 'PAYLOADER';

export interface WarehouseStaffMember {
  worker_id: string;
  name: string;
  phone: string;
  role: WarehouseStaffRole | string;
  is_active: boolean;
  created_at: string;
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

export interface WarehouseFleetDriver {
  driver_id: string;
  name: string;
  phone: string;
  driver_type?: string;
  vehicle_type?: string;
  license_plate?: string;
  is_active: boolean;
  truck_status: string;
  created_at: string;
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
  status: 'ASSIGNED' | 'UNASSIGNED' | string;
  driver_id: string;
  vehicle_id?: string;
  previously_assigned_driver?: string;
}

export type WarehouseVehicleUnavailableReason =
  | 'MAINTENANCE'
  | 'TRUCK_DAMAGED'
  | 'REGULATORY_HOLD'
  | 'MANUAL_HOLD'
  | string;

export interface WarehouseFleetVehicle {
  vehicle_id: string;
  vehicle_class: string;
  class_label: string;
  label: string;
  license_plate: string;
  max_volume_vu: number;
  capacity_vu: number;
  is_active: boolean;
  status: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  created_at: string;
  assigned_driver_id?: string;
  assigned_driver_name?: string;
  driver_truck_status?: string;
}

export interface WarehouseFleetVehicleListResponse {
  vehicles: WarehouseFleetVehicle[];
  total?: number;
}

export interface WarehouseUpdateVehicleRequest {
  label?: string;
  license_plate?: string;
  is_active?: boolean;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
}

export interface WarehouseVehicleMutationResponse {
  status: string;
  vehicle_id: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
}

export interface WarehouseDispatchOrder {
  order_id: string;
  retailer_name: string;
  total_uzs: number;
  item_count: number;
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
  vehicle_label?: string;
}

export interface WarehouseUnavailableDispatchDriver extends WarehouseDispatchDriver {
  unavailable_reason?: WarehouseVehicleUnavailableReason;
}

export interface WarehouseDispatchPreview {
  orders?: WarehouseDispatchOrder[];
  undispatched_orders: WarehouseDispatchOrder[];
  drivers?: WarehouseDispatchDriver[];
  available_drivers: WarehouseDispatchDriver[];
  unavailable_drivers?: WarehouseUnavailableDispatchDriver[];
  pending_count?: number;
  available_driver_count?: number;
}

export interface WarehouseSupplyRequest {
  request_id: string;
  warehouse_id: string;
  factory_id: string;
  supplier_id: string;
  state: string;
  priority: string;
  requested_delivery_date?: string;
  total_volume_vu: number;
  notes?: string;
  transfer_order_id?: string;
  created_by: string;
  created_at: string;
  updated_at?: string;
}

export interface WarehouseSupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  recommended_qty: number;
  unit_volume_vu: number;
}

export interface WarehouseSupplyRequestDetail extends WarehouseSupplyRequest {
  demand_breakdown?: unknown;
  items: WarehouseSupplyRequestItem[];
}

export interface CreateWarehouseSupplyRequestResponse {
  request_id: string;
  state: string;
  priority: string;
  total_volume_vu: number;
  items_count: number;
}

export interface WarehouseDispatchLock {
  lock_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  lock_type: string;
  locked_at: string;
  unlocked_at?: string;
  locked_by: string;
}

export interface CreateWarehouseDispatchLockResponse {
  lock_id: string;
  lock_type: string;
  status: 'LOCKED' | string;
}

export interface ReleaseWarehouseDispatchLockResponse {
  lock_id: string;
  status: 'RELEASED' | string;
}

export interface WarehouseSupplyRequestUpdateEvent {
  type: 'SUPPLY_REQUEST_UPDATE';
  warehouse_id: string;
  request_id: string;
  state: string;
  timestamp: string;
}

export interface WarehouseDispatchLockChangeEvent {
  type: 'DISPATCH_LOCK_CHANGE';
  warehouse_id: string;
  lock_id: string;
  action: 'ACQUIRED' | 'RELEASED' | string;
  timestamp: string;
}

export interface WarehouseOutboxFailureEvent {
  type: 'OUTBOX_FAILED';
  event_id: string;
  aggregate_id: string;
  topic: string;
  reason: string;
  timestamp: string;
}

export type WarehouseLiveEvent =
  | WarehouseSupplyRequestUpdateEvent
  | WarehouseDispatchLockChangeEvent
  | WarehouseOutboxFailureEvent;