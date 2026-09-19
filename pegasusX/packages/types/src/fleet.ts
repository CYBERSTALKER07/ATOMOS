// ── Fleet Telemetry & Capacity Command Center ─────────────────────────────────
export interface PartnerFilterMetric {
  id: string;
  name: string;
  count: number;
}

export interface VehicleShipmentCard {
  id: string;
  code: string;
  status: "WAITING" | "ON_ROUTE" | "COMPLETED" | "DELAYED";
  vehicle_type: "VAN" | "SEMI_TRUCK" | "BOX_TRUCK" | "FLATBED";
  eta_seconds: number;
  distance_miles_left: number;
  stops_count: number;
  stops_summary: string[];
  driver_name?: string;
  driver_phone?: string;
  partner_id?: string;
  partner_name?: string;
}

export interface VehicleCapacityMetrics {
  vehicle_id: string;
  code: string;
  capacity_percentage: number;
  current_volume_cubic_meters?: number;
  max_volume_cubic_meters?: number;
  current_weight_kg?: number;
  max_weight_kg?: number;
}

export interface RouteTelemetryPoint {
  latitude: number;
  longitude: number;
  timestamp: string;
  speed_mph?: number;
  heading_degrees?: number;
}

export interface RouteTelemetryDetails {
  route_id: string;
  vehicle_code: string;
  status: string;
  eta_seconds: number;
  distance_miles_left: number;
  start_point: { name: string; latitude: number; longitude: number };
  destination_point: { name: string; latitude: number; longitude: number };
  current_location: RouteTelemetryPoint;
  waypoints: Array<{ name: string; latitude: number; longitude: number; status: "VISITED" | "PENDING" }>;
}

export interface PoDPhotoReport {
  id: string;
  title: string;
  location_name: string;
  timestamp: string;
  photo_url: string;
  step_number: number;
}

export interface FleetDispatchOverview {
  total_count: number;
  active_count: number;
  inactive_count: number;
  partner_filters: PartnerFilterMetric[];
  shipments: VehicleShipmentCard[];
}

export interface FleetDispatchFilter {
  partner_id?: string;
  status_filter?: "ALL" | "ACTIVE" | "INACTIVE";
  search_query?: string;
}

export * from "../forecast-confidence";

export type SupplierSettingsResponse = any;
export type RecommendReassignRequest = any;
export type RecommendReassignResponse = { candidates?: ReassignmentCandidate[] };
export type ApplyReassignRequest = any;
export type StatusResponse = any;
export type ReassignmentCandidate = any;

/** Partner Integration Layer (Gate 3 / §8.9) */
export type PartnerTenantType = "RETAILER" | "SUPPLIER";

export interface PartnerIssuedKey {
  key_id: string;
  tenant_type: PartnerTenantType;
  tenant_id: string;
  scopes: string[];
  key_prefix: string;
  /** Shown once at issuance — store securely. */
  secret: string;
  expires_at?: string;
}

export interface PartnerApiKeyMeta {
  key_id: string;
  tenant_type: PartnerTenantType;
  tenant_id: string;
  key_prefix: string;
  scopes: string[];
  status: string;
}

export interface PartnerWebhookCreated {
  subscription_id: string;
  url: string;
  event_types: string[];
  signing_secret: string;
}

export interface PartnerAvailabilityRow {
  product_id: string;
  supplier_id: string;
  available_stock?: number;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  show_stock_counts?: boolean;
}

export interface PartnerWebhookSubscription {
  subscription_id: string;
  url: string;
  event_types: string[];
  is_active: boolean;
  secret_prefix?: string;
  created_at?: string;
}

export interface PartnerDeadLetterAttempt {
  attempt_id: string;
  subscription_id: string;
  event_id: string;
  event_type: string;
  status: string;
  last_error?: string;
  attempt_count?: number;
}

export type PartnerExportResource = "orders" | "invoices" | "inventory" | "ledger" | "journals";
export type PartnerExportFormat = "csv" | "json" | "xml";

export interface PartnerExportJob {
  job_id: string;
  tenant_type?: string;
  tenant_id?: string;
  resource: string;
  format: string;
  status: string;
  row_count?: number;
  sftp_status?: string;
  error?: string;
  download_url?: string;
  from?: string;
  to?: string;
  created_at?: string;
  finished_at?: string;
}

export interface PartnerSftpConfig {
  configured: boolean;
  host?: string;
  port?: number;
  username?: string;
  secret_ref?: string;
  remote_dir?: string;
  is_active?: boolean;
  inbound_dir?: string;
  outbound_dir?: string;
  archive_dir?: string;
  edi_enabled?: boolean;
}

/** Per-tenant AS2 station (§8.9) — cert PEMs via secret refs only. */
export interface PartnerAs2Config {
  configured: boolean;
  as2_enabled?: boolean;
  our_as2_id?: string;
  partner_as2_id?: string;
  partner_url?: string;
  our_cert_secret_ref?: string;
  our_key_secret_ref?: string;
  partner_cert_secret_ref?: string;
  sign_required?: boolean;
  encrypt_required?: boolean;
}

/** Per-tenant 1C chart of accounts for journals exports. */
export interface PartnerCoaMap {
  account_ar: string;
  account_revenue: string;
  account_bank_cash: string;
  using_defaults?: boolean;
  updated_at?: string;
  updated_by?: string;
}

export interface PartnerEdiDocument {
  document_id: string;
  tenant_type?: string;
  tenant_id?: string;
  direction: string;
  doc_type: string;
  external_doc_id: string;
  order_id?: string;
  status: string;
  remote_name?: string;
  object_path?: string;
  error?: string;
  created_at?: string;
  finished_at?: string;
}


