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
import type {
  SupplierDemandAccuracyResponse,
  SupplierDemandHistoryResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

function pctLabel(ratio: number | null | undefined): string {
  if (ratio == null || Number.isNaN(ratio)) return "—";
  return `${Math.round(ratio * 100)}%`;
}

export default function DemandAnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [refreshTick, setRefreshTick] = useState(0);
  useSupplierSessionReconcile(() => setRefreshTick(t => t + 1));
  const [error, setError] = useState<string | null>(null);
  const [summary, setSummary] = useState<SupplierDemandSummaryResponse | null>(null);
  const [history, setHistory] = useState<SupplierDemandHistoryResponse | null>(null);
  const [accuracy, setAccuracy] = useState<SupplierDemandAccuracyResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.getSupplierDemandToday(),
      api.getSupplierDemandHistory(),
      api.getSupplierDemandAccuracy({ days: 28 }),
    ])
      .then(([summaryResp, historyResp, accuracyResp]) => {
        if (cancelled) return;
        setSummary(summaryResp);
        setHistory(historyResp);
        setAccuracy(accuracyResp);
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

  return (
    <PageChrome
      icon="analytics"
      title="Demand forecast"
      description="Predicted versus actual order volume from supplier analytics authority."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex items-center gap-2">
          <Link href={"/analytics/demand/flywheel" as Route} className="md-btn md-btn-filled-tonal md-typescale-label-large px-4 py-2">
            POS flywheel
          </Link>
          <Link href={"/analytics/demand/signals" as Route} className="md-btn md-btn-filled-tonal md-typescale-label-large px-4 py-2">
            Manage Signals
          </Link>
          <Link href={"/analytics" as Route} className="md-btn md-btn-text">
            Back to analytics
          </Link>
        </div>
      }
    >
      {summary ? (
        <>
          <KpiStatGrid columns={3}>
            <KpiStatCard label="Predictions" value={summary.prediction_count} />
            <KpiStatCard label="Retailers" value={summary.total_retailers} />
            <KpiStatCard label="Forecast units" value={summary.total_pallets} />
          </KpiStatGrid>
          {accuracy?.enabled ? (
            <div className="mt-4">
              <KpiStatGrid columns={4}>
                <KpiStatCard label="28d forecast units" value={accuracy.forecast_units} />
                <KpiStatCard label="28d actual units" value={accuracy.actual_units} />
                <KpiStatCard
                  label="WAPE (28d)"
                  value={accuracy.actual_units > 0 ? pctLabel(accuracy.wape_28) : "—"}
                  sub={
                    accuracy.actual_units > 0
                      ? `Bias ${pctLabel(accuracy.bias_28)} · TS ${accuracy.tracking_signal.toFixed(2)}`
                      : "No scored actuals yet"
                  }
                />
                <KpiStatCard
                  label="Tracking alerts"
                  value={accuracy.alert_count}
                  sub={accuracy.alert_count > 0 ? "|TS| > 4 on one or more series" : "All series in control"}
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
          Baseline units from DemandForecastBaseline; actual units are completed order line quantities
          (SKU-day grain, rolled to supplier-day). Accuracy KPIs above are server WAPE / bias / tracking
          signal — not client-side MAPE.
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
