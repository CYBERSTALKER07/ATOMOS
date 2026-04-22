'use client';

import {
  LineChart, Line, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts';
import type { ThroughputBucket } from '@/hooks/useAdvancedAnalytics';

interface Props { data: ThroughputBucket[] }

export default function ThroughputLine({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No throughput data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Order Throughput
      </h3>
      <div className="flex-1 min-h-[200px]">
        <ResponsiveContainer width="100%" height="100%">
          <LineChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
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
            <Line type="monotone" dataKey="order_count" name="Total" stroke="var(--color-md-primary)" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="completed_count" name="Completed" stroke="oklch(0.723 0.219 149.579)" strokeWidth={2} dot={false} />
            <Line type="monotone" dataKey="cancelled_count" name="Cancelled" stroke="oklch(0.577 0.245 27.325)" strokeWidth={1.5} dot={false} strokeDasharray="4 4" />
          </LineChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
