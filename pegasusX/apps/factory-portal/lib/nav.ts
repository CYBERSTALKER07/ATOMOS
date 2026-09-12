export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };

/** GS-UN: Home · Bay · Payload · Transfers. */
export const NAV: NavSection[] = [
  {
    items: [
      { href: "/", icon: "dashboard", labelKey: "portal.nav.dashboard" },
      { href: "/loading-bay", icon: "loadingBay", labelKey: "portal.nav.loading_bay" },
      { href: "/payload", icon: "loadingBay", labelKey: "portal.nav.payload" },
      { href: "/transfers", icon: "transfers", labelKey: "portal.nav.transfers" },
    ],
  },
  {
    labelKey: "portal.nav.section.operations",
    items: [
      { href: "/fleet", icon: "fleet", labelKey: "portal.nav.fleet" },
      { href: "/staff", icon: "staff", labelKey: "portal.nav.staff" },
      { href: "/settings/location", icon: "loadingBay", labelKey: "portal.nav.location" },
      { href: "/supply-requests", icon: "transfers", labelKey: "portal.nav.supply_requests" },
      { href: "/payload-override", icon: "loadingBay", labelKey: "portal.nav.payload_override" },
      { href: "/manifests", icon: "manifests", labelKey: "portal.nav.manifests" },
      { href: "/manifest-exceptions", icon: "insights", labelKey: "portal.nav.gate_exceptions" },
    ],
  },
  {
    labelKey: "portal.nav.section.insights",
    items: [
      { href: "/insights", icon: "insights", labelKey: "portal.nav.insights" },
      { href: "/analytics", icon: "analytics", labelKey: "portal.nav.analytics" },
    ],
  },
];

export const ALL_NAV_ITEMS = NAV.flatMap((section) => section.items);

export function primaryNavHrefs(nav: NavSection[] = NAV): string[] {
  return nav[0]?.items.map((item) => item.href) ?? [];
}

export function allNavHrefs(nav: NavSection[] = NAV): string[] {
  return nav.flatMap((section) => section.items.map((item) => item.href));
}

export function navSectionIsActive(
  section: NavSection,
  pathname: string,
  isActive: (pathname: string, href: string) => boolean,
): boolean {
  return section.items.some((item) => isActive(pathname, item.href));
}
