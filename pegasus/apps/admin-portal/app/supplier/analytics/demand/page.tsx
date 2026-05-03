'use client';

import Link from 'next/link';
import { useState, useEffect, useMemo, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { usePagination } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts';

interface TimeSeriesPoint {
  date: string;
  predicted: number;
  actual: number;
  predicted_qty: number;
  actual_qty: number;
}

interface UpcomingRow {
  date: string;
  retailer_name: string;
  sku_id: string;
  product_name: string;
  predicted_qty: number;
}

interface DemandHistory {
  time_series: TimeSeriesPoint[];
  upcoming: UpcomingRow[];
}

export default function DemandAnalyticsPage() {
  const [data, setData] = useState<DemandHistory | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const ts = useMemo(() => data?.time_series || [], [data]);
  const upcoming = useMemo(() => data?.upcoming || [], [data]);
  const upcomingPagination = usePagination(upcoming, 25);
  const avgAccuracy = useMemo(() => {
    const accuracyPairs = ts.filter((p) => p.predicted > 0 && p.actual > 0);
    if (accuracyPairs.length === 0) return 0;
    return Math.round(
      accuracyPairs.reduce((sum, p) => sum + Math.min(p.actual / p.predicted, 1) * 100, 0) / accuracyPairs.length,
    );
  }, [ts]);
  const formatAmount = useCallback((v: number) => new Intl.NumberFormat('uz-UZ').format(v), []);

  useEffect(() => {
    apiFetch('/v1/supplier/analytics/demand/history')
      .then(res => { if (!res.ok) throw new Error(`Demand history fetch failed: ${res.status}`); return res.json(); })
      .then(json => setData(json))
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="w-6 h-6 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="md-card md-card-elevated p-6" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>{error}</div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <div className="flex items-center gap-4 mb-8">
        <Link
          href="/supplier/analytics"
          className="w-10 h-10 rounded-full flex items-center justify-center transition-colors"
          style={{ background: 'var(--surface)' }}
        >
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"><path d="M19 12H5m7-7-7 7 7 7"/></svg>
        </Link>
        <div>
          <h1 className="md-typescale-headline-medium">AI Demand Analytics</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Predicted vs actual volume — 30-day window
          </p>
        </div>
      </div>

      {/* KPI Row */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="md-card md-card-elevated p-5">
          <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Prediction Accuracy</p>
          <p className="md-typescale-headline-small">{avgAccuracy}%</p>
          <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>last 30 days</p>
        </div>
        <div className="md-card md-card-elevated p-5">
          <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Upcoming AI Orders</p>
          <p className="md-typescale-headline-small">{upcoming.length}</p>
          <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>line items queued</p>
        </div>
        <div className="md-card md-card-elevated p-5">
          <p className="md-typescale-label-small mb-2" style={{ color: 'var(--muted)' }}>Data Points</p>
          <p className="md-typescale-headline-small">{ts.length}</p>
          <p className="md-typescale-label-small mt-1" style={{ color: 'var(--border)' }}>days tracked</p>
        </div>
      </div>

      {/* Dual-Axis Line Chart: Predicted vs Actual */}
      <div className="md-card md-card-outlined p-6 mb-8">
        <h2 className="md-typescale-title-small mb-6" style={{ color: 'var(--muted)' }}>
          Predicted Volume vs Actual Ordered Volume (Amount)
        </h2>
        {ts.length > 0 ? (
          <div className="h-80">
            <ResponsiveContainer width="100%" height="100%">
              <LineChart data={ts} margin={{ top: 5, right: 20, left: 20, bottom: 5 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
                <XAxis
                  dataKey="date"
                  tick={{ fontSize: 11, fill: 'var(--muted)' }}
                  tickLine={false}
                  axisLine={{ stroke: 'var(--border)' }}
                />
                <YAxis
                  yAxisId="amount"
                  tick={{ fontSize: 11, fill: 'var(--muted)' }}
                  tickLine={false}
                  axisLine={false}
                  tickFormatter={(v: number) => `${(v / 1000000).toFixed(1)}M`}
                />
                <YAxis
                  yAxisId="qty"
                  orientation="right"
                  tick={{ fontSize: 11, fill: 'var(--muted)' }}
                  tickLine={false}
                  axisLine={false}
                />
                <Tooltip
                  contentStyle={{
                    background: 'var(--surface)',
                    border: '1px solid var(--border)',
                    borderRadius: '12px',
                    fontSize: '12px',
                  }}
                  formatter={(value, name) => {
                    const v = Number(value ?? 0);
                    if (String(name).includes('Amount')) return [formatAmount(v) , String(name)];
                    return [v, String(name)];
                  }}
                />
                <Legend
                  wrapperStyle={{ fontSize: '11px', color: 'var(--muted)' }}
                />
                <Line
                  yAxisId="amount"
                  type="monotone"
                  dataKey="predicted"
                  name="Predicted (Amount)"
                  stroke="var(--accent)"
                  strokeWidth={2}
                  strokeDasharray="6 3"
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  yAxisId="amount"
                  type="monotone"
                  dataKey="actual"
                  name="Actual (Amount)"
                  stroke="var(--muted)"
                  strokeWidth={2}
                  dot={false}
                  activeDot={{ r: 4 }}
                />
                <Line
                  yAxisId="qty"
                  type="monotone"
                  dataKey="predicted_qty"
                  name="Predicted Qty"
                  stroke="var(--muted)"
                  strokeWidth={1.5}
                  strokeDasharray="4 2"
                  dot={false}
                />
                <Line
                  yAxisId="qty"
                  type="monotone"
                  dataKey="actual_qty"
                  name="Actual Qty"
                  stroke="var(--danger)"
                  strokeWidth={1.5}
                  dot={false}
                />
              </LineChart>
            </ResponsiveContainer>
          </div>
        ) : (
          <div className="flex items-center justify-center h-64" style={{ color: 'var(--muted)' }}>
            <p className="md-typescale-body-medium">No time-series data available yet</p>
          </div>
        )}
      </div>

      {/* Upcoming AI Orders Data Grid */}
      <div className="md-card md-card-outlined p-0 overflow-hidden">
        <div className="px-6 py-4" style={{ borderBottom: '1px solid var(--border)' }}>
          <h2 className="md-typescale-title-small" style={{ color: 'var(--muted)' }}>
            Upcoming AI-Planned Orders
          </h2>
        </div>
        {upcoming.length > 0 ? (
          <>
          <table className="md-table">
            <thead>
              <tr>
                <th>Date</th>
                <th>Retailer</th>
                <th>SKU</th>
                <th>Product</th>
                <th className="text-right">Predicted Qty</th>
              </tr>
            </thead>
            <tbody>
              {upcomingPagination.pageItems.map((row, i) => (
                <tr key={i} className="transition-colors">
                  <td className="md-typescale-body-small font-mono" style={{ color: 'var(--muted)' }}>{row.date}</td>
                  <td className="md-typescale-body-small">{row.retailer_name}</td>
                  <td className="md-typescale-body-small font-mono" style={{ color: 'var(--muted)' }}>{row.sku_id}</td>
                  <td className="md-typescale-body-small">{row.product_name || '—'}</td>
                  <td className="text-right md-typescale-body-small font-semibold">{row.predicted_qty}</td>
                </tr>
              ))}
            </tbody>
          </table>
          <PaginationControls pagination={upcomingPagination} />
          </>
        ) : (
          <div className="flex items-center justify-center h-32" style={{ color: 'var(--muted)' }}>
            <p className="md-typescale-body-medium">No upcoming AI orders for your catalog</p>
          </div>
        )}
      </div>
    </div>
  );
}
