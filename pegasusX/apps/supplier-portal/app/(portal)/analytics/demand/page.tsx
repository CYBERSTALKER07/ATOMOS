"use client";

import Link from "next/link";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { Route } from "next";
import { useEffect, useMemo, useState } from "react";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import { createSupplierApi } from "@/lib/api";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
import type { SupplierDemandHistoryResponse, SupplierDemandSummaryResponse } from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

export default function DemandAnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [summary, setSummary] = useState<SupplierDemandSummaryResponse | null>(null);
  const [history, setHistory] = useState<SupplierDemandHistoryResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([api.getSupplierDemandToday(), api.getSupplierDemandHistory()])
      .then(([summaryResp, historyResp]) => {
        if (cancelled) return;
        setSummary(summaryResp);
        setHistory(historyResp);
      })
      .catch(() => {
        if (!cancelled) setError("load_demand_analytics_failed");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [refreshTick]);

  const chartData = useMemo(
    () =>
      (history?.time_series ?? []).map((point) => ({
        date: point.date.slice(5),
        baseline: point.predicted_qty,
        actual: point.actual_qty,
      })),
    [history],
  );

  const accuracy = useMemo(() => {
    const points = history?.time_series ?? [];
    if (points.length === 0) return null;
    let baselineTotal = 0;
    let actualTotal = 0;
    let absError = 0;
    let compareDays = 0;
    for (const point of points) {
      baselineTotal += point.predicted_qty;
      actualTotal += point.actual_qty;
      if (point.actual_qty > 0 || point.predicted_qty > 0) {
        compareDays += 1;
        absError += Math.abs(point.predicted_qty - point.actual_qty);
      }
    }
    const variancePct =
      actualTotal > 0 ? Math.round(((baselineTotal - actualTotal) / actualTotal) * 100) : null;
    const mapePct = actualTotal > 0 ? Math.round((absError / actualTotal) * 100) : null;
    return { baselineTotal, actualTotal, variancePct, mapePct, compareDays, days: points.length };
  }, [history]);

  return (
    <PageChrome
      icon="analytics"
      title="Demand forecast"
      description="Predicted versus actual order volume from supplier analytics authority."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <Link href={"/analytics" as Route} className="md-btn md-btn-text">
          Back to analytics
        </Link>
      }
    >
      {summary ? (
        <>
          <KpiStatGrid columns={3}>
            <KpiStatCard label="Predictions" value={summary.prediction_count} />
            <KpiStatCard label="Retailers" value={summary.total_retailers} />
            <KpiStatCard label="Forecast units" value={summary.total_pallets} />
          </KpiStatGrid>
          {accuracy ? (
            <div className="mt-4">
              <KpiStatGrid columns={3}>
                <KpiStatCard label="14d baseline units" value={accuracy.baselineTotal} />
                <KpiStatCard label="14d actual units" value={accuracy.actualTotal} />
                <KpiStatCard
                  label="Variance (baseline − actual)"
                  value={accuracy.variancePct == null ? "—" : `${accuracy.variancePct >= 0 ? "+" : ""}${accuracy.variancePct}%`}
                  sub={accuracy.mapePct == null ? "No completed orders in window" : `MAPE ${accuracy.mapePct}% over ${accuracy.compareDays}d`}
                />
              </KpiStatGrid>
            </div>
          ) : null}
          <div className="mt-6">
            <ForecastConfidenceCard
              confidence={forecastConfidenceFromDemand(summary)}
              updatedAt={formatForecastUpdatedAt(summary.generated_at)}
              stale={isForecastStale(summary.generated_at)}
            />
          </div>
        </>
      ) : null}

      <section className="desk-card p-6 mt-6">
        <h2 className="bento-card-title">Baseline vs actual (14d)</h2>
        <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
          Math-only v1: baseline units come from pending AI demand recommendations by day; actual units count
          completed retailer orders. No ML inference — compare coverage, not causal accuracy.
        </p>
        <div className="h-72 mt-4">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
              <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="baseline" name="Baseline" stroke="var(--desk-accent)" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="actual" name="Actual" stroke="var(--desk-success)" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      {history && history.upcoming.length > 0 ? (
        <section className="desk-card p-6 mt-6 overflow-x-auto">
          <h2 className="bento-card-title">Upcoming demand rows</h2>
          <table className="desk-table w-full mt-4">
            <thead>
              <tr style={{ color: "var(--desk-text-secondary)" }}>
                <th className="md-typescale-label-medium p-3 text-left font-medium">SKU</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Product</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">Qty</th>
                <th className="md-typescale-label-medium p-3 text-left font-medium">When</th>
              </tr>
            </thead>
            <tbody>
              {history.upcoming.slice(0, 25).map((row, index) => (
                <tr key={`${row.sku_id}-${index}`} style={{ borderTop: "1px solid var(--desk-border)" }}>
                  <td className="p-3 md-typescale-body-medium font-mono text-sm">{row.sku_id}</td>
                  <td className="p-3 md-typescale-body-medium">{row.product_name}</td>
                  <td className="p-3 md-typescale-body-medium">{row.predicted_qty}</td>
                  <td className="p-3 md-typescale-body-medium">{row.date || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
    </PageChrome>
  );
}
