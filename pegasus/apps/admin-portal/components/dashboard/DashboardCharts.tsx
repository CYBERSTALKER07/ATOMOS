"use client";

/**
 * Dashboard Charts — extracted from app/page.tsx so recharts is dynamically
 * imported and excluded from the dashboard's first-paint bundle.
 */

import {
  PieChart, Pie, Cell, ResponsiveContainer,
  BarChart, Bar, XAxis, YAxis, Tooltip,
} from "recharts";

type PipelineDatum = { name: string; value: number };
type RevDatum = { gateway: string; amount: number };

const tooltipStyle = {
  background: 'var(--surface)',
  border: '1px solid var(--border)',
  borderRadius: 0,
  fontSize: 13,
  color: 'var(--foreground)',
};

export function PipelinePieChart({ data, shades }: { data: PipelineDatum[]; shades: string[] }) {
  return (
    <ResponsiveContainer width="50%" height="100%">
      <PieChart>
        <Pie
          data={data}
          innerRadius={55}
          outerRadius={80}
          paddingAngle={3}
          dataKey="value"
          strokeWidth={0}
        >
          {data.map((_, i) => (
            <Cell key={i} fill={shades[i % shades.length]} />
          ))}
        </Pie>
        <Tooltip contentStyle={tooltipStyle} />
      </PieChart>
    </ResponsiveContainer>
  );
}

export function RevenueBarChart({ data, shades }: { data: RevDatum[]; shades: string[] }) {
  return (
    <ResponsiveContainer width="100%" height="100%" minHeight={140}>
      <BarChart data={data} barSize={40}>
        <XAxis
          dataKey="gateway"
          tick={{ fill: 'var(--muted)', fontSize: 12 }}
          axisLine={false}
          tickLine={false}
        />
        <YAxis
          tick={{ fill: 'var(--muted)', fontSize: 11 }}
          axisLine={false}
          tickLine={false}
          tickFormatter={(v: number) =>
            v >= 1000000 ? `${(v / 1000000).toFixed(1)}M` : v >= 1000 ? `${(v / 1000).toFixed(0)}K` : String(v)
          }
        />
        <Tooltip
          contentStyle={tooltipStyle}
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          formatter={(value: any) => [`${Number(value ?? 0).toLocaleString()}`, 'Revenue']}
        />
        <Bar dataKey="amount" radius={[0, 0, 0, 0]}>
          {data.map((_, i) => (
            <Cell key={i} fill={shades[i % shades.length]} />
          ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}
