import type { ElementType } from "react";
import {
  Activity,
  BarChart3,
  Box,
  Building2,
  Clock,
  Container,
  FileBarChart,
  HandHelping,
  LayoutDashboard,
  LayoutGrid,
  MapPin,
  PackageSearch,
  Radar,
  RefreshCcw,
  Settings,
  ShoppingCart,
  Store,
} from "lucide-react";

export type NavEntry = {
  href: string;
  icon: ElementType;
  labelKey: string;
  perm?: string;
  pack?: string;
};
export type NavSection = { labelKey?: string; items: NavEntry[] };

/** GS-UN: Home · Buy · Incoming · Store. Packs reveal Store children, not a 6th tab. */
export const NAV: NavSection[] = [
  {
    items: [
      { href: "/dashboard", icon: LayoutDashboard, labelKey: "portal.nav.dashboard" },
      { href: "/catalog", icon: PackageSearch, labelKey: "portal.nav.buy", perm: "order.place" },
      { href: "/tracking", icon: MapPin, labelKey: "portal.nav.incoming", perm: "dock.receive" },
      { href: "/stock", icon: PackageSearch, labelKey: "portal.nav.store", perm: "stock.view", pack: "STORE_STOCK" },
    ],
  },
  {
    labelKey: "portal.nav.section.operations",
    items: [
      { href: "/orders", icon: ShoppingCart, labelKey: "portal.nav.orders", perm: "order.place" },
      { href: "/procurement", icon: Activity, labelKey: "portal.nav.procurement", perm: "order.place" },
      { href: "/my-suppliers", icon: Store, labelKey: "portal.nav.my_suppliers", perm: "order.place" },
      { href: "/dock", icon: Container, labelKey: "portal.nav.dock", perm: "dock.receive" },
    ],
  },
  {
    labelKey: "portal.nav.section.inventory",
    items: [
      { href: "/stock/local-skus", icon: Box, labelKey: "portal.nav.local_skus", perm: "stock.view", pack: "STORE_STOCK" },
      { href: "/pos", icon: ShoppingCart, labelKey: "portal.nav.pos", perm: "pos.sell", pack: "POS" },
      { href: "/shifts", icon: Clock, labelKey: "portal.nav.shifts", perm: "shift.open", pack: "SHIFTS" },
      { href: "/sections", icon: LayoutGrid, labelKey: "portal.nav.sections", perm: "stock.view", pack: "SECTIONS" },
      { href: "/assist", icon: HandHelping, labelKey: "portal.nav.assist", perm: "assist.respond", pack: "CUSTOMER_ASSIST" },
    ],
  },
  {
    labelKey: "portal.nav.section.insights",
    items: [
      { href: "/credit", icon: Store, labelKey: "portal.nav.credit_partners", perm: "order.place" },
      { href: "/auto-order", icon: RefreshCcw, labelKey: "portal.nav.auto_order", perm: "order.place" },
      { href: "/insights", icon: BarChart3, labelKey: "portal.nav.insights", perm: "reports.view" },
      { href: "/control-tower", icon: Radar, labelKey: "portal.nav.control_tower" },
      { href: "/reports", icon: FileBarChart, labelKey: "portal.nav.reports_pro", perm: "reports.view", pack: "REPORTS_PRO" },
      { href: "/hq", icon: Building2, labelKey: "portal.nav.franchise_hq", perm: "reports.view", pack: "REPORTS_PRO" },
    ],
  },
  {
    labelKey: "portal.nav.section.system",
    items: [{ href: "/settings", icon: Settings, labelKey: "portal.nav.settings" }],
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
