'use client';

import type { ForecastConfidence } from '@pegasusx/types';
import {
  formatSourceBadge,
  isForecastBlocked,
  isSeasonalTemplateActive,
} from '@/lib/forecast-confidence';

type Props = {
  confidence: ForecastConfidence;
  compact?: boolean;
};

function confidenceColor(pct?: number): string {
  if (pct == null) return 'text-[var(--muted)]';
  if (pct >= 80) return 'text-[var(--success)]';
  if (pct >= 60) return 'text-[var(--warning)]';
  return 'text-[var(--danger)]';
}

export function ForecastConfidenceView({ confidence, compact = false }: Props) {
  const blocked = isForecastBlocked(confidence);
  const seasonal = isSeasonalTemplateActive(confidence);
  const low = confidence.low_units ?? 0;
  const high = confidence.high_units ?? low;
  const pct = confidence.confidence_pct;

  if (compact) {
    return (
      <div className="flex flex-wrap items-center gap-1.5">
        {blocked ? (
          <span className="text-[10px] font-semibold uppercase tracking-wide text-amber-700 bg-amber-50 px-1.5 py-0.5 rounded-full border border-amber-200">
            Insufficient history
          </span>
        ) : (
          <span className="text-xs font-mono tabular-nums">
            {low.toLocaleString()} – {high.toLocaleString()}
          </span>
        )}
        {confidence.baseline_source && (
          <span className="text-[10px] uppercase tracking-wide rounded-full bg-[var(--surface)] border border-[var(--border)] px-1.5 py-0.5 text-[var(--muted)]">
            {formatSourceBadge(confidence.baseline_source)}
          </span>
        )}
        {seasonal && (
          <span className="text-[10px] font-medium text-[var(--warning)]">Seasonal template active</span>
        )}
      </div>
    );
  }

  return (
    <div className="rounded-lg border border-[var(--border)] bg-[var(--surface)] p-3 space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs font-medium text-[var(--muted)]">Forecast confidence</span>
        {confidence.baseline_source && (
          <span className="text-[10px] uppercase tracking-wide rounded-full bg-[var(--default)] px-2 py-0.5 text-[var(--muted)]">
            {formatSourceBadge(confidence.baseline_source)}
          </span>
        )}
      </div>
      {blocked ? (
        <p className="text-sm text-amber-700">Insufficient history — predictive forecast blocked</p>
      ) : (
        <p className="text-sm font-semibold tabular-nums font-mono">
          {low.toLocaleString()} – {high.toLocaleString()} units
        </p>
      )}
      {pct != null && !blocked && (
        <p className={`text-xs ${confidenceColor(pct)}`}>{pct}% confidence</p>
      )}
      {seasonal && (
        <p className="text-xs text-[var(--warning)]">Seasonal template active — ML extrapolation disabled</p>
      )}
    </div>
  );
}
