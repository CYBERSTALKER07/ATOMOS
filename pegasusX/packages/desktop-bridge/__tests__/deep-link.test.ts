import { describe, expect, it } from "vitest";
import { desktopDeepLinkToPath } from "../deep-link";

describe("desktopDeepLinkToPath", () => {
  it("maps custom scheme to app path", () => {
    expect(desktopDeepLinkToPath("pegasusx-retailer://notifications")).toBe(
      "/notifications",
    );
    expect(desktopDeepLinkToPath("pegasusx-warehouse://dispatch")).toBe("/dispatch");
    expect(desktopDeepLinkToPath("pegasusx-supplier://orders")).toBe("/orders");
    expect(desktopDeepLinkToPath("pegasusx-factory://loading-bay")).toBe("/loading-bay");
  });

  it("preserves query strings", () => {
    expect(desktopDeepLinkToPath("pegasusx-retailer://orders?review=1")).toBe(
      "/orders?review=1",
    );
  });

  it("accepts plain paths", () => {
    expect(desktopDeepLinkToPath("/notifications")).toBe("/notifications");
  });
});
