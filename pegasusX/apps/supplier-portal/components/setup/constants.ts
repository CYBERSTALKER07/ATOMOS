export const SETUP_STEPS = [
  {
    id: "account",
    label: "Account",
    description: "Identity & verification",
    href: "/auth/register",
  },
  {
    id: "business",
    label: "Business details",
    description: "Tax ID & headquarters",
    href: "/setup/business",
  },
  {
    id: "billing",
    label: "Billing & gateways",
    description: "Payouts & checkout",
    href: "/setup/billing",
  },
] as const;

export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];

export function setupStepIndex(pathname: string): number {
  if (pathname.startsWith("/setup/billing")) return 2;
  if (pathname.startsWith("/setup/business")) return 1;
  return 0;
}
