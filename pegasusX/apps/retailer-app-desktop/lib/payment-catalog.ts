import { packCurrency, readCachedAuthSession, selectablePackPsps } from "@pegasusx/api-client";
import type { PSPListing } from "@pegasusx/types";

const FOREIGN_RAILS = new Set(["STRIPE", "ADYEN", "AIRWALLEX"]);

export function catalogGatewayCodes(catalog: Array<{ code?: string }> | null | undefined): string[] {
  return (catalog || [])
    .map((row) => String(row.code || "").trim().toUpperCase())
    .filter(Boolean);
}

export function catalogOmits(
  catalog: Array<{ code?: string }> | null | undefined,
  codes: string[],
): boolean {
  const seen = new Set(catalogGatewayCodes(catalog));
  return codes.every((code) => !seen.has(code.trim().toUpperCase()));
}

/** Card rails the retailer may tap. Empty catalog never invents Adyen/Stripe. */
export function filterRetailerCardGateways(
  incoming: string[] | null | undefined,
  catalogCodes: string[] | null | undefined,
): string[] {
  const allowed = (catalogCodes || [])
    .map((code) => code.trim().toUpperCase())
    .filter((code) => code && code !== "CASH");
  const raw = (incoming || [])
    .map((code) => code.trim().toUpperCase())
    .filter(Boolean);
  if (allowed.length > 0) {
    return raw.length === 0 ? allowed : raw.filter((code) => allowed.includes(code));
  }
  return raw.filter((code) => !FOREIGN_RAILS.has(code));
}

export function checkoutGatewayForMethod(
  method: string,
  catalogCodes: string[] | null | undefined,
): string {
  const map: Record<string, string> = {
    cash: "CASH",
    global_pay: "GLOBAL_PAY",
  };
  const allowed = (catalogCodes || []).map((code) => code.trim().toUpperCase()).filter(Boolean);
  const mapped = map[method] || "";
  if (mapped && (allowed.length === 0 || allowed.includes(mapped))) {
    return mapped;
  }
  if (allowed.includes("CASH")) return "CASH";
  return allowed[0] || "CASH";
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

export function retailerCatalogGateways(catalog: PSPListing[] | null | undefined): string[] {
  return selectablePackPsps(catalog).filter((code) => !FOREIGN_RAILS.has(code));
}

export { packCurrency };
