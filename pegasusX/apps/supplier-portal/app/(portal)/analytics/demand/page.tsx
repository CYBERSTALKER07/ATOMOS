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
import { createSupplierApi } from "@/lib/api";
import type { SupplierDemandHistoryResponse, SupplierDemandSummaryResponse } from "@pegasusx/types";
import { PortalSurface } from "../../_components/PortalSurface";

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
    <PortalSurface
      title="Demand forecast"
      description="Predicted versus actual order volume from supplier analytics authority."
      loading={loading}
      error={error}
    >
      <p className="mb-4 md-typescale-body-medium">
        <Link href={"/analytics" as Route} className="text-[var(--color-md-primary)] underline">
          Back to analytics
        </Link>
      </p>

      {summary ? (
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
          <div className="md-card p-6">
            <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Predictions</p>
            <p className="md-typescale-display-small mt-2">{summary.prediction_count}</p>
          </div>
          <div className="md-card p-6">
            <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Retailers</p>
            <p className="md-typescale-display-small mt-2">{summary.total_retailers}</p>
          </div>
          <div className="md-card p-6">
            <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Forecast units</p>
            <p className="md-typescale-display-small mt-2">{summary.total_pallets}</p>
          </div>
        </div>
      ) : null}

      <section className="md-card p-6 mb-6">
        <h2 className="md-typescale-title-large">Predicted vs actual (14d)</h2>
        <div className="h-72 mt-4">
          <ResponsiveContainer width="100%" height="100%">
            <LineChart data={chartData}>
              <CartesianGrid stroke="var(--color-md-outline-variant)" strokeDasharray="3 3" />
              <XAxis dataKey="date" />
              <YAxis allowDecimals={false} />
              <Tooltip />
              <Legend />
              <Line type="monotone" dataKey="predicted" stroke="var(--color-md-primary)" strokeWidth={2} dot={false} />
              <Line type="monotone" dataKey="actual" stroke="var(--color-md-success)" strokeWidth={2} dot={false} />
            </LineChart>
          </ResponsiveContainer>
        </div>
      </section>

      {history && history.upcoming.length > 0 ? (
        <section className="md-card p-6 overflow-x-auto">
          <h2 className="md-typescale-title-large">Upcoming demand rows</h2>
          <table className="min-w-full text-left mt-4">
            <thead>
              <tr className="md-typescale-label-medium text-[var(--color-md-outline)]">
                <th className="py-2 pr-4">SKU</th>
                <th className="py-2 pr-4">Product</th>
                <th className="py-2 pr-4">Qty</th>
                <th className="py-2 pr-4">When</th>
              </tr>
            </thead>
            <tbody>
              {history.upcoming.slice(0, 25).map((row, index) => (
                <tr key={`${row.sku_id}-${index}`} className="md-typescale-body-medium border-t border-[var(--color-md-outline-variant)]">
                  <td className="py-2 pr-4 font-mono text-sm">{row.sku_id}</td>
                  <td className="py-2 pr-4">{row.product_name}</td>
                  <td className="py-2 pr-4">{row.predicted_qty}</td>
                  <td className="py-2 pr-4">{row.date || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </section>
      ) : null}
    </PortalSurface>
  );
}
