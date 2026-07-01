import type { ForecastConfidence } from '@pegasusx/types';

function asNumber(value: unknown): number | undefined {
  if (typeof value === 'number' && Number.isFinite(value)) return value;
  if (typeof value === 'string' && value.trim() !== '') {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) return parsed;
  }
  return undefined;
}

function asString(value: unknown): string | undefined {
  if (typeof value === 'string' && value.trim() !== '') return value;
  return undefined;
}

/** Parse planning-brain confidence fields from demand_breakdown JSON. */
export function parseForecastConfidence(
  breakdown?: Record<string, unknown> | null,
): ForecastConfidence | null {
  if (!breakdown || typeof breakdown !== 'object') return null;

  const low = asNumber(breakdown.low_units);
  const high = asNumber(breakdown.high_units);
  let confidencePct = asNumber(breakdown.confidence_pct);
  const rawConfidence = breakdown.confidence;
  if (confidencePct == null && typeof rawConfidence === 'number') {
    confidencePct = rawConfidence <= 1
      ? Math.round(rawConfidence * 100)
      : Math.round(rawConfidence);
  }
  const baselineSource = asString(breakdown.baseline_source);
  const blockedReason = asString(breakdown.blocked_reason);
  const label = asString(breakdown.label) as ForecastConfidence['label'];

  const predicted = asNumber(breakdown.predictedQty) ?? asNumber(breakdown.predicted_qty);
  const derivedLow = low ?? (predicted != null ? Math.max(0, Math.floor(predicted * 0.9)) : undefined);
  const derivedHigh = high ?? (predicted != null ? Math.ceil(predicted * 1.1) : undefined);

  if (
    derivedLow == null
    && derivedHigh == null
    && confidencePct == null
    && !blockedReason
    && !baselineSource
    && !label
  ) {
    return null;
  }

  return {
    low_units: derivedLow,
    high_units: derivedHigh ?? derivedLow,
    confidence_pct: confidencePct,
    baseline_source: baselineSource,
    blocked_reason: blockedReason,
    label,
  };
}

export function isForecastBlocked(confidence?: ForecastConfidence | null): boolean {
  if (!confidence) return false;
  return Boolean(confidence.blocked_reason) || confidence.label === 'insufficient_history';
}

export function isSeasonalTemplateActive(confidence?: ForecastConfidence | null): boolean {
  return confidence?.baseline_source === 'seasonal_template';
}

export function formatSourceBadge(source?: string): string {
  if (!source) return '';
  switch (source) {
    case 'ml': return 'ML';
    case 'moving_average': return 'Baseline';
    case 'seasonal_template': return 'Seasonal';
    case 'mixed': return 'Mixed';
    default: return source.replaceAll('_', ' ');
  }
}
