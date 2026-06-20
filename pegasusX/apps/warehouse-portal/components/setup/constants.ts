export const SETUP_STEPS = [
  {
    id: "account",
    label: "Account",
    description: "Identity & verification",
    href: "/auth/register",
  },
  {
    id: "location",
    label: "Warehouse location",
    description: "Address & geolocation",
    href: "/setup/location",
  },
] as const;

export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];

export function setupStepIndex(pathname: string): number {
  if (pathname.startsWith("/setup/location")) return 1;
  return 0;
}
