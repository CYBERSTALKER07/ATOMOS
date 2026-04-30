/**
 * @file packages/types/entities.ts
 * @description Canonical entity types for the Pegasus ecosystem.
 * Sync with Spanner DDL: apps/backend-go/schema/spanner.ddl
 * Sync with Go structs across: supplier/, order/, fleet/, auth/
 */

// ─── Supplier ───────────────────────────────────────────────────────────────
// Maps to Suppliers table. The "ADMIN" in the JWT is always a SUPPLIER.
export interface Supplier {
  supplier_id: string;
  company_name: string;
  contact_person: string;
  phone: string;
  email: string;
  warehouse_lat: number;
  warehouse_lng: number;
  warehouse_address: string;
  tax_id: string;
  company_registration_number: string;
  billing_address: string;
  bank_name: string;
  bank_account_number: string;
  payment_gateway_preference: string;
  product_categories: string[];      // JSON-encoded in Spanner
  onboarding_status: SupplierOnboardingStatus;
  manual_off_shift: boolean;
  operating_schedule: DayWindow[] | null;
  created_at: string;                // ISO 8601
  updated_at: string | null;
}

export type SupplierOnboardingStatus = 'PENDING' | 'CONFIGURED' | 'ACTIVE' | 'SUSPENDED';

export interface DayWindow {
  day: string;       // "monday" | "tuesday" | ... | "sunday"
  start: string;     // "09:00"
  end: string;       // "18:00"
  enabled: boolean;
}

// ─── Retailer ───────────────────────────────────────────────────────────────
// Maps to Retailers table.
export interface Retailer {
  retailer_id: string;
  name: string;
  shop_name: string;
  phone: string;
  tax_identification_number: string;
  shop_lat: number;
  shop_lng: number;
  shop_address: string;
  status: RetailerStatus;
  telegram_chat_id: string | null;
  created_at: string;
  updated_at: string | null;
}

export type RetailerStatus = 'PENDING_KYC' | 'ACTIVE' | 'SUSPENDED' | 'BLOCKED';

// ─── Driver ─────────────────────────────────────────────────────────────────
// Maps to Drivers table.
export interface Driver {
  driver_id: string;
  name: string;
  phone: string;
  supplier_id: string;
  vehicle_id: string | null;
  is_active: boolean;
  truck_status: DriverTruckStatus;
  last_known_lat: number | null;
  last_known_lng: number | null;
  last_ping_at: string | null;
  created_at: string;
}

export type DriverTruckStatus =
  | 'AVAILABLE'
  | 'LOADING'
  | 'DISPATCHED'
  | 'IN_TRANSIT'
  | 'RETURNING'
  | 'OFFLINE';

// ─── Vehicle ────────────────────────────────────────────────────────────────
// Maps to Vehicles table.
export interface Vehicle {
  vehicle_id: string;
  supplier_id: string;
  plate_number: string;
  model: string;
  capacity_kg: number;
  capacity_volume_m3: number;
  status: VehicleStatus;
  created_at: string;
}

export type VehicleStatus = 'ACTIVE' | 'MAINTENANCE' | 'DECOMMISSIONED';

// ─── Route ──────────────────────────────────────────────────────────────────
// Maps to Routes table.
export interface Route {
  route_id: string;
  driver_id: string;
  vehicle_id: string;
  supplier_id: string;
  status: RouteStatus;
  planned_departure: string | null;
  actual_departure: string | null;
  completed_at: string | null;
  created_at: string;
}

export type RouteStatus =
  | 'PLANNED'
  | 'ACCEPTED'
  | 'REJECTED'
  | 'DISPATCHED'
  | 'IN_PROGRESS'
  | 'COMPLETED'
  | 'CANCELLED';

// ─── Category ───────────────────────────────────────────────────────────────
// Maps to SupplierCategories table (platform-level categories).
export interface Category {
  category_id: string;
  name: string;
  description: string;
  icon_url: string | null;
  sort_order: number;
}

// ─── Supplier Product (Catalog) ─────────────────────────────────────────────
// Maps to SupplierProducts table.
export interface SupplierProduct {
  sku_id: string;
  supplier_id: string;
  category_id: string;
  name: string;
  description: string;
  unit_price: number;
  currency: string;
  unit_label: string;
  image_url: string | null;
  stock_quantity: number;
  is_active: boolean;
  created_at: string;
  updated_at: string | null;
}

// ─── Cart Item ──────────────────────────────────────────────────────────────
// Maps to RetailerCarts table (server-side cart persistence).
export interface CartItem {
  cart_id: string;
  retailer_id: string;
  sku_id: string;
  supplier_id: string;
  quantity: number;
  unit_price: number;
  currency: string;
  added_at: string;
}

// ─── Master Invoice ─────────────────────────────────────────────────────────
// Maps to MasterInvoices table.
export interface MasterInvoice {
  invoice_id: string;
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  total: number;
  currency: string;
  state: InvoiceState;
  payment_gateway: string;
  global_pay_order_id: string | null;
  created_at: string;
  settled_at: string | null;
}

export type InvoiceState = 'PENDING' | 'COLLECTED' | 'SETTLED' | 'CANCELLED' | 'REFUNDED';

// ─── Ledger Entry ───────────────────────────────────────────────────────────
// Maps to LedgerEntries table.
export interface LedgerEntry {
  entry_id: string;
  account_id: string;     // "ACC-SUPPLIER-{id}" or "ACC-PLATFORM"
  type: LedgerEntryType;
  amount: number;
  currency: string;
  reference_id: string;   // OrderId or InvoiceId
  description: string;
  created_at: string;
}

export type LedgerEntryType = 'CREDIT' | 'DEBIT' | 'ADJUSTMENT';

// ─── Notification ───────────────────────────────────────────────────────────
// Maps to Notifications table.
export interface Notification {
  notification_id: string;
  recipient_id: string;
  type: string;           // "BROADCAST" | "ORDER_UPDATE" | "PAYMENT" | etc.
  title: string;
  body: string;
  payload: string | null; // Optional JSON payload
  channel: string;        // "FCM" | "TELEGRAM" | "WS"
  read_at: string | null;
  created_at: string;
}

// ─── Device Token ───────────────────────────────────────────────────────────
// Maps to DeviceTokens table.
export interface DeviceToken {
  token_id: string;
  user_id: string;
  role: string;
  platform: 'ANDROID' | 'IOS' | 'WEB';
  token: string;
  created_at: string;
}

// ─── Audit Log ──────────────────────────────────────────────────────────────
// Maps to AuditLog table.
export interface AuditLogEntry {
  log_id: string;
  actor_id: string;
  actor_role: string;
  action: string;
  resource_type: string;
  resource_id: string;
  metadata: Record<string, unknown> | null;
  created_at: string;
}

// ─── Supplier Return ────────────────────────────────────────────────────────
// Maps to SupplierReturns table.
export interface SupplierReturn {
  return_id: string;
  order_id: string;
  sku_id: string;
  rejected_qty: number;
  reason: ReturnReason;
  resolved: boolean;
  created_at: string;
}

export type ReturnReason = 'DAMAGED' | 'MISSING' | 'WRONG_ITEM' | 'OTHER';

// ─── Payloader (Warehouse Staff) ────────────────────────────────────────────
// Maps to Payloaders table.
export interface Payloader {
  payloader_id: string;
  supplier_id: string;
  name: string;
  phone: string;
  pin: string;
  is_active: boolean;
  created_at: string;
}
