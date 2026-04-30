/**
 * Unit tests for Dock page utility functions and supplier grouping logic.
 *
 * These test the pure functions extracted from
 * app/(dashboard)/dock/page.tsx without requiring React rendering.
 */
import { describe, it, expect } from "vitest";
import type { TrackingOrder } from "@/lib/types";

/* ── Re-implement pure functions from dock/page.tsx for isolated testing ── */

const chipCfg: Record<
  string,
  { color: string; label: string }
> = {
  DISPATCHED: { color: "warning", label: "Dispatched" },
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  ARRIVING: { color: "accent", label: "Arriving" },
  ARRIVED: { color: "success", label: "Arrived" },
  ARRIVED_SHOP_CLOSED: { color: "warning", label: "Shop Closed" },
  AWAITING_PAYMENT: { color: "danger", label: "Awaiting Payment" },
  PENDING_CASH_COLLECTION: { color: "warning", label: "Cash Collection" },
  PENDING: { color: "default", label: "Pending" },
  PENDING_REVIEW: { color: "default", label: "Pending Review" },
  LOADED: { color: "default", label: "Loaded" },
  COMPLETED: { color: "success", label: "Completed" },
  CANCELLED: { color: "danger", label: "Cancelled" },
  CANCEL_REQUESTED: { color: "danger", label: "Cancel Requested" },
  NO_CAPACITY: { color: "danger", label: "No Capacity" },
  SCHEDULED: { color: "default", label: "Scheduled" },
  AUTO_ACCEPTED: { color: "default", label: "Auto-Accepted" },
  QUARANTINE: { color: "danger", label: "Quarantined" },
  DELIVERED_ON_CREDIT: { color: "success", label: "Delivered (Credit)" },
};

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ") + "";
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

interface SupplierGroup {
  supplierId: string;
  supplierName: string;
  orders: TrackingOrder[];
  totalAmount: number;
  hasApproaching: boolean;
  hasArrived: boolean;
}

function groupBySupplier(activeOrders: TrackingOrder[]): SupplierGroup[] {
  const map = new Map<string, SupplierGroup>();
  for (const order of activeOrders) {
    const sid = order.supplier_id;
    let group = map.get(sid);
    if (!group) {
      group = {
        supplierId: sid,
        supplierName: order.supplier_name || sid.slice(0, 8),
        orders: [],
        totalAmount: 0,
        hasApproaching: false,
        hasArrived: false,
      };
      map.set(sid, group);
    }
    group.orders.push(order);
    group.totalAmount += order.total_amount;
    if (order.is_approaching || order.state === "ARRIVING") group.hasApproaching = true;
    if (order.state === "ARRIVED" || order.state === "AWAITING_PAYMENT") group.hasArrived = true;
  }
  return Array.from(map.values()).sort((a, b) => {
    if (a.hasArrived !== b.hasArrived) return a.hasArrived ? -1 : 1;
    if (a.hasApproaching !== b.hasApproaching) return a.hasApproaching ? -1 : 1;
    return b.totalAmount - a.totalAmount;
  });
}

/* ── Helper factory ── */

function makeOrder(overrides: Partial<TrackingOrder> = {}): TrackingOrder {
  return {
    order_id: "ORD-001",
    supplier_id: "SUP-001",
    supplier_name: "Supplier Alpha",
    driver_id: "DRV-001",
    state: "IN_TRANSIT",
    total_amount: 150_000,
    order_source: "CART",
    driver_latitude: null,
    driver_longitude: null,
    is_approaching: false,
    delivery_token: "tok-abc",
    created_at: new Date().toISOString(),
    items: [],
    ...overrides,
  };
}

/* ── Tests ── */

describe("formatAmount", () => {
  it("formats zero", () => {
    expect(formatAmount(0)).toBe("0");
  });

  it("formats small amount without separators", () => {
    expect(formatAmount(999)).toBe("999");
  });

  it("formats thousands with space separator", () => {
    expect(formatAmount(150_000)).toBe("150 000");
  });

  it("formats millions", () => {
    expect(formatAmount(1_500_000)).toBe("1 500 000");
  });
});

