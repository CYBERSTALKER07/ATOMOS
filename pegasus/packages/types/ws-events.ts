/**
 * GENERATED FILE - DO NOT EDIT MANUALLY.
 * Source: kafka/events.go
 * Generated at: 2026-05-14T15:49:35Z
 */

export enum WSEventType {
  AI_ORDER_CONFIRMED = 'AI_ORDER_CONFIRMED',
  AI_ORDER_REJECTED = 'AI_ORDER_REJECTED',
  AI_PLAN_DATE_SHIFT = 'AI_PLAN_DATE_SHIFT',
  AI_PLAN_SKU_MODIFIED = 'AI_PLAN_SKU_MODIFIED',
  AI_PREDICTION = 'AI_PREDICTION',
  AI_PREDICTION_CORRECTED = 'AI_PREDICTION_CORRECTED',
  BYPASS_TOKEN_ISSUED = 'BYPASS_TOKEN_ISSUED',
  CANCEL_APPROVED = 'CANCEL_APPROVED',
  CANCEL_REQUESTED = 'CANCEL_REQUESTED',
  CART_SYNC_UPDATED = 'CART_SYNC_UPDATED',
  CASH_COLLECTION_REQUIRED = 'CASH_COLLECTION_REQUIRED',
  COMMAND_DISPATCHED = 'COMMAND_DISPATCHED',
  COMMAND_RECEIVED = 'COMMAND_RECEIVED',
  COMMAND_SETTLED = 'COMMAND_SETTLED',
  CREDIT_DELIVERY_MARKED = 'CREDIT_DELIVERY_MARKED',
  CREDIT_DELIVERY_RESOLVED = 'CREDIT_DELIVERY_RESOLVED',
  DELIVERY_DISPUTED = 'DELIVERY_DISPUTED',
  DELIVERY_SESSION_UPDATED = 'DELIVERY_SESSION_UPDATED',
  DEMAND_FORECAST_READY = 'DEMAND_FORECAST_READY',
  DISPATCH_LOCK_ACQUIRED = 'DISPATCH_LOCK_ACQUIRED',
  DISPATCH_LOCK_CHANGE = 'DISPATCH_LOCK_CHANGE',
  DISPATCH_LOCK_RELEASED = 'DISPATCH_LOCK_RELEASED',
  DRIVER_APPROACHING = 'DRIVER_APPROACHING',
  DRIVER_ARRIVED = 'DRIVER_ARRIVED',
  DRIVER_AVAILABILITY_CHANGED = 'DRIVER_AVAILABILITY_CHANGED',
  DRIVER_CREATED = 'DRIVER_CREATED',
  EARLY_COMPLETE_APPROVED = 'EARLY_COMPLETE_APPROVED',
  EARLY_COMPLETE_REQUESTED = 'EARLY_COMPLETE_REQUESTED',
  ETA_UPDATED = 'ETA_UPDATED',
  FACTORY_CREATED = 'FACTORY_CREATED',
  FACTORY_MANIFEST_CREATED = 'FACTORY_MANIFEST_CREATED',
  FACTORY_MANIFEST_UPDATE = 'FACTORY_MANIFEST_UPDATE',
  FACTORY_OUTBOX_FAILED = 'FACTORY_OUTBOX_FAILED',
  FACTORY_SLA_BREACH = 'FACTORY_SLA_BREACH',
  FACTORY_SUPPLY_REQUEST_UPDATE = 'FACTORY_SUPPLY_REQUEST_UPDATE',
  FACTORY_TRANSFER_UPDATE = 'FACTORY_TRANSFER_UPDATE',
  FEE_RATE_ADJUSTED = 'FEE_RATE_ADJUSTED',
  FLEET_DISPATCHED = 'FLEET_DISPATCHED',
  FORCE_SEAL_ALERT = 'FORCE_SEAL_ALERT',
  FREEZE_LOCK_ACQUIRED = 'FREEZE_LOCK_ACQUIRED',
  FREEZE_LOCK_RELEASED = 'FREEZE_LOCK_RELEASED',
  FULFILLMENT_PAID = 'FULFILLMENT_PAID',
  FULFILLMENT_PAYMENT_COMPLETED = 'FULFILLMENT_PAYMENT_COMPLETED',
  INBOUND_FREIGHT_UNANNOUNCED = 'INBOUND_FREIGHT_UNANNOUNCED',
  INSIGHT_APPROVED_TRANSFER_CREATED = 'INSIGHT_APPROVED_TRANSFER_CREATED',
  INTERNAL_LOAD_CONFIRMED = 'INTERNAL_LOAD_CONFIRMED',
  INVENTORY_IMPORT_STATUS_UPDATE = 'INVENTORY_IMPORT_STATUS_UPDATE',
  INVENTORY_IMPORT_UPLOADED = 'INVENTORY_IMPORT_UPLOADED',
  INVENTORY_SYNC_COMPLETE = 'INVENTORY_SYNC_COMPLETE',
  LOOK_AHEAD_COMPLETED = 'LOOK_AHEAD_COMPLETED',
  MANIFEST_CANCELLED = 'MANIFEST_CANCELLED',
  MANIFEST_COMPLETED = 'MANIFEST_COMPLETED',
  MANIFEST_DISPATCHED = 'MANIFEST_DISPATCHED',
  MANIFEST_DLQ_ESCALATION = 'MANIFEST_DLQ_ESCALATION',
  MANIFEST_DRAFT_CREATED = 'MANIFEST_DRAFT_CREATED',
  MANIFEST_FORCE_SEALED = 'MANIFEST_FORCE_SEALED',
  MANIFEST_LOADING_STARTED = 'MANIFEST_LOADING_STARTED',
  MANIFEST_ORDER_EXCEPTION = 'MANIFEST_ORDER_EXCEPTION',
  MANIFEST_ORDER_INJECTED = 'MANIFEST_ORDER_INJECTED',
  MANIFEST_ORDER_REASSIGNED = 'MANIFEST_ORDER_REASSIGNED',
  MANIFEST_REBALANCED = 'MANIFEST_REBALANCED',
  MANIFEST_SEALED = 'MANIFEST_SEALED',
  MANIFEST_SETTLED = 'MANIFEST_SETTLED',
  MISSING_ITEMS_REPORTED = 'MISSING_ITEMS_REPORTED',
  NEGOTIATION_PROPOSED = 'NEGOTIATION_PROPOSED',
  NEGOTIATION_RESOLVED = 'NEGOTIATION_RESOLVED',
  NETWORK_MODE_CHANGED = 'NETWORK_MODE_CHANGED',
  OFFLOAD_CONFIRMED = 'OFFLOAD_CONFIRMED',
  ORDER_AMENDED = 'ORDER_AMENDED',
  ORDER_ASSIGNED = 'ORDER_ASSIGNED',
  ORDER_CANCELLED = 'ORDER_CANCELLED',
  ORDER_CANCELLED_BY_ORIGIN = 'ORDER_CANCELLED_BY_ORIGIN',
  ORDER_CANCEL_LOCKED = 'ORDER_CANCEL_LOCKED',
  ORDER_COMPLETED = 'ORDER_COMPLETED',
  ORDER_CREATED = 'ORDER_CREATED',
  ORDER_DELAYED = 'ORDER_DELAYED',
  ORDER_DISPATCHED = 'ORDER_DISPATCHED',
  ORDER_FINALIZED = 'ORDER_FINALIZED',
  ORDER_MODIFIED = 'ORDER_MODIFIED',
  ORDER_REASSIGNED = 'ORDER_REASSIGNED',
  ORDER_REJECTED_BY_SUPPLIER = 'ORDER_REJECTED_BY_SUPPLIER',
  ORDER_REROUTED = 'ORDER_REROUTED',
  ORDER_STATE_CHANGED = 'ORDER_STATE_CHANGED',
  ORDER_STATUS_CHANGED = 'ORDER_STATUS_CHANGED',
  ORDER_SYNC = 'ORDER_SYNC',
  ORDER_VALIDATION_FAILED = 'ORDER_VALIDATION_FAILED',
  OUTBOX_FAILED = 'OUTBOX_FAILED',
  OUT_OF_STOCK = 'OUT_OF_STOCK',
  PAYLOAD_OVERFLOW = 'PAYLOAD_OVERFLOW',
  PAYLOAD_READY_TO_SEAL = 'PAYLOAD_READY_TO_SEAL',
  PAYLOAD_SEALED = 'PAYLOAD_SEALED',
  PAYLOAD_SYNC = 'PAYLOAD_SYNC',
  PAYMENT_BYPASS_COMPLETED = 'PAYMENT_BYPASS_COMPLETED',
  PAYMENT_BYPASS_ISSUED = 'PAYMENT_BYPASS_ISSUED',
  PAYMENT_CLEARED = 'PAYMENT_CLEARED',
  PAYMENT_EXPIRED = 'PAYMENT_EXPIRED',
  PAYMENT_FAILED = 'PAYMENT_FAILED',
  PAYMENT_INTENT_CREATED = 'PAYMENT_INTENT_CREATED',
  PAYMENT_REFUNDED = 'PAYMENT_REFUNDED',
  PAYMENT_REQUIRED = 'PAYMENT_REQUIRED',
  PAYMENT_SETTLED = 'PAYMENT_SETTLED',
  POWER_OUTAGE_REPORTED = 'POWER_OUTAGE_REPORTED',
  PRE_ORDER_AUTO_ACCEPTED = 'PRE_ORDER_AUTO_ACCEPTED',
  PRE_ORDER_CANCELLED = 'PRE_ORDER_CANCELLED',
  PRE_ORDER_CONFIRMATION = 'PRE_ORDER_CONFIRMATION',
  PRE_ORDER_CONFIRMED = 'PRE_ORDER_CONFIRMED',
  PRE_ORDER_EDITED = 'PRE_ORDER_EDITED',
  PRE_ORDER_NOTIFIED = 'PRE_ORDER_NOTIFIED',
  PRE_ORDER_NUDGE = 'PRE_ORDER_NUDGE',
  PULL_MATRIX_COMPLETED = 'PULL_MATRIX_COMPLETED',
  REPLENISHMENT_LOCK_ACQUIRED = 'REPLENISHMENT_LOCK_ACQUIRED',
  REPLENISHMENT_LOCK_RELEASED = 'REPLENISHMENT_LOCK_RELEASED',
  REPLENISHMENT_TRANSFER_CREATED = 'REPLENISHMENT_TRANSFER_CREATED',
  RETAILER_PRICE_OVERRIDE = 'RETAILER_PRICE_OVERRIDE',
  RETAILER_REGISTERED = 'RETAILER_REGISTERED',
  RETURN_RESOLVED = 'RETURN_RESOLVED',
  ROUTE_CREATED = 'ROUTE_CREATED',
  ROUTE_FINALIZED = 'ROUTE_FINALIZED',
  SETTLEMENT_REQUIRED = 'SETTLEMENT_REQUIRED',
  SHOP_CLOSED = 'SHOP_CLOSED',
  SHOP_CLOSED_ALERT = 'SHOP_CLOSED_ALERT',
  SHOP_CLOSED_ESCALATED = 'SHOP_CLOSED_ESCALATED',
  SHOP_CLOSED_RESOLVED = 'SHOP_CLOSED_RESOLVED',
  SHOP_CLOSED_RESPONSE = 'SHOP_CLOSED_RESPONSE',
  SMS_QUICK_COMPLETE = 'SMS_QUICK_COMPLETE',
  SPLIT_PAYMENT_CREATED = 'SPLIT_PAYMENT_CREATED',
  STOCK_BACKORDERED = 'STOCK_BACKORDERED',
  STOCK_THRESHOLD_BREACH = 'STOCK_THRESHOLD_BREACH',
  SUPPLY_LANE_TRANSIT_UPDATED = 'SUPPLY_LANE_TRANSIT_UPDATED',
  SUPPLY_REQUEST_ACKNOWLEDGED = 'SUPPLY_REQUEST_ACKNOWLEDGED',
  SUPPLY_REQUEST_CANCELLED = 'SUPPLY_REQUEST_CANCELLED',
  SUPPLY_REQUEST_FULFILLED = 'SUPPLY_REQUEST_FULFILLED',
  SUPPLY_REQUEST_READY = 'SUPPLY_REQUEST_READY',
  SUPPLY_REQUEST_SUBMITTED = 'SUPPLY_REQUEST_SUBMITTED',
  SUPPLY_REQUEST_UPDATE = 'SUPPLY_REQUEST_UPDATE',
  SYSTEM_BROADCAST = 'SYSTEM_BROADCAST',
  TOKEN_REFRESH_NEEDED = 'TOKEN_REFRESH_NEEDED',
  TRANSFER_APPROVED = 'TRANSFER_APPROVED',
  TRANSFER_RECEIVED = 'TRANSFER_RECEIVED',
  TRANSFER_STATE_CHANGED = 'TRANSFER_STATE_CHANGED',
  TRANSFER_UNASSIGNED = 'TRANSFER_UNASSIGNED',
  UNIFIED_CHECKOUT_COMPLETED = 'UNIFIED_CHECKOUT_COMPLETED',
  VEHICLE_CREATED = 'VEHICLE_CREATED',
  WAREHOUSE_CREATED = 'WAREHOUSE_CREATED',
  WAREHOUSE_SPATIAL_UPDATED = 'WAREHOUSE_SPATIAL_UPDATED',
  WAREHOUSE_STATUS_CHANGED = 'WAREHOUSE_STATUS_CHANGED',
}

