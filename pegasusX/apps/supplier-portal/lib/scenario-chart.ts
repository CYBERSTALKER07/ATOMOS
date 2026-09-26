import type { PlanningScenarioResult } from "@pegasusx/types";

export type ScenarioChartRow = {
  name: string;
  baseline: number;
  upside: number;
};

/** Pick published baseline + best (lowest RaR) draft for control-tower compare bars. */
export function pickScenarioPair(scenarios: PlanningScenarioResult[]): {
  baseline: PlanningScenarioResult | null;
  upside: PlanningScenarioResult | null;
} {
  const published = scenarios.filter((s) => s.status === "PUBLISHED");
  const drafts = scenarios.filter((s) => s.status === "DRAFT");
  const baseline =
    published.sort((a, b) => (b.version ?? 0) - (a.version ?? 0))[0] ??
    scenarios[0] ??
    null;
  let upside: PlanningScenarioResult | null = null;
  for (const d of drafts) {
    if (!upside) {
      upside = d;
      continue;
    }
    const rar = d.revenue_at_risk_minor ?? Number.MAX_SAFE_INTEGER;
    const best = upside.revenue_at_risk_minor ?? Number.MAX_SAFE_INTEGER;
    if (rar < best || (rar === best && d.sla_risk_pct < upside.sla_risk_pct)) {
      upside = d;
    }
  }
  if (!upside && scenarios.length > 1) {
    upside = scenarios.find((s) => s.scenario_id !== baseline?.scenario_id) ?? null;
  }
  return { baseline, upside };
}

/** Build baseline vs upside metric rows for the control-tower bar chart. */
export function buildScenarioChartRows(scenarios: PlanningScenarioResult[]): ScenarioChartRow[] {
  const { baseline, upside } = pickScenarioPair(scenarios);
  if (!baseline) return [];
  const b = baseline;
  const u = upside ?? baseline;
  return [
    {
      name: "SLA risk %",
      baseline: Math.round(b.sla_risk_pct),
      upside: Math.round(u.sla_risk_pct),
    },
    {
      name: "Fleet",
      baseline: b.fleet_volume_orders,
      upside: u.fleet_volume_orders,
    },
    {
      name: "Stockouts",
      baseline: b.stockout_skus?.length ?? 0,
      upside: u.stockout_skus?.length ?? 0,
    },
    {
      name: "RaR (k)",
      baseline: Math.round((b.revenue_at_risk_minor ?? 0) / 1000),
      upside: Math.round((u.revenue_at_risk_minor ?? 0) / 1000),
    },
  ];
}
