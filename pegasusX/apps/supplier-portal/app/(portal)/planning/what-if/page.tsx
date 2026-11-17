"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";
import { resolvePortalSessionRole } from "@/lib/connectors-auditor-api";
import { usePortalT } from "@/lib/i18n";
import {
  createP2bApi,
  extractP2bError,
  formatFinanceMinor,
  formatRateBps,
  parseCommittedProposalIds,
  parsePercentToBps,
  parseWhatIfBreakdown,
  parseWhatIfBreakdownCurrency,
  parseWhatIfShocks,
} from "@/lib/p2b-api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";
import type { P2BProposalSummary, WhatIfSimulation } from "@pegasusx/types";

const CURRENCIES = ["UZS", "USD", "EUR", "RUB"] as const;
const HORIZONS = [7, 14, 30, 60, 90] as const;

export default function WhatIfSimulatorPage() {
  return (
    <Suspense fallback={null}>
      <WhatIfSimulatorInner />
    </Suspense>
  );
}

function WhatIfSimulatorInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const p2bApi = useMemo(() => createP2bApi(createSupplierApi()), []);

  const initialSimId = (searchParams.get("sim_id") || "").trim();
  const initialScenarioId = (searchParams.get("scenario_id") || "").trim();
  const initialEntityId = (searchParams.get("entity_id") || "").trim();

  const [roleResolved, setRoleResolved] = useState(false);
  const [isReadOnlyVisitor, setIsReadOnlyVisitor] = useState(false);

  const [simulations, setSimulations] = useState<WhatIfSimulation[]>([]);
  const [activeSim, setActiveSim] = useState<WhatIfSimulation | null>(null);
  const [committedProposals, setCommittedProposals] = useState<P2BProposalSummary[]>([]);

  // Simulation form state (percentages converted to integer bps without floats)
  const [simName, setSimName] = useState("Q4 Stress Simulation");
  const [scenarioId, setScenarioId] = useState(initialScenarioId);
  const [entityId, setEntityId] = useState(initialEntityId);
  const [horizonDays, setHorizonDays] = useState("30");
  const [currency, setCurrency] = useState("UZS");
  const [demandDeltaPct, setDemandDeltaPct] = useState("15");
  const [downtimeHours, setDowntimeHours] = useState("12");
  const [fxShockPct, setFxShockPct] = useState("5");
  const [creditDeltaPct, setCreditDeltaPct] = useState("-10");
  const [priceDeltaPct, setPriceDeltaPct] = useState("0");

  const [loading, setLoading] = useState(true);
  const [refreshing, setRefreshing] = useState(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const syncRoleFromToken = useCallback(() => {
    const token = getSupplierToken() || readTokenFromCookie();
    if (!token) {
      setIsReadOnlyVisitor(false);
      setRoleResolved(true);
      return;
    }
    const roleInfo = resolvePortalSessionRole(decodeJwtPayload(token));
    setIsReadOnlyVisitor(roleInfo.isAuditor || roleInfo.isAgent);
    setRoleResolved(true);
  }, []);

  useEffect(() => {
    syncRoleFromToken();
  }, [syncRoleFromToken]);

  const loadAll = useCallback(
    async (silent = false) => {
      syncRoleFromToken();
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const listRes = await p2bApi.listWhatIfSimulations({ limit: 50 });
        const list = listRes.simulations ?? [];
        setSimulations(list);
        if (initialSimId) {
          const detail = await p2bApi.getWhatIfSimulation(initialSimId);
          setActiveSim(detail);
        } else if (list.length > 0) {
          setActiveSim((prev) => {
            if (!prev) return list[0];
            return list.find((s) => s.simulation_id === prev.simulation_id) ?? prev;
          });
        }
      } catch (err) {
        setError(
          extractP2bError(err, tRef.current("supplier_portal.what_if.load_failed")).message,
        );
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [initialSimId, p2bApi, syncRoleFromToken],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const canMutate = roleResolved && !isReadOnlyVisitor;

  const handleRunSimulation = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!canMutate) return;
    const demandBps = parsePercentToBps(demandDeltaPct);
    const fxBps = parsePercentToBps(fxShockPct);
    const creditBps = parsePercentToBps(creditDeltaPct);
    const priceBps = parsePercentToBps(priceDeltaPct);
    const downtimeInt = /^\d+$/.test(downtimeHours.trim()) ? Number(downtimeHours.trim()) : NaN;
    const horizonInt = /^\d+$/.test(horizonDays.trim()) ? Number(horizonDays.trim()) : 30;

    if (
      !simName.trim() ||
      demandBps === null ||
      fxBps === null ||
      creditBps === null ||
      priceBps === null ||
      !Number.isSafeInteger(downtimeInt) ||
      downtimeInt < 0
    ) {
      setError(t("supplier_portal.what_if.invalid_shock_input"));
      return;
    }

    setBusyAction("simulate");
    setError(null);
    setNotice(null);
    setCommittedProposals([]);
    try {
      const sim = await p2bApi.runWhatIfSimulation({
        name: simName.trim(),
        scenario_id: scenarioId.trim() || undefined,
        entity_id: entityId.trim() || undefined,
        horizon_days: horizonInt > 0 ? horizonInt : 30,
        demand_delta_bps: demandBps,
        factory_downtime_hours: downtimeInt,
        fx_rate_shock_bps: fxBps,
        credit_limit_delta_bps: creditBps,
        unit_price_delta_bps: priceBps,
        currency,
      });
      setActiveSim(sim);
      setNotice(
        t("supplier_portal.what_if.simulate_success", {
          simId: sim.simulation_id,
          netPnl: formatFinanceMinor(sim.net_pnl_delta_minor, currency),
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.what_if.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleInspectSimulation = async (sim: WhatIfSimulation) => {
    setBusyAction(`inspect:${sim.simulation_id}`);
    setError(null);
    setCommittedProposals([]);
    try {
      const detail = await p2bApi.getWhatIfSimulation(sim.simulation_id);
      setActiveSim(detail);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.what_if.load_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleCommitProposals = async () => {
    if (!canMutate || !activeSim) return;
    setBusyAction("commit");
    setError(null);
    setNotice(null);
    try {
      const res = await p2bApi.commitWhatIfProposals(activeSim.simulation_id, {});
      setActiveSim(res.simulation);
      setCommittedProposals(res.proposals ?? []);
      setNotice(
        t("supplier_portal.what_if.commit_success", {
          count: (res.committed_proposal_ids ?? []).length,
          simId: res.simulation.simulation_id,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(extractP2bError(err, t("supplier_portal.what_if.action_failed")).message);
    } finally {
      setBusyAction(null);
    }
  };

  const activeShocks = useMemo(() => parseWhatIfShocks(activeSim), [activeSim]);
  const activeBreakdown = useMemo(() => parseWhatIfBreakdown(activeSim), [activeSim]);
  const activeCommittedIds = useMemo(() => parseCommittedProposalIds(activeSim), [activeSim]);
  const displayCurrency = parseWhatIfBreakdownCurrency(activeSim) || currency;

  return (
    <PageChrome
      icon="analytics"
      title={t("supplier_portal.what_if.title")}
      description={t("supplier_portal.what_if.subtitle")}
      loading={loading && simulations.length === 0 && !activeSim && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/planning" as Route} className="portal-btn portal-btn--outline text-xs">
            <Icon name="overview" size={16} />
            <span>{t("portal.nav.planning")}</span>
          </Link>
          <Link href={"/proposals" as Route} className="portal-btn portal-btn--outline text-xs">
            <Icon name="ai" size={16} />
            <span>{t("portal.nav.proposals")}</span>
          </Link>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--outline text-xs"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {roleResolved && isReadOnlyVisitor ? (
          <div
            data-testid="whatif-read-only-banner"
            className="portal-alert portal-alert--info"
            role="status"
          >
            <p className="portal-alert-text text-xs">
              {t("supplier_portal.auditor.safe_mode_badge")} —{" "}
              {t("supplier_portal.what_if.read_only_notice")}
            </p>
          </div>
        ) : null}

        <div className="portal-alert portal-alert--info" role="note">
          <p className="portal-alert-text text-xs">
            {t("supplier_portal.what_if.governance_guard_note")}
          </p>
        </div>

        {error ? (
          <div className="portal-alert portal-alert--error" role="alert">
            <p className="portal-alert-text font-semibold">{error}</p>
          </div>
        ) : null}

        {notice ? (
          <div className="portal-alert portal-alert--info" role="status">
            <p className="portal-alert-text">{notice}</p>
          </div>
        ) : null}

        {canMutate ? (
          <PortalSection
            icon="analytics"
            title={t("supplier_portal.what_if.runner_title")}
            description={t("supplier_portal.what_if.runner_subtitle")}
          >
            <form onSubmit={(e) => void handleRunSimulation(e)} className="space-y-4">
              <div className="grid grid-cols-1 md:grid-cols-5 gap-3 items-end">
                <div className="md:col-span-2">
                  <label htmlFor="whatif-name" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.name_label")}
                  </label>
                  <PortalInput
                    id="whatif-name"
                    value={simName}
                    onChange={(e) => setSimName(e.target.value)}
                    required
                  />
                </div>
                <div>
                  <label htmlFor="whatif-scenario-id" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.scenario_id_label")}
                  </label>
                  <PortalInput
                    id="whatif-scenario-id"
                    value={scenarioId}
                    onChange={(e) => setScenarioId(e.target.value)}
                    placeholder="scn-optional"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-horizon" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.horizon_label")}
                  </label>
                  <PortalSelect
                    id="whatif-horizon"
                    value={horizonDays}
                    onChange={(e) => setHorizonDays(e.target.value)}
                  >
                    {HORIZONS.map((d) => (
                      <option key={d} value={String(d)}>
                        {d}d
                      </option>
                    ))}
                  </PortalSelect>
                </div>
                <div>
                  <label htmlFor="whatif-currency" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.currency_label")}
                  </label>
                  <PortalSelect
                    id="whatif-currency"
                    value={currency}
                    onChange={(e) => setCurrency(e.target.value)}
                  >
                    {CURRENCIES.map((cur) => (
                      <option key={cur} value={cur}>
                        {cur}
                      </option>
                    ))}
                  </PortalSelect>
                </div>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-3 items-end">
                <div>
                  <label htmlFor="whatif-demand-pct" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.demand_delta_label")}
                  </label>
                  <PortalInput
                    id="whatif-demand-pct"
                    value={demandDeltaPct}
                    onChange={(e) => setDemandDeltaPct(e.target.value)}
                    placeholder="15"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-downtime-hrs" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.downtime_hours_label")}
                  </label>
                  <PortalInput
                    id="whatif-downtime-hrs"
                    value={downtimeHours}
                    onChange={(e) => setDowntimeHours(e.target.value)}
                    placeholder="12"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-fx-pct" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.fx_shock_label")}
                  </label>
                  <PortalInput
                    id="whatif-fx-pct"
                    value={fxShockPct}
                    onChange={(e) => setFxShockPct(e.target.value)}
                    placeholder="5"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-credit-pct" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.credit_delta_label")}
                  </label>
                  <PortalInput
                    id="whatif-credit-pct"
                    value={creditDeltaPct}
                    onChange={(e) => setCreditDeltaPct(e.target.value)}
                    placeholder="-10"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-price-pct" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.price_delta_label")}
                  </label>
                  <PortalInput
                    id="whatif-price-pct"
                    value={priceDeltaPct}
                    onChange={(e) => setPriceDeltaPct(e.target.value)}
                    placeholder="0"
                  />
                </div>
                <div>
                  <label htmlFor="whatif-entity-id" className="portal-label block mb-1">
                    {t("supplier_portal.what_if.entity_id_label")}
                  </label>
                  <PortalInput
                    id="whatif-entity-id"
                    value={entityId}
                    onChange={(e) => setEntityId(e.target.value)}
                    placeholder="ent-optional"
                  />
                </div>
              </div>

              <div className="flex justify-end">
                <button
                  type="submit"
                  disabled={busyAction !== null || !simName.trim()}
                  className="portal-btn portal-btn--primary text-xs"
                >
                  {busyAction === "simulate"
                    ? t("common.status.loading")
                    : t("supplier_portal.what_if.run_button")}
                </button>
              </div>
            </form>
          </PortalSection>
        ) : null}

        {activeSim ? (
          <>
            <KpiStatGrid columns={4}>
              <KpiStatCard
                label={t("supplier_portal.what_if.kpi_forecast_delta")}
                value={formatFinanceMinor(activeSim.forecast_delta_minor, displayCurrency)}
                sub={`${activeSim.horizon_days}d · ${activeSim.status}`}
              />
              <KpiStatCard
                label={t("supplier_portal.what_if.kpi_replenishment_cost")}
                value={formatFinanceMinor(activeSim.replenishment_cost_minor, displayCurrency)}
                sub={t("supplier_portal.what_if.kpi_dispatch_delta", {
                  amount: formatFinanceMinor(activeSim.dispatch_cost_delta_minor, displayCurrency),
                })}
              />
              <KpiStatCard
                label={t("supplier_portal.what_if.kpi_credit_exposure_delta")}
                value={formatFinanceMinor(activeSim.credit_exposure_delta_minor, displayCurrency)}
                sub={activeSim.entity_id || "ALL"}
              />
              <KpiStatCard
                label={t("supplier_portal.what_if.kpi_net_pnl_delta")}
                value={formatFinanceMinor(activeSim.net_pnl_delta_minor, displayCurrency)}
                sub={activeSim.simulation_id}
              />
            </KpiStatGrid>

            <PortalSection
              icon="ledger"
              title={`${activeSim.name} (${activeSim.simulation_id})`}
              description={
                activeShocks
                  ? `Demand ${formatRateBps(activeShocks.demand_delta_bps)} · Downtime ${activeShocks.factory_downtime_hours}h · FX ${formatRateBps(activeShocks.fx_rate_shock_bps)} · Credit ${formatRateBps(activeShocks.credit_limit_delta_bps)} · Price ${formatRateBps(activeShocks.unit_price_delta_bps)}`
                  : undefined
              }
              actions={
                <div className="flex flex-wrap items-center gap-2">
                  <StatusBadge state={activeSim.status} />
                  {canMutate ? (
                    <button
                      type="button"
                      disabled={busyAction !== null || activeSim.status === "COMMITTED"}
                      onClick={() => void handleCommitProposals()}
                      className="portal-btn portal-btn--primary text-xs"
                    >
                      {busyAction === "commit"
                        ? t("common.status.loading")
                        : t("supplier_portal.what_if.commit_proposals_button")}
                    </button>
                  ) : null}
                </div>
              }
            >
              {activeBreakdown.length === 0 ? (
                <pre
                  className="rounded-lg border p-3 text-xs font-mono overflow-x-auto"
                  style={{
                    borderColor: "var(--desk-border)",
                    background: "var(--desk-surface-raised)",
                  }}
                >
                  {activeSim.breakdown_json || "{}"}
                </pre>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left border-collapse text-xs">
                    <thead>
                      <tr
                        className="border-b uppercase"
                        style={{
                          borderColor: "var(--desk-border)",
                          color: "var(--desk-text-tertiary)",
                        }}
                      >
                        <th className="py-2 px-2.5">{t("supplier_portal.what_if.col_stage")}</th>
                        <th className="py-2 px-2.5 text-right">{t("supplier_portal.what_if.col_delta")}</th>
                        <th className="py-2 px-2.5 text-right">{t("supplier_portal.what_if.col_cost")}</th>
                        <th className="py-2 px-2.5 text-right">{t("supplier_portal.what_if.col_metric_bps")}</th>
                        <th className="py-2 px-2.5">{t("supplier_portal.what_if.col_detail")}</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                      {activeBreakdown.map((row, idx) => (
                        <tr key={`${row.stage || "stage"}-${idx}`}>
                          <td className="py-2 px-2.5 font-mono font-semibold">
                            {row.label || row.stage}
                          </td>
                          <td className="py-2 px-2.5 font-mono text-right">
                            {row.delta_minor !== undefined
                              ? formatFinanceMinor(row.delta_minor, displayCurrency)
                              : "—"}
                          </td>
                          <td className="py-2 px-2.5 font-mono text-right">
                            {row.cost_minor !== undefined
                              ? formatFinanceMinor(row.cost_minor, displayCurrency)
                              : "—"}
                          </td>
                          <td className="py-2 px-2.5 font-mono text-right">
                            {row.metric_bps !== undefined ? formatRateBps(row.metric_bps) : "—"}
                          </td>
                          <td className="py-2 px-2.5">{row.detail || "—"}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}

              {(activeCommittedIds.length > 0 || committedProposals.length > 0) ? (
                <div className="mt-4 pt-3 border-t flex flex-col gap-2" style={{ borderColor: "var(--desk-border)" }}>
                  <div className="text-xs font-semibold uppercase" style={{ color: "var(--desk-text-tertiary)" }}>
                    {t("supplier_portal.what_if.committed_proposals_title")}
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {committedProposals.length > 0
                      ? committedProposals.map((p) => (
                          <Link
                            key={p.proposal_id}
                            href={`/proposals?proposal_id=${encodeURIComponent(p.proposal_id)}` as Route}
                            className="rounded-lg border px-2.5 py-1 text-xs font-mono underline"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            {p.proposal_id} · {p.proposal_type} ({p.status})
                          </Link>
                        ))
                      : activeCommittedIds.map((pid) => (
                          <Link
                            key={pid}
                            href={`/proposals?proposal_id=${encodeURIComponent(pid)}` as Route}
                            className="rounded-lg border px-2.5 py-1 text-xs font-mono underline"
                            style={{ borderColor: "var(--desk-border)" }}
                          >
                            {pid}
                          </Link>
                        ))}
                  </div>
                </div>
              ) : null}
            </PortalSection>
          </>
        ) : null}

        <PortalSection icon="overview" title={t("supplier_portal.what_if.history_title")}>
          {simulations.length === 0 ? (
            <p className="text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.what_if.no_simulations")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-xs">
                <thead>
                  <tr
                    className="border-b uppercase"
                    style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}
                  >
                    <th className="py-2 px-2.5">{t("supplier_portal.what_if.name_label")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.what_if.horizon_label")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.what_if.kpi_net_pnl_delta")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_status")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.what_if.col_proposals")}</th>
                    <th className="py-2 px-2.5">{t("supplier_portal.anomalies.col_detected")}</th>
                    <th className="py-2 px-2.5 text-right">{t("supplier_portal.anomalies.col_action")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {simulations.map((sim) => {
                    const pids = parseCommittedProposalIds(sim);
                    return (
                      <tr key={sim.simulation_id}>
                        <td className="py-2 px-2.5">
                          <div className="font-semibold">{sim.name}</div>
                          <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                            {sim.simulation_id}
                          </div>
                        </td>
                        <td className="py-2 px-2.5 font-mono">{sim.horizon_days}d</td>
                        <td className="py-2 px-2.5 font-mono text-right font-semibold">
                          {formatFinanceMinor(sim.net_pnl_delta_minor, displayCurrency)}
                        </td>
                        <td className="py-2 px-2.5">
                          <StatusBadge state={sim.status} />
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">{pids.length}</td>
                        <td className="py-2 px-2.5 whitespace-nowrap">
                          {new Date(sim.created_at).toLocaleString()}
                        </td>
                        <td className="py-2 px-2.5 text-right">
                          <button
                            type="button"
                            disabled={busyAction !== null}
                            onClick={() => void handleInspectSimulation(sim)}
                            className="portal-btn portal-btn--outline text-xs"
                          >
                            {t("supplier_portal.what_if.inspect_button")}
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </PortalSection>
      </div>
    </PageChrome>
  );
}
