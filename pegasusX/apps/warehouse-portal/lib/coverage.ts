import type { CoverageMode, PSPListing, ServicePin } from "@pegasusx/types";
import { selectablePackPsps } from '@pegasusx/api-core';

export function coverageModeLabel(mode: string | undefined): string {
  switch (String(mode || "").toUpperCase()) {
    case "PINNED":
      return "Pinned";
    case "CITY_CELLS":
      return "City cells";
    case "COUNTRY_CLOSEST":
      return "Closest in country";
    default:
      return "Closest in country";
  }
}

export function normalizeCoverageMode(mode: string | undefined): CoverageMode {
  const raw = String(mode || "").toUpperCase();
  if (raw === "PINNED" || raw === "CITY_CELLS" || raw === "COUNTRY_CLOSEST") {
    return raw;
  }
  return "COUNTRY_CLOSEST";
}

export function pinKey(pin: Pick<ServicePin, "target_type" | "target_id">): string {
  return `${String(pin.target_type || "").toUpperCase()}:${String(pin.target_id || "").trim()}`;
}

export function gatewayLabel(code: string): string {
  switch (String(code || "").toUpperCase()) {
    case "GLOBAL_PAY":
      return "Global Pay";
    case "CASH":
      return "Cash on delivery";
    case "PAYME":
      return "Payme";
    case "CLICK":
      return "Click";
    case "STRIPE":
      return "Stripe";
    case "ADYEN":
      return "Adyen";
    default:
      return String(code || "").replace(/_/g, " ");
  }
}

export function catalogGatewayCodes(catalog: PSPListing[] | null | undefined): string[] {
  return selectablePackPsps(catalog);
}

export function catalogOmits(catalog: PSPListing[] | null | undefined, codes: string[]): boolean {
  const have = new Set((catalog || []).map((row) => String(row.code || "").toUpperCase()));
  return codes.every((code) => !have.has(code.toUpperCase()));
}
