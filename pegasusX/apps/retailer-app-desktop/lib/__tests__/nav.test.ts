import { describe, expect, it } from "vitest";
import { allNavHrefs, primaryNavHrefs } from "../nav";

const PRESERVED = [
  "/dashboard",
  "/orders",
  "/tracking",
  "/dock",
  "/catalog",
  "/procurement",
  "/my-suppliers",
  "/credit",
  "/auto-order",
  "/stock",
  "/stock/local-skus",
  "/pos",
  "/shifts",
  "/sections",
  "/assist",
  "/insights",
  "/control-tower",
  "/reports",
  "/hq",
  "/settings",
];

describe("GS-UN retailer nav", () => {
  it("keeps the first group at most 5 and one-click Home / Buy / Incoming / Store", () => {
    const primary = primaryNavHrefs();
    expect(primary.length).toBeLessThanOrEqual(5);
    expect(primary).toEqual(["/dashboard", "/catalog", "/tracking", "/stock"]);
  });

  it("does not delete existing routes or add a 6th primary tab", () => {
    const hrefs = allNavHrefs();
    for (const href of PRESERVED) {
      expect(hrefs).toContain(href);
    }
    expect(primaryNavHrefs()).not.toContain("/pos");
    expect(primaryNavHrefs()).not.toContain("/hq");
  });
});
