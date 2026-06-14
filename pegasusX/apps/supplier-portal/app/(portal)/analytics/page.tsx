"use client";

import Link from "next/link";
import type { Route } from "next";
import { useEffect, useMemo, useState } from "react";
import {
  CartesianGrid,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { createSupplierApi } from "@/lib/api";
import type {
  SupplierAnalyticsRevenueResponse,
  SupplierAnalyticsVelocityResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

function formatMoney(minor: number, currency: string) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: "currency",
      currency,
      maximumFractionDigits: 0,
    }).format(minor / 100);
  } catch {
    return `${minor} ${currency}`;
  }
}

export default function AnalyticsPage() {
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [velocity, setVelocity] = useState<SupplierAnalyticsVelocityResponse | null>(null);
  const [revenue, setRevenue] = useState<SupplierAnalyticsRevenueResponse | null>(null);
  const [demand, setDemand] = useState<SupplierDemandSummaryResponse | null>(null);

  useEffect(() => {
    let cancelled = false;
    Promise.all([
      api.getSupplierAnalyticsVelocity(),
      api.getSupplierAnalyticsRevenue(),
      api.getSupplierDemandToday(),
    ])
      .then(([velocityResp, revenueResp, demandResp]) => {
        if (cancelled) return;
        setVelocity(velocityResp);
        setRevenue(revenueResp);
        setDemand(demandResp);
      })
      .catch(() => {
        if (!cancelled) setError("load_supplier_analytics_failed");
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const velocityChart = useMemo(
    () =>
      (velocity?.points ?? []).map((point) => ({
        date: point.date.slice(5),
        created: point.orders_created,
        completed: point.orders_completed,
      })),
    [velocity],
  );

  const revenueChart = useMemo(
    () =>
      (revenue?.series ?? []).map((point) => ({
        date: point.date.slice(5),
        revenue: point.revenue_minor,
      })),
    [revenue],
  );

  return (
    <PortalSurface
      title="Analytics"
      description="Order velocity, revenue trend, and near-term demand signals."
      loading={loading}
      error={error}
    >
      <div className="flex flex-wrap gap-3 mb-6">
        <Link href={"/analytics/demand" as Route} className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2">
          Demand forecast
        </Link>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-6">
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">30-day revenue</p>
          <p className="md-typescale-display-small mt-2">
            {revenue ? formatMoney(revenue.total_minor, revenue.currency) : "—"}
          </p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Demand predictions</p>
          <p className="md-typescale-display-small mt-2">{demand?.prediction_count ?? 0}</p>
        </div>
        <div className="md-card p-6">
          <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Forecast units (24h)</p>
          <p className="md-typescale-display-small mt-2">{demand?.total_pallets ?? 0}</p>
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        <section className="md-card p-6">
          <h2 className="md-typescale-title-large">Order velocity (7d)</h2>
          <div className="h-64 mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={velocityChart}>
                <CartesianGrid stroke="var(--color-md-outline-variant)" strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis allowDecimals={false} />
                <Tooltip />
                <Line type="monotone" dataKey="created" stroke="var(--color-md-primary)" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey="completed" stroke="var(--color-md-success)" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </section>

        <section className="md-card p-6">
          <h2 className="md-typescale-title-large">Revenue trend (30d)</h2>
          <div className="h-64 mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={revenueChart}>
                <CartesianGrid stroke="var(--color-md-outline-variant)" strokeDasharray="3 3" />
                <XAxis dataKey="date" />
                <YAxis allowDecimals={false} />
                <Tooltip />
                <Line type="monotone" dataKey="revenue" stroke="var(--color-md-primary)" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </section>
      </div>

      {demand && demand.items.length > 0 ? (
        <section className="md-card p-6 mt-6">
          <h2 className="md-typescale-title-large">Top demand SKUs (today)</h2>
          <ul className="mt-4 divide-y divide-[var(--color-md-outline-variant)]">
            {demand.items.slice(0, 5).map((item) => (
              <li key={item.sku_id} className="py-3 md-typescale-body-medium flex justify-between gap-4">
                <span>{item.product_name || item.sku_id}</span>
                <span className="text-[var(--color-md-outline)]">{item.total_qty} units</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}
    </PortalSurface>
  );
}
