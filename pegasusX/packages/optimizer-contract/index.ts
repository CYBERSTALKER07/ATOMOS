// Stable RPC contract between the dispatcher in apps/backend-go and the
// optimizer (optimizer-core / ai-worker). Treat this file as a wire contract:
// bump `OptimizerContractVersion` whenever a field is renamed or removed.
// Additive fields keep version 1 (matches Go packages/optimizer-contract V="v1").

import type {
  DriverId,
  H3Cell,
  Money,
  OrderId,
  RouteId,
  SupplierId,
  VehicleId,
  WarehouseId,
} from "@pegasusx/types";

export const OptimizerContractVersion = 1 as const;

/** A single delivery target the optimizer must visit. */
export interface Stop {
  order_id: OrderId;
  retailer_id: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
  /** Volumetric units consumed at this stop (wire alias: volume_vu in Go). */
  vu: number;
  volume_vu?: number;
  /** Optional retailer-declared receiving window. HH:MM or RFC3339. */
  window_open?: string;
  window_close?: string;
  service_minutes?: number;
  priority?: number;
  handling_class?: string;
  requires_cold_chain?: boolean;
  is_hazardous?: boolean;
  access_restriction?: string;
  /** Total payable on delivery (cash-on-delivery flows). */
  cod_total?: Money;
}

/** One available truck / driver pair. */
export interface Vehicle {
  vehicle_id: VehicleId;
  driver_id: DriverId;
  max_volume_vu: number;
  start_lat: number;
  start_lng: number;
  avg_speed_kmph?: number;
  end_lat?: number;
  end_lng?: number;
  has_refrigeration?: boolean;
  hazmat_certified?: boolean;
  shift_start?: string;
  shift_end?: string;
  max_route_minutes?: number;
}

/** A planned, optimized route — output of one solver run. */
export interface Route {
  route_id?: RouteId;
  driver_id: DriverId;
  vehicle_id: VehicleId;
  /** Geographic start point (warehouse / factory). */
  origin_node_id?: WarehouseId;
  /** Stops in execution order. */
  stops: Stop[];
  /** Cumulative VU after all stops. */
  total_vu: number;
  util_pct?: number;
  distance_km?: number;
  duration_min?: number;
  /** Estimated drive time in seconds, end-to-end. */
  estimated_duration_seconds?: number;
}

export interface OptimizerTunables {
  tetris_buffer?: number;
  two_opt_iterations?: number;
  max_stops_per_route?: number;
  /** OR-Tools search budget; Go HTTP timeout must exceed this. */
  time_limit_ms?: number;
  max_runtime_ms?: number;
  strategy?: "clarke_wright_h3" | "greedy_nearest" | "or_tools_vrp";
}

export interface OptimizerRequest {
  /** Wire contract version string ("v1"). */
  v?: string;
  /** Caller-generated correlation id; echoed back. */
  trace_id: string;
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  home_node_id?: WarehouseId;
  departure_time?: string;
  /** Candidate drivers + their available capacity (legacy shape). */
  fleet?: Array<{
    driver_id: DriverId;
    vehicle_id: VehicleId;
    max_vu: number;
    home_h3_cell: H3Cell;
  }>;
  /** Canonical fleet shape (matches Go SolveRequest.vehicles). */
  vehicles?: Vehicle[];
  /** Stops awaiting dispatch. */
  stops: Stop[];
  /**
   * Optional NxN meter matrix. Layout: vehicle starts then customer stops
   * (see Go SolveRequest.DistanceMatrixM).
   */
  distance_matrix_m?: number[][];
  tunables?: OptimizerTunables;
  /** Optional knobs the dispatcher may pass through (legacy). */
  options?: OptimizerTunables;
}

export interface OptimizerResponse {
  v?: string;
  trace_id: string;
  source?: string;
  routes: Route[];
  orphans?: Array<{ order_id: OrderId; reason: string }>;
  /** Stops the solver could not assign (capacity / window infeasibility). */
  unassigned?: Array<{ order_id: OrderId; reason: string }>;
  stats?: {
    elapsed_ms: number;
    stops_considered: number;
    stops_placed: number;
    stops_orphaned: number;
    vehicles_used: number;
    avg_utilisation_pct: number;
    two_opt_improvement_pct: number;
  };
  /** Total elapsed solver wall-clock, milliseconds. */
  runtime_ms?: number;
  contract_version?: typeof OptimizerContractVersion;
}
