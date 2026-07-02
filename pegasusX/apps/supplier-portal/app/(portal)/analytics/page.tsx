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
import Icon from "@/components/Icon";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { createSupplierApi } from "@/lib/api";
import type {
  SupplierAnalyticsRevenueResponse,
  SupplierAnalyticsVelocityResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import PlanningBrainPanel from "@/components/PlanningBrainPanel";
import RevenueHeatmap from "./RevenueHeatmap";

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
    <PageChrome
      icon="analytics"
      title="Analytics"
      description="Financial overview and operational intelligence."
      loading={loading}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex flex-wrap items-center gap-3">
          <Link href={"/analytics/demand" as Route} className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2">
            Demand forecast
          </Link>
          <Link href={"/dispatch" as Route} className="md-btn md-btn-filled md-typescale-label-large px-4 py-2 inline-flex items-center gap-2">
            <Icon name="dispatch" size={18} />
            Dispatch room
          </Link>
        </div>
      }
    >
      {demand && demand.prediction_count > 0 ? (
        <div
          className="desk-card p-6 mb-6"
          style={{
            background: "color-mix(in srgb, var(--desk-accent) 12%, var(--desk-surface))",
            borderColor: "color-mix(in srgb, var(--desk-accent) 28%, var(--desk-border))",
          }}
        >
          <div className="flex flex-wrap items-start justify-between gap-4 mb-4">
            <div className="flex items-center gap-3">
              <div
                className="w-10 h-10 rounded-full flex items-center justify-center shrink-0"
                style={{ background: "var(--desk-accent)", color: "var(--desk-accent-on)" }}
              >
                <Icon name="overview" size={20} />
              </div>
              <div>
                <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-primary)" }}>
                  AI future demand (next 24h)
                </h2>
                <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                  Empathy engine predictions for your catalog
                </p>
              </div>
            </div>
            <span
              className="md-chip h-7 px-3"
              style={{
                background: "var(--desk-accent)",
                color: "var(--desk-accent-on)",
                borderColor: "transparent",
              }}
            >
              {demand.prediction_count} predictions
            </span>
          </div>

          <KpiStatGrid columns={3}>
            <KpiStatCard label="Retailers" value={demand.total_retailers} />
            <KpiStatCard label="Total pallets" value={demand.total_pallets.toLocaleString()} />
            <KpiStatCard
              label="Forecast value"
              value={new Intl.NumberFormat("uz-UZ").format(demand.total_value)}
            />
          </KpiStatGrid>

          {demand.items.length > 0 ? (
            <div className="flex flex-wrap gap-2 mt-4">
              {demand.items.slice(0, 5).map((item) => (
                <span
                  key={item.sku_id}
                  className="md-chip h-7 px-3 text-xs"
                  style={{ background: "var(--desk-surface-raised)", color: "var(--desk-text-primary)" }}
                >
                  {item.total_qty}× {item.product_name || item.sku_id}
                </span>
              ))}
              {demand.items.length > 5 ? (
                <span className="md-chip h-7 px-3 text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                  +{demand.items.length - 5} more
                </span>
              ) : null}
            </div>
          ) : null}

          <Link
            href={"/analytics/demand" as Route}
            className="md-btn md-btn-filled inline-flex items-center gap-2 mt-5"
          >
            View demand forecast
            <Icon name="right" size={16} />
          </Link>
        </div>
      ) : null}

      <KpiStatGrid columns={3}>
        <KpiStatCard
          label="30-day revenue"
          value={revenue ? formatMoney(revenue.total_minor, revenue.currency) : "—"}
        />
        <KpiStatCard label="Demand predictions" value={demand?.prediction_count ?? 0} />
        <KpiStatCard label="Forecast units (24h)" value={demand?.total_pallets ?? 0} />
      </KpiStatGrid>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 mt-6">
        <section className="desk-card p-6">
          <h2 className="bento-card-title">Order velocity (7d)</h2>
          <div className="h-64 mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={velocityChart}>
                <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
                <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                <Tooltip />
                <Line type="monotone" dataKey="created" stroke="var(--desk-accent)" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey="completed" stroke="var(--desk-success)" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </section>

        <section className="desk-card p-6">
          <h2 className="bento-card-title">Revenue trend (30d)</h2>
          <div className="h-64 mt-4">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={revenueChart}>
                <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
                <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                <Tooltip />
                <Line type="monotone" dataKey="revenue" stroke="var(--desk-accent)" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          </div>
        </section>
      </div>

      {demand && demand.items.length > 0 ? (
        <section className="desk-card p-6 mt-6">
          <h2 className="bento-card-title">Top demand SKUs (today)</h2>
          <ul className="mt-4 divide-y" style={{ borderColor: "var(--desk-border)" }}>
            {demand.items.slice(0, 8).map((item) => (
              <li
                key={item.sku_id}
                className="py-3 md-typescale-body-medium flex justify-between gap-4"
                style={{ borderColor: "var(--desk-border)" }}
              >
                <span style={{ color: "var(--desk-text-primary)" }}>{item.product_name || item.sku_id}</span>
                <span style={{ color: "var(--desk-text-secondary)" }}>{item.total_qty} units</span>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      <section className="desk-card p-0 mt-6 overflow-hidden min-h-[320px]">
        <PlanningBrainPanel />
      </section>

      <section className="desk-card p-0 mt-6 overflow-hidden h-[400px]">
        <h2 className="bento-card-title p-6 pb-0">Revenue spatial distribution</h2>
        <RevenueHeatmap className="w-full h-full p-4" />
      </section>
    </PageChrome>
  );
}
