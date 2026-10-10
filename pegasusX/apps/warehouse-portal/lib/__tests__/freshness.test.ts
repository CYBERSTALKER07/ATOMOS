import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from "../fleet-ws-events";

describe("GS-UF warehouse freshness", () => {
  it("does not refetch the fleet GET on location ticks", () => {
    expect(WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS.has("DRIVER_LOCATION_UPDATED")).toBe(false);
  });

  it("dashboard uses conditional GET and 60s polling", () => {
    const here = dirname(fileURLToPath(import.meta.url));
    const source = readFileSync(join(here, "../../app/page.tsx"), "utf8");
    expect(source).toContain("getWarehouseOpsDashboardConditional");
    expect(source).toContain("usePolling");
    expect(source).toContain("60_000");
  });
});
