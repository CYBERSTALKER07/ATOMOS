'use client';

import {
  RadialBarChart,
  RadialBar,
  ResponsiveContainer,
  PolarAngleAxis,
} from 'recharts';

type VelocityGaugeProps = {
  className?: string;
  /** Average dispatch minutes from SoT; omit/undefined = no data (honest empty). */
  avgDispatchMinutes?: number | null;
};

export default function VelocityGauge({ className, avgDispatchMinutes }: VelocityGaugeProps) {
  if (avgDispatchMinutes == null || Number.isNaN(avgDispatchMinutes)) {
    return (
      <div
        className={className}
        style={{
          display: 'flex',
          flexDirection: 'column',
          alignItems: 'center',
          justifyContent: 'center',
          minHeight: 250,
          color: 'var(--desk-text-secondary)',
        }}
      >
        <p className="md-typescale-body-medium">No dispatch velocity data</p>
      </div>
    );
  }

  const data = [
    { name: 'Avg Dispatch', value: Math.max(0, Math.min(100, avgDispatchMinutes)), fill: 'var(--desk-accent)' },
  ];

  return (
    <div className={className} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
      <ResponsiveContainer width="100%" height={250}>
        <RadialBarChart
          cx="50%"
          cy="70%"
          innerRadius="70%"
          outerRadius="100%"
          barSize={20}
          data={data}
          startAngle={180}
          endAngle={0}
        >
          <PolarAngleAxis type="number" domain={[0, 100]} angleAxisId={0} tick={false} />
          <RadialBar
            background={{ fill: 'var(--color-md-surface-container-high)' }}
            dataKey="value"
            cornerRadius={10}
          />
          <text
            x="50%"
            y="65%"
            textAnchor="middle"
            dominantBaseline="middle"
            className="md-typescale-display-small"
            style={{ fill: 'var(--desk-text-primary)' }}
          >
            {avgDispatchMinutes}
          </text>
          <text
            x="50%"
            y="75%"
            textAnchor="middle"
            dominantBaseline="middle"
            className="md-typescale-label-large"
            style={{ fill: 'var(--desk-text-secondary)' }}
          >
            min / dispatch
          </text>
        </RadialBarChart>
      </ResponsiveContainer>
    </div>
  );
}
