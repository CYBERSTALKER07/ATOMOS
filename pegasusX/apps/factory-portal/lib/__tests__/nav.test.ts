import { describe, expect, it } from "vitest";
import { allNavHrefs, primaryNavHrefs } from "../nav";

const PRESERVED = [
  "/",
  "/loading-bay",
  "/transfers",
  "/fleet",
  "/staff",
  "/settings/location",
  "/insights",
  "/analytics",
  "/supply-requests",
  "/payload",
  "/payload-override",
  "/manifests",
  "/manifest-exceptions",
];

describe("GS-UN factory nav", () => {
  it("keeps the first group at most 5 and one-click Home / Bay / Payload / Transfers", () => {
    const primary = primaryNavHrefs();
    expect(primary.length).toBeLessThanOrEqual(5);
    expect(primary).toEqual(["/", "/loading-bay", "/payload", "/transfers"]);
  });

  it("does not delete existing routes", () => {
    const hrefs = allNavHrefs();
    for (const href of PRESERVED) {
      expect(hrefs).toContain(href);
    }
  });
});
