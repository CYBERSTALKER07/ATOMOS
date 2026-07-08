'use client';

import {
  RadialBarChart,
  RadialBar,
  ResponsiveContainer,
  PolarAngleAxis,
} from 'recharts';

type VelocityGaugeProps = {
  className?: string;
};

// Removed Mock Data
// We would pass this via props in production
const DEFAULT_VELOCITY_DATA = [
  { name: 'Avg Dispatch', value: 0, fill: 'var(--desk-accent)' }
];

export default function VelocityGauge({ className }: VelocityGaugeProps) {
  return (
    <div className={className} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
      <ResponsiveContainer width="100%" height={250}>
        <RadialBarChart 
          cx="50%" 
          cy="70%" 
          innerRadius="70%" 
          outerRadius="100%" 
          barSize={20} 
          data={DEFAULT_VELOCITY_DATA}
          startAngle={180} 
          endAngle={0}
        >
          <PolarAngleAxis
            type="number"
            domain={[0, 100]}
            angleAxisId={0}
            tick={false}
          />
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
            {DEFAULT_VELOCITY_DATA[0].value}
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
