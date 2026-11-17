import type { ApiClient } from "@pegasusx/api-core";
import { ApiError } from "@pegasusx/api-core";
import { createWarehouseApi } from "@/lib/api";
import { formatMinorUnits, parseMajorToMinorBigInt } from "@/lib/money";

function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && String(v).trim() !== "") {
      params.set(k, String(v).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

export interface WarehouseRequestTransport {
  request: ApiClient["request"];
}

export function normalizeMinorBigInt(minor: number | bigint | null | undefined): bigint {
  if (typeof minor === "bigint") return minor;
  if (typeof minor !== "number" || !Number.isFinite(minor)) return BigInt(0);
  return BigInt(Math.trunc(minor));
}

export function formatWarehouseMinor(
  minor: number | bigint | null | undefined,
  currency = "UZS",
  options?: { signDisplay?: "auto" | "always" },
): string {
  const safe = normalizeMinorBigInt(minor);
  const abs = safe < BigInt(0) ? -safe : safe;
  const minFrac: 0 | 2 = abs % BigInt(100) === BigInt(0) ? 0 : 2;
  return formatMinorUnits(safe, {
    currency: (currency || "UZS").trim().toUpperCase(),
    minimumFractionDigits: minFrac,
    maximumFractionDigits: 2,
    signDisplay: options?.signDisplay ?? "auto",
  });
}

export function parseMajorToSafeMinor(major: string, allowNegative = false): number | null {
  const parsed = parseMajorToMinorBigInt(major, { allowNegative });
  if (parsed === null) return null;
  if (parsed > BigInt(Number.MAX_SAFE_INTEGER) || parsed < BigInt(Number.MIN_SAFE_INTEGER)) {
    return null;
  }
  return Number(parsed);
}

export function extractValuationError(err: unknown, fallback: string): string {
  if (err instanceof ApiError && err.payload && typeof err.payload === "object") {
    const payload = err.payload as Record<string, unknown>;
    const parts = [
      typeof payload.code === "string" ? payload.code.trim() : "",
      typeof payload.error === "string" ? payload.error.trim() : "",
      typeof payload.message === "string" ? payload.message.trim() : "",
      typeof payload.detail === "string" ? payload.detail.trim() : "",
    ].filter((v): v is string => typeof v === "string" && v.length > 0);
    const unique = Array.from(new Set(parts));
    if (unique.length > 0) {
      return unique.join(" · ");
    }
  }
  return err instanceof Error && err.message.trim() ? err.message : fallback;
}

export interface SkuValuationSummary {
  entity_id: string;
  warehouse_id: string;
  product_id: string;
  currency: string;
  total_qty_remaining: number;
  total_value_minor: number;
  moving_avg_unit_cost_minor: number;
  layer_count: number;
}

export interface InventoryCostLayer {
  entity_id: string;
  warehouse_id: string;
  product_id: string;
  layer_id: string;
  lot_id?: string;
  receipt_ref_type?: string;
  receipt_ref_id?: string;
  qty_remaining: number;
  original_qty: number;
  unit_cost_minor: number;
  total_value_minor: number;
  currency: string;
  received_at: string;
  expiry_date?: string;
  near_expiry?: boolean;
  days_to_expiry?: number;
}

export interface InventoryValuationResponse {
  entity_id: string;
  warehouse_id?: string;
  currency: string;
  total_qty_remaining: number;
  total_value_minor: number;
  expiry_risk_days: number;
  skus: SkuValuationSummary[];
  layers: InventoryCostLayer[];
}

export type InventoryMovementType =
  | "RECEIPT"
  | "ISSUE_COGS"
  | "ADJUSTMENT_GAIN"
  | "ADJUSTMENT_SHRINK"
  | "RETURN"
  | string;

export interface InventoryMovement {
  entity_id: string;
  movement_id: string;
  movement_type: InventoryMovementType;
  reference_type?: string;
  reference_id?: string;
  warehouse_id: string;
  product_id: string;
  lot_id?: string;
  layer_id?: string;
  qty_delta: number;
  unit_cost_minor: number;
  total_cost_minor: number;
  currency: string;
  gl_entry_id?: string;
  idempotency_key?: string;
  created_at: string;
}

export interface InventoryMovementsResponse {
  entity_id: string;
  movements: InventoryMovement[];
  count: number;
}

export interface RecordCostLayerReceiptRequest {
  entity_id?: string;
  warehouse_id: string;
  product_id: string;
  lot_id?: string;
  receipt_ref_type?: string;
  receipt_ref_id: string;
  quantity: number;
  unit_cost_minor: number;
  vat_minor?: number;
  credit_account?: string;
  vendor_id?: string;
  currency?: string;
  expiry_date?: string;
  idempotency_key?: string;
}

export interface RecordCostLayerReceiptResponse {
  layer: InventoryCostLayer;
  movement: InventoryMovement;
  gl_entry_id?: string;
  total_cost_minor: number;
  vat_minor: number;
  expiry_risk_emitted?: boolean;
  already_processed?: boolean;
}

export function createWarehouseValuationApi(client?: WarehouseRequestTransport) {
  const transport = client ?? createWarehouseApi();

  return {
    getValuation: (
      query: {
        entity_id?: string;
        warehouse_id?: string;
        product_id?: string;
        expiry_risk_days?: number;
      } = {},
    ): Promise<InventoryValuationResponse> =>
      transport.request<InventoryValuationResponse>(
        withQuery("/v1/inventory/valuation", query),
        "GET",
      ),

    listMovements: (
      query: {
        entity_id?: string;
        warehouse_id?: string;
        product_id?: string;
        movement_type?: string;
        limit?: number;
      } = {},
    ): Promise<InventoryMovementsResponse> =>
      transport.request<InventoryMovementsResponse>(
        withQuery("/v1/inventory/movements", query),
        "GET",
      ),

    recordCostLayerReceipt: (
      body: RecordCostLayerReceiptRequest,
      idempotencyKey?: string,
    ): Promise<RecordCostLayerReceiptResponse> => {
      const key =
        idempotencyKey?.trim() ||
        body.idempotency_key?.trim() ||
        (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? `cost-layer-rcpt-${crypto.randomUUID()}`
          : `cost-layer-rcpt-${Date.now()}`);
      return transport.request<RecordCostLayerReceiptResponse>(
        "/v1/inventory/cost-layers/receipt",
        "POST",
        {
          body: { ...body, idempotency_key: key },
          idempotencyKey: key,
        },
      );
    },
  };
}

export type WarehouseValuationApi = ReturnType<typeof createWarehouseValuationApi>;
