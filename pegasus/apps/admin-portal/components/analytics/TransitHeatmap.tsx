'use client';

import type { TransitPoint } from '@/hooks/useAnalytics';

export default function TransitHeatmap({ data }: { data: TransitPoint[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm font-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        No active transit data
      </div>
    );
  }

  // Group by state for legend counts
  const stateCounts: Record<string, number> = {};
  for (const p of data) {
    stateCounts[p.state] = (stateCounts[p.state] || 0) + p.count;
  }

  const stateColors: Record<string, string> = {
    PENDING: 'var(--color-md-warning)',
    LOADED: 'var(--color-md-info)',
    IN_TRANSIT: 'var(--color-md-primary)',
    ARRIVED: 'var(--color-md-success)',
  };

  return (
    <div className="flex flex-col gap-4 h-full">
      <div className="flex items-center gap-1">
        <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
          Transit Density
        </h3>
        <span className="md-typescale-label-small ml-auto" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          {data.length} clusters
        </span>
      </div>
      <div className="flex flex-wrap gap-3">
        {Object.entries(stateCounts).map(([state, count]) => (
          <div key={state} className="flex items-center gap-2">
            <span
              className="inline-block w-3 h-3 md-shape-full"
              style={{ background: stateColors[state] || 'var(--color-md-outline)' }}
            />
            <span className="md-typescale-label-medium" style={{ color: 'var(--color-md-on-surface)' }}>
              {state.replace(/_/g, ' ')}
            </span>
            <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {count}
            </span>
          </div>
        ))}
      </div>
      {/* Grid-based mini heatmap — full MapLibre integration is a follow-up */}
      <div className="flex-1 grid grid-cols-4 gap-1 min-h-[120px]">
        {data.slice(0, 16).map((p, i) => (
          <div
            key={i}
            className="md-shape-sm flex items-center justify-center md-typescale-label-small"
            style={{
              background: stateColors[p.state] || 'var(--color-md-surface-container)',
              color: 'var(--color-md-on-primary)',
              opacity: Math.min(0.4 + (p.count / Math.max(...data.map(d => d.count))) * 0.6, 1),
            }}
          >
            {p.count}
          </div>
        ))}
      </div>
    </div>
  );
}
