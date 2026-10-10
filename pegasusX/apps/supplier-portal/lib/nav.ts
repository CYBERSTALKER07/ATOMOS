export type NavEntry = { href: string; icon: string; labelKey: string };
export type NavSection = { labelKey?: string; items: NavEntry[] };

/** GS-UN: first group is the ritual row. Overflow stays reachable. */
export const NAV: NavSection[] = [
  {
    items: [
      { href: "/dashboard", icon: "overview", labelKey: "portal.nav.overview" },
      { href: "/orders", icon: "orders", labelKey: "portal.nav.orders" },
      { href: "/dispatch", icon: "dispatch", labelKey: "portal.nav.dispatch" },
      { href: "/planning", icon: "overview", labelKey: "portal.nav.planning" },
    ],
  },
  {
    labelKey: "portal.nav.section.fleet",
    items: [
      { href: "/ops/map", icon: "dispatch", labelKey: "portal.nav.live_ops_map" },
      { href: "/labor-capacity", icon: "fleet", labelKey: "portal.nav.labor_capacity" },
      { href: "/manifests", icon: "manifests", labelKey: "portal.nav.manifests" },
      { href: "/fleet", icon: "fleet", labelKey: "portal.nav.fleet" },
      { href: "/fleet/orders", icon: "orders", labelKey: "portal.nav.fleet_orders" },
      { href: "/operations", icon: "dispatch", labelKey: "portal.nav.operations" },
      { href: "/org-fleet", icon: "person-add", labelKey: "portal.nav.org_fleet" },
    ],
  },
  {
    labelKey: "portal.nav.section.catalog",
    items: [
      { href: "/inventory", icon: "inventory", labelKey: "portal.nav.inventory" },
      { href: "/inventory/import", icon: "inventory", labelKey: "portal.nav.import_csv" },
      { href: "/catalog", icon: "catalog", labelKey: "portal.nav.catalog" },
      { href: "/pricing", icon: "pricing", labelKey: "portal.nav.pricing" },
      { href: "/pricing/retailer-overrides", icon: "pricing", labelKey: "portal.nav.retailer_overrides" },
      { href: "/promotions", icon: "pricing", labelKey: "portal.nav.promotions" },
      { href: "/operations/replenishment-policies", icon: "inventory", labelKey: "portal.nav.replenishment_policies" },
      { href: "/replenishment/suggestions", icon: "inventory", labelKey: "portal.nav.reorder_suggestions" },
    ],
  },
  {
    labelKey: "portal.nav.section.network",
    items: [
      { href: "/topology", icon: "topology", labelKey: "portal.nav.topology" },
      { href: "/crm", icon: "crm", labelKey: "portal.nav.crm" },
      { href: "/loyalty", icon: "pricing", labelKey: "portal.nav.loyalty" },
      { href: "/entity-resolution", icon: "topology", labelKey: "portal.nav.entity_resolution" },
      { href: "/factories", icon: "factory", labelKey: "portal.nav.factories" },
      { href: "/warehouses", icon: "warehouse", labelKey: "portal.nav.warehouses" },
      { href: "/delivery-zones", icon: "global", labelKey: "portal.nav.delivery_zones" },
      { href: "/supply-lanes", icon: "fleet", labelKey: "portal.nav.supply_lanes" },
      { href: "/geo-report", icon: "global", labelKey: "portal.nav.geo_report" },
    ],
  },
  {
    labelKey: "portal.nav.section.exceptions",
    items: [
      { href: "/exceptions", icon: "warning", labelKey: "portal.nav.exceptions" },
      { href: "/control-tower", icon: "global", labelKey: "portal.nav.control_tower" },
      { href: "/activity", icon: "overview", labelKey: "portal.nav.activity" },
      { href: "/returns", icon: "returns", labelKey: "portal.nav.returns" },
    ],
  },
  {
    labelKey: "portal.nav.section.insights",
    items: [
      { href: "/analytics", icon: "overview", labelKey: "portal.nav.analytics" },
      { href: "/analytics/demand", icon: "overview", labelKey: "portal.nav.demand_forecast" },
      { href: "/analytics/demand/flywheel", icon: "campaign", labelKey: "portal.nav.pos_flywheel" },
      { href: "/analytics/route-performance", icon: "overview", labelKey: "portal.nav.route_performance" },
      { href: "/analytics/demand/signals", icon: "campaign", labelKey: "portal.nav.demand_signals" },
      { href: "/demand/payday-calendar", icon: "campaign", labelKey: "portal.nav.payday_calendar" },
      { href: "/analytics/knowledge-graph", icon: "topology", labelKey: "portal.nav.knowledge_graph" },
      { href: "/ai/recommendations", icon: "overview", labelKey: "portal.nav.ai_recommendations" },
    ],
  },
  {
    labelKey: "portal.nav.section.finance",
    items: [
      { href: "/treasury", icon: "treasury", labelKey: "portal.nav.treasury" },
      { href: "/treasury/cash-reconciliations", icon: "reconcile", labelKey: "portal.nav.cash_reconciliations" },
      { href: "/finance/credit-notes", icon: "returns", labelKey: "portal.nav.credit_notes" },
      { href: "/reconciliation", icon: "reconcile", labelKey: "portal.nav.reconciliation" },
      { href: "/compliance", icon: "warning", labelKey: "portal.nav.compliance_audit" },
      { href: "/payments", icon: "payment", labelKey: "portal.nav.payments" },
      { href: "/earnings", icon: "pricing", labelKey: "portal.nav.earnings" },
      { href: "/finance/payouts", icon: "treasury", labelKey: "portal.nav.payouts" },
      { href: "/credit/policy", icon: "treasury", labelKey: "portal.nav.credit_policy" },
      { href: "/credit/collections", icon: "treasury", labelKey: "portal.nav.credit_collections" },
      { href: "/credit/admin-disable", icon: "warning", labelKey: "portal.nav.credit_admin_disable" },
      { href: "/chargebacks", icon: "warning", labelKey: "portal.nav.chargebacks" },
      { href: "/chargebacks/claims", icon: "warning", labelKey: "portal.nav.claim_chargebacks" },
      { href: "/ledger", icon: "orders", labelKey: "portal.nav.ledger" },
    ],
  },
  {
    labelKey: "portal.nav.section.settings",
    items: [
      { href: "/profile", icon: "supplier", labelKey: "portal.nav.profile" },
      { href: "/settings/tax-regimes", icon: "overview", labelKey: "portal.nav.tax_regimes" },
      { href: "/settings/fx-rates", icon: "overview", labelKey: "portal.nav.fx_rates" },
      { href: "/settings/planning", icon: "overview", labelKey: "portal.nav.planning_settings" },
      { href: "/settings/return-policy", icon: "returns", labelKey: "portal.nav.return_policy" },
      { href: "/settings/notification-preferences", icon: "overview", labelKey: "portal.nav.notifications" },
      { href: "/settings/integrations", icon: "overview", labelKey: "portal.nav.integrations" },
      { href: "/settings/segmentation", icon: "overview", labelKey: "portal.nav.segmentation" },
      { href: "/settings/playbooks", icon: "overview", labelKey: "portal.nav.playbooks" },
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
