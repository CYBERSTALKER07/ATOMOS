'use client';

import {
  AreaChart, Area, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid, Legend,
} from 'recharts';
import type { RevenueDayBucket } from '@/hooks/useAdvancedAnalytics';

interface Props { data: RevenueDayBucket[] }

export default function RevenueChart({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No revenue data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Revenue Breakdown
      </h3>
      <div className="flex-1 min-h-[200px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
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
              tickFormatter={(v) => v >= 1000 ? `${(v / 1000).toFixed(0)}k` : v}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
              formatter={(value: any) => value.toLocaleString()}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
            <Area type="monotone" dataKey="payme" name="Payme" stroke="oklch(0.6 0.2 240)" fill="oklch(0.6 0.2 240 / 0.12)" strokeWidth={2} stackId="1" />
            <Area type="monotone" dataKey="click" name="Click" stroke="oklch(0.65 0.2 150)" fill="oklch(0.65 0.2 150 / 0.12)" strokeWidth={2} stackId="1" />
            <Area type="monotone" dataKey="card" name="Card" stroke="oklch(0.6 0.15 300)" fill="oklch(0.6 0.15 300 / 0.1)" strokeWidth={2} stackId="1" />
            <Area type="monotone" dataKey="cash" name="Cash" stroke="oklch(0.65 0.12 60)" fill="oklch(0.65 0.12 60 / 0.1)" strokeWidth={2} stackId="1" />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
