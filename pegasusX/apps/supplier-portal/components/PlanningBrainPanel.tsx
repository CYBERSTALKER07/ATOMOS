"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import { supplierPlanningScenarioKey } from "@pegasusx/api-client/idempotency";
import type {
  PlanningSAndOPSnapshot,
  PlanningScenarioCompareResult,
  PlanningScenarioResult,
} from "@pegasusx/types";

const api = createSupplierApi();

export default function PlanningBrainPanel() {
  const t = usePortalT();
  const [sandop, setSandop] = useState<PlanningSAndOPSnapshot | null>(null);
  const [scenario, setScenario] = useState<PlanningScenarioResult | null>(null);
  const [scenarios, setScenarios] = useState<PlanningScenarioResult[]>([]);
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [compare, setCompare] = useState<PlanningScenarioCompareResult | null>(null);
  const [downtimeHours, setDowntimeHours] = useState(8);
  const [demandDeltaPct, setDemandDeltaPct] = useState(10);
  const [loading, setLoading] = useState(true);
  const [running, setRunning] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const refreshList = useCallback(async () => {
    try {
      const list = await api.listPlanningScenarios();
      setScenarios(list.scenarios ?? []);
    } catch {
      // list is best-effort after run; surface only when user interacts
    }
  }, []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [snap, list] = await Promise.all([api.getPlanningSAndOP(), api.listPlanningScenarios()]);
        if (!cancelled) {
          setSandop(snap);
          setScenarios(list.scenarios ?? []);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.planning_unavailable"));
        }
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [t]);

  const runScenario = useCallback(async () => {
    setRunning(true);
    setError(null);
    setCompare(null);
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
      setSelectedIds([result.scenario_id]);
      await refreshList();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.scenario_failed"));
    } finally {
      setRunning(false);
    }
  }, [demandDeltaPct, downtimeHours, refreshList, t]);

  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      if (prev.includes(id)) return prev.filter((x) => x !== id);
      if (prev.length >= 2) return [prev[1], id];
      return [...prev, id];
    });
    setCompare(null);
  };

  const cloneSelected = async () => {
    const id = selectedIds[0] ?? scenario?.scenario_id;
    if (!id) return;
    setBusy(true);
    setError(null);
    try {
      const cloned = await api.clonePlanningScenario(
        id,
        { label: `Clone ${id.slice(0, 8)}` },
        `scenario-clone-${id}-${Date.now()}`,
      );
      setScenario(cloned);
      setSelectedIds([cloned.scenario_id]);
      await refreshList();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Clone failed");
    } finally {
      setBusy(false);
    }
  };

  const compareSelected = async () => {
    if (selectedIds.length !== 2) {
      setError("Select exactly two scenarios to compare");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const result = await api.comparePlanningScenarios(
        [selectedIds[0], selectedIds[1]],
        `scenario-compare-${selectedIds[0]}-${selectedIds[1]}-${Date.now()}`,
      );
      setCompare(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Compare failed");
    } finally {
      setBusy(false);
    }
  };

  const publishSelected = async () => {
    const id = selectedIds[0] ?? scenario?.scenario_id;
    if (!id) return;
    const row = scenarios.find((s) => s.scenario_id === id) ?? scenario;
    if (row?.status && row.status !== "DRAFT") {
      setError("Only DRAFT scenarios can be published");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      const published = await api.publishPlanningScenario(id, `scenario-publish-${id}-${Date.now()}`);
      setScenario(published);
      setSelectedIds([published.scenario_id]);
      await refreshList();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Publish failed");
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="p-5 flex flex-col gap-5 h-full">
      <div>
        <h2 className="md-typescale-title-medium" style={{ color: "var(--desk-text-secondary)" }}>
          Planning sandbox
        </h2>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Clone, compare, and publish what-if scenarios for your network baseline.
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
            <Stat
              label={
                sandop.horizon_days && sandop.horizon_days !== 7
                  ? `Factory cap (${sandop.horizon_days}d)`
                  : t("supplier_portal.residual.text.factory_cap_7d")
              }
              value={sandop.factory_capacity_units}
            />
            <Stat label="Projected demand" value={sandop.projected_demand_units ?? 0} />
            <Stat label={t("supplier_portal.residual.text.utilization")} value={Math.round(sandop.utilization_pct)} suffix="%" />
            <Stat
              label={t("factory_portal.dashboard.dashboard_alerts.text.alert")}
              value={sandop.capacity_alert ? 1 : 0}
              labelOverride={sandop.capacity_alert ? "Breach" : "OK"}
            />
            {sandop.capacity_source || sandop.capacity_model ? (
              <p className="md-typescale-body-small col-span-full" style={{ color: "var(--desk-text-secondary)" }}>
                {sandop.capacity_model ? `model ${sandop.capacity_model}` : "model —"}
                {" · "}
                {sandop.capacity_source ? `source ${sandop.capacity_source}` : "source unavailable"}
              </p>
            ) : null}
          </>
        ) : null}
      </section>

      <section className="flex flex-col gap-3 rounded-lg p-4" style={{ background: "var(--desk-surface-raised)" }}>
        <h3 className="md-typescale-title-small">{t("supplier_portal.planning_brain_panel.text.scenario_run")}</h3>
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
        <div className="flex flex-wrap gap-2">
          <button
            type="button"
            className="portal-btn portal-btn--primary w-fit"
            disabled={running || busy}
            onClick={() => void runScenario()}
          >
            {running ? "Running…" : "Run scenario"}
          </button>
          <button type="button" className="portal-btn w-fit" disabled={busy || !selectedIds[0]} onClick={() => void cloneSelected()}>
            Clone
          </button>
          <button type="button" className="portal-btn w-fit" disabled={busy || selectedIds.length !== 2} onClick={() => void compareSelected()}>
            Compare
          </button>
          <button type="button" className="portal-btn w-fit" disabled={busy || !selectedIds[0]} onClick={() => void publishSelected()}>
            Publish
          </button>
        </div>
        {scenario ? <ScenarioMetrics scenario={scenario} t={t} /> : null}
      </section>

      <section className="flex flex-col gap-3 rounded-lg p-4" style={{ background: "var(--desk-surface-raised)" }}>
        <h3 className="md-typescale-title-small">Recent scenarios</h3>
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          Select up to two drafts for compare. Publish marks the planning baseline only (no sealed-manifest rewrite).
        </p>
        {scenarios.length === 0 ? (
          <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
            No saved scenarios yet.
          </p>
        ) : (
          <ul className="flex flex-col gap-2">
            {scenarios.map((row) => {
              const checked = selectedIds.includes(row.scenario_id);
              return (
                <li key={row.scenario_id}>
                  <label
                    className="flex items-start gap-3 rounded-lg p-3 cursor-pointer"
                    style={{ background: checked ? "var(--desk-surface)" : "transparent" }}
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() => toggleSelect(row.scenario_id)}
                      className="mt-1"
                    />
                    <span className="flex flex-col gap-1 md-typescale-body-small">
                      <span>
                        {(row.label || row.scenario_id.slice(0, 8))} · v{row.version ?? 1} · {row.status ?? "DRAFT"}
                      </span>
                      <span style={{ color: "var(--desk-text-secondary)" }}>
                        downtime {row.factory_downtime_hours ?? 0}h · demand {row.demand_delta_pct ?? 0}% · SLA{" "}
                        {Math.round(row.sla_risk_pct)}% · RaR {row.revenue_at_risk_minor ?? 0}
                      </span>
                    </span>
                  </label>
                </li>
              );
            })}
          </ul>
        )}
      </section>

      {compare ? (
        <section className="flex flex-col gap-3 rounded-lg p-4" style={{ background: "var(--desk-surface-raised)" }}>
          <h3 className="md-typescale-title-small">Compare</h3>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-3">
            <Stat label="Δ SLA risk" value={Math.round(compare.deltas.sla_risk_pct_delta)} suffix="%" />
            <Stat label="Δ Fleet" value={compare.deltas.fleet_volume_orders_delta} />
            <Stat label="Δ RaR" value={compare.deltas.revenue_at_risk_minor_delta} suffix=" tiyin" />
            <Stat label="Δ Stockouts" value={compare.deltas.stockout_count_delta} />
            <Stat
              label="Capacity changed"
              value={compare.deltas.capacity_breach_changed ? 1 : 0}
              labelOverride={compare.deltas.capacity_breach_changed ? "Yes" : "No"}
            />
          </div>
        </section>
      ) : null}
    </div>
  );
}

