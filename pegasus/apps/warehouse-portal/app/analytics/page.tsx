'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface AnalyticsData {
  period: string;
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
  const fmtCurrency = (n: number) => new Intl.NumberFormat('uz-UZ', { maximumFractionDigits: 0 }).format(n);

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div className="md-skeleton md-skeleton-title" />
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-card" />)}
        </div>
      </div>
    );
  }

  const d = data || { period: '30d', total_revenue: 0, total_orders: 0, avg_order_value: 0, top_products: [], daily: [], fleet_utilization_pct: 0 };
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
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Analytics</h1>
        <div className="flex gap-2">
          {['7d', '30d'].map(p => (
            <button
              key={p}
              onClick={() => { setPeriod(p); setLoading(true); }}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium ${p === period ? 'button--primary' : 'button--secondary'}`}
            >
              {p === '7d' ? '7 Days' : '30 Days'}
            </button>
          ))}
        </div>
      </div>

      {/* KPI Cards */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Revenue</div>
          <div className="text-2xl font-bold">{fmtCurrency(d.total_revenue)} UZS</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Orders</div>
          <div className="text-2xl font-bold">{fmt(d.total_orders)}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Avg Order Value</div>
          <div className="text-2xl font-bold">{fmtCurrency(d.avg_order_value)} UZS</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Fleet Utilization</div>
          <div className="text-2xl font-bold">{fleetUtilizationPct.toFixed(0)}%</div>
        </div>
      </div>

      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Icon name="refresh" size={16} className="text-[var(--accent)]" />
          <h2 className="text-sm font-semibold">Import Freshness</h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
          <div>
            <div className="text-xs text-[var(--muted)]">Rows Imported (30d)</div>
            <div className="text-xl font-bold">{fmt(importFreshness.applied_rows_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">SKUs Updated (30d)</div>
            <div className="text-xl font-bold">{fmt(importFreshness.applied_skus_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">Quantity Delta (30d)</div>
            <div className="text-xl font-bold">{fmt(importFreshness.quantity_delta_30d)}</div>
          </div>
        </div>
        <div className="mt-3 text-xs text-[var(--muted)]">
          Last Session: {importFreshness.last_session_id || 'N/A'} • {lastImportAppliedAt}
        </div>
      </div>

      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
        <div className="flex items-center gap-2 mb-3">
          <Icon name="warning" size={16} className="text-[var(--warning)]" />
          <h2 className="text-sm font-semibold">Import Anomaly Queue</h2>
        </div>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <div>
            <div className="text-xs text-[var(--muted)]">Open Anomaly Rows (30d)</div>
            <div className="text-xl font-bold">{fmt(importAnomalyQueue.open_rows_30d)}</div>
          </div>
          <div>
            <div className="text-xs text-[var(--muted)]">Affected Sessions (30d)</div>
            <div className="text-xl font-bold">{fmt(importAnomalyQueue.affected_sessions_30d)}</div>
          </div>
        </div>
        <div className="mt-3 text-xs text-[var(--muted)]">
          Last Session: {importAnomalyQueue.last_session_id || 'N/A'} • {lastAnomalyDetectedAt}
        </div>
        {importAnomalyQueue.last_detail ? (
          <div className="mt-2 text-xs text-[var(--muted)]">Latest Detail: {importAnomalyQueue.last_detail}</div>
        ) : null}
      </div>

      {/* Daily Revenue Chart */}
      {dailySeries.length > 0 && (
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <h2 className="text-sm font-semibold mb-4">Daily Revenue</h2>
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={dailySeries}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis dataKey="date" tick={{ fontSize: 11 }} stroke="var(--muted)" />
              <YAxis tick={{ fontSize: 11 }} stroke="var(--muted)" />
              <Tooltip />
              <Bar dataKey="revenue" fill="var(--accent)" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>
      )}

      {/* Top Products */}
      {d.top_products.length > 0 && (
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <h2 className="text-sm font-semibold mb-3">Top Products</h2>
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)]">
                  <th className="text-left py-2 px-3 font-medium">Product</th>
                  <th className="text-right py-2 px-3 font-medium">Units Sold</th>
                  <th className="text-right py-2 px-3 font-medium">Revenue (UZS)</th>
                </tr>
              </thead>
              <tbody>
                {d.top_products.map((p, i) => (
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
  );
}
