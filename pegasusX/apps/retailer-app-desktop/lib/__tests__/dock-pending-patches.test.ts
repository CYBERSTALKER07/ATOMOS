import { describe, expect, it, beforeEach } from "vitest";
import type { TrackingOrder } from "@/lib/types";
import {
  applyDockPendingPatches,
  clearDockPendingPatches,
  queueDockOrderPatch,
} from "../dock-pending-patches";

function makeOrder(id: string, state = "IN_TRANSIT"): TrackingOrder {
  return {
    order_id: id,
    supplier_id: "sup-1",
    supplier_name: "Alpha",
    driver_id: "drv-1",
    state,
    total_amount: 1000,
    order_source: "CART",
    driver_latitude: null,
    driver_longitude: null,
    is_approaching: false,
    delivery_token: "tok",
    created_at: new Date().toISOString(),
    items: [],
  };
}

describe("dock-pending-patches", () => {
  beforeEach(() => {
    clearDockPendingPatches();
  });

  it("applies queued patch overlay on hydrate", () => {
    queueDockOrderPatch("ord-1", { state: "ARRIVING", is_approaching: true });
    const merged = applyDockPendingPatches([makeOrder("ord-1")]);
    expect(merged[0].state).toBe("ARRIVING");
    expect(merged[0].is_approaching).toBe(true);
  });

  it("clears patches after reconcile", () => {
    queueDockOrderPatch("ord-1", { state: "ARRIVED" });
    clearDockPendingPatches();
    const merged = applyDockPendingPatches([makeOrder("ord-1")]);
    expect(merged[0].state).toBe("IN_TRANSIT");
  });
});
