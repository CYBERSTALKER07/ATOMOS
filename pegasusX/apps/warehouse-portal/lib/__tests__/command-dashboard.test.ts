import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import {
  TRUCK_DUTY_STATUSES,
  canonicalizeTruckDuty,
  emptyOrderStatusCounts,
  incrementOrderStatusCount,
} from "@pegasusx/types";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/page.tsx"), "utf8");
const kpiSource = readFileSync(join(here, "../../components/KpiStatCard.tsx"), "utf8");

describe("GS-U4 warehouse command", () => {
  it("does not draw invented history or a date-range theatre control", () => {
    expect(pageSource).not.toMatch(/sparkline_/);
    expect(pageSource).not.toMatch(/spark=/);
    expect(pageSource).not.toMatch(/dateRange/);
    expect(pageSource).not.toMatch(/last_7d/);
    expect(kpiSource).toMatch(/guardHistorySeries/);
  });

  it("binds full order and truck-duty stacks plus demand source", () => {
    expect(pageSource).toMatch(/ORDER_STATUS_FUNNEL/);
    expect(pageSource).toMatch(/TRUCK_DUTY_STATUSES/);
    expect(pageSource).toMatch(/gs-u-demand-source/);
    expect(pageSource).toMatch(/gs-u-hold-reasons/);
    expect(pageSource).toMatch(/\/orders\?state=/);
    expect(TRUCK_DUTY_STATUSES).toEqual(
      expect.arrayContaining(["OFF_SHIFT", "RETURNING_TO_WAREHOUSE", "UNASSIGNED", "VEHICLE_INACTIVE"]),
    );
  });

  it("increments FISCAL_FAILED and maps idle to AVAILABLE", () => {
    const next = incrementOrderStatusCount(emptyOrderStatusCounts(), "FISCAL_FAILED");
    expect(next.FISCAL_FAILED).toBe(1);
    expect(Object.keys(next)).toHaveLength(17);
    expect(canonicalizeTruckDuty("IDLE")).toBe("AVAILABLE");
    expect(canonicalizeTruckDuty("RETURNING")).toBe("RETURNING_TO_WAREHOUSE");
  });
});
