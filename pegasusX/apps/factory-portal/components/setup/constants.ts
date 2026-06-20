export const SETUP_STEPS = [
  {
    id: "account",
    label: "Account",
    description: "Identity & verification",
    href: "/auth/register",
  },
  {
    id: "factory",
    label: "Factory details",
    description: "Facility & location",
    href: "/setup/factory",
  },
] as const;

export type SetupStepId = (typeof SETUP_STEPS)[number]["id"];

export function setupStepIndex(pathname: string): number {
  if (pathname.startsWith("/setup/factory")) return 1;
  return 0;
}
