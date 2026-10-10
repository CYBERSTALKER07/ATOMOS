import { describe, expect, it } from "vitest";
import { NAV, allNavHrefs, primaryNavHrefs } from "../nav";

const PRESERVED = [
  "/dashboard",
  "/orders",
  "/dispatch",
  "/ops/map",
  "/labor-capacity",
  "/manifests",
  "/fleet",
  "/fleet/orders",
  "/operations",
  "/operations/replenishment-policies",
  "/replenishment/suggestions",
  "/exceptions",
  "/activity",
  "/inventory",
  "/inventory/import",
  "/catalog",
  "/pricing",
  "/pricing/retailer-overrides",
  "/promotions",
  "/topology",
  "/crm",
  "/loyalty",
  "/entity-resolution",
  "/factories",
  "/warehouses",
  "/delivery-zones",
  "/supply-lanes",
  "/geo-report",
  "/analytics",
  "/control-tower",
  "/analytics/demand",
  "/analytics/demand/flywheel",
  "/analytics/route-performance",
  "/analytics/demand/signals",
  "/demand/payday-calendar",
  "/analytics/knowledge-graph",
  "/ai/recommendations",
  "/planning",
  "/treasury",
  "/treasury/cash-reconciliations",
  "/finance/credit-notes",
  "/reconciliation",
  "/compliance",
  "/payments",
  "/earnings",
  "/finance/payouts",
  "/credit/policy",
  "/credit/collections",
  "/credit/admin-disable",
  "/chargebacks",
  "/chargebacks/claims",
  "/ledger",
  "/profile",
  "/settings/tax-regimes",
  "/settings/fx-rates",
  "/settings/planning",
  "/settings/return-policy",
  "/settings/notification-preferences",
  "/settings/integrations",
  "/settings/segmentation",
  "/settings/playbooks",
  "/org-fleet",
  "/returns",
];

describe("GS-UN supplier nav", () => {
  it("keeps the first group at most 5 and one-click Home / Orders / Dispatch / Plan", () => {
    const primary = primaryNavHrefs();
    expect(primary.length).toBeLessThanOrEqual(5);
    expect(primary).toEqual(["/dashboard", "/orders", "/dispatch", "/planning"]);
  });

  it("does not delete existing routes", () => {
    const hrefs = allNavHrefs();
    for (const href of PRESERVED) {
      expect(hrefs).toContain(href);
    }
  });

  it("labels /planning and /settings/planning differently", () => {
    const plan = NAV.flatMap((s) => s.items).find((item) => item.href === "/planning");
    const flags = NAV.flatMap((s) => s.items).find((item) => item.href === "/settings/planning");
    expect(plan?.labelKey).toBe("portal.nav.planning");
    expect(flags?.labelKey).toBe("portal.nav.planning_settings");
    expect(plan?.labelKey).not.toBe(flags?.labelKey);
  });
});
