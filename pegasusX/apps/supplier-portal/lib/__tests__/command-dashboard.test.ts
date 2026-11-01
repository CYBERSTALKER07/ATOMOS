import { describe, expect, it } from "vitest";
import { incrementOrderStatusCount, emptyOrderStatusCounts } from "@pegasusx/types";
import { formatPackMoney } from '@pegasusx/api-core';
import { orderStatusFromWsRaw } from "../dashboard-command";

describe("GS-U2 command dashboard honesty", () => {
  it("increments FISCAL_FAILED from a WS payload", () => {
    const status = orderStatusFromWsRaw(
      JSON.stringify({ type: "ORDER_STATUS_CHANGED", data: { status: "FISCAL_FAILED" } }),
    );
    const next = incrementOrderStatusCount(emptyOrderStatusCounts(), status);
    expect(next.FISCAL_FAILED).toBe(1);
    expect(Object.keys(next)).toHaveLength(17);
  });

  it("formats pack money without inventing UZS", () => {
    expect(formatPackMoney(1500, null)).toBe("15");
    expect(formatPackMoney(1500, { currency_code: "EUR", currency_decimal_places: 2 })).toBe("15 EUR");
  });
});
