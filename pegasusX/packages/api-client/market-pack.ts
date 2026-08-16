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
  distance_unit?: string;
  checkout_reads_this?: boolean;
};

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
