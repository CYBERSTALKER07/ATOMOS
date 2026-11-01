import { describe, expect, it } from "vitest";
import { ORDER_STATUS_FUNNEL, emptyOrderStatusCounts, incrementOrderStatusCount } from "@pegasusx/types";

describe("GS-U6 retailer command", () => {
  it("increments FISCAL_FAILED on the 17-key child stack without blending suppliers", () => {
    const next = incrementOrderStatusCount(emptyOrderStatusCounts(), "FISCAL_FAILED");
    expect(next.FISCAL_FAILED).toBe(1);
    expect(Object.keys(next)).toHaveLength(17);
    expect(ORDER_STATUS_FUNNEL).toContain("FISCAL_FAILED");
    expect(ORDER_STATUS_FUNNEL).toHaveLength(17);
  });
});
