import { describe, expect, it } from "vitest";
import { allNavHrefs, primaryNavHrefs } from "../nav";

const PRESERVED = [
  "/",
  "/control-tower",
  "/orders",
  "/preorders",
  "/tomorrow-board",
  "/dispatch",
  "/dispatch/rescues",
  "/dispatch-settings",
  "/manifests",
  "/inventory",
  "/bins",
  "/pick-waves",
  "/cycle-counts",
  "/cold-chain",
  "/stock-commitments",
  "/products",
  "/supply-requests",
  "/coverage",
  "/settings",
  "/replenishment",
  "/demand-forecast",
  "/drivers",
  "/labor-capacity",
  "/vehicles",
  "/fleet-live-map",
  "/dispatch-locks",
  "/staff",
  "/crm",
  "/operations",
  "/returns",
  "/claims",
  "/exceptions",
  "/transfers",
  "/analytics",
  "/treasury",
  "/payment-config",
];

describe("GS-UN warehouse nav", () => {
  it("keeps the first group at most 5 and one-click Home / Dispatch / Floor / Plan", () => {
    const primary = primaryNavHrefs();
    expect(primary.length).toBeLessThanOrEqual(5);
    expect(primary).toEqual(["/", "/dispatch", "/inventory", "/demand-forecast"]);
  });

  it("does not delete existing routes or promote dispatch settings", () => {
    const hrefs = allNavHrefs();
    for (const href of PRESERVED) {
      expect(hrefs).toContain(href);
    }
    expect(primaryNavHrefs()).not.toContain("/dispatch-settings");
    expect(primaryNavHrefs()).not.toContain("/dispatch-locks");
  });
});
