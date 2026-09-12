import type { ForecastConfidence, HistorySeries, SupplierDemandSummaryResponse } from "./index";

export type PlanBrainTab = "planning" | "brain";

export function planBrainTabFromQuery(raw?: string | null): PlanBrainTab {
  return String(raw || "").trim().toLowerCase() === "brain" ? "brain" : "planning";
}

/** Never invent a forecast line when belief is blocked or accuracy is short. */
export function brainForecastLine(
  confidence?: ForecastConfidence | null,
  accuracyPoints?: number[] | null,
): HistorySeries | null {
  if (isForecastBlocked(confidence)) return null;
  const points = (accuracyPoints ?? []).filter((n) => Number.isFinite(n));
  if (points.length < 2) return null;
  return { points, source: "live", available: true };
}

export function factoryPlanningDisabledCode(status: number, body?: unknown): string | null {
  if (status !== 409) return null;
  const text = typeof body === "string" ? body : JSON.stringify(body ?? "");
  return text.includes("factory_planning_disabled") ? "factory_planning_disabled" : null;
}

const STALE_MS = 30 * 60 * 1000;

/** Canonical mapper — keep Android/iOS util copies aligned with this module. */
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

export function mapBaselineSource(
  src?: string,
): ForecastConfidence["baseline_source"] {
  if (src === "demand_forecast_baseline") return "moving_average";
  if (src === "ai_recommendations" || src === "inventory_hint") return "inventory_hint";
  if (src === "ml") return "moving_average";
  if (src === "mixed") return "mixed";
  return src as ForecastConfidence["baseline_source"];
}

export function formatBaselineSourceLabel(src?: string): string {
  const mapped = mapBaselineSource(src);
  switch (mapped) {
    case "moving_average":
      return "Baseline";
    case "seasonal_template":
      return "Seasonal";
    case "inventory_hint":
      return "Hint";
    case "mixed":
      return "Mixed";
    default:
      return mapped?.replace(/_/g, " ") ?? "";
  }
}

export function isForecastBlocked(confidence?: ForecastConfidence | null): boolean {
  return (
    confidence?.label === "insufficient_history" || Boolean(confidence?.blocked_reason)
  );
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
  else if (src === "inventory_hint") confidence = 72;
  else if (summary.prediction_count >= 5) confidence = 72;
  return {
    low_units: Math.max(0, mid - spread),
    high_units: mid + spread,
    confidence_pct: confidence,
    baseline_source: src,
    label: summary.prediction_count < 3 ? "early_signal" : "standard",
  };
}
