// Stable RPC contract between the dispatcher in apps/backend-go and the
// optimizer in apps/ai-worker. Treat this file as a wire contract: bump
// `OptimizerContractVersion` whenever a field is renamed or removed.

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
  /** Volumetric units consumed at this stop. */
  vu: number;
  /** Optional retailer-declared receiving window. RFC3339. */
  window_open?: string;
  window_close?: string;
  /** Total payable on delivery (cash-on-delivery flows). */
  cod_total?: Money;
}

/** A planned, optimized route — output of one solver run. */
export interface Route {
  route_id: RouteId;
  driver_id: DriverId;
  vehicle_id: VehicleId;
  /** Geographic start point (warehouse / factory). */
  origin_node_id: WarehouseId;
  /** Stops in execution order. */
  stops: Stop[];
  /** Cumulative VU after all stops. */
  total_vu: number;
  /** Estimated drive time in seconds, end-to-end. */
  estimated_duration_seconds: number;
}

export interface OptimizerRequest {
  /** Caller-generated correlation id; echoed back. */
  trace_id: string;
  supplier_id: SupplierId;
  warehouse_id: WarehouseId;
  /** Candidate drivers + their available capacity. */
  fleet: Array<{
    driver_id: DriverId;
    vehicle_id: VehicleId;
    max_vu: number;
    home_h3_cell: H3Cell;
  }>;
  /** Stops awaiting dispatch. */
  stops: Stop[];
  /** Optional knobs the dispatcher may pass through. */
  options?: {
    /** Maximum runtime budget for the solver in milliseconds. */
    max_runtime_ms?: number;
    /** Algorithm hint; defaults to "clarke_wright_h3". */
    strategy?: "clarke_wright_h3" | "greedy_nearest";
  };
}

export interface OptimizerResponse {
  trace_id: string;
  routes: Route[];
  /** Stops the solver could not assign (capacity / window infeasibility). */
  unassigned: Array<{ order_id: OrderId; reason: string }>;
  /** Total elapsed solver wall-clock, milliseconds. */
  runtime_ms: number;
  contract_version: typeof OptimizerContractVersion;
}
