"use client";

import type { ForecastConfidence } from "@pegasusx/types";
import { formatBaselineSourceLabel } from "@/lib/forecast-confidence";

type Props = {
  confidence: ForecastConfidence;
  updatedAt?: string;
  stale?: boolean;
};

function confidenceColor(pct?: number): string {
  if (pct == null) return "var(--desk-text-secondary)";
  if (pct >= 80) return "var(--desk-success)";
  if (pct >= 60) return "var(--desk-warning)";
  return "var(--desk-danger)";
}

export function ForecastConfidenceCard({ confidence, updatedAt, stale }: Props) {
  const blocked = Boolean(confidence.blocked_reason || confidence.label === "insufficient_history");
  const low = confidence.low_units ?? 0;
  const high = confidence.high_units ?? low;
  const pct = confidence.confidence_pct;

  return (
    <div
      className="rounded-lg p-4 flex flex-col gap-2"
      style={{ background: "var(--desk-surface-raised)", border: "1px solid var(--desk-border)" }}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="md-typescale-title-small">Forecast confidence</span>
        {confidence.baseline_source ? (
          <span className="md-chip text-[10px] uppercase tracking-wide">
            {formatBaselineSourceLabel(confidence.baseline_source)}
          </span>
        ) : null}
      </div>
      {blocked ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-warning)" }}>
          Insufficient history — predictive forecast blocked
        </p>
      ) : (
        <p className="md-kpi-value text-xl tabular-nums">
          {low.toLocaleString()} – {high.toLocaleString()} units
        </p>
      )}
      {pct != null && !blocked ? (
        <p className="md-typescale-body-small font-medium" style={{ color: confidenceColor(pct) }}>
          {pct}% confidence
        </p>
      ) : null}
      {updatedAt ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          {stale ? "Stale · " : ""}Updated {updatedAt}
        </p>
      ) : null}
    </div>
  );
}
