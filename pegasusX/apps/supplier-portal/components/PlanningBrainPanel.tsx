"use client";

import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierPlanningScenarioKey } from "@pegasusx/api-client/idempotency";
import type { PlanningSAndOPSnapshot, PlanningScenarioResult } from "@pegasusx/types";

const api = createSupplierApi();

export default function PlanningBrainPanel() {
  const [sandop, setSandop] = useState<PlanningSAndOPSnapshot | null>(null);
  const [scenario, setScenario] = useState<PlanningScenarioResult | null>(null);
  const [downtimeHours, setDowntimeHours] = useState(8);
  const [demandDeltaPct, setDemandDeltaPct] = useState(10);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const snap = await api.getPlanningSAndOP();
        if (!cancelled) setSandop(snap);
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "planning_unavailable");
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const runScenario = useCallback(async () => {
    setRunning(true);
    setError(null);
    try {
      const result = await api.runPlanningScenario(
        {
          factory_downtime_hours: downtimeHours,
          demand_delta_pct: demandDeltaPct,
          horizon_days: 7,
        },
        supplierPlanningScenarioKey("portal", downtimeHours, demandDeltaPct),
      );
      setScenario(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "scenario_failed");
    } finally {
      setRunning(false);
    }
  }, [demandDeltaPct, downtimeHours]);

  return (
    <div className="p-5 flex flex-col gap-5 h-full">
      <div>
        <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
          Planning sandbox
        </h2>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Read-only what-if and lightweight S&amp;OP for your network.
        </p>
      </div>

      {error ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-danger)" }}>
          {error}
        </p>
      ) : null}

      <section className="grid grid-cols-2 md:grid-cols-4 gap-3">
        {loading ? (
          <p className="md-typescale-body-small col-span-full" style={{ color: "var(--desk-text-secondary)" }}>
            Loading S&amp;OP…
          </p>
        ) : sandop ? (
          <>
            <Stat label="Factory cap (7d)" value={sandop.factory_capacity_units} />
            <Stat label="WH inbound cap" value={sandop.warehouse_inbound_cap_units} />
            <Stat label="Utilization" value={Math.round(sandop.utilization_pct)} suffix="%" />
            <Stat label="Alert" value={sandop.capacity_alert ? 1 : 0} labelOverride={sandop.capacity_alert ? "Breach" : "OK"} />
          </>
        ) : null}
      </section>

      <section className="flex flex-col gap-3 rounded-lg p-4" style={{ background: "var(--desk-surface-raised)" }}>
        <h3 className="md-typescale-title-small">Scenario run</h3>
        <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
          <label className="flex flex-col gap-1 md-typescale-body-small">
            Factory downtime (hours)
            <input
              type="number"
              min={0}
              max={168}
              value={downtimeHours}
              onChange={(e) => setDowntimeHours(Number(e.target.value))}
              className="portal-input"
            />
          </label>
          <label className="flex flex-col gap-1 md-typescale-body-small">
            Demand delta (%)
            <input
              type="number"
              min={-50}
              max={200}
              value={demandDeltaPct}
              onChange={(e) => setDemandDeltaPct(Number(e.target.value))}
              className="portal-input"
            />
          </label>
        </div>
        <button
          type="button"
          className="portal-btn portal-btn--primary w-fit"
          disabled={running}
          onClick={() => void runScenario()}
        >
          {running ? "Running…" : "Run scenario"}
        </button>
        {scenario ? (
          <div className="flex flex-col gap-3 pt-2">
            {scenario.mode ? (
              <span className="md-chip h-6 text-xs w-fit">
                {scenario.mode === "twin_snapshot" ? "Twin snapshot" : "Heuristic"}
              </span>
            ) : null}
            <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
              <Stat label="SLA risk" value={Math.round(scenario.sla_risk_pct)} suffix="%" />
              {scenario.baseline_sla_risk_pct != null ? (
                <Stat label="Baseline SLA" value={Math.round(scenario.baseline_sla_risk_pct)} suffix="%" />
              ) : null}
              <Stat label="Fleet volume" value={scenario.fleet_volume_orders} />
              <Stat label="Stockout SKUs" value={scenario.stockout_skus.length} />
              <Stat label="Capacity breach" value={scenario.capacity_breach ? 1 : 0} labelOverride={scenario.capacity_breach ? "Yes" : "No"} />
              {scenario.revenue_at_risk_minor != null && scenario.revenue_at_risk_minor > 0 ? (
                <Stat label="Revenue at risk" value={scenario.revenue_at_risk_minor} suffix=" tiyin" />
              ) : null}
            </div>
            {scenario.stockout_skus.length > 0 ? (
              <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
                Stockouts: {scenario.stockout_skus.slice(0, 12).join(", ")}
                {scenario.stockout_skus.length > 12 ? "…" : ""}
              </p>
            ) : null}
          </div>
        ) : null}
      </section>
    </div>
  );
}

function Stat({
  label,
  value,
  suffix,
  labelOverride,
}: {
  label: string;
  value: number;
  suffix?: string;
  labelOverride?: string;
}) {
  return (
    <div className="rounded-lg p-3" style={{ background: "var(--desk-surface)" }}>
      <div className="md-typescale-label-medium" style={{ color: "var(--desk-text-secondary)" }}>
        {label}
      </div>
      <div className="md-kpi-value text-xl">
        {labelOverride ?? `${value}${suffix ?? ""}`}
      </div>
    </div>
  );
}
