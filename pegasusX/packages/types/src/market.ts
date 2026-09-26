// ── Market Pack & Geo Types ───────────────────────────────────────────────────
export interface MarketPack {
  pack_id: string;
  country_code: string;
  currency_code: string;
  currency_decimal_places?: number;
  market_code?: string;
  display_name?: string;
  default_locale?: string;
  status: string;
  map_center_lat?: number;
  map_center_lng?: number;
  psp_adapters?: string[];
  fiscal_provider?: string;
  enabled?: boolean;
}

export interface PackMapCenter {
  lat: number;
  lng: number;
}