export interface UnknownEventPayload {
  [key: string]: any;
}

export interface CommandDispatchedPayload {
  command_id: string;
  command_state: string;
  event_type: string;
  target_role: string;
  target_id: string;
  trace_id: string;
  timestamp: string;
}

export interface CommandReceivedPayload {
  command_id: string;
  command_state: string;
  event_type: string;
  target_role: string;
  target_id: string;
  ack_by_user_id?: string;
  ack_by_role?: string;
  trace_id: string;
  timestamp: string;
}

export interface CommandSettledPayload {
  command_id: string;
  command_state: string;
  event_type: string;
  target_role: string;
  target_id: string;
  ack_by_user_id?: string;
  ack_by_role?: string;
  trace_id: string;
  timestamp: string;
}

export interface DeliveryDisputedPayload {
  session_id: string;
  order_id: string;
  retailer_id: string;
  driver_id?: string;
  supplier_id?: string;
  reason?: string;
  disputed_by?: string;
  timestamp: string;
}

export interface DeliverySessionUpdatedPayload {
  session_id: string;
  order_id: string;
  retailer_id: string;
  driver_id?: string;
  state: string;
  original_amount: number;
  adjusted_amount: number;
  fee_basis_points: number;
  fee_amount: number;
  currency: string;
  timestamp: string;
}

