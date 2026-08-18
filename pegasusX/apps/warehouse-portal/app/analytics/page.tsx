'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';
import AnalyticsChartGrid from '@/components/analytics/AnalyticsChartGrid';
import { moneyCurrency } from '@pegasusx/api-client';
// VelocityGauge unmounted — no avg-dispatch SoT on warehouse ops analytics

interface AnalyticsData {
  period: string;
  currency?: string;
  total_revenue: number;
  total_orders: number;
  avg_order_value: number;
  top_products: { product_name: string; total_sold?: number; total_qty?: number; revenue: number }[];
  daily: { date: string; revenue: number; orders: number }[];
  daily_breakdown?: { date: string; revenue: number; orders: number }[];
  fleet_utilization_pct: number;
  fleet_utilization?: { utilization_pct: number };
  import_freshness?: {
    applied_rows_30d: number;
    applied_skus_30d: number;
    quantity_delta_30d: number;
    last_session_id?: string;
    last_applied_at?: string;
  };
  import_anomaly_queue?: {
    open_rows_30d: number;
    affected_sessions_30d: number;
    last_session_id?: string;
    last_detected_at?: string;
    last_detail?: string;
  };
}

export default function AnalyticsPage() {
  const t = usePortalT();
  const [data, setData] = useState<AnalyticsData | null>(null);
  const [loading, setLoading] = useState(true);
  const [period, setPeriod] = useState('30d');

  const load = useCallback(async () => {
    try {
      const res = await apiFetch(`/v1/warehouse/ops/analytics?period=${period}`);
      if (res.ok) setData(await res.json());
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [period]);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);
  const packCode = moneyCurrency(data?.currency);
  const fmtCurrency = (n: number) => {
    const formatted = new Intl.NumberFormat('uz-UZ', { maximumFractionDigits: 0 }).format(n);
    return packCode ? `${formatted} ${packCode}` : formatted;
  };

  const d = data || { period: '30d', total_revenue: 0, total_orders: 0, avg_order_value: 0, top_products: [], daily: [], fleet_utilization_pct: 0 };
  const topProducts = d.top_products ?? [];
  const dailySeries = d.daily_breakdown || d.daily || [];
  const fleetUtilizationPct = d.fleet_utilization?.utilization_pct ?? d.fleet_utilization_pct ?? 0;
  const importFreshness = d.import_freshness || {
    applied_rows_30d: 0,
    applied_skus_30d: 0,
    quantity_delta_30d: 0,
    last_session_id: '',
    last_applied_at: '',
  };
  const parsedImportTime = importFreshness.last_applied_at ? new Date(importFreshness.last_applied_at) : null;
  const lastImportAppliedAt = parsedImportTime && !Number.isNaN(parsedImportTime.getTime())
    ? parsedImportTime.toLocaleString('uz-UZ')
    : 'No imports applied yet';
  const importAnomalyQueue = d.import_anomaly_queue || {
    open_rows_30d: 0,
    affected_sessions_30d: 0,
    last_session_id: '',
    last_detected_at: '',
    last_detail: '',
  };
  const parsedAnomalyTime = importAnomalyQueue.last_detected_at ? new Date(importAnomalyQueue.last_detected_at) : null;
  const lastAnomalyDetectedAt = parsedAnomalyTime && !Number.isNaN(parsedAnomalyTime.getTime())
    ? parsedAnomalyTime.toLocaleString('uz-UZ')
    : 'No anomalies detected';

  return (
    <PageTransition>
      <PageChrome
        icon="analytics"
        title={t("portal.nav.analytics")}
        description={t("warehouse_portal.residual.text.warehouse_revenue_order_health_fleet_utilization_and_import_tele")}
        loading={loading}
        actions={
          <div className="flex gap-2">
            {['7d', '30d'].map(p => (
              <button
                key={p}
                type="button"
                onClick={() => { setPeriod(p); setLoading(true); }}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium ${p === period ? 'button--primary' : 'button--secondary'}`}
              >
                {p === '7d' ? '7 Days' : '30 Days'}
              </button>
            ))}
          </div>
        }
      >
      <div className="space-y-6">
      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.analytics.text.revenue")}</div>
          <div className="text-2xl font-light">{fmtCurrency(d.total_revenue)}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("portal.nav.orders")}</div>
          <div className="text-2xl font-light">{fmt(d.total_orders)}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.analytics.text.avg_order_value")}</div>
          <div className="text-2xl font-light">{fmtCurrency(d.avg_order_value)}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.analytics.text.fleet_utilization")}</div>
          <div className="text-2xl font-light">{fleetUtilizationPct.toFixed(0)}%</div>
        </div>
      </div>

      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Icon name="refresh" size={16} className="text-[var(--accent)]" />
          <h2 className="text-sm font-semibold">{t("warehouse_portal.analytics.text.import_freshness")}</h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <div className="text-xs text-[var(--muted)]">{t("warehouse_portal.analytics.text.rows_imported_30d")}</div>
            <div className="text-xl font-light">{fmt(importFreshness.applied_rows_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">{t("warehouse_portal.analytics.text.skus_updated_30d")}</div>
            <div className="text-xl font-light">{fmt(importFreshness.applied_skus_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">{t("warehouse_portal.analytics.text.quantity_delta_30d")}</div>
            <div className="text-xl font-light">{fmt(importFreshness.quantity_delta_30d)}</div>
          </div>
        </div>
        <div className="mt-3 text-xs text-[var(--muted)]">
          Last Session: {importFreshness.last_session_id || 'N/A'} • {lastImportAppliedAt}
        </div>
      </div>

      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Icon name="warning" size={16} className="text-[var(--warning)]" />
          <h2 className="text-sm font-semibold">{t("warehouse_portal.analytics.text.import_anomaly_queue")}</h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div className="text-xs text-[var(--muted)]">{t("warehouse_portal.analytics.text.open_anomaly_rows_30d")}</div>
            <div className="text-xl font-light">{fmt(importAnomalyQueue.open_rows_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">{t("warehouse_portal.analytics.text.affected_sessions_30d")}</div>
            <div className="text-xl font-light">{fmt(importAnomalyQueue.affected_sessions_30d)}</div>
          </div>
        </div>
        <div className="mt-3 text-xs text-[var(--muted)]">
          Last Session: {importAnomalyQueue.last_session_id || 'N/A'} • {lastAnomalyDetectedAt}
        </div>
        {importAnomalyQueue.last_detail ? (
          <div className="mt-2 text-xs text-[var(--muted)]">Latest Detail: {importAnomalyQueue.last_detail}</div>
        ) : null}
      </div>

      {/* Analytics Charts */}
      <AnalyticsChartGrid dailySeries={dailySeries} fmtCurrency={fmtCurrency} />

      {/* Top Products */}
      {topProducts.length > 0 && (
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <h2 className="text-sm font-semibold mb-3">{t("warehouse_portal.analytics.text.top_products")}</h2>
          <div className="overflow-x-auto">
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)]">
                  <th className="text-left py-2 px-3 font-medium">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                  <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.analytics.text.units_sold")}</th>
                  <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.analytics.text.revenue_uzs")}</th>
                </tr>
              </thead>
              <tbody>
                {topProducts.map((p, i) => (
                  <tr key={i} className="border-b border-[var(--border)]">
                    <td className="py-2 px-3">{p.product_name}</td>
                    <td className="py-2 px-3 text-right font-mono">{fmt(p.total_sold ?? p.total_qty ?? 0)}</td>
                    <td className="py-2 px-3 text-right font-mono">{fmtCurrency(p.revenue)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </div>
      )}
      </div>
      </PageChrome>
    </PageTransition>
  );
}
