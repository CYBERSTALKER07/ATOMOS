// Canonical TypeScript DTO mirrors of the backend contract. Hand-aligned with
// contracts/events.schema.json. Bump `WireVersion` and individual event
// `schema_version` fields in lockstep with the JSON Schema when shapes change.

export const WireVersion = 1 as const;

// ── Role + scalar primitives ────────────────────────────────────────────────
export type Role =
  | "ADMIN"
  | "RETAILER"
  | "DRIVER"
  | "PAYLOAD"
  | "FACTORY_ADMIN"
  | "WAREHOUSE_ADMIN";

export type SupplierId = string;
export type RetailerId = string;
export type OrderId = string;
export type DriverId = string;
export type VehicleId = string;
export type WarehouseId = string;
export type FactoryId = string;
export type RouteId = string;
export type ManifestId = string;
export type SessionId = string;
export type CommandId = string;
export type H3Cell = string; // 15-char hex
export type Iso4217 = string; // 3-char currency code

export interface Money {
  /** int64 minor units (e.g. tiyin/cents). Never float. */
  amount: number;
  currency: Iso4217;
}

export type HomeNodeType = "WAREHOUSE" | "FACTORY";
export interface HomeNode {
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
}

export type PaymentGateway = "GLOBAL_PAY" | "ADYEN" | "AIRWALLEX" | "CASH";

export type OrderStatus =
  | "PENDING"
  | "LOADED"
  | "IN_TRANSIT"
  | "ARRIVED"
  | "COMPLETED"
  | "CANCELLED";

// ── Envelope + event-type union ─────────────────────────────────────────────
export interface WsEventEnvelope<T extends string = string, P = unknown> {
  type: T;
  trace_id: string;
  timestamp: string; // RFC3339Nano
  v: number;
  schema_version?: number;
  data?: P;
}

export type EventType =
  | "SUPPLIER_CREATED"
  | "RETAILER_REGISTERED"
  | "DRIVER_CREATED"
  | "VEHICLE_CREATED"
  | "WAREHOUSE_CREATED"
  | "FACTORY_CREATED"
  | "ORDER_CREATED"
  | "ORDER_VALIDATION_FAILED"
  | "ORDER_ASSIGNED"
  | "ORDER_REASSIGNED"
  | "ORDER_FINALIZED"
  | "ROUTE_CREATED"
  | "MANIFEST_DRAFT_CREATED"
  | "MANIFEST_LOADING_STARTED"
  | "MANIFEST_ORDER_INJECTED"
  | "MANIFEST_ORDER_EXCEPTION"
  | "MANIFEST_DLQ_ESCALATION"
  | "MANIFEST_REBALANCED"
  | "MANIFEST_CANCELLED"
  | "MANIFEST_SEALED"
  | "MANIFEST_DISPATCHED"
  | "MANIFEST_COMPLETED"
  | "PAYMENT_CLEARED"
  | "PAYMENT_REQUIRED"
  | "SETTLEMENT_REQUIRED"
  | "DELIVERY_SESSION_UPDATED"
  | "DELIVERY_DISPUTED"
  | "DRIVER_AVAILABILITY_CHANGED"
  | "SHOP_CLOSED"
  | "SHOP_CLOSED_RESPONSE"
  | "CART_SYNC_UPDATED"
  | "INVENTORY_SYNC_COMPLETE"
  | "COMMAND_DISPATCHED"
  | "COMMAND_RECEIVED"
  | "COMMAND_SETTLED"
  | "SYSTEM_APP_OUTDATED";

// ── Event payloads ──────────────────────────────────────────────────────────
export interface SupplierCreated {
  supplier_id: SupplierId;
  name: string;
  country: string;
  currency: Iso4217;
}

export interface RetailerRegistered {
  retailer_id: RetailerId;
  phone: string;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
}

export interface DriverCreated extends HomeNode {
  driver_id: DriverId;
  supplier_id: SupplierId;
  phone: string;
  name?: string;
}

export interface VehicleCreated extends HomeNode {
  vehicle_id: VehicleId;
  supplier_id: SupplierId;
  plate: string;
  max_vu?: number;
}

export interface WarehouseCreated {
  warehouse_id: WarehouseId;
  supplier_id: SupplierId;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
  max_vu?: number;
}

export interface FactoryCreated {
  factory_id: FactoryId;
  supplier_id: SupplierId;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
}

export interface OrderCreated {
  order_id: OrderId;
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  total: Money;
}

export interface OrderValidationFailed {
  order_id: OrderId;
  reason: string;
}

