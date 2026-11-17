export const SETUP_STEPS = [
  {
    id: "account",
    label: "Account",
    description: "Identity & verification",
    href: "/auth/register",
  },
  {
    id: "tax",
    label: "Tax details",
    description: "Business tax ID",
    href: "/setup/tax",
  },
  {
    id: "address",
    label: "Delivery address",
    description: "Billing & shipping",
    href: "/setup/address",
  },
] as const;

export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];

export function setupStepIndex(pathname: string): number {
  if (pathname.startsWith("/setup/address")) return 2;
  if (pathname.startsWith("/setup/tax")) return 1;
  return 0;
}

export const SETUP_TAX_KEY = "pegasus_retailer_setup_tax";

export type RetailerTaxSetupMode = "LEGAL_VERIFIED" | "INFORMAL_PHONE";

export interface TaxIdentitySetupDraft {
  mode: RetailerTaxSetupMode;
  taxId: string;
  legalName?: string;
  persisted?: boolean;
}

export function readTaxSetupDraft(): TaxIdentitySetupDraft | null {
  if (typeof sessionStorage === "undefined") return null;
  const raw = sessionStorage.getItem(SETUP_TAX_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Partial<TaxIdentitySetupDraft>;
    if (parsed && (parsed.mode === "LEGAL_VERIFIED" || parsed.mode === "INFORMAL_PHONE")) {
      return {
        mode: parsed.mode,
        taxId: typeof parsed.taxId === "string" ? parsed.taxId : "",
        legalName: typeof parsed.legalName === "string" ? parsed.legalName : undefined,
        persisted: Boolean(parsed.persisted),
      };
    }
  } catch {
    // Backward compatibility with legacy plain-string tax IDs.
  }
  const trimmed = raw.trim();
  if (!trimmed) return null;
  return { mode: "LEGAL_VERIFIED", taxId: trimmed, persisted: false };
}

export function writeTaxSetupDraft(draft: TaxIdentitySetupDraft): void {
  if (typeof sessionStorage === "undefined") return;
  sessionStorage.setItem(SETUP_TAX_KEY, JSON.stringify(draft));
}

export function clearTaxSetupDraft(): void {
  if (typeof sessionStorage === "undefined") return;
  sessionStorage.removeItem(SETUP_TAX_KEY);
}
