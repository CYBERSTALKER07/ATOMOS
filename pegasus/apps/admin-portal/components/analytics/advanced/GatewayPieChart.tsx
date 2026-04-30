'use client';

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import type { GatewayBreakdown } from '@/hooks/useAdvancedAnalytics';

const COLORS = [
  'oklch(0.6 0.2 240)',   // Payme
  'oklch(0.65 0.2 150)',  // Click
  'oklch(0.6 0.15 300)',  // Card
  'oklch(0.65 0.12 60)',  // Cash
  'oklch(0.5 0.15 200)',  // Other
];

interface Props { data: GatewayBreakdown[] }

export default function GatewayPieChart({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No gateway data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Payment Gateways
      </h3>
      <div className="flex-1 min-h-[180px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              dataKey="total"
              nameKey="gateway"
              cx="50%"
              cy="50%"
              innerRadius="40%"
              outerRadius="70%"
              paddingAngle={2}
            >
              {data.map((_, i) => (
                <Cell key={i} fill={COLORS[i % COLORS.length]} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
              // eslint-disable-next-line @typescript-eslint/no-explicit-any
              formatter={(value: any) => value.toLocaleString()}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}