export interface OrderAssigned {
  order_id: OrderId;
  driver_id: DriverId;
  route_id: RouteId;
}

export interface OrderReassigned {
  order_id: OrderId;
  from_driver_id: DriverId;
  to_driver_id: DriverId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
}

export interface OrderFinalized {
  order_id: OrderId;
  total: Money;
  fee_amount?: number;
  net_payout_amount?: number;
}

export interface PaymentCleared {
  order_id: OrderId;
  amount: Money;
  gateway?: PaymentGateway;
  provider_reference?: string;
}

export interface SettlementRequired {
  order_id: OrderId;
  session_id: SessionId;
  amount: Money;
}

export interface DeliverySessionUpdated {
  session_id: SessionId;
  order_id: OrderId;
  amount?: number;
  original_amount?: number;
  adjusted_amount?: number;
  currency?: Iso4217;
}

export interface DriverAvailabilityChanged {
  driver_id: DriverId;
  available: boolean;
  reason?: string;
}

export interface ManifestSealed {
  manifest_id: ManifestId;
  route_id: RouteId;
  driver_id: DriverId;
  vehicle_id?: VehicleId;
  order_count?: number;
}

export interface ManifestOrderInjected {
  manifest_id: ManifestId;
  order_id: OrderId;
  total_volume_vu?: number;
  stop_count?: number;
}

export interface ManifestOrderException {
  manifest_id: ManifestId;
  order_id?: OrderId;
  transfer_id?: string;
  reason: string;
  attempt_count?: number;
  escalated?: boolean;
  exception_id?: string;
  metadata?: string;
}

export interface ManifestRebalanced {
  manifest_id: ManifestId;
  transfer_id?: string;
  order_id?: OrderId;
  from_driver_id?: DriverId;
  to_driver_id?: DriverId;
  from_vehicle_id?: VehicleId;
  to_vehicle_id?: VehicleId;
  from_route_id?: RouteId | string;
  to_route_id?: RouteId | string;
  from_manifest_id?: ManifestId;
  to_manifest_id?: ManifestId;
  depth?: number;
  reason?: string;
}

export interface ManifestCancelled {
  manifest_id: ManifestId;
  reason?: string;
}

export type CommandState = "INITIATED" | "DISPATCHED" | "RECEIVED" | "SETTLED";

export interface CommandLifecycle {
  command_id: CommandId;
  command_state: CommandState;
  target_role?: Role;
  target_id?: string;
}

// ── Discriminated WS event union ────────────────────────────────────────────
export type WsEvent =
  | WsEventEnvelope<"SUPPLIER_CREATED", SupplierCreated>
  | WsEventEnvelope<"RETAILER_REGISTERED", RetailerRegistered>
  | WsEventEnvelope<"DRIVER_CREATED", DriverCreated>
  | WsEventEnvelope<"VEHICLE_CREATED", VehicleCreated>
  | WsEventEnvelope<"WAREHOUSE_CREATED", WarehouseCreated>
  | WsEventEnvelope<"FACTORY_CREATED", FactoryCreated>
  | WsEventEnvelope<"ORDER_CREATED", OrderCreated>
  | WsEventEnvelope<"ORDER_VALIDATION_FAILED", OrderValidationFailed>
  | WsEventEnvelope<"ORDER_ASSIGNED", OrderAssigned>
  | WsEventEnvelope<"ORDER_REASSIGNED", OrderReassigned>
  | WsEventEnvelope<"ORDER_FINALIZED", OrderFinalized>
  | WsEventEnvelope<"PAYMENT_CLEARED", PaymentCleared>
  | WsEventEnvelope<"SETTLEMENT_REQUIRED", SettlementRequired>
  | WsEventEnvelope<"DELIVERY_SESSION_UPDATED", DeliverySessionUpdated>
  | WsEventEnvelope<"DRIVER_AVAILABILITY_CHANGED", DriverAvailabilityChanged>
  | WsEventEnvelope<"MANIFEST_ORDER_INJECTED", ManifestOrderInjected>
  | WsEventEnvelope<"MANIFEST_ORDER_EXCEPTION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_DLQ_ESCALATION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_REBALANCED", ManifestRebalanced>
  | WsEventEnvelope<"MANIFEST_CANCELLED", ManifestCancelled>
  | WsEventEnvelope<"MANIFEST_SEALED", ManifestSealed>
  | WsEventEnvelope<"COMMAND_DISPATCHED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_RECEIVED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_SETTLED", CommandLifecycle>
  | WsEventEnvelope<"SYSTEM_APP_OUTDATED", { minimum_version: string }>;
