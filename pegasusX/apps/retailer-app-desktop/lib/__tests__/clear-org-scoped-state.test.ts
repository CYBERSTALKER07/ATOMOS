import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { clearOrgScopedState, ORG_SWITCHED_EVENT } from "../clear-org-scoped-state";

vi.mock("../pending-pos-sales", () => ({
  clearParkedPosCart: vi.fn(async () => undefined),
}));

describe("clearOrgScopedState", () => {
  beforeEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  afterEach(() => {
    localStorage.clear();
    sessionStorage.clear();
  });

  it("clears cart, POS, count drafts, and assist keys", async () => {
    localStorage.setItem("retailer_cart", "[]");
    localStorage.setItem("retailer_pos_parked_cart_v1", "{}");
    localStorage.setItem("retailer_stock_count_draft_v1", "{}");
    localStorage.setItem("retailer_assist_context_v1", "{}");
    sessionStorage.setItem("retailer_pos_session_v1", "sess-1");

    const events: Event[] = [];
    const handler = (e: Event) => events.push(e);
    window.addEventListener(ORG_SWITCHED_EVENT, handler);

    const result = await clearOrgScopedState();

    window.removeEventListener(ORG_SWITCHED_EVENT, handler);

    expect(localStorage.getItem("retailer_cart")).toBeNull();
    expect(localStorage.getItem("retailer_pos_parked_cart_v1")).toBeNull();
    expect(localStorage.getItem("retailer_stock_count_draft_v1")).toBeNull();
    expect(localStorage.getItem("retailer_assist_context_v1")).toBeNull();
    expect(sessionStorage.getItem("retailer_pos_session_v1")).toBeNull();
    expect(result.clearedLocalKeys.length).toBeGreaterThan(0);
    expect(events.length).toBe(1);
  });

  it("does not throw when storage is empty", async () => {
    await expect(clearOrgScopedState()).resolves.toBeDefined();
  });
});
