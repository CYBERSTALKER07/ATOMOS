/**
 * GS-R — bind GET /v1/auth/session pack fields on every client.
 * CheckoutReadsThis stays false; this is display bind, not a checkout flip.
 */

export type MarketPack = {
  code: string;
  name: string;
  status: string;
  home_cell: string;
  timezone: string;
  currency_code: string;
  currency_decimal_places?: number;
  fiscal_adapter: string;
  psp_adapters?: string[];
  maps_adapter?: string;
  map_center_lat?: number;
  map_center_lng?: number;
  distance_unit?: string;
  checkout_reads_this?: boolean;
};

export type PackMapCenter = { lat: number; lng: number };

export type AuthSession = {
  subject?: string;
  role?: string;
  supplier_id?: string;
  market_code: string;
  home_cell: string;
  api_url?: string;
  ws_url?: string;
  source?: string;
  pack: MarketPack | null;
  checkout_reads_this?: boolean;
  pack_error?: string;
};

const PACK_CACHE_KEY = "pegasusx_market_pack_session";

export function fiscalReceiptLabel(adapter?: string | null): string {
  const a = String(adapter || "").trim().toUpperCase();
  if (a === "MY_SOLIQ") return "Soliq";
  if (a === "COMMERCIAL" || a === "PEGASUS" || a === "FAKE") return "commercial";
  if (a === "PEPPOL") return "PEPPOL";
  if (a === "PLANNED" || a === "") return "planned";
  return a;
}

/** Pack currency for display. Empty pack does not invent UZS. */
export function packCurrency(pack?: Pick<MarketPack, "currency_code"> | null, fallback = ""): string {
  const code = String(pack?.currency_code || fallback || "").trim().toUpperCase();
  return code;
}

export function displayPackCurrency(raw?: string | null, packCurrencyCode = ""): string {
  const fromEvent = String(raw || "").trim().toUpperCase();
  if (fromEvent) return fromEvent;
  return String(packCurrencyCode || "").trim().toUpperCase();
}

export function sessionPackCurrency(): string {
  return packCurrency(readCachedAuthSession()?.pack);
}

/** Stored/event currency, else session pack. Empty pack does not invent UZS. */
export function moneyCurrency(raw?: string | null): string {
  return displayPackCurrency(raw, sessionPackCurrency());
}

/** Shipped pack camera. Empty/planned pack does not invent Tashkent. */
export function packMapCenter(
  pack?: { map_center_lat?: number; map_center_lng?: number; status?: string } | null,
): PackMapCenter | null {
  if (pack && pack.status && pack.status !== "shipped") return null;
  const lat = Number(pack?.map_center_lat);
  const lng = Number(pack?.map_center_lng);
  if (!Number.isFinite(lat) || !Number.isFinite(lng) || (lat === 0 && lng === 0)) {
    return null;
  }
  return { lat, lng };
}

export function sessionMapCenter(): PackMapCenter | null {
  return packMapCenter(readCachedAuthSession()?.pack);
}

export function mapInitialViewState(
  pack?: Pick<MarketPack, "map_center_lat" | "map_center_lng" | "status"> | null,
  zoom = 12,
): { latitude: number; longitude: number; zoom: number } {
  const c = packMapCenter(pack);
  if (!c) return { latitude: 0, longitude: 0, zoom: 1 };
  return { latitude: c.lat, longitude: c.lng, zoom };
}

/** Selectable catalog codes. Planned rows stay visible but are not choosable. */
export function selectablePackPsps(
  catalog: Array<{ code?: string; selectable?: boolean }> | null | undefined,
): string[] {
  return (catalog || [])
    .filter((row) => row.selectable !== false)
    .map((row) => String(row.code || "").trim().toUpperCase())
    .filter(Boolean);
}

export function packAllowsPsp(pack: MarketPack | null | undefined, psp: string): boolean {
  const want = String(psp || "").trim().toUpperCase();
  if (!want) return false;
  const list = (pack?.psp_adapters || []).map((p) => String(p).toUpperCase());
  return list.includes(want);
}

export function formatPackMoney(minor: number, pack?: Pick<MarketPack, "currency_code" | "currency_decimal_places"> | null): string {
  const places = pack?.currency_decimal_places ?? 2;
  const denom = 10 ** places;
  const units = Number.isFinite(minor) ? minor / denom : 0;
  const formatted = units.toLocaleString("en-US").replace(/,/g, " ");
  const ccy = packCurrency(pack);
  return ccy ? `${formatted} ${ccy}` : formatted;
}

export function readCachedAuthSession(): AuthSession | null {
  if (typeof window === "undefined") return null;
  try {
    const raw = window.localStorage.getItem(PACK_CACHE_KEY);
    if (!raw) return null;
    return JSON.parse(raw) as AuthSession;
  } catch {
    return null;
  }
}

export function cacheAuthSession(session: AuthSession | null): void {
  if (typeof window === "undefined") return;
  try {
    if (!session) {
      window.localStorage.removeItem(PACK_CACHE_KEY);
      return;
    }
    window.localStorage.setItem(PACK_CACHE_KEY, JSON.stringify(session));
  } catch {
    // ignore quota
  }
}

export async function fetchAuthSession(
  baseUrl: string,
  token: string,
  fetchImpl: typeof fetch = fetch,
): Promise<AuthSession> {
  const base = String(baseUrl || "").replace(/\/$/, "");
  const res = await fetchImpl(`${base}/v1/auth/session`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error(`auth_session_${res.status}`);
  }
  const body = (await res.json()) as AuthSession;
  cacheAuthSession(body);
  return body;
}
