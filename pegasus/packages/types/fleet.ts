/**
 * @file packages/types/fleet.ts
 * @description Fleet, dispatch, and telemetry types.
 * Sync with: apps/backend-go/fleet/, apps/backend-go/supplier/dispatcher.go
 */

import type { OrderState, PaymentGateway } from './order';
import type { RouteStatus, DriverTruckStatus } from './entities';

// ─── Fleet Capacity ─────────────────────────────────────────────────────────
export interface CapacityInfo {
  available_slots: number;
  total_capacity: number;
  assignments: TruckAssignment[];
}

export interface TruckAssignment {
  truck_id: string;
  plate_number: string;
  capacity_kg: number;
  assigned_orders: number;
  driver_id: string | null;
  driver_name: string | null;
}

// ─── Auto-Dispatch ──────────────────────────────────────────────────────────
export interface AutoDispatchRequest {
  order_ids: string[];
  excluded_truck_ids?: string[];
}

export interface AutoDispatchResult {
  manifests: TruckManifest[];
  orphans: OrphanOrder[];
}

export interface TruckManifest {
  truck_id: string;
  plate_number: string;
  driver_id: string;
  driver_name: string;
  order_ids: string[];
  total_weight_kg: number;
  route_distance_km: number;
}

export interface OrphanOrder {
  order_id: string;
  reason: string;
}

// ─── Waiting Room ───────────────────────────────────────────────────────────
export interface WaitingRoomOrder {
  order_id: string;
  retailer_id: string;
  retailer_name: string;
  total: number;
  currency: string;
  item_count: number;
  created_at: string;
  state: OrderState;
}

// ─── Driver Earnings ────────────────────────────────────────────────────────
export interface DriverEarningsResponse {
  total_deliveries: number;
  total_volume: number;
  total_routes: number;
  last_30_days: DailyEarning[];
}

export interface DailyEarning {
  date: string;        // "2026-04-10"
  deliveries: number;
  volume: number;
}

// ─── Delivery History ───────────────────────────────────────────────────────
export interface DeliveryHistoryItem {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  state: OrderState;
  amount: number;
  currency: string;
  route_id: string;
  completed_at: string | null;
}

// ─── Telemetry (GPS) ────────────────────────────────────────────────────────
export interface GPSPing {
  driver_id: string;
  latitude: number;
  longitude: number;
  timestamp?: string;
  battery?: number;
}

// ─── Fleet Map Driver Info ──────────────────────────────────────────────────
export interface FleetDriverInfo {
  driver_id: string;
  name: string;
  phone: string;
  vehicle_plate: string;
  vehicle_model: string;
  truck_status: DriverTruckStatus;
  is_active: boolean;
  current_route_id: string | null;
  route_status: RouteStatus | null;
  assigned_orders: number;
  current_order_id: string | null;
  last_lat: number | null;
  last_lng: number | null;
  last_ping_at: string | null;
}
