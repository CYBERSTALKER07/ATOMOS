import { describe, expect, it } from "vitest";
import {
  canOrderTransition,
  getNextOrderStatuses,
  isTerminalOrderStatus,
  CANONICAL_ORDER_GRAPH,
  ORDER_LIFECYCLE_NODES,
} from "@pegasusx/types";

describe("Canonical Order State Machine - TypeScript Graph", () => {
  it("enforces ADR-009 fiscal hard-gate on delivery states", () => {
    expect(canOrderTransition("ARRIVED", "COMPLETED")).toBe(false);
    expect(canOrderTransition("AWAITING_PAYMENT", "COMPLETED")).toBe(false);
    expect(canOrderTransition("PENDING_CASH_COLLECTION", "COMPLETED")).toBe(false);
    expect(canOrderTransition("DELIVERED_ON_CREDIT", "COMPLETED")).toBe(false);

    // Capture to fiscalizing is valid
    expect(canOrderTransition("AWAITING_PAYMENT", "FISCALIZING")).toBe(true);
    expect(canOrderTransition("PENDING_CASH_COLLECTION", "FISCALIZING")).toBe(true);
    expect(canOrderTransition("DELIVERED_ON_CREDIT", "FISCALIZING")).toBe(true);
    expect(canOrderTransition("FISCALIZING", "COMPLETED")).toBe(true);
  });

  it("verifies terminal status invariant", () => {
    expect(isTerminalOrderStatus("COMPLETED")).toBe(true);
    expect(getNextOrderStatuses("COMPLETED")).toEqual([]);

    expect(isTerminalOrderStatus("PENDING")).toBe(false);
    expect(isTerminalOrderStatus("CANCELLED")).toBe(false); // Has reconciliation loop
  });

  it("prevents bricking orders on cancel request", () => {
    const cancelExits = getNextOrderStatuses("CANCEL_REQUESTED");
    expect(cancelExits).toContain("CANCELLED");
    expect(cancelExits).toContain("LOADED");
    expect(cancelExits).toContain("IN_TRANSIT");
    expect(cancelExits).toContain("ARRIVED");
    expect(cancelExits).not.toContain("COMPLETED");
  });

  it("supports idempotent transitions", () => {
    expect(canOrderTransition("PENDING", "PENDING")).toBe(true);
    expect(canOrderTransition("IN_TRANSIT", "IN_TRANSIT")).toBe(true);
    expect(canOrderTransition("COMPLETED", "COMPLETED")).toBe(true);
  });

  it("registers all 18 statuses in metadata nodes", () => {
    const statuses = Object.keys(CANONICAL_ORDER_GRAPH);
    expect(statuses.length).toBe(18);
    for (const status of statuses) {
      expect(ORDER_LIFECYCLE_NODES[status]).toBeDefined();
      expect(ORDER_LIFECYCLE_NODES[status].phase).toBeDefined();
    }
  });
});
