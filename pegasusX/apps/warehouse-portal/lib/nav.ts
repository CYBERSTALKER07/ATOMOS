export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };

/** GS-UN: Home · Dispatch · Floor · Plan. Dispatch settings stay overflow. */
export const NAV: NavSection[] = [
  {
    items: [
      { href: "/", icon: "dashboard", labelKey: "portal.nav.dashboard" },
      { href: "/dispatch", icon: "dispatch", labelKey: "portal.nav.dispatch" },
      { href: "/inventory", icon: "inventory", labelKey: "portal.nav.floor" },
      { href: "/demand-forecast", icon: "forecast", labelKey: "portal.nav.planning" },
    ],
  },
  {
    labelKey: "portal.nav.section.operations",
    items: [
      { href: "/control-tower", icon: "global", labelKey: "portal.nav.control_tower" },
      { href: "/orders", icon: "orders", labelKey: "portal.nav.orders" },
      { href: "/preorders", icon: "orders", labelKey: "portal.nav.preorders" },
      { href: "/tomorrow-board", icon: "orders", labelKey: "portal.nav.tomorrow_board" },
      { href: "/dispatch/rescues", icon: "dispatch", labelKey: "portal.nav.rescues" },
      { href: "/dispatch-settings", icon: "settings", labelKey: "portal.nav.dispatch_settings" },
      { href: "/dispatch-locks", icon: "lock", labelKey: "portal.nav.dispatch_locks" },
      { href: "/manifests", icon: "manifests", labelKey: "portal.nav.manifests" },
    ],
  },
  {
    labelKey: "portal.nav.section.inventory",
    items: [
      { href: "/bins", icon: "inventory", labelKey: "portal.nav.bins_lots" },
      { href: "/pick-waves", icon: "inventory", labelKey: "portal.nav.pick_waves" },
      { href: "/cycle-counts", icon: "inventory", labelKey: "portal.nav.cycle_counts" },
      { href: "/cold-chain", icon: "warning", labelKey: "portal.nav.cold_chain" },
      { href: "/stock-commitments", icon: "inventory", labelKey: "portal.nav.stock_commitments" },
      { href: "/products", icon: "catalog", labelKey: "portal.nav.products" },
      { href: "/replenishment", icon: "forecast", labelKey: "portal.nav.replenishment" },
    ],
  },
  {
    labelKey: "portal.nav.section.fleet",
    items: [
      { href: "/drivers", icon: "fleet", labelKey: "portal.nav.drivers" },
      { href: "/labor-capacity", icon: "fleet", labelKey: "portal.nav.labor_capacity" },
      { href: "/vehicles", icon: "fleet", labelKey: "portal.nav.trucks" },
      { href: "/fleet-live-map", icon: "map", labelKey: "portal.nav.live_fleet" },
    ],
  },
  {
    labelKey: "portal.nav.section.supply_chain",
    items: [
      { href: "/supply-requests", icon: "supplyRequests", labelKey: "portal.nav.supply_requests" },
      { href: "/coverage", icon: "warehouse", labelKey: "warehouse_portal.coverage.text.coverage_and_supply" },
      { href: "/transfers", icon: "transfers", labelKey: "portal.nav.transfers" },
    ],
  },
  {
    labelKey: "portal.nav.section.exceptions",
    items: [
      { href: "/staff", icon: "staff", labelKey: "portal.nav.staff" },
      { href: "/crm", icon: "crm", labelKey: "portal.nav.retailers" },
      { href: "/operations", icon: "send", labelKey: "portal.nav.operations" },
      { href: "/returns", icon: "returns", labelKey: "portal.nav.returns" },
      { href: "/claims", icon: "warning", labelKey: "portal.nav.claims" },
      { href: "/exceptions", icon: "warning", labelKey: "portal.nav.exceptions" },
      { href: "/analytics", icon: "analytics", labelKey: "portal.nav.analytics" },
    ],
  },
  {
    labelKey: "portal.nav.section.finance",
    items: [
      { href: "/treasury", icon: "treasury", labelKey: "portal.nav.treasury" },
      { href: "/payment-config", icon: "payment", labelKey: "portal.nav.payment_config" },
    ],
  },
  {
    labelKey: "portal.nav.section.settings",
    items: [{ href: "/settings", icon: "settings", labelKey: "portal.nav.settings" }],
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
