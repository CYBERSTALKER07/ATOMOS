import { describe, expect, it } from "vitest";
import {
  ORDER_STATUS_FUNNEL,
  emptyOrderStatusCounts,
  guardHistorySeries,
  incrementOrderStatusCount,
  statusStackModel,
  yesterdayRevenueDeltaPct,
  type HistorySeries,
} from "@pegasusx/types";

describe("GS-U1 HistorySeries guard", () => {
  it("rejects null, unavailable, empty, short, and non-finite series", () => {
    expect(guardHistorySeries(null)).toBeNull();
    expect(guardHistorySeries(undefined)).toBeNull();
    expect(
      guardHistorySeries({ points: [1, 2], source: "unavailable", available: true }),
    ).toBeNull();
    expect(
      guardHistorySeries({ points: [1, 2], source: "empty", available: true }),
    ).toBeNull();
    expect(
      guardHistorySeries({ points: [1, 2], source: "live", available: false }),
    ).toBeNull();
    expect(
      guardHistorySeries({ points: [4], source: "live", available: true }),
    ).toBeNull();
    expect(
      guardHistorySeries({ points: [1, Number.NaN], source: "live", available: true }),
    ).toBeNull();
  });

  it("returns the live series when it has at least two finite points", () => {
    const series: HistorySeries = { points: [1, 4, 2], source: "live", available: true };
    expect(guardHistorySeries(series)).toBe(series);
  });
});

describe("GS-U1 statusStackModel", () => {
  it("keeps empty, zero, unavailable, and live distinct", () => {
    expect(statusStackModel(ORDER_STATUS_FUNNEL, null).mode).toBe("empty");
    expect(statusStackModel(ORDER_STATUS_FUNNEL, null).rows).toEqual([]);

    const zero = statusStackModel(ORDER_STATUS_FUNNEL, emptyOrderStatusCounts());
    expect(zero.mode).toBe("zero");
    expect(zero.rows).toHaveLength(17);
    expect(zero.rows.every((row) => row.count === 0 && row.share === 0)).toBe(true);

    const unavailable = statusStackModel(ORDER_STATUS_FUNNEL, { PENDING: 9 }, false);
    expect(unavailable.mode).toBe("unavailable");
    expect(unavailable.rows).toHaveLength(17);
    expect(unavailable.rows.every((row) => row.count === null)).toBe(true);

    const live = statusStackModel(ORDER_STATUS_FUNNEL, {
      ...emptyOrderStatusCounts(),
      PENDING: 2,
      COMPLETED: 2,
    });
    expect(live.mode).toBe("live");
    expect(live.total).toBe(4);
    expect(live.rows).toHaveLength(17);
    expect(live.rows.find((row) => row.key === "PENDING")?.share).toBe(0.5);
    expect(live.rows.find((row) => row.key === "LOADED")?.count).toBe(0);
  });
});

describe("GS-U2 increment + yesterday delta", () => {
  it("increments FISCAL_FAILED and keeps 17 funnel keys", () => {
    const next = incrementOrderStatusCount(emptyOrderStatusCounts(), "FISCAL_FAILED");
    expect(next.FISCAL_FAILED).toBe(1);
    expect(Object.keys(next)).toHaveLength(17);
    expect(incrementOrderStatusCount(next, "DISPATCHED").LOADED).toBe(1);
    expect(incrementOrderStatusCount(next, "UNKNOWN").FISCAL_FAILED).toBe(1);
  });

  it("does not invent a yesterday delta without a previous bucket", () => {
    expect(yesterdayRevenueDeltaPct([{ date: "2026-08-16", revenue_minor: 100 }], "2026-08-16")).toBeNull();
    expect(
      yesterdayRevenueDeltaPct(
        [
          { date: "2026-08-16", revenue_minor: 200 },
          { date: "2026-08-14", revenue_minor: 100 },
        ],
        "2026-08-16",
      ),
    ).toBeNull();
    expect(
      yesterdayRevenueDeltaPct(
        [
          { date: "2026-08-16", revenue_minor: 200 },
          { date: "2026-08-15", revenue_minor: 100 },
        ],
        "2026-08-16",
      ),
    ).toBe(100);
    expect(
      yesterdayRevenueDeltaPct(
        [
          { date: "2026-08-16", revenue_minor: 200 },
          { date: "2026-08-15", revenue_minor: 0 },
        ],
        "2026-08-16",
      ),
    ).toBeNull();
  });
});
