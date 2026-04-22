/**
 * @file packages/types/order.ts
 * @description Canonical shared types for the Lab Industries distribution ecosystem.
 *
 * GOLDEN RULE: Every frontend (Next.js Admin, Retailer App, Driver App) and every
 * backend handler MUST import from this package. No local type redeclarations.
 *
 * Sync with Spanner DDL: apps/backend-go/schema/spanner.ddl
 * Sync with Go structs:   apps/backend-go/order/service.go
 */

// ─── Order State Machine ────────────────────────────────────────────────────
// Maps 1:1 to the Orders.State column in Spanner.
// GOLDEN PATH: PENDING → LOADED → IN_TRANSIT → ARRIVED → COMPLETED
export type OrderState =
  | 'PENDING'
  | 'PENDING_REVIEW'
  | 'LOADED'
  | 'IN_TRANSIT'
  | 'ARRIVING'
  | 'ARRIVED'
  | 'AWAITING_PAYMENT'
  | 'PENDING_CASH_COLLECTION'
  | 'COMPLETED'
  | 'CANCELLED'
  | 'SCHEDULED'
  | 'QUARANTINE'
  | 'DELIVERED_ON_CREDIT';

// ─── Payment Gateways ───────────────────────────────────────────────────────
export type PaymentGateway = 'CLICK' | 'PAYME' | 'GLOBAL_PAY' | 'UZCARD' | 'CASH';

// ─── Payment Status ─────────────────────────────────────────────────────────
// Maps 1:1 to the Orders.PaymentStatus column in Spanner.
export type PaymentStatus =
  | 'PENDING'
  | 'PENDING_CASH_COLLECTION'
  | 'AWAITING_GATEWAY_WEBHOOK'
  | 'PAID'
  | 'FAILED';

// ─── Payment Session Status ─────────────────────────────────────────────────
// Maps 1:1 to PaymentSessions.Status in Spanner.
export type PaymentSessionStatus =
  | 'CREATED'
  | 'PENDING'
  | 'SETTLED'
  | 'FAILED'
  | 'EXPIRED'
  | 'CANCELLED';

// ─── Payment Attempt Status ─────────────────────────────────────────────────
// Maps 1:1 to PaymentAttempts.Status in Spanner.
export type PaymentAttemptStatus =
  | 'INITIATED'
  | 'REDIRECTED'
  | 'PROCESSING'
  | 'SUCCESS'
  | 'FAILED'
  | 'CANCELLED'
  | 'TIMED_OUT';

// ─── Payment Session ────────────────────────────────────────────────────────
// Canonical session record returned by POST /v1/order/card-checkout
// and GET /v1/payment/sessions/:sessionId.
export interface PaymentSession {
  session_id: string;
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  gateway: PaymentGateway;
  locked_amount: number;
  currency: string;
  status: PaymentSessionStatus;
  current_attempt_no: number;
  invoice_id: string | null;
  redirect_url: string | null;
  expires_at: string | null;       // ISO 8601
  last_error_code: string | null;
  last_error_message: string | null;
  created_at: string;              // ISO 8601
  updated_at: string | null;       // ISO 8601
  settled_at: string | null;       // ISO 8601
}

// ─── Payment Attempt ────────────────────────────────────────────────────────
// Individual attempt record within a session.
export interface PaymentAttempt {
  attempt_id: string;
  session_id: string;
  attempt_no: number;
  gateway: PaymentGateway;
  provider_transaction_id: string | null;
  status: PaymentAttemptStatus;
  failure_code: string | null;
  failure_message: string | null;
  started_at: string;              // ISO 8601
  finished_at: string | null;      // ISO 8601
}

// ─── Payment Session Detail (with attempts) ─────────────────────────────────
// Returned by GET /v1/payment/sessions/:sessionId?include=attempts
export interface PaymentSessionDetail extends PaymentSession {
  attempts: PaymentAttempt[];
}

// ─── WebSocket Payment Events ───────────────────────────────────────────────
export interface PaymentRequiredEvent {
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

export interface PaymentSettledEvent {
  type: 'PAYMENT_SETTLED';
  order_id: string;
  session_id: string;
  amount: number;
  currency: string;
  gateway: PaymentGateway;
  message: string;
}

export interface PaymentFailedEvent {
  type: 'PAYMENT_FAILED';
  order_id: string;
  session_id: string;
  error_code: string;
  error_message: string;
  gateway: PaymentGateway;
  retryable: boolean;
}

// ─── Line Item ──────────────────────────────────────────────────────────────
// Maps to OrderLineItems table. Status reflects physical delivery disposition.
export type LineItemStatus = 'PENDING' | 'DELIVERED' | 'REJECTED_DAMAGED';

export interface OrderLineItem {
  line_item_id: string;   // UUID — PK on OrderLineItems
  order_id: string;       // FK → Orders.OrderId
  sku_id: string;         // e.g. "SKU-COKE-1L-PALLET"
  quantity: number;       // INT64 in Spanner
  unit_price: number; // INT64 in Spanner (minor units)
  currency: string;
  status: LineItemStatus;
}

// ─── Order ──────────────────────────────────────────────────────────────────
// Core aggregate. items[] is populated only when fetched via
// GET /v1/orders/{id}/items — not embedded in the base list response.
export interface Order {
  order_id: string;         // Opaque UUID v4
  retailer_id: string;      // e.g. "SHOP-TASH-01"
  driver_id: string | null; // null until dispatched
  payment_gateway: PaymentGateway;
  payment_status: PaymentStatus;
  total: number;        // INT64 — recalculated on amend
  currency: string;
  state: OrderState;
  shop_location: string;    // WKT: "POINT(lon lat)" — Spanner GEOGRAPHY
  delivery_token?: string;  // Present ONLY server-side; never sent to client
  items?: OrderLineItem[];  // Lazy-loaded via Idx_OrderItems_ByOrder
}

// ─── Active Mission (Fleet Radar) ───────────────────────────────────────────
// Returned by GET /v1/fleet/active — lightweight projection for the map view.
export interface ActiveMission {
  order_id: string;
  state: OrderState;
  target_lat: number;
  target_lng: number;
  amount: number;
  currency: string;
  gateway: PaymentGateway;
}

// ─── Route Manifest (Offline Crypto Protocol) ───────────────────────────────
// Downloaded by the driver at depot. Hashes only — raw tokens stay server-side.
export interface RouteManifest {
  driver_id: string;
  date: string;           // ISO date: "2026-03-12"
  expires_at: number;     // Unix epoch — valid for one calendar day
  hashes: Record<string, string>; // OrderId → SHA256(DeliveryToken)
}

// ─── Amend Request (Partial Delivery) ───────────────────────────────────────
// Body for POST /v1/order/amend — mutates OrderLineItems status.
export interface AmendLineItem {
  line_item_id: string;
  status: Exclude<LineItemStatus, 'PENDING'>; // Must settle: DELIVERED | REJECTED_DAMAGED
  delivered_quantity?: number; // If partial — overrides original quantity
}

export interface AmendOrderRequest {
  order_id: string;
  driver_id: string;
  line_items: AmendLineItem[];
}

export interface AmendOrderResponse {
  status: 'AMENDED';
  order_id: string;
  new_total: number; // Recalculated after rejections
  currency: string;
}
