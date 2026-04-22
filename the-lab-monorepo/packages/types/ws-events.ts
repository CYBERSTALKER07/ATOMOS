/**
 * @file packages/types/ws-events.ts
 * @description All WebSocket event types pushed from backend to clients.
 * Sync with: apps/backend-go/ws/, main.go WS push calls, cron.go
 */

import type { OrderState, PaymentGateway } from './order';

// ─── Discriminated Union of All WS Events ───────────────────────────────────
export type WSEvent =
  | OrderStateChangedEvent
  | DriverApproachingEvent
  | ETAUpdatedEvent
  | PaymentRequiredWSEvent
  | CashCollectionRequiredEvent
  | OrderCompletedWSEvent
  | PaymentSettledWSEvent
  | OrderAmendedEvent
  | PaymentFailedWSEvent
  | PaymentExpiredEvent
  | FleetDispatchedWSEvent
  | DispatchLockAcquiredWSEvent
  | DispatchLockReleasedWSEvent
  | FreezeLockAcquiredWSEvent
  | FreezeLockReleasedWSEvent
  | DriverCreatedWSEvent
  | VehicleCreatedWSEvent;

// ─── Order State Changed (Supplier Portal) ──────────────────────────────────
export interface OrderStateChangedEvent {
  type: 'ORDER_STATE_CHANGED';
  order_id: string;
  old_state: OrderState;
  new_state: OrderState;
  driver_id?: string;
  timestamp: string;
}

// ─── Driver Approaching (Retailer + Supplier) ──────────────────────────────
export interface DriverApproachingEvent {
  type: 'DRIVER_APPROACHING';
  order_id: string;
  supplier_id: string;
  retailer_id: string;
  delivery_token: string;
  driver_latitude: number;
  driver_longitude: number;
}

// ─── ETA Updated (Supplier Portal) ─────────────────────────────────────────
export interface ETAUpdatedEvent {
  type: 'ETA_UPDATED';
  order_id: string;
  driver_id: string;
  eta_minutes: number;
  distance_km: number;
}

// ─── Payment Required (Retailer) ────────────────────────────────────────────
export interface PaymentRequiredWSEvent {
  type: 'PAYMENT_REQUIRED';
  order_id: string;
  session_id: string;
  invoice_id: string | null;
  amount: number;
  original_amount: number;
  currency: string;
  payment_method: string;
  gateway: PaymentGateway;
  message: string;
}

// ─── Cash Collection Required (Driver + Retailer) ──────────────────────────
export interface CashCollectionRequiredEvent {
  type: 'CASH_COLLECTION_REQUIRED';
  order_id: string;
  amount: number;
  currency: string;
  retailer_id: string;
  message: string;
}

// ─── Order Completed (Retailer) ─────────────────────────────────────────────
export interface OrderCompletedWSEvent {
  type: 'ORDER_COMPLETED';
  order_id: string;
  amount: number;
  currency: string;
  message: string;
}

// ─── Payment Settled (Driver) ───────────────────────────────────────────────
export interface PaymentSettledWSEvent {
  type: 'PAYMENT_SETTLED';
  order_id: string;
  amount: number;
  currency: string;
  message: string;
}

// ─── Order Amended (Retailer + Driver) ──────────────────────────────────────
export interface OrderAmendedEvent {
  type: 'ORDER_AMENDED';
  order_id: string;
  new_total: number;
  old_total: number;
  currency: string;
  status: string;
  message: string;
}

// ─── Payment Failed (Retailer + Driver) ─────────────────────────────────────
export interface PaymentFailedWSEvent {
  type: 'PAYMENT_FAILED';
  order_id: string;
  session_id?: string;
  gateway: string;
  reason: string;
  message: string;
}

// ─── Payment Expired (Retailer) ─────────────────────────────────────────────
export interface PaymentExpiredEvent {
  type: 'PAYMENT_EXPIRED';
  order_id: string;
  session_id: string;
  gateway: string;
  message: string;
}

// ─── Fleet Dispatched (Driver + Supplier) ──────────────────────────────────
// Mirror of backend-go/kafka/events.go::FleetDispatchedEvent, fan-out by
// notification_dispatcher onto both DRIVER and SUPPLIER hubs.
export interface FleetDispatchedWSEvent {
  type: 'FLEET_DISPATCHED';
  route_id: string;
  manifest_id?: string;
  order_ids: string[];
  driver_id?: string;
  supplier_id?: string;
  warehouse_id?: string;
  geo_zone?: string;
  timestamp: string;
}

// ─── Dispatch Lock Lifecycle (Supplier) ────────────────────────────────────
export interface DispatchLockAcquiredWSEvent {
  type: 'DISPATCH_LOCK_ACQUIRED';
  lock_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  lock_type: string;
  locked_by: string;
  timestamp: string;
}

export interface DispatchLockReleasedWSEvent {
  type: 'DISPATCH_LOCK_RELEASED';
  lock_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  lock_type: string;
  locked_by: string;
  timestamp: string;
}

// ─── Freeze Lock Lifecycle (Supplier) ──────────────────────────────────────
// Emitted alongside DISPATCH_LOCK_* when LockType == MANUAL_DISPATCH; the AI
// worker halts dispatch automation for the affected scope while the lock holds.
export interface FreezeLockAcquiredWSEvent {
  type: 'FREEZE_LOCK_ACQUIRED';
  lock_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  lock_type: string;
  locked_by: string;
  timestamp: string;
}

export interface FreezeLockReleasedWSEvent {
  type: 'FREEZE_LOCK_RELEASED';
  lock_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  lock_type: string;
  locked_by: string;
  timestamp: string;
}

// ─── Fleet Entity Lifecycle (Supplier) ─────────────────────────────────────
export interface DriverCreatedWSEvent {
  type: 'DRIVER_CREATED';
  driver_id: string;
  supplier_id: string;
  name: string;
  phone: string;
  driver_type: string;
  home_node_type?: string;
  home_node_id?: string;
  created_by: string;
  timestamp: string;
}

export interface VehicleCreatedWSEvent {
  type: 'VEHICLE_CREATED';
  vehicle_id: string;
  supplier_id: string;
  vehicle_class: string;
  label: string;
  license_plate: string;
  max_volume_vu: number;
  home_node_type?: string;
  home_node_id?: string;
  created_by: string;
  timestamp: string;
}
