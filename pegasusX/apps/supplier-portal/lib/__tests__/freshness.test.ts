import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  applyDriverLocationPatch,
  dashboardDirtySlice,
  parseDriverLocationPatch,
  shouldRefetchDashboardRollup,
} from "@pegasusx/ws-refresh-contract";

describe("GS-UF freshness", () => {
  it("50 coalesced location events produce 0 dashboard GETs", () => {
    const events = Array.from({ length: 50 }, () => "DRIVER_LOCATION_UPDATED");
    const dashboardGets = events.filter(shouldRefetchDashboardRollup).length;
    expect(dashboardGets).toBe(0);
    expect(events.every((type) => dashboardDirtySlice(type) === "map")).toBe(true);
  });

  it("order and money events dirty the dashboard rollup", () => {
    expect(shouldRefetchDashboardRollup("ORDER_STATUS_CHANGED")).toBe(true);
    expect(shouldRefetchDashboardRollup("MANIFEST_DISPATCHED")).toBe(true);
    expect(shouldRefetchDashboardRollup("PAYMENT_CLEARED")).toBe(true);
    expect(shouldRefetchDashboardRollup("SHOP_CLOSED")).toBe(true);
    expect(shouldRefetchDashboardRollup("PULSE_TICK")).toBe(false);
    expect(shouldRefetchDashboardRollup("SCENARIO_PUBLISHED")).toBe(false);
  });

  it("patches a matching fleet marker without inventing a route", () => {
    const patch = parseDriverLocationPatch(
      JSON.stringify({
        type: "DRIVER_LOCATION_UPDATED",
        data: { driver_id: "drv-1", lat: 41.3, lng: 69.2 },
      }),
    );
    expect(patch).toEqual({ driver_id: "drv-1", route_id: undefined, lat: 41.3, lng: 69.2 });
    const next = applyDriverLocationPatch(
      [
        {
          driver_id: "drv-1",
          route_id: "r1",
          live_location_available: false,
          driver_location: undefined as { lat?: number; lng?: number; latitude?: number; longitude?: number } | undefined,
        },
        {
          driver_id: "drv-2",
          route_id: "r2",
          live_location_available: false,
          driver_location: undefined as { lat?: number; lng?: number; latitude?: number; longitude?: number } | undefined,
        },
      ],
      patch!,
    );
    expect(next[0]?.driver_location).toEqual({
      lat: 41.3,
      lng: 69.2,
      latitude: 41.3,
      longitude: 69.2,
    });
    expect(next[1]?.driver_location).toBeUndefined();
  });

  it("supplier dashboard hook does not subscribe location into the rollup set", () => {
    const here = dirname(fileURLToPath(import.meta.url));
    const source = readFileSync(join(here, "../../app/(portal)/dashboard/use-dashboard-data.ts"), "utf8");
    expect(source).toContain("shouldRefetchDashboardRollup");
    expect(source).toContain("DASHBOARD_ROLLUP_REFRESH_EVENTS");
    expect(source).toContain("getSupplierDashboardConditional");
    expect(source).not.toContain("DRIVER_LOCATION_UPDATED");
  });
});
