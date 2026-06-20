"use client";

import Link from "next/link";
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
import { createSupplierApi } from "@/lib/api";
import type { SupplierDemandHistoryResponse, SupplierDemandSummaryResponse } from "@pegasusx/types";
import { PageChrome } from '@/components/PageChrome';

const api = createSupplierApi();

export default function DemandAnalyticsPage() {
  const [loading, setLoading] = useState(true);
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
  }, []);

  const chartData = useMemo(
    () =>
      (history?.time_series ?? []).map((point) => ({
        date: point.date.slice(5),
        predicted: point.predicted_qty,
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
        <Link href={"/analytics" as Route} className="md-btn md-btn-text">
          Back to analytics
        </Link>
      }
    >
      {summary ? (
        <KpiStatGrid columns={3}>
          <KpiStatCard label="Predictions" value={summary.prediction_count} />
          <KpiStatCard label="Retailers" value={summary.total_retailers} />
          <KpiStatCard label="Forecast units" value={summary.total_pallets} />
        </KpiStatGrid>
      ) : null}

      <section className="desk-card p-6 mt-6">
        <h2 className="bento-card-title">Predicted vs actual (14d)</h2>
        <div className="h-72 mt-4">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
              <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
              <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="predicted" stroke="var(--desk-accent)" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="actual" stroke="var(--desk-success)" strokeWidth={2} dot={false} />
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
