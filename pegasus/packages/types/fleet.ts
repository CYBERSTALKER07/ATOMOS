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
  queued?: boolean;
  job_id?: string;
  status?: string;
  snapshot_timestamp?: string;
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

export interface DispatchJobSummary {
  job_id: string;
  status: string;
  solver_type: string;
  requested_at: string;
  updated_at: string;
  ready: boolean;
}

export interface DispatchJobActiveListResponse {
  jobs: DispatchJobSummary[];
  source: string;
  degraded: boolean;
}

export interface DispatchProjectionDepot {
  node_uuid: string;
  label?: string;
  lat: number;
  lng: number;
}

export interface DispatchProjectionStop {
  sequence?: number;
  node_uuid: string;
  order_id?: string;
  retailer_id?: string;
  retailer_name?: string;
  lat: number;
  lng: number;
  amount?: number;
  demand_vu: number;
  receiving_window_open?: string;
  receiving_window_close?: string;
}

export interface DispatchProjectionRoute {
  route_id?: string;
  manifest_id?: string;
  vehicle_uuid: string;
  driver_uuid: string;
  driver_name?: string;
  vehicle_type?: string;
  vehicle_class?: string;
  capacity_vu: number;
  load_vu: number;
  route_cost_km: number;
  stops: DispatchProjectionStop[];
}

export interface DispatchJobProjection {
  job_id: string;
  status: string;
  solver_type: string;
  ready: boolean;
  requested_at: string;
  updated_at: string;
  completed_at?: string;
  failure_code?: string;
  failure_message?: string;
  timed_out?: boolean;
  matrix_size?: number;
  objective_cost_km?: number;
  warnings: string[];
  depot?: DispatchProjectionDepot;
  routes: DispatchProjectionRoute[];
  unassigned: DispatchProjectionStop[];
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
