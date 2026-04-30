'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts';
import type { SLAEntry } from '@/hooks/useAnalytics';

export default function SLAHealth({ data }: { data: SLAEntry[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm font-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        No SLA data
      </div>
    );
  }

  // Summary KPIs
  const totals = data.reduce(
    (acc, e) => ({
      onTime: acc.onTime + e.on_time,
      late: acc.late + e.late,
      breached: acc.breached + e.breached,
      total: acc.total + e.total_orders,
    }),
    { onTime: 0, late: 0, breached: 0, total: 0 },
  );
  const complianceRate = totals.total > 0 ? ((totals.onTime / totals.total) * 100) : 0;

  return (
    <div className="flex flex-col gap-3 h-full">
      <div className="flex items-center gap-2">
        <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
          SLA Compliance — 30 Days
        </h3>
        <span
          className="md-typescale-label-large ml-auto tabular-nums font-medium"
          style={{
            color: complianceRate >= 90
              ? 'var(--color-md-success)'
              : complianceRate >= 70
                ? 'var(--color-md-warning)'
                : 'var(--color-md-error)',
          }}
        >
          {complianceRate.toFixed(1)}%
        </span>
      </div>
      <div className="flex-1 min-h-[200px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-md-outline-variant)" vertical={false} />
            <XAxis
              dataKey="date"
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
              tickFormatter={(v) => v.slice(5)}
            />
            <YAxis
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
            <Bar dataKey="on_time" name="On-Time" stackId="sla" fill="oklch(0.723 0.219 149.579)" radius={[0, 0, 0, 0]} />
            <Bar dataKey="late" name="Late" stackId="sla" fill="oklch(0.769 0.188 70.08)" />
            <Bar dataKey="breached" name="Breached" stackId="sla" fill="oklch(0.577 0.245 27.325)" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
      {/* Summary row */}
      <div className="flex gap-4 md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span>On-Time: <strong className="tabular-nums">{totals.onTime.toLocaleString()}</strong></span>
        <span>Late: <strong className="tabular-nums">{totals.late.toLocaleString()}</strong></span>
        <span>Breached: <strong className="tabular-nums">{totals.breached.toLocaleString()}</strong></span>
      </div>
    </div>
  );
}