export interface DemandForecastReadyPayload {
  retailer_id: string;
  warehouse_id: string;
  supplier_id: string;
  sku_count: number;
  timestamp: string;
}

export interface DriverArrivedPayload {
  order_id: string;
  retailer_id: string;
  driver_id: string;
  supplier_id: string;
  warehouse_id?: string;
  timestamp: string;
}

export interface DriverAvailabilityChangedPayload {
  driver_id: string;
  supplier_id: string;
  warehouse_id?: string;
  available: boolean;
  reason?: string;
  note?: string;
  truck_id?: string;
  timestamp: string;
}

export interface DriverCreatedPayload {
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

export interface EarlyCompleteRequestedPayload {
  driver_id: string;
  supplier_id: string;
  route_id: string;
  order_ids?: string[];
  reason: string;
  note?: string;
  timestamp: string;
}

export interface FactoryCreatedPayload {
  factory_id: string;
  supplier_id: string;
  name: string;
  lat: number;
  lng: number;
  h3_index?: string;
  region_code: string;
  lead_time_days: number;
  production_capacity_vu: number;
  product_types?: string[];
  warehouses_linked: number;
  timestamp: string;
}

export interface FactorySLABreachPayload {
  transfer_id: string;
  factory_id: string;
  warehouse_id: string;
  supplier_id: string;
  escalation_level: string;
  sla_breach_minutes: number;
  replacement_transfer_id?: string;
  timestamp: string;
}

export interface FeeRateAdjustedPayload {
  previous_fee_basis_points: number;
  new_fee_basis_points: number;
  milestone_order_count: number;
  global_order_count: number;
  milestone_index: number;
  trigger_order_id: string;
  timestamp: string;
}

export interface FleetDispatchedPayload {
  route_id: string;
  manifest_id?: string;
  order_ids?: string[];
  driver_id?: string;
  supplier_id?: string;
  warehouse_id?: string;
  geo_zone?: string;
  timestamp: string;
}

export interface ForceSealAlertPayload {
  supplier_id: string;
  warehouse_id?: string;
  manifest_id: string;
  count_24h: number;
  quota: number;
  sealed_by: string;
  timestamp: string;
}

export interface InboundFreightUnannouncedPayload {
  transfer_id: string;
  warehouse_id: string;
  supplier_id: string;
  items_count: number;
  received_by: string;
  timestamp: string;
}

export interface InternalLoadConfirmedPayload {
  order_id: string;
  manifest_id: string;
  supplier_id: string;
  warehouse_id?: string;
  driver_id?: string;
  truck_id?: string;
  confirmed_by: string;
  volume_vu: number;
  timestamp: string;
}

export interface InventoryImportStatusUpdatePayload {
  type: string;
  session_id: string;
  supplier_id: string;
  status: string;
  suggested_mappings: number;
  timestamp: string;
}

export interface InventoryImportUploadedPayload {
  session_id: string;
  supplier_id: string;
  gcs_path: string;
}

export interface LookAheadCompletedPayload {
  run_id: string;
  supplier_id: string;
  source: string;
  duration_ms: number;
  horizon_days?: number;
  timestamp: string;
}

export interface ManifestCancelledPayload {
  manifest_id: string;
  supplier_id: string;
  factory_id?: string;
  warehouse_id?: string;
  released_ids?: string[];
  released_kind: string;
  reason: string;
  cancelled_by: string;
  timestamp: string;
}

export interface ManifestOrderExceptionPayload {
  exception_id: string;
  manifest_id: string;
  order_id: string;
  supplier_id: string;
  reason: string;
  attempt_count: number;
  escalated: boolean;
  metadata?: string;
  timestamp: string;
}

export interface ManifestOrderInjectedPayload {
  manifest_id: string;
  order_id: string;
  supplier_id: string;
  new_total_volume_vu: number;
  injected_by: string;
  timestamp: string;
}

export interface ManifestOrderReassignedPayload {
  order_id: string;
  source_manifest_id: string;
  target_manifest_id: string;
  old_driver_id?: string;
  new_driver_id?: string;
  supplier_id: string;
  warehouse_id?: string;
  reason?: string;
  reassigned_by: string;
  timestamp: string;
}

export interface ManifestRebalancedPayload {
  factory_id: string;
  supplier_id: string;
  source_manifest_id: string;
  target_manifest_id: string;
  transfer_ids?: string[];
  reason: string;
  rebalanced_by: string;
  timestamp: string;
}

export interface NegotiationProposedPayload {
  proposal_id: string;
  order_id: string;
  driver_id: string;
  supplier_id: string;
  retailer_id: string;
  timestamp: string;
}

export interface NegotiationResolvedPayload {
  proposal_id: string;
  order_id: string;
  supplier_id: string;
  action: string;
  timestamp: string;
}

export interface NetworkModeChangedPayload {
  supplier_id: string;
  old_mode: string;
  new_mode: string;
  changed_by: string;
  reason?: string;
  timestamp: string;
}

export interface OffloadConfirmedPayload {
  order_id: string;
  retailer_id: string;
  amount: number;
  original_amount: number;
  payment_method: string;
  timestamp: string;
}

export interface OrderAssignedPayload {
  order_id: string;
  route_id: string;
  driver_id: string;
  supplier_id: string;
  warehouse_id?: string;
  timestamp: string;
}

export interface OrderCancelLockedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  reason: string;
  timestamp: string;
}

export interface OrderCancelledByOriginPayload {
  order_id: string;
  supplier_id: string;
  warehouse_id: string;
  retailer_id: string;
  manifest_id?: string;
  reason: string;
  cancelled_by: string;
  amount: number;
  timestamp: string;
}

export interface OrderCompletedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id?: string;
  warehouse_id?: string;
  amount?: number;
  currency?: string;
  timestamp: string;
}

export interface OrderCreatedPayload {
  invoice_id?: string;
  order_id: string;
  supplier_id?: string;
  retailer_id?: string;
  warehouse_id?: string;
  warehouse_name?: string;
  tin?: string;
  terminal_id?: string;
  receipt_type?: string;
  fiscal_sign?: string;
  total: number;
  currency: string;
  items?: any[];
  timestamp: string;
}

export interface OrderDelayedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  warehouse_id?: string;
  manifest_id?: string;
  reason: string;
  timestamp: string;
}

export interface OrderDispatchedPayload {
  route_id: string;
  order_ids?: string[];
  driver_id: string;
  supplier_id: string;
  warehouse_id?: string;
  timestamp: string;
}

