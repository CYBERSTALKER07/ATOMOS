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
