import sys

with open('pegasusX/packages/types/index.ts', 'a') as f:
    f.write("""
export interface TaxRegimeVersion {
  id: string;
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rates_bps: number[];
  simplified_rules?: any;
  created_at: string;
  created_by: string;
  updated_at: string;
}

export interface CreateRegimeRequest {
  country_code: string;
  effective_from: string;
  effective_to?: string;
  currency: string;
  vat_rates_bps: number[];
  simplified_rules?: any;
}

export interface OrderLineFiscalSnapshot {
  order_id: string;
  order_line_id: string;
  regime_id: string;
  vat_rate_bps: number;
  net_minor: number;
  vat_minor: number;
  gross_minor: number;
  snapshot_at: string;
  created_at: string;
}
""")
