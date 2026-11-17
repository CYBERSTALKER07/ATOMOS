import { useCallback, useEffect, useState } from "react";
import { ApiError } from '@pegasusx/api-core';
import type { DashboardHistoryRange, HistorySeries } from "@pegasusx/types";
import {
  historySeriesFromValues,
  sliceDatedSeries,
  yesterdayRevenueDeltaPct,
} from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";

const api = createSupplierApi();

export type DashboardHistory = {
  range: DashboardHistoryRange;
  setRange: (range: DashboardHistoryRange) => void;
  velocity: HistorySeries;
  revenue: HistorySeries;
  revenueDeltaPct: number | null;
  loading: boolean;
};

const unavailable: HistorySeries = { points: [], source: "unavailable", available: false };

export function useDashboardHistory(): DashboardHistory {
  const [range, setRange] = useState<DashboardHistoryRange>("7d");
  const [velocity, setVelocity] = useState<HistorySeries>(unavailable);
  const [revenue, setRevenue] = useState<HistorySeries>(unavailable);
  const [revenueDeltaPct, setRevenueDeltaPct] = useState<number | null>(null);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async (nextRange: DashboardHistoryRange) => {
    setLoading(true);
    try {
      const [vel, rev] = await Promise.all([
        api.getSupplierAnalyticsVelocity(),
        api.getSupplierAnalyticsRevenue(),
      ]);
      const now = new Date();
      const velSlice = sliceDatedSeries(vel.points, nextRange, now);
      const revSlice = sliceDatedSeries(rev.series, nextRange, now);
      setVelocity(historySeriesFromValues(velSlice.map((p) => p.orders_completed), true));
      setRevenue(historySeriesFromValues(revSlice.map((p) => p.revenue_minor), true));
      setRevenueDeltaPct(yesterdayRevenueDeltaPct(rev.series, now.toISOString().slice(0, 10)));
    } catch (err) {
      const failed = err instanceof ApiError || err instanceof Error;
      setVelocity(historySeriesFromValues([], !failed));
      setRevenue(historySeriesFromValues([], !failed));
      setRevenueDeltaPct(null);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh(range);
  }, [range, refresh]);

  return { range, setRange, velocity, revenue, revenueDeltaPct, loading };
}
