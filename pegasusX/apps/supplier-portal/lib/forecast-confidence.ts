import type { ForecastConfidence, SupplierDemandSummaryResponse } from "@pegasusx/types";

const STALE_MS = 30 * 60 * 1000;

export function isForecastStale(generatedAt?: string): boolean {
  if (!generatedAt) return false;
  const t = Date.parse(generatedAt);
  if (Number.isNaN(t)) return false;
  return Date.now() - t > STALE_MS;
}

export function formatForecastUpdatedAt(generatedAt?: string): string | undefined {
  if (!generatedAt) return undefined;
  const t = Date.parse(generatedAt);
  if (Number.isNaN(t)) return generatedAt;
  const mins = Math.round((Date.now() - t) / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  return new Date(t).toLocaleString();
}

function mapBaselineSource(src?: string): ForecastConfidence["baseline_source"] {
  if (src === "demand_forecast_baseline") return "moving_average";
  if (src === "ai_recommendations") return "ml";
  if (src === "mixed") return "mixed";
  return src as ForecastConfidence["baseline_source"];
}

export function forecastConfidenceFromDemand(
  summary: SupplierDemandSummaryResponse,
): ForecastConfidence {
  if (summary.confidence) {
    return summary.confidence;
  }
  if (summary.prediction_count === 0) {
    return {
      label: "insufficient_history",
      blocked_reason: "no_predictions",
      baseline_source: mapBaselineSource(summary.baseline_source),
    };
  }
  const mid = summary.total_pallets;
  const spread = Math.max(1, Math.round(mid * 0.1));
  let confidence = 65;
  const src = mapBaselineSource(summary.baseline_source);
  if (src === "seasonal_template") confidence = 75;
  else if (src === "ml") confidence = 85;
  else if (summary.prediction_count >= 5) confidence = 72;
  return {
    low_units: Math.max(0, mid - spread),
    high_units: mid + spread,
    confidence_pct: confidence,
    baseline_source: src,
    label: summary.prediction_count < 3 ? "early_signal" : "standard",
  };
}