export interface OrderFinalizedPayload {
  order_id: string;
  invoice_id?: string;
  supplier_id?: string;
  retailer_id?: string;
  fiscal_sign: string;
  timestamp: string;
}

export interface OrderModifiedPayload {
  order_id: string;
  amendment_id: string;
  driver_id: string;
  supplier_id: string;
  warehouse_id?: string;
  retailer_id: string;
  new_amount: number;
  refunded: number;
  currency: string;
  timestamp: string;
}

export interface OrderReassignedPayload {
  order_ids?: string[];
  old_route_id: string;
  new_route_id: string;
  old_driver_id?: string;
  new_driver_id?: string;
  supplier_id?: string;
  timestamp: string;
}

export interface OrderReroutedPayload {
  order_id?: string;
  supplier_id: string;
  original_warehouse_id: string;
  new_warehouse_id: string;
  original_load_percent: number;
  new_load_percent: number;
  retailer_lat: number;
  retailer_lng: number;
  distance_km: number;
  timestamp: string;
}

export interface OrderStatusChangedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  warehouse_id?: string;
  old_state: string;
  new_state: string;
  timestamp: string;
}

export interface OrderValidationFailedPayload {
  order_id: string;
  invoice_id?: string;
  retailer_id?: string;
  reason: string;
  timestamp: string;
}

export interface OutOfStockPayload {
  order_id?: string;
  retailer_id: string;
  supplier_id: string;
  warehouse_id: string;
  shortfall_map?: Record<string, number>;
  timestamp: string;
}

export interface PayloadOverflowPayload {
  order_id: string;
  supplier_id: string;
  warehouse_id: string;
  manifest_id: string;
  reason: string;
  attempt_count: number;
  timestamp: string;
}

export interface PayloadReadyToSealPayload {
  route_id: string;
  order_ids?: string[];
  supplier_id: string;
  warehouse_id?: string;
  timestamp: string;
}

export interface PayloadSealedPayload {
  order_id: string;
  terminal_id: string;
  delivery_token: string;
  timestamp: string;
}

export interface PayloadSyncPayload {
  supplier_id: string;
  warehouse_id?: string;
  manifest_id: string;
  reason: string;
  timestamp: string;
}

export interface PaymentClearedPayload {
  order_id: string;
  invoice_id: string;
  supplier_id?: string;
  retailer_id?: string;
  status: string;
  timestamp: string;
}

export interface PaymentFailedPayload {
  order_id: string;
  invoice_id: string;
  retailer_id: string;
  warehouse_id?: string;
  gateway: string;
  reason: string;
  timestamp: string;
}

export interface PaymentSettledPayload {
  order_id: string;
  invoice_id: string;
  retailer_id: string;
  driver_id: string;
  warehouse_id?: string;
  gateway: string;
  amount: number;
  currency: string;
  timestamp: string;
}

export interface PreOrderAutoAcceptedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  delivery_date: string;
  timestamp: string;
}

export interface PreOrderCancelledPayload {
  order_id: string;
  cancelled_by: string;
  reason: string;
  timestamp: string;
}

export interface PreOrderConfirmedPayload {
  order_id: string;
  confirmed_by: string;
  timestamp: string;
}

export interface PreOrderEditedPayload {
  order_id: string;
  edited_by: string;
  new_date?: string;
  timestamp: string;
}

export interface PreOrderNotifiedPayload {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  delivery_date: string;
  timestamp: string;
}

export interface PullMatrixCompletedPayload {
  run_id: string;
  supplier_id: string;
  transfers_generated: number;
  skus_processed: number;
  duration_ms: number;
  source: string;
  timestamp: string;
}

export interface RetailerPriceOverridePayload {
  override_id: string;
  supplier_id: string;
  retailer_id: string;
  sku_id: string;
  price: number;
  action: string;
  set_by: string;
  set_by_role: string;
  timestamp: string;
}

export interface RetailerRegisteredPayload {
  retailer_id: string;
  owner_name: string;
  shop_name: string;
  phone_number: string;
  lat: number;
  lng: number;
  h3_cell: string;
  region_code: string;
  timestamp: string;
}

export interface RouteCreatedPayload {
  route_id: string;
  manifest_id?: string;
  driver_id: string;
  truck_id: string;
  supplier_id: string;
  warehouse_id?: string;
  factory_id?: string;
  stop_count: number;
  volume_vu: number;
  timestamp: string;
}

export interface RouteFinalizedPayload {
  manifest_id: string;
  driver_id: string;
  stop_count: number;
  route_json?: string;
  timestamp: string;
}

export interface SettlementRequiredPayload {
  session_id: string;
  order_id: string;
  retailer_id: string;
  driver_id?: string;
  supplier_id?: string;
  payment_session_id?: string;
  invoice_id?: string;
  state: string;
  amount: number;
  original_amount: number;
  currency: string;
  timestamp: string;
}

export interface ShopClosedEscalatedPayload {
  order_id: string;
  attempt_id: string;
  supplier_id: string;
  escalated_to: string;
  timestamp: string;
}

export interface ShopClosedPayload {
  order_id: string;
  driver_id: string;
  retailer_id: string;
  supplier_id: string;
  attempt_id: string;
  gps_lat: number;
  gps_lng: number;
  timestamp: string;
}

export interface ShopClosedResolvedPayload {
  order_id: string;
  attempt_id: string;
  resolution: string;
  resolved_by: string;
  timestamp: string;
}

export interface ShopClosedResponsePayload {
  order_id: string;
  retailer_id: string;
  attempt_id: string;
  response: string;
  timestamp: string;
}

export interface StockBackorderedPayload {
  invoice_id: string;
  backorder_order_id: string;
  backorder_id?: string;
  supplier_id: string;
  retailer_id: string;
  warehouse_id?: string;
  warehouse_name?: string;
  items?: any[];
  total: number;
  currency: string;
  timestamp: string;
}

export interface StockThresholdBreachPayload {
  supplier_id: string;
  warehouse_id: string;
  product_id: string;
  current_stock: number;
  safety_level: number;
  timestamp: string;
}

export interface SupplyLaneTransitUpdatedPayload {
  lane_id: string;
  supplier_id: string;
  factory_id: string;
  warehouse_id: string;
  old_dampened_hours: number;
  new_dampened_hours: number;
  raw_transit_hours: number;
  timestamp: string;
}

export interface TransferUnassignedPayload {
  manifest_id: string;
  transfer_id: string;
  factory_id: string;
  supplier_id: string;
  reason?: string;
  unassigned_by: string;
  timestamp: string;
}

export interface UnifiedCheckoutCompletedPayload {
  invoice_id: string;
  retailer_id: string;
  total: number;
  currency: string;
  order_count: number;
  timestamp: string;
}

export interface VehicleCreatedPayload {
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

export interface WarehouseCreatedPayload {
  warehouse_id: string;
  supplier_id: string;
  name: string;
  lat: number;
  lng: number;
  h3_count: number;
  coverage_radius_km: number;
  timestamp: string;
}

export interface WarehouseSpatialUpdatedPayload {
  warehouse_id: string;
  supplier_id: string;
  old_h3_count: number;
  new_h3_count: number;
  coverage_radius_km: number;
  timestamp: string;
}

export interface WarehouseStatusChangedPayload {
  warehouse_id: string;
  supplier_id: string;
  field: string;
  old_value: boolean;
  new_value: boolean;
  reason?: string;
  timestamp: string;
}

/** Strong-typed map from event type to payload shape. */
export interface WSEventPayloadMap {
  'AI_ORDER_CONFIRMED': UnknownEventPayload;
  'AI_ORDER_REJECTED': UnknownEventPayload;
  'AI_PLAN_DATE_SHIFT': UnknownEventPayload;
  'AI_PLAN_SKU_MODIFIED': UnknownEventPayload;
  'AI_PREDICTION': UnknownEventPayload;
  'AI_PREDICTION_CORRECTED': UnknownEventPayload;
  'BYPASS_TOKEN_ISSUED': UnknownEventPayload;
  'CANCEL_APPROVED': UnknownEventPayload;
  'CANCEL_REQUESTED': UnknownEventPayload;
  'CART_SYNC_UPDATED': UnknownEventPayload;
  'CASH_COLLECTION_REQUIRED': UnknownEventPayload;
  'COMMAND_DISPATCHED': CommandDispatchedPayload;
  'COMMAND_RECEIVED': CommandReceivedPayload;
  'COMMAND_SETTLED': CommandSettledPayload;
  'CREDIT_DELIVERY_MARKED': UnknownEventPayload;
  'CREDIT_DELIVERY_RESOLVED': UnknownEventPayload;
  'DELIVERY_DISPUTED': DeliveryDisputedPayload;
  'DELIVERY_SESSION_UPDATED': DeliverySessionUpdatedPayload;
  'DEMAND_FORECAST_READY': DemandForecastReadyPayload;
  'DISPATCH_LOCK_ACQUIRED': UnknownEventPayload;
  'DISPATCH_LOCK_CHANGE': UnknownEventPayload;
  'DISPATCH_LOCK_RELEASED': UnknownEventPayload;
  'DRIVER_APPROACHING': UnknownEventPayload;
  'DRIVER_ARRIVED': DriverArrivedPayload;
  'DRIVER_AVAILABILITY_CHANGED': DriverAvailabilityChangedPayload;
  'DRIVER_CREATED': DriverCreatedPayload;
  'EARLY_COMPLETE_APPROVED': UnknownEventPayload;
  'EARLY_COMPLETE_REQUESTED': EarlyCompleteRequestedPayload;
  'ETA_UPDATED': UnknownEventPayload;
  'FACTORY_CREATED': FactoryCreatedPayload;
  'FACTORY_MANIFEST_CREATED': UnknownEventPayload;
  'FACTORY_MANIFEST_UPDATE': UnknownEventPayload;
  'FACTORY_OUTBOX_FAILED': UnknownEventPayload;
  'FACTORY_SLA_BREACH': FactorySLABreachPayload;
  'FACTORY_SUPPLY_REQUEST_UPDATE': UnknownEventPayload;
  'FACTORY_TRANSFER_UPDATE': UnknownEventPayload;
  'FEE_RATE_ADJUSTED': FeeRateAdjustedPayload;
  'FLEET_DISPATCHED': FleetDispatchedPayload;
  'FORCE_SEAL_ALERT': ForceSealAlertPayload;
  'FREEZE_LOCK_ACQUIRED': UnknownEventPayload;
  'FREEZE_LOCK_RELEASED': UnknownEventPayload;
  'FULFILLMENT_PAID': UnknownEventPayload;
  'FULFILLMENT_PAYMENT_COMPLETED': UnknownEventPayload;
  'INBOUND_FREIGHT_UNANNOUNCED': InboundFreightUnannouncedPayload;
  'INSIGHT_APPROVED_TRANSFER_CREATED': UnknownEventPayload;
  'INTERNAL_LOAD_CONFIRMED': InternalLoadConfirmedPayload;
  'INVENTORY_IMPORT_STATUS_UPDATE': InventoryImportStatusUpdatePayload;
  'INVENTORY_IMPORT_UPLOADED': InventoryImportUploadedPayload;
  'INVENTORY_SYNC_COMPLETE': UnknownEventPayload;
  'LOOK_AHEAD_COMPLETED': LookAheadCompletedPayload;
  'MANIFEST_CANCELLED': ManifestCancelledPayload;
  'MANIFEST_COMPLETED': UnknownEventPayload;
  'MANIFEST_DISPATCHED': UnknownEventPayload;
  'MANIFEST_DLQ_ESCALATION': UnknownEventPayload;
  'MANIFEST_DRAFT_CREATED': UnknownEventPayload;
  'MANIFEST_FORCE_SEALED': UnknownEventPayload;
  'MANIFEST_LOADING_STARTED': UnknownEventPayload;
  'MANIFEST_ORDER_EXCEPTION': ManifestOrderExceptionPayload;
  'MANIFEST_ORDER_INJECTED': ManifestOrderInjectedPayload;
  'MANIFEST_ORDER_REASSIGNED': ManifestOrderReassignedPayload;
  'MANIFEST_REBALANCED': ManifestRebalancedPayload;
  'MANIFEST_SEALED': UnknownEventPayload;
  'MANIFEST_SETTLED': UnknownEventPayload;
  'MISSING_ITEMS_REPORTED': UnknownEventPayload;
  'NEGOTIATION_PROPOSED': NegotiationProposedPayload;
  'NEGOTIATION_RESOLVED': NegotiationResolvedPayload;
  'NETWORK_MODE_CHANGED': NetworkModeChangedPayload;
  'OFFLOAD_CONFIRMED': OffloadConfirmedPayload;
  'ORDER_AMENDED': UnknownEventPayload;
  'ORDER_ASSIGNED': OrderAssignedPayload;
  'ORDER_CANCELLED': UnknownEventPayload;
  'ORDER_CANCELLED_BY_ORIGIN': OrderCancelledByOriginPayload;
  'ORDER_CANCEL_LOCKED': OrderCancelLockedPayload;
  'ORDER_COMPLETED': OrderCompletedPayload;
  'ORDER_CREATED': OrderCreatedPayload;
  'ORDER_DELAYED': OrderDelayedPayload;
  'ORDER_DISPATCHED': OrderDispatchedPayload;
  'ORDER_FINALIZED': OrderFinalizedPayload;
  'ORDER_MODIFIED': OrderModifiedPayload;
  'ORDER_REASSIGNED': OrderReassignedPayload;
  'ORDER_REJECTED_BY_SUPPLIER': UnknownEventPayload;
  'ORDER_REROUTED': OrderReroutedPayload;
  'ORDER_STATE_CHANGED': UnknownEventPayload;
  'ORDER_STATUS_CHANGED': OrderStatusChangedPayload;
  'ORDER_SYNC': UnknownEventPayload;
  'ORDER_VALIDATION_FAILED': OrderValidationFailedPayload;
  'OUTBOX_FAILED': UnknownEventPayload;
  'OUT_OF_STOCK': OutOfStockPayload;
  'PAYLOAD_OVERFLOW': PayloadOverflowPayload;
  'PAYLOAD_READY_TO_SEAL': PayloadReadyToSealPayload;
  'PAYLOAD_SEALED': PayloadSealedPayload;
  'PAYLOAD_SYNC': PayloadSyncPayload;
  'PAYMENT_BYPASS_COMPLETED': UnknownEventPayload;
  'PAYMENT_BYPASS_ISSUED': UnknownEventPayload;
  'PAYMENT_CLEARED': PaymentClearedPayload;
  'PAYMENT_EXPIRED': UnknownEventPayload;
  'PAYMENT_FAILED': PaymentFailedPayload;
  'PAYMENT_INTENT_CREATED': UnknownEventPayload;
  'PAYMENT_REFUNDED': UnknownEventPayload;
  'PAYMENT_REQUIRED': UnknownEventPayload;
  'PAYMENT_SETTLED': PaymentSettledPayload;
  'POWER_OUTAGE_REPORTED': UnknownEventPayload;
  'PRE_ORDER_AUTO_ACCEPTED': PreOrderAutoAcceptedPayload;
  'PRE_ORDER_CANCELLED': PreOrderCancelledPayload;
  'PRE_ORDER_CONFIRMATION': UnknownEventPayload;
  'PRE_ORDER_CONFIRMED': PreOrderConfirmedPayload;
  'PRE_ORDER_EDITED': PreOrderEditedPayload;
  'PRE_ORDER_NOTIFIED': PreOrderNotifiedPayload;
  'PRE_ORDER_NUDGE': UnknownEventPayload;
  'PULL_MATRIX_COMPLETED': PullMatrixCompletedPayload;
  'REPLENISHMENT_LOCK_ACQUIRED': UnknownEventPayload;
  'REPLENISHMENT_LOCK_RELEASED': UnknownEventPayload;
  'REPLENISHMENT_TRANSFER_CREATED': UnknownEventPayload;
  'RETAILER_PRICE_OVERRIDE': RetailerPriceOverridePayload;
  'RETAILER_REGISTERED': RetailerRegisteredPayload;
  'RETURN_RESOLVED': UnknownEventPayload;
  'ROUTE_CREATED': RouteCreatedPayload;
  'ROUTE_FINALIZED': RouteFinalizedPayload;
  'SETTLEMENT_REQUIRED': SettlementRequiredPayload;
  'SHOP_CLOSED': ShopClosedPayload;
  'SHOP_CLOSED_ALERT': UnknownEventPayload;
  'SHOP_CLOSED_ESCALATED': ShopClosedEscalatedPayload;
  'SHOP_CLOSED_RESOLVED': ShopClosedResolvedPayload;
  'SHOP_CLOSED_RESPONSE': ShopClosedResponsePayload;
  'SMS_QUICK_COMPLETE': UnknownEventPayload;
  'SPLIT_PAYMENT_CREATED': UnknownEventPayload;
  'STOCK_BACKORDERED': StockBackorderedPayload;
  'STOCK_THRESHOLD_BREACH': StockThresholdBreachPayload;
  'SUPPLY_LANE_TRANSIT_UPDATED': SupplyLaneTransitUpdatedPayload;
  'SUPPLY_REQUEST_ACKNOWLEDGED': UnknownEventPayload;
  'SUPPLY_REQUEST_CANCELLED': UnknownEventPayload;
  'SUPPLY_REQUEST_FULFILLED': UnknownEventPayload;
  'SUPPLY_REQUEST_READY': UnknownEventPayload;
  'SUPPLY_REQUEST_SUBMITTED': UnknownEventPayload;
  'SUPPLY_REQUEST_UPDATE': UnknownEventPayload;
  'SYSTEM_BROADCAST': UnknownEventPayload;
  'TOKEN_REFRESH_NEEDED': UnknownEventPayload;
  'TRANSFER_APPROVED': UnknownEventPayload;
  'TRANSFER_RECEIVED': UnknownEventPayload;
  'TRANSFER_STATE_CHANGED': UnknownEventPayload;
  'TRANSFER_UNASSIGNED': TransferUnassignedPayload;
  'UNIFIED_CHECKOUT_COMPLETED': UnifiedCheckoutCompletedPayload;
  'VEHICLE_CREATED': VehicleCreatedPayload;
  'WAREHOUSE_CREATED': WarehouseCreatedPayload;
  'WAREHOUSE_SPATIAL_UPDATED': WarehouseSpatialUpdatedPayload;
  'WAREHOUSE_STATUS_CHANGED': WarehouseStatusChangedPayload;
}

export type WSEventTypeValue = keyof WSEventPayloadMap;
export type WSEventMessage<T extends WSEventTypeValue = WSEventTypeValue> = { type: T } & WSEventPayloadMap[T];
export type WSEvent = { [K in WSEventTypeValue]: WSEventMessage<K> }[WSEventTypeValue];

export type AiOrderConfirmedWSEvent = WSEventMessage<'AI_ORDER_CONFIRMED'>;
export type AiOrderRejectedWSEvent = WSEventMessage<'AI_ORDER_REJECTED'>;
export type AiPlanDateShiftWSEvent = WSEventMessage<'AI_PLAN_DATE_SHIFT'>;
export type AiPlanSkuModifiedWSEvent = WSEventMessage<'AI_PLAN_SKU_MODIFIED'>;
export type AiPredictionWSEvent = WSEventMessage<'AI_PREDICTION'>;
export type AiPredictionCorrectedWSEvent = WSEventMessage<'AI_PREDICTION_CORRECTED'>;
export type BypassTokenIssuedWSEvent = WSEventMessage<'BYPASS_TOKEN_ISSUED'>;
export type CancelApprovedWSEvent = WSEventMessage<'CANCEL_APPROVED'>;
export type CancelRequestedWSEvent = WSEventMessage<'CANCEL_REQUESTED'>;
export type CartSyncUpdatedWSEvent = WSEventMessage<'CART_SYNC_UPDATED'>;
export type CashCollectionRequiredWSEvent = WSEventMessage<'CASH_COLLECTION_REQUIRED'>;
export type CommandDispatchedWSEvent = WSEventMessage<'COMMAND_DISPATCHED'>;
export type CommandReceivedWSEvent = WSEventMessage<'COMMAND_RECEIVED'>;
export type CommandSettledWSEvent = WSEventMessage<'COMMAND_SETTLED'>;
export type CreditDeliveryMarkedWSEvent = WSEventMessage<'CREDIT_DELIVERY_MARKED'>;
export type CreditDeliveryResolvedWSEvent = WSEventMessage<'CREDIT_DELIVERY_RESOLVED'>;
export type DeliveryDisputedWSEvent = WSEventMessage<'DELIVERY_DISPUTED'>;
export type DeliverySessionUpdatedWSEvent = WSEventMessage<'DELIVERY_SESSION_UPDATED'>;
export type DemandForecastReadyWSEvent = WSEventMessage<'DEMAND_FORECAST_READY'>;
export type DispatchLockAcquiredWSEvent = WSEventMessage<'DISPATCH_LOCK_ACQUIRED'>;
export type DispatchLockChangeWSEvent = WSEventMessage<'DISPATCH_LOCK_CHANGE'>;
export type DispatchLockReleasedWSEvent = WSEventMessage<'DISPATCH_LOCK_RELEASED'>;
export type DriverApproachingWSEvent = WSEventMessage<'DRIVER_APPROACHING'>;
export type DriverArrivedWSEvent = WSEventMessage<'DRIVER_ARRIVED'>;
export type DriverAvailabilityChangedWSEvent = WSEventMessage<'DRIVER_AVAILABILITY_CHANGED'>;
export type DriverCreatedWSEvent = WSEventMessage<'DRIVER_CREATED'>;
export type EarlyCompleteApprovedWSEvent = WSEventMessage<'EARLY_COMPLETE_APPROVED'>;
export type EarlyCompleteRequestedWSEvent = WSEventMessage<'EARLY_COMPLETE_REQUESTED'>;
export type EtaUpdatedWSEvent = WSEventMessage<'ETA_UPDATED'>;
export type FactoryCreatedWSEvent = WSEventMessage<'FACTORY_CREATED'>;
export type FactoryManifestCreatedWSEvent = WSEventMessage<'FACTORY_MANIFEST_CREATED'>;
export type FactoryManifestUpdateWSEvent = WSEventMessage<'FACTORY_MANIFEST_UPDATE'>;
export type FactoryOutboxFailedWSEvent = WSEventMessage<'FACTORY_OUTBOX_FAILED'>;
export type FactorySlaBreachWSEvent = WSEventMessage<'FACTORY_SLA_BREACH'>;
export type FactorySupplyRequestUpdateWSEvent = WSEventMessage<'FACTORY_SUPPLY_REQUEST_UPDATE'>;
export type FactoryTransferUpdateWSEvent = WSEventMessage<'FACTORY_TRANSFER_UPDATE'>;
export type FeeRateAdjustedWSEvent = WSEventMessage<'FEE_RATE_ADJUSTED'>;
export type FleetDispatchedWSEvent = WSEventMessage<'FLEET_DISPATCHED'>;
export type ForceSealAlertWSEvent = WSEventMessage<'FORCE_SEAL_ALERT'>;
export type FreezeLockAcquiredWSEvent = WSEventMessage<'FREEZE_LOCK_ACQUIRED'>;
export type FreezeLockReleasedWSEvent = WSEventMessage<'FREEZE_LOCK_RELEASED'>;
export type FulfillmentPaidWSEvent = WSEventMessage<'FULFILLMENT_PAID'>;
export type FulfillmentPaymentCompletedWSEvent = WSEventMessage<'FULFILLMENT_PAYMENT_COMPLETED'>;
export type InboundFreightUnannouncedWSEvent = WSEventMessage<'INBOUND_FREIGHT_UNANNOUNCED'>;
export type InsightApprovedTransferCreatedWSEvent = WSEventMessage<'INSIGHT_APPROVED_TRANSFER_CREATED'>;
export type InternalLoadConfirmedWSEvent = WSEventMessage<'INTERNAL_LOAD_CONFIRMED'>;
export type InventoryImportStatusUpdateWSEvent = WSEventMessage<'INVENTORY_IMPORT_STATUS_UPDATE'>;
export type InventoryImportUploadedWSEvent = WSEventMessage<'INVENTORY_IMPORT_UPLOADED'>;
export type InventorySyncCompleteWSEvent = WSEventMessage<'INVENTORY_SYNC_COMPLETE'>;
export type LookAheadCompletedWSEvent = WSEventMessage<'LOOK_AHEAD_COMPLETED'>;
export type ManifestCancelledWSEvent = WSEventMessage<'MANIFEST_CANCELLED'>;
export type ManifestCompletedWSEvent = WSEventMessage<'MANIFEST_COMPLETED'>;
export type ManifestDispatchedWSEvent = WSEventMessage<'MANIFEST_DISPATCHED'>;
export type ManifestDlqEscalationWSEvent = WSEventMessage<'MANIFEST_DLQ_ESCALATION'>;
export type ManifestDraftCreatedWSEvent = WSEventMessage<'MANIFEST_DRAFT_CREATED'>;
export type ManifestForceSealedWSEvent = WSEventMessage<'MANIFEST_FORCE_SEALED'>;
export type ManifestLoadingStartedWSEvent = WSEventMessage<'MANIFEST_LOADING_STARTED'>;
export type ManifestOrderExceptionWSEvent = WSEventMessage<'MANIFEST_ORDER_EXCEPTION'>;
export type ManifestOrderInjectedWSEvent = WSEventMessage<'MANIFEST_ORDER_INJECTED'>;
export type ManifestOrderReassignedWSEvent = WSEventMessage<'MANIFEST_ORDER_REASSIGNED'>;
export type ManifestRebalancedWSEvent = WSEventMessage<'MANIFEST_REBALANCED'>;
export type ManifestSealedWSEvent = WSEventMessage<'MANIFEST_SEALED'>;
export type ManifestSettledWSEvent = WSEventMessage<'MANIFEST_SETTLED'>;
export type MissingItemsReportedWSEvent = WSEventMessage<'MISSING_ITEMS_REPORTED'>;
export type NegotiationProposedWSEvent = WSEventMessage<'NEGOTIATION_PROPOSED'>;
export type NegotiationResolvedWSEvent = WSEventMessage<'NEGOTIATION_RESOLVED'>;
export type NetworkModeChangedWSEvent = WSEventMessage<'NETWORK_MODE_CHANGED'>;
export type OffloadConfirmedWSEvent = WSEventMessage<'OFFLOAD_CONFIRMED'>;
export type OrderAmendedWSEvent = WSEventMessage<'ORDER_AMENDED'>;
export type OrderAssignedWSEvent = WSEventMessage<'ORDER_ASSIGNED'>;
export type OrderCancelledWSEvent = WSEventMessage<'ORDER_CANCELLED'>;
export type OrderCancelledByOriginWSEvent = WSEventMessage<'ORDER_CANCELLED_BY_ORIGIN'>;
export type OrderCancelLockedWSEvent = WSEventMessage<'ORDER_CANCEL_LOCKED'>;
export type OrderCompletedWSEvent = WSEventMessage<'ORDER_COMPLETED'>;
export type OrderCreatedWSEvent = WSEventMessage<'ORDER_CREATED'>;
export type OrderDelayedWSEvent = WSEventMessage<'ORDER_DELAYED'>;
export type OrderDispatchedWSEvent = WSEventMessage<'ORDER_DISPATCHED'>;
export type OrderFinalizedWSEvent = WSEventMessage<'ORDER_FINALIZED'>;
export type OrderModifiedWSEvent = WSEventMessage<'ORDER_MODIFIED'>;
export type OrderReassignedWSEvent = WSEventMessage<'ORDER_REASSIGNED'>;
export type OrderRejectedBySupplierWSEvent = WSEventMessage<'ORDER_REJECTED_BY_SUPPLIER'>;
export type OrderReroutedWSEvent = WSEventMessage<'ORDER_REROUTED'>;
export type OrderStateChangedWSEvent = WSEventMessage<'ORDER_STATE_CHANGED'>;
export type OrderStatusChangedWSEvent = WSEventMessage<'ORDER_STATUS_CHANGED'>;
export type OrderSyncWSEvent = WSEventMessage<'ORDER_SYNC'>;
export type OrderValidationFailedWSEvent = WSEventMessage<'ORDER_VALIDATION_FAILED'>;
export type OutboxFailedWSEvent = WSEventMessage<'OUTBOX_FAILED'>;
export type OutOfStockWSEvent = WSEventMessage<'OUT_OF_STOCK'>;
export type PayloadOverflowWSEvent = WSEventMessage<'PAYLOAD_OVERFLOW'>;
export type PayloadReadyToSealWSEvent = WSEventMessage<'PAYLOAD_READY_TO_SEAL'>;
export type PayloadSealedWSEvent = WSEventMessage<'PAYLOAD_SEALED'>;
export type PayloadSyncWSEvent = WSEventMessage<'PAYLOAD_SYNC'>;
export type PaymentBypassCompletedWSEvent = WSEventMessage<'PAYMENT_BYPASS_COMPLETED'>;
export type PaymentBypassIssuedWSEvent = WSEventMessage<'PAYMENT_BYPASS_ISSUED'>;
export type PaymentClearedWSEvent = WSEventMessage<'PAYMENT_CLEARED'>;
export type PaymentExpiredWSEvent = WSEventMessage<'PAYMENT_EXPIRED'>;
export type PaymentFailedWSEvent = WSEventMessage<'PAYMENT_FAILED'>;
export type PaymentIntentCreatedWSEvent = WSEventMessage<'PAYMENT_INTENT_CREATED'>;
export type PaymentRefundedWSEvent = WSEventMessage<'PAYMENT_REFUNDED'>;
export type PaymentRequiredWSEvent = WSEventMessage<'PAYMENT_REQUIRED'>;
export type PaymentSettledWSEvent = WSEventMessage<'PAYMENT_SETTLED'>;
export type PowerOutageReportedWSEvent = WSEventMessage<'POWER_OUTAGE_REPORTED'>;
export type PreOrderAutoAcceptedWSEvent = WSEventMessage<'PRE_ORDER_AUTO_ACCEPTED'>;
export type PreOrderCancelledWSEvent = WSEventMessage<'PRE_ORDER_CANCELLED'>;
export type PreOrderConfirmationWSEvent = WSEventMessage<'PRE_ORDER_CONFIRMATION'>;
export type PreOrderConfirmedWSEvent = WSEventMessage<'PRE_ORDER_CONFIRMED'>;
export type PreOrderEditedWSEvent = WSEventMessage<'PRE_ORDER_EDITED'>;
export type PreOrderNotifiedWSEvent = WSEventMessage<'PRE_ORDER_NOTIFIED'>;
export type PreOrderNudgeWSEvent = WSEventMessage<'PRE_ORDER_NUDGE'>;
export type PullMatrixCompletedWSEvent = WSEventMessage<'PULL_MATRIX_COMPLETED'>;
export type ReplenishmentLockAcquiredWSEvent = WSEventMessage<'REPLENISHMENT_LOCK_ACQUIRED'>;
export type ReplenishmentLockReleasedWSEvent = WSEventMessage<'REPLENISHMENT_LOCK_RELEASED'>;
export type ReplenishmentTransferCreatedWSEvent = WSEventMessage<'REPLENISHMENT_TRANSFER_CREATED'>;
export type RetailerPriceOverrideWSEvent = WSEventMessage<'RETAILER_PRICE_OVERRIDE'>;
export type RetailerRegisteredWSEvent = WSEventMessage<'RETAILER_REGISTERED'>;
export type ReturnResolvedWSEvent = WSEventMessage<'RETURN_RESOLVED'>;
export type RouteCreatedWSEvent = WSEventMessage<'ROUTE_CREATED'>;
export type RouteFinalizedWSEvent = WSEventMessage<'ROUTE_FINALIZED'>;
export type SettlementRequiredWSEvent = WSEventMessage<'SETTLEMENT_REQUIRED'>;
export type ShopClosedWSEvent = WSEventMessage<'SHOP_CLOSED'>;
export type ShopClosedAlertWSEvent = WSEventMessage<'SHOP_CLOSED_ALERT'>;
export type ShopClosedEscalatedWSEvent = WSEventMessage<'SHOP_CLOSED_ESCALATED'>;
export type ShopClosedResolvedWSEvent = WSEventMessage<'SHOP_CLOSED_RESOLVED'>;
export type ShopClosedResponseWSEvent = WSEventMessage<'SHOP_CLOSED_RESPONSE'>;
export type SmsQuickCompleteWSEvent = WSEventMessage<'SMS_QUICK_COMPLETE'>;
export type SplitPaymentCreatedWSEvent = WSEventMessage<'SPLIT_PAYMENT_CREATED'>;
export type StockBackorderedWSEvent = WSEventMessage<'STOCK_BACKORDERED'>;
export type StockThresholdBreachWSEvent = WSEventMessage<'STOCK_THRESHOLD_BREACH'>;
export type SupplyLaneTransitUpdatedWSEvent = WSEventMessage<'SUPPLY_LANE_TRANSIT_UPDATED'>;
export type SupplyRequestAcknowledgedWSEvent = WSEventMessage<'SUPPLY_REQUEST_ACKNOWLEDGED'>;
export type SupplyRequestCancelledWSEvent = WSEventMessage<'SUPPLY_REQUEST_CANCELLED'>;
export type SupplyRequestFulfilledWSEvent = WSEventMessage<'SUPPLY_REQUEST_FULFILLED'>;
export type SupplyRequestReadyWSEvent = WSEventMessage<'SUPPLY_REQUEST_READY'>;
export type SupplyRequestSubmittedWSEvent = WSEventMessage<'SUPPLY_REQUEST_SUBMITTED'>;
export type SupplyRequestUpdateWSEvent = WSEventMessage<'SUPPLY_REQUEST_UPDATE'>;
export type SystemBroadcastWSEvent = WSEventMessage<'SYSTEM_BROADCAST'>;
export type TokenRefreshNeededWSEvent = WSEventMessage<'TOKEN_REFRESH_NEEDED'>;
export type TransferApprovedWSEvent = WSEventMessage<'TRANSFER_APPROVED'>;
export type TransferReceivedWSEvent = WSEventMessage<'TRANSFER_RECEIVED'>;
export type TransferStateChangedWSEvent = WSEventMessage<'TRANSFER_STATE_CHANGED'>;
export type TransferUnassignedWSEvent = WSEventMessage<'TRANSFER_UNASSIGNED'>;
export type UnifiedCheckoutCompletedWSEvent = WSEventMessage<'UNIFIED_CHECKOUT_COMPLETED'>;
export type VehicleCreatedWSEvent = WSEventMessage<'VEHICLE_CREATED'>;
export type WarehouseCreatedWSEvent = WSEventMessage<'WAREHOUSE_CREATED'>;
export type WarehouseSpatialUpdatedWSEvent = WSEventMessage<'WAREHOUSE_SPATIAL_UPDATED'>;
export type WarehouseStatusChangedWSEvent = WSEventMessage<'WAREHOUSE_STATUS_CHANGED'>;
