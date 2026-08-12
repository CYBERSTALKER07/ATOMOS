import { buildScenarioChartRows, pickScenarioPair } from "./scenario-chart";
import type { PlanningScenarioResult } from "@pegasusx/types";

function sc(partial: Partial<PlanningScenarioResult> & { scenario_id: string }): PlanningScenarioResult {
  return {
    supplier_id: "sup-1",
    sla_risk_pct: 10,
    fleet_volume_orders: 100,
    stockout_skus: [],
    capacity_breach: false,
    ...partial,
  };
}

describe("scenario-chart", () => {
  it("picks published baseline and lowest-RaR draft as upside", () => {
    const { baseline, upside } = pickScenarioPair([
      sc({ scenario_id: "d1", status: "DRAFT", revenue_at_risk_minor: 9000, sla_risk_pct: 40 }),
      sc({ scenario_id: "p1", status: "PUBLISHED", version: 2, revenue_at_risk_minor: 5000, sla_risk_pct: 20 }),
      sc({ scenario_id: "d2", status: "DRAFT", revenue_at_risk_minor: 1000, sla_risk_pct: 15 }),
    ]);
    expect(baseline?.scenario_id).toBe("p1");
    expect(upside?.scenario_id).toBe("d2");
  });

  it("builds chart rows from pair", () => {
    const rows = buildScenarioChartRows([
      sc({
        scenario_id: "p1",
        status: "PUBLISHED",
        sla_risk_pct: 22.4,
        fleet_volume_orders: 50,
        stockout_skus: ["a", "b"],
        revenue_at_risk_minor: 25000,
      }),
      sc({
        scenario_id: "d1",
        status: "DRAFT",
        sla_risk_pct: 10,
        fleet_volume_orders: 40,
        stockout_skus: ["a"],
        revenue_at_risk_minor: 12000,
      }),
    ]);
    expect(rows).toHaveLength(4);
    expect(rows[0]).toEqual({ name: "SLA risk %", baseline: 22, upside: 10 });
    expect(rows[3]).toEqual({ name: "RaR (k)", baseline: 25, upside: 12 });
  });
});
