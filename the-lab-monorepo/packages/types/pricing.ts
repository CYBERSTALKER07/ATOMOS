/**
 * Per-retailer pricing override types.
 * Resolution hierarchy: RetailerPricingOverride > PricingTier > BasePriceUZS.
 */

/** A per-retailer price set by GLOBAL_ADMIN or NODE_ADMIN. */
export interface RetailerPricingOverride {
  retailer_id: string;
  supplier_id: string;
  product_id: string;
  price: number;
  currency: string;
  created_by: string;
  created_by_role: 'GLOBAL_ADMIN' | 'NODE_ADMIN';
  warehouse_id: string | null;
  last_updated_at: string;
}

/** Summary row returned by GET /v1/supplier/pricing/overrides. */
export interface PricingOverrideListRow {
  retailer_id: string;
  retailer_name: string;
  product_id: string;
  product_name: string;
  price: number;
  currency: string;
  created_by_role: string;
  warehouse_id?: string;
  last_updated_at: string;
}

/** Detail row returned by GET /v1/supplier/pricing/overrides/retailer/{id}. */
export interface RetailerPricingOverrideDetail {
  retailer_id: string;
  supplier_id: string;
  product_id: string;
  product_name: string;
  base_price_uzs: number;
  price: number;
  currency: string;
  created_by: string;
  created_by_role: string;
  warehouse_id?: string;
  last_updated_at: string;
}

/** Payload for POST /v1/supplier/pricing/overrides (single). */
export interface CreatePricingOverrideRequest {
  retailer_id: string;
  product_id: string;
  price: number;
  currency?: string;
}

/** Payload for POST /v1/supplier/pricing/overrides (bulk). */
export interface BulkPricingOverrideRequest {
  retailer_id: string;
  overrides: Array<{
    product_id: string;
    price: number;
    currency?: string;
  }>;
}