function ScenarioMetrics({
  scenario,
  t,
}: {
  scenario: PlanningScenarioResult;
  t: (key: string) => string;
}) {
  return (
    <div className="flex flex-col gap-3 pt-2">
      <div className="flex flex-wrap gap-2">
        {scenario.mode ? (
          <span className="md-chip h-6 text-xs w-fit">
            {scenario.mode === "twin_snapshot" ? "Twin snapshot" : "Heuristic"}
          </span>
        ) : null}
        {scenario.status ? <span className="md-chip h-6 text-xs w-fit">{scenario.status}</span> : null}
        {scenario.version != null ? <span className="md-chip h-6 text-xs w-fit">v{scenario.version}</span> : null}
        {scenario.unit_value_source ? (
          <span className="md-chip h-6 text-xs w-fit">RaR {scenario.unit_value_source}</span>
        ) : null}
      </div>
      <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
        <Stat label={t("supplier_portal.residual.text.sla_risk")} value={Math.round(scenario.sla_risk_pct)} suffix="%" />
        {scenario.baseline_sla_risk_pct != null ? (
          <Stat label={t("supplier_portal.residual.text.baseline_sla")} value={Math.round(scenario.baseline_sla_risk_pct)} suffix="%" />
        ) : null}
        <Stat label={t("supplier_portal.residual.text.fleet_volume")} value={scenario.fleet_volume_orders} />
        <Stat label={t("supplier_portal.residual.text.stockout_skus")} value={scenario.stockout_skus.length} />
        <Stat
          label={t("supplier_portal.residual.text.capacity_breach")}
          value={scenario.capacity_breach ? 1 : 0}
          labelOverride={scenario.capacity_breach ? "Yes" : "No"}
        />
        {scenario.revenue_at_risk_minor != null && scenario.revenue_at_risk_minor > 0 ? (
          <Stat label={t("supplier_portal.residual.text.revenue_at_risk")} value={scenario.revenue_at_risk_minor} suffix=" tiyin" />
        ) : null}
      </div>
      {scenario.stockout_skus.length > 0 ? (
        <p className="md-typescale-body-small" style={{ color: "var(--desk-text-secondary)" }}>
          Stockouts: {scenario.stockout_skus.slice(0, 12).join(", ")}
          {scenario.stockout_skus.length > 12 ? "…" : ""}
        </p>
      ) : null}
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
      <div className="md-kpi-value text-xl">{labelOverride ?? `${value}${suffix ?? ""}`}</div>
    </div>
  );
}
