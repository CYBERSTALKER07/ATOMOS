import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { shouldRefetchDashboardRollup } from "@pegasusx/ws-refresh-contract";

describe("GS-UF factory fleet freshness", () => {
  it("uses usePolling and does not raw-setInterval", () => {
    const here = dirname(fileURLToPath(import.meta.url));
    const source = readFileSync(join(here, "../use-factory-fleet-live-map.ts"), "utf8");
    expect(source).toContain("usePolling");
    expect(source).toContain("hiddenIntervalMs: 60_000");
    expect(source).toContain("parseDriverLocationPatch");
    expect(source).not.toMatch(/window\.setInterval/);
    expect(source).not.toMatch(/[^.]setInterval\(/);
  });

  it("factory status events dirty the dashboard; location does not", () => {
    expect(shouldRefetchDashboardRollup("FACTORY_TRANSFER_UPDATE")).toBe(true);
    expect(shouldRefetchDashboardRollup("FACTORY_MANIFEST_UPDATE")).toBe(true);
    expect(shouldRefetchDashboardRollup("DRIVER_LOCATION_UPDATED")).toBe(false);
  });

  it("50 coalesced location events produce 0 dashboard GETs", () => {
    const events = Array.from({ length: 50 }, () => "DRIVER_LOCATION_UPDATED");
    expect(events.filter(shouldRefetchDashboardRollup)).toHaveLength(0);
  });
});