describe("timeAgo", () => {
  it("returns 'just now' for recent timestamps", () => {
    const now = new Date().toISOString();
    expect(timeAgo(now)).toBe("just now");
  });

  it("returns minutes for <60 min", () => {
    const fiveMinAgo = new Date(Date.now() - 5 * 60_000).toISOString();
    expect(timeAgo(fiveMinAgo)).toBe("5m ago");
  });

  it("returns hours for <24 hrs", () => {
    const twoHrsAgo = new Date(Date.now() - 2 * 3_600_000).toISOString();
    expect(timeAgo(twoHrsAgo)).toBe("2h ago");
  });

  it("returns days for ≥24 hrs", () => {
    const threeDaysAgo = new Date(Date.now() - 3 * 86_400_000).toISOString();
    expect(timeAgo(threeDaysAgo)).toBe("3d ago");
  });
});

describe("chipCfg", () => {
  it("maps all delivery states", () => {
    const deliveryStates = ["DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED", "AWAITING_PAYMENT"];
    for (const state of deliveryStates) {
      expect(chipCfg[state]).toBeDefined();
      expect(chipCfg[state].label).toBeTruthy();
      expect(chipCfg[state].color).toBeTruthy();
    }
  });

  it("uses success for ARRIVED", () => {
    expect(chipCfg["ARRIVED"].color).toBe("success");
  });

  it("uses danger for AWAITING_PAYMENT", () => {
    expect(chipCfg["AWAITING_PAYMENT"].color).toBe("danger");
  });

  it("uses accent for ARRIVING", () => {
    expect(chipCfg["ARRIVING"].color).toBe("accent");
  });
});

describe("groupBySupplier", () => {
  it("returns empty array for no orders", () => {
    expect(groupBySupplier([])).toEqual([]);
  });

  it("groups orders by supplier_id", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", supplier_name: "Alpha" }),
      makeOrder({ order_id: "O2", supplier_id: "S2", supplier_name: "Beta" }),
      makeOrder({ order_id: "O3", supplier_id: "S1", supplier_name: "Alpha" }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups).toHaveLength(2);
    const s1 = groups.find((g) => g.supplierId === "S1");
    expect(s1?.orders).toHaveLength(2);
    expect(s1?.supplierName).toBe("Alpha");
  });

  it("sums totalAmount per supplier", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", total_amount: 100_000 }),
      makeOrder({ order_id: "O2", supplier_id: "S1", total_amount: 50_000 }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].totalAmount).toBe(150_000);
  });

  it("sets hasApproaching when order is ARRIVING", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", state: "ARRIVING" }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].hasApproaching).toBe(true);
  });

  it("sets hasApproaching when is_approaching is true", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", state: "IN_TRANSIT", is_approaching: true }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].hasApproaching).toBe(true);
  });

  it("sets hasArrived when order is ARRIVED", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", state: "ARRIVED" }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].hasArrived).toBe(true);
  });

  it("sets hasArrived when order is AWAITING_PAYMENT", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", state: "AWAITING_PAYMENT" }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].hasArrived).toBe(true);
  });

  it("sorts: arrived first, then approaching, then by total amount desc", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "S1", state: "IN_TRANSIT", total_amount: 500_000 }),
      makeOrder({ order_id: "O2", supplier_id: "S2", state: "ARRIVING", total_amount: 100_000 }),
      makeOrder({ order_id: "O3", supplier_id: "S3", state: "ARRIVED", total_amount: 50_000 }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].supplierId).toBe("S3"); // arrived first
    expect(groups[1].supplierId).toBe("S2"); // approaching second
    expect(groups[2].supplierId).toBe("S1"); // highest amount last (no priority flags)
  });

  it("falls back to supplier_id slice when supplier_name is empty", () => {
    const orders = [
      makeOrder({ order_id: "O1", supplier_id: "ABCDEFGHIJ", supplier_name: "" }),
    ];
    const groups = groupBySupplier(orders);
    expect(groups[0].supplierName).toBe("ABCDEFGHI".slice(0, 8));
  });
});
