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