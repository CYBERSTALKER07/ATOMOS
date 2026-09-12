import { RetailerId } from "./primitives";

// ── Envelope + event-type union ─────────────────────────────────────────────
export interface WsEventEnvelope<T extends string = string, P = unknown> {
  type: T;
  trace_id: string;
  timestamp: string; // RFC3339Nano
  v: number;
  schema_version?: number;
  data?: P;
}

/** Retail OS Phase 0 capability pack ids. */
export type RetailerCapabilityPackId =
  | "CORE"
  | "TEAM"
  | "LOCATIONS"
  | "STORE_STOCK"
  | "SECTIONS"
  | "POS"
  | "SHIFTS"
  | "REPORTS_PRO"
  | "CUSTOMER_ASSIST";

export type RetailerStaffRole =
  | "OWNER"
  | "ADMIN"
  | "MANAGER"
  | "BUYER"
  | "RECEIVER"
  | "CASHIER"
  | "STOCK_CLERK"
  | "SECTION_LEAD"
  | "VIEWER";

export interface RetailerCapabilityPackMeta {
  id: RetailerCapabilityPackId | string;
  name: string;
  description: string;
  hard_deps?: string[];
  soft_deps?: string[];
  always_on?: boolean;
}

export interface RetailerCapabilityPackStatus extends RetailerCapabilityPackMeta {
  enabled: boolean;
  config?: Record<string, unknown>;
}

export interface RetailerMeResponse {
  user_id: string;
  retailer_id: RetailerId;
  retailer_org_id: RetailerId;
  retailer_role: RetailerStaffRole | string;
  name: string;
  phone?: string;
  is_owner: boolean;
  is_configured: boolean;
  permissions: string[];
  capabilities: string[];
  packs: RetailerCapabilityPackStatus[];
  location_ids?: string[];
}

export interface RetailerCapabilitiesResponse {
  retailer_id: RetailerId;
  capabilities: string[];
  packs: RetailerCapabilityPackStatus[];
}

export interface RetailerCapabilityMutationResult {
  status: "OK" | "BLOCKED" | "WARN" | string;
  pack_id?: string;
  enabled?: boolean;
  capabilities?: string[];
  missing_hard_deps?: string[];
  missing_soft_deps?: string[];
  enable_all?: string[];
  message?: string;
  evaluation?: RetailerCapabilityMutationResult;
}

/** Retail OS Phase 1 team member. */
export interface RetailerOrgMember {
  user_id: string;
  retailer_id: RetailerId;
  name: string;
  phone: string;
  retailer_role: RetailerStaffRole | string;
  is_owner: boolean;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface RetailerOrgMembersResponse {
  retailer_id: RetailerId;
  items: RetailerOrgMember[];
  updated_at: string;
}

export interface CreateRetailerOrgMemberRequest {
  name: string;
  phone: string;
  password: string;
  retailer_role: RetailerStaffRole | string;
  is_active?: boolean;
}

export interface UpdateRetailerOrgMemberRequest {
  name?: string;
  retailer_role?: RetailerStaffRole | string;
  password?: string;
  is_active?: boolean;
}

export type RetailerHomeSurface =
  | "dashboard"
  | "pos"
  | "dock"
  | "catalog"
  | "insights";

/** Retail OS Phase 2 store branch. */
export interface RetailerLocation {
  location_id: string;
  retailer_id: RetailerId;
  name: string;
  delivery_address?: string;
  place_id?: string;
  lat?: number;
  lng?: number;
  h3_cell?: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
  is_primary: boolean;
  is_active: boolean;
  created_at?: string;
  updated_at?: string;
}

export interface RetailerLocationsResponse {
  retailer_id: RetailerId;
  active_location_id?: string;
  items: RetailerLocation[];
}

export interface CreateRetailerLocationRequest {
  name: string;
  delivery_address?: string;
  place_id?: string;
  lat?: number;
  lng?: number;
  h3_cell?: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
  is_primary?: boolean;
}

export interface SwitchRetailerLocationRequest {
  location_id: string;
}

export interface SwitchRetailerLocationResponse {
  token: string;
  refresh_token: string;
  active_location_id: string;
  location: RetailerLocation;
}

/** Wave C1.2/C1.3 multi-org membership (person → retailer orgs). */
export interface RetailerMembershipDTO {
  user_id: string;
  retailer_id: string;
  retailer_role: string;
  name?: string;
  phone?: string;
  is_active: boolean;
}

export type RetailerAuthTokenType = "full" | "pending_org_select";

/** Login may return full JWT or intermediate PendingOrgSelect (C1.2). */
export interface RetailerLoginResponse {
  token: string;
  token_type?: RetailerAuthTokenType;
  refresh_token?: string;
  is_configured?: boolean;
  retailer_id?: string;
  retailer_org_id?: string;
  user_id?: string;
  retailer_role?: string;
  memberships?: RetailerMembershipDTO[];
  membership_count?: number;
  expires_in_sec?: number;
  user?: {
    id: string;
    retailer_id?: string;
    name: string;
    company?: string;
    email?: string;
    avatar_url?: string | null;
    role?: string;
  };
  home_surface?: string;
  capabilities?: string[];
  active_location_id?: string;
  location_ids?: string[];
  firebase_token?: string;
}

export interface RetailerSelectOrgRequest {
  retailer_id: string;
}

export interface RetailerSwitchOrgRequest {
  retailer_id: string;
}

export interface RetailerMembershipsResponse {
  memberships: RetailerMembershipDTO[];
}

/** Wave C3.1/C3.2 parked POS cart (server hold). Never reserves stock. */
export type RetailerPosHoldStatus = "HELD" | "RESUMED" | "EXPIRED" | "VOIDED";

export interface RetailerPosHold {
  hold_id: string;
  retailer_id: string;
  location_id: string;
  register_id?: string;
  user_id: string;
  status: RetailerPosHoldStatus | string;
  cart: unknown;
  note?: string;
  expires_at: string;
  created_at?: string;
  updated_at?: string;
  resumed_at?: string;
  voided_at?: string;
}

export interface RetailerPosHoldsListResponse {
  retailer_id: string;
  location_id?: string;
  items: RetailerPosHold[];
}

export interface ParkRetailerPosHoldRequest {
  location_id: string;
  register_id?: string;
  cart: unknown;
  note?: string;
}

export interface ResumeRetailerPosHoldRequest {
  location_id: string;
}

/** Wave C2.2 franchise HQ analytics reads. */
export interface RetailerHqSummary {
  retailer_id: string;
  day: string;
  location_count: number;
  sku_count: number;
  qty_sold: number;
  qty_voided: number;
  gross_minor: number;
  net_minor: number;
  currency: string;
  honest_empty?: boolean;
}

export interface RetailerHqLocationSales {
  location_id: string;
  qty_sold: number;
  qty_voided: number;
  gross_minor: number;
  net_minor: number;
  currency: string;
}

export interface RetailerHqSalesByLocationResponse {
  retailer_id: string;
  day: string;
  items: RetailerHqLocationSales[];
  org_net_minor: number;
  sum_locations: number;
  balanced: boolean;
  honest_empty?: boolean;
}

/** Wave C3.3 offline stock count version protocol. */
export interface RetailerStockCountVersionResponse {
  retailer_id: string;
  location_id: string;
  stock_bin: string;
  version: number;
}

export interface RetailerStockCountCommitRequest {
  location_id: string;
  stock_bin?: string;
  base_version: number;
  lines: { sku_id?: string; sku?: string; counted_qty: number }[];
  force?: boolean;
  force_reason?: string;
}

/** Response of POST /v1/retailer/stock/counts/commit (mirrors backend-go retailer/stock_count_commit.go). */
export interface RetailerStockCountCommitResponse {
  count_id: string;
  location_id: string;
  stock_bin: string;
  status: string;
  base_version: number;
  server_version: number;
  forced: boolean;
  lines: { sku: string; system_qty: number; counted_qty: number; variance: number }[];
}

export interface RetailerStockCountVersionConflict {
  error: "COUNT_VERSION_CONFLICT";
  server_version: number;
  server_lines: { sku_id: string; counted_qty: number; on_hand: number }[];
  message: string;
}

/** Retail OS Phase 3 store stock. */
export type RetailerStockBin = "BACKROOM" | "FLOOR" | "QUARANTINE";

export interface RetailerStockBalance {
  location_id: string;
  stock_bin: RetailerStockBin | string;
  sku: string;
  on_hand: number;
  reserved: number;
  available: number;
}

export interface RetailerStockListResponse {
  retailer_id: string;
  location_id: string;
  items: RetailerStockBalance[];
}

export interface RetailerReceiveLine {
  sku: string;
  product_name?: string;
  ordered_qty: number;
  accepted_qty: number;
  damaged_qty?: number;
  missing_qty?: number;
}

export interface RetailerReceiveSession {
  session_id: string;
  retailer_id: string;
  location_id: string;
  order_id: string;
  status: "DRAFT" | "CONFIRMED" | string;
  lines: RetailerReceiveLine[];
  created_at?: string;
  confirmed_at?: string;
}

export interface CreateRetailerReceiveSessionRequest {
  order_id: string;
  location_id?: string;
  confirm?: boolean;
  stock_bin?: RetailerStockBin | string;
  lines?: { sku: string; accepted_qty: number; damaged_qty?: number; missing_qty?: number }[];
}

export interface RetailerStockTransferRequest {
  location_id?: string;
  from_location_id?: string;
  to_location_id?: string;
  sku: string;
  qty: number;
  from_bin?: RetailerStockBin | string;
  to_bin?: RetailerStockBin | string;
  note?: string;
}

export interface RetailerStockAdjustRequest {
  location_id?: string;
  sku: string;
  qty_delta: number;
  stock_bin?: RetailerStockBin | string;
  note?: string;
}

export interface RetailerStockCountRequest {
  location_id?: string;
  stock_bin?: RetailerStockBin | string;
  commit?: boolean;
  lines: { sku: string; counted_qty: number }[];
}

/** Retail OS Phase 4 POS */
export interface RetailerRegister {
  register_id: string;
  retailer_id: string;
  location_id: string;
  label: string;
  status: string;
  created_at?: string;
}

export interface RetailerPosSession {
  session_id: string;
  register_id: string;
  location_id: string;
  retailer_id: string;
  opened_by_user_id: string;
  closed_by_user_id?: string;
  status: "OPEN" | "CLOSED" | string;
  opening_float_minor: number;
  closing_cash_minor?: number;
  expected_cash_minor?: number;
  variance_minor?: number;
  currency: string;
  opened_at?: string;
  closed_at?: string;
}

export interface RetailerPosSaleLine {
  sku: string;
  name?: string;
  qty: number;
  unit_price_minor: number;
  line_total_minor: number;
}

export interface RetailerPosTender {
  method: "CASH" | "CARD" | "OTHER" | string;
  amount_minor: number;
}

export interface RetailerPosSale {
  sale_id: string;
  session_id: string;
  register_id: string;
  location_id: string;
  retailer_id: string;
  cashier_user_id: string;
  status: "COMPLETED" | "VOIDED" | string;
  total_minor: number;
  currency: string;
  receipt_number: string;
  lines: RetailerPosSaleLine[];
  tenders: RetailerPosTender[];
  stock_bin: string;
  created_at?: string;
  voided_at?: string;
  voided_by_user_id?: string;
  void_reason?: string;
}

export interface CreateRetailerPosSaleRequest {
  session_id: string;
  stock_bin?: string;
  currency?: string;
  lines: { sku: string; name?: string; qty: number; unit_price_minor: number }[];
  tenders?: RetailerPosTender[];
}

/** Retail OS Phase 5 shifts & time clock */
export interface RetailerTimeEntry {
  entry_id: string;
  retailer_id: string;
  user_id: string;
  location_id: string;
  status: "OPEN" | "CLOSED" | string;
  clock_in_at?: string;
  clock_out_at?: string;
  auto_closed?: boolean;
  note?: string;
}

export interface RetailerTimeEntriesResponse {
  items: RetailerTimeEntry[];
  open_entry?: RetailerTimeEntry;
  clocked_in: boolean;
}

export interface RetailerShift {
  shift_id: string;
  retailer_id: string;
  location_id: string;
  register_id?: string;
  opened_by_user_id: string;
  closed_by_user_id?: string;
  status: "OPEN" | "CLOSED" | string;
  opening_float_minor: number;
  closing_cash_minor?: number;
  expected_cash_minor?: number;
  variance_minor?: number;
  currency: string;
  linked_pos_session_id?: string;
  opened_at?: string;
  closed_at?: string;
}

export interface RetailerShiftsResponse {
  items: RetailerShift[];
}

export interface OpenRetailerShiftRequest {
  location_id?: string;
  register_id?: string;
  opening_float_minor: number;
  currency?: string;
}

export interface CloseRetailerShiftRequest {
  closing_cash_minor: number;
}

/** Retail OS Phase 6 sections */
export interface RetailerSection {
  section_id: string;
  retailer_id: string;
  location_id: string;
  name: string;
  aisle_tag?: string;
  shelf_tag?: string;
  sort_order?: number;
  status: "ACTIVE" | "INACTIVE" | string;
  sku_count?: number;
  staff_ids?: string[];
  created_at?: string;
  updated_at?: string;
}

export interface RetailerSectionsResponse {
  items: RetailerSection[];
}

/** Retail OS Phase 6 reports pro */
export interface RetailerReportsSummary {
  from?: string;
  to?: string;
  location_id?: string;
  sales_minor: number;
  sale_count: number;
  on_hand_sku_count: number;
  low_stock_count: number;
  open_variances: number;
  top_skus?: { sku: string; sales_minor: number; units: number }[];
  pack?: string;
}

export interface RetailerReportsSalesItem {
  key: string;
  sales_minor: number;
  sale_count: number;
  units: number;
}

/** Retail OS Phase 6 assist */
export type RetailerAssistTicketStatus =
  | "OPEN"
  | "CLAIMED"
  | "DONE"
  | "CANCELLED"
  | string;

export interface RetailerAssistTicket {
  ticket_id: string;
  retailer_id: string;
  location_id: string;
  section_id: string;
  note: string;
  status: RetailerAssistTicketStatus;
  created_by_user_id: string;
  claimed_by_user_id?: string;
  completed_by_user_id?: string;
  created_at?: string;
  claimed_at?: string;
  completed_at?: string;
  sla_due_at?: string;
}

export interface RetailerAssistTicketsResponse {
  items: RetailerAssistTicket[];
}

export interface RetailerSupplierOrderFacet {
  supplier_id: string;
  orders_by_status: Record<string, number>;
}

export interface RetailerPulseLoyalty {
  enrolled: boolean;
}

/** Retail OS Phase 7 — honest ops pulse (never demo) */
export interface RetailerControlTowerPulse {
  retailer_id: string;
  generated_at: string;
  open_orders: number;
  active_fulfillments: number;
  dock_pending: number;
  pos_open_sessions: number;
  open_shifts: number;
  open_assist_tickets: number;
  low_stock_sku_bins: number;
  shift_variances_7d: number;
  sales_minor_7d: number;
  capabilities: string[];
  empty: boolean;
  source?: "spanner" | "memory" | "empty" | string;
  orders_by_status?: Record<string, number>;
  orders_by_supplier?: RetailerSupplierOrderFacet[];
  loyalty?: RetailerPulseLoyalty;
}

export type EventType =
  | "SUPPLIER_CREATED"
  | "SUPPLIER_UPDATED"
  | "SUPPLIER_PROFILE_UPDATED"
  | "SUPPLIER_BILLING_UPDATED"
  | "SUPPLIER_BILLING_CONFIGURED"
  | "SUPPLIER_MEMBER_ADDED"
  | "SUPPLIER_SERVICE_PROMISE_CREATED"
  | "SUPPLIER_SERVICE_PROMISE_BREACHED"
  | "SUPPLIER_BROADCAST"
  | "RETAILER_REGISTERED"
  | "RETAILER_STAFF_CREATED"
  | "RETAILER_CAPABILITY_PACK_CHANGED"
  | "RETAILER_AUTO_ORDER_UPDATED"
  | "RETAILER_LOCATION_CREATED"
  | "RETAILER_LOCATION_UPDATED"
  | "STORE_STOCK_RECEIVED"
  | "STORE_STOCK_ADJUSTED"
  | "STORE_STOCK_TRANSFERRED"
  | "STORE_STOCK_COUNTED"
  | "POS_SESSION_OPENED"
  | "POS_SESSION_CLOSED"
  | "POS_SALE_COMPLETED"
  | "POS_SALE_VOIDED"
  | "RETAILER_SELL_THROUGH_UPDATED"
  | "DEMAND_SIGNAL"
  | "RETAILER_CLOCK_IN"
  | "RETAILER_CLOCK_OUT"
  | "RETAILER_SHIFT_OPENED"
  | "RETAILER_SHIFT_CLOSED"
  | "RETAILER_SHIFT_CASH_VARIANCE"
  | "RETAILER_SECTION_CREATED"
  | "RETAILER_SECTION_UPDATED"
  | "RETAILER_SECTION_SKU_MAPPED"
  | "RETAILER_STAFF_SECTION_ASSIGNED"
  | "RETAILER_ASSIST_TICKET_OPENED"
  | "RETAILER_ASSIST_TICKET_CLAIMED"
  | "RETAILER_ASSIST_TICKET_COMPLETED"
  | "RETAILER_ASSIST_TICKET_CANCELLED"
  | "DRIVER_CREATED"
  | "DRIVER_AVAILABILITY_CHANGED"
  | "DRIVER_LOCATION_UPDATED"
  | "DRIVER_RETURN_APPROACHING"
  | "VEHICLE_CREATED"
  | "VEHICLE_AVAILABILITY_CHANGED"
  | "WAREHOUSE_CREATED"
  | "WAREHOUSE_BROADCAST"
  | "WAREHOUSE_LOCATION_UPDATED"
  | "WAREHOUSE_DISPATCH_LOCK_CHANGED"
  | "WAREHOUSE_SUPPLY_REQUEST_OPENED"
  | "WAREHOUSE_TRANSFER_CREATED"
  | "WAREHOUSE_TRANSFER_RECEIVED"
  | "SUPPLY_TRANSFER_APPROACHING"
  | "FACTORY_CREATED"
  | "FACTORY_LOCATION_UPDATED"
  | "FACTORY_STAFF_CREATED"
  | "FACTORY_STAFF_PASSWORD_SET"
  | "FACTORY_SUPPLY_REQUEST_UPDATE"
  | "TRANSFER_CREATED"
  | "ORDER_CREATED"
  | "ORDER_STATUS_CHANGED"
  | "ORDER_VALIDATION_FAILED"
  | "ORDER_ASSIGNED"
  | "ORDER_REASSIGNED"
  | "ORDER_FINALIZED"
  | "ORDER_AMENDED"
  | "MISSING_ITEMS_REPORTED"
  | "ROUTE_CREATED"
  | "ROUTE_REORDERED"
  | "MANIFEST_DRAFT_CREATED"
  | "MANIFEST_LOADING_STARTED"
  | "MANIFEST_ORDER_INJECTED"
  | "MANIFEST_ORDER_EXCEPTION"
  | "MANIFEST_EXCEPTION_RESOLVED"
  | "MANIFEST_DLQ_ESCALATION"
  | "MANIFEST_REBALANCED"
  | "MANIFEST_CANCELLED"
  | "MANIFEST_SEALED"
  | "MANIFEST_DISPATCHED"
  | "MANIFEST_COMPLETED"
  | "SPLIT_PAYMENT_CREATED"
  | "PAYMENT_CLEARED"
  | "PAYMENT_FAILED"
  | "PAYMENT_REQUIRED"
  | "SETTLEMENT_REQUIRED"
  | "AR_INVOICE_OPENED"
  | "AR_INVOICE_PAYMENT"
  | "AR_INVOICE_DUNNED"
  | "AR_INVOICE_SETTLED"
  | "PAYOUT_BATCH_GENERATED"
  | "PAYOUT_BATCH_EXPORTED"
  | "PAYOUT_BATCH_DISPATCHED"
  | "PAYOUT_BATCH_PAID"
  | "PAYOUT_POLICY_UPDATED"
  | "FISCAL_RECEIPT_REQUESTED"
  | "FISCAL_RECEIPT_SUCCEEDED"
  | "FISCAL_RECEIPT_FAILED"
  | "BUYER_ACCEPTANCE_PENDING"
  | "BUYER_ACCEPTANCE_ACCEPTED"
  | "BUYER_ACCEPTANCE_REJECTED"
  | "BUYER_ACCEPTANCE_EXPIRED"
  | "ORDER_FORCE_COMPLETED"
  | "CASH_SHORTFALL"
  | "CASH_OVERAGE"
  | "DELIVERY_SESSION_UPDATED"
  | "DELIVERY_DISPUTED"
  | "SHOP_CLOSED"
  | "SHOP_CLOSED_RESPONSE"
  | "SHOP_CLOSED_ESCALATED"
  | "SHOP_CLOSED_RESOLVED"
  | "SHOP_CLOSED_BYPASS_OFFLOAD"
  | "SHOP_CLOSED_TIMEOUT"
  | "PROXIMITY_UNLOCKED"
  | "PARTIAL_OFFLOAD"
  | "CREDIT_LEAVE"
  | "CREDIT_DELIVERY_MARKED"
  | "CREDIT_DELIVERY_RESOLVED"
  | "NEGOTIATION_PROPOSED"
  | "NEGOTIATION_RESOLVED"
  | "CART_SYNC_UPDATED"
  | "INVENTORY_SYNC_COMPLETE"
  | "INVENTORY_IMPORT_UPLOADED"
  | "INVENTORY_IMPORT_STATUS_UPDATE"
  | "PROMOTION_CHANGED"
  | "RETAILER_PRICE_OVERRIDE"
  | "COMMAND_DISPATCHED"
  | "COMMAND_RECEIVED"
  | "COMMAND_SETTLED"
  | "SYSTEM_APP_OUTDATED"
  | "FREEZE_LOCK_ACQUIRED"
  | "FREEZE_LOCK_RELEASED"
  | "SUPPLIER_RETURN_CREATED"
  | "SUPPLIER_RETURN_RESOLVED"
  | "RETURN_RECEIVED_AT_WAREHOUSE"
  | "DRIVER_RETURN_APPROACHING"
  | "ORDER_CONDITION_REPORTED"
  | "RETAILER_CREDIT_PROFILE_CHANGED"
  | "RETAILER_CREDIT_LIMIT_BREACHED"
  | "PRODUCT_HANDLING_UPDATED"
  | "PRE_ORDER_NOTIFIED"
  | "PRE_ORDER_NUDGE"
  | "PRE_ORDER_CONFIRMATION"
  | "PRE_ORDER_CONFIRMED"
  | "PRE_ORDER_EDITED"
  | "PRE_ORDER_CANCELLED"
  | "PRE_ORDER_AUTO_ACCEPTED"
  | "PRE_ORDER_DATE_PROPOSED"
  | "PRE_ORDER_DATE_ACCEPTED"
  | "PRE_ORDER_DATE_REJECTED"
  | "REPLENISHMENT_AUTO_APPROVED"
  | "REPLENISHMENT_INSIGHT_CREATED"
  | "DISPATCH_ZONE_OVERRIDE"
  | "planning.meio.recommendation.v1"
  | "planning.scenario.published.v1"
  | "planning.signal.ingest.v1"
  | "DEMAND_BASELINE_UPDATED"
  | "PLANNING_AGENT_BROADCAST"
  | "PLANNING_FORECAST_UPDATED"
  | "PLANNING_PROMO_SIMULATION_READY"
  | "PLANNING_CONFIDENCE_DOWNGRADED";

