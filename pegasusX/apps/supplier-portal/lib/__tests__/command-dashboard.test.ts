import { readFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import { describe, expect, it } from "vitest";
import { incrementOrderStatusCount, emptyOrderStatusCounts } from "@pegasusx/types";
import { formatPackMoney } from '@pegasusx/api-core';
import { orderStatusFromWsRaw } from "../dashboard-command";

const here = dirname(fileURLToPath(import.meta.url));
const pageSource = readFileSync(join(here, "../../app/(portal)/dashboard/page.tsx"), "utf8");

describe("GS-U2 command dashboard honesty", () => {
  it("does not hardcode UZS or a fake yesterday delta", () => {
    expect(pageSource).not.toMatch(/UZS/);
    expect(pageSource).not.toMatch(/vs yesterday"/);
    expect(pageSource).not.toMatch(/revenueChangePct/);
    expect(pageSource).toMatch(/formatPackMoney/);
    expect(pageSource).toMatch(/ORDER_STATUS_FUNNEL/);
    expect(pageSource).toMatch(/\/orders\?status=/);
  });

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
