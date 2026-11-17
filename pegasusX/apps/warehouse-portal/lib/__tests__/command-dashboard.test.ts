import { describe, expect, it } from "vitest";
import {
  TRUCK_DUTY_STATUSES,
  canonicalizeTruckDuty,
  emptyOrderStatusCounts,
  incrementOrderStatusCount,
} from "@pegasusx/types";

describe("GS-U4 warehouse command", () => {
  it("keeps canonical truck duty statuses and order states", () => {
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
