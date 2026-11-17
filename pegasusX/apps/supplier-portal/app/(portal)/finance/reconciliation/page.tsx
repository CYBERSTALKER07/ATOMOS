"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import {
  RECON_DIMENSIONS,
  createSupplierFinanceApi,
  currentAccountingPeriod,
  formatFinanceMinor,
  type ReconBreakSeverity,
  type ReconBreakStatus,
  type ReconDimension,
  type ReconciliationBreak,
  type ReconciliationRun,
} from "@/lib/finance-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;

function dimensionLabelKey(dim: ReconDimension): string {
  switch (dim) {
    case "ORDERS_AR":
      return "supplier_portal.finance.four_way_recon.dim_orders_ar";
    case "STOCK_GL":
      return "supplier_portal.finance.four_way_recon.dim_stock_gl";
    case "CASH_PSP":
      return "supplier_portal.finance.four_way_recon.dim_cash_psp";
    case "TAX_FISCAL":
      return "supplier_portal.finance.four_way_recon.dim_tax_fiscal";
  }
}

export default function FourWayReconciliationPage() {
  return (
    <Suspense fallback={null}>
      <FourWayReconciliationInner />
    </Suspense>
  );
}

function FourWayReconciliationInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const financeApi = useMemo(() => createSupplierFinanceApi(createSupplierApi()), []);

  const initialRunId = (searchParams.get("runId") || "").trim();
  const initialDimension = (searchParams.get("dimension") || "ALL").trim().toUpperCase();

  const [period, setPeriod] = useState<string>(() => currentAccountingPeriod());
  const [runs, setRuns] = useState<ReconciliationRun[]>([]);
  const [breaks, setBreaks] = useState<ReconciliationBreak[]>([]);
  const [selectedRunId, setSelectedRunId] = useState<string>(initialRunId);
  const [dimensionFilter, setDimensionFilter] = useState<string>(
    RECON_DIMENSIONS.includes(initialDimension as ReconDimension) ? initialDimension : "ALL",
  );
  const [statusFilter, setStatusFilter] = useState<string>("OPEN");
  const [severityFilter, setSeverityFilter] = useState<string>("ALL");
  const [resolutionNotes, setResolutionNotes] = useState<Record<string, string>>({});

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [runningRecon, setRunningRecon] = useState<boolean>(false);
  const [busyBreakId, setBusyBreakId] = useState<string | null>(null);
  const [historyTarget, setHistoryTarget] = useState<{
    objectType: string;
    fallbackObjectTypes?: string[];
    objectId: string;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadAll = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      try {
        const [runsRes, breaksRes] = await Promise.all([
          financeApi.listReconRuns({ limit: 20 }),
          financeApi.listReconBreaks({
            run_id: selectedRunId || undefined,
            dimension: dimensionFilter !== "ALL" ? dimensionFilter : undefined,
            status: statusFilter !== "ALL" ? statusFilter : undefined,
            severity: severityFilter !== "ALL" ? severityFilter : undefined,
            limit: 200,
          }),
        ]);
        const loadedRuns = runsRes.runs ?? [];
        setRuns(loadedRuns);
        setBreaks(breaksRes.breaks ?? []);
      } catch (err) {
        setError(err instanceof Error ? err.message : tRef.current("supplier_portal.finance.four_way_recon.load_failed"));
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [dimensionFilter, financeApi, selectedRunId, severityFilter, statusFilter],
  );

  useEffect(() => {
    void loadAll();
  }, [loadAll]);

  useSupplierSessionReconcile(() => {
    void loadAll(true);
  });

  const activeRun = useMemo<ReconciliationRun | null>(() => {
    if (runs.length === 0) return null;
    if (selectedRunId) {
      const exact = runs.find((r) => r.run_id === selectedRunId);
      if (exact) return exact;
    }
    return runs[0];
  }, [runs, selectedRunId]);

  const openBreakCount = useMemo(
    () => breaks.filter((b) => b.status === "OPEN" || b.status === "ACKNOWLEDGED").length,
    [breaks],
  );

  const handleTriggerRun = async () => {
    setRunningRecon(true);
    setError(null);
    setNotice(null);
    try {
      const targetPeriod = PERIOD_RE.test(period.trim()) ? period.trim() : undefined;
      const res = await financeApi.triggerReconRun({
        period_id: targetPeriod,
        mode: "MANUAL",
      });
      setSelectedRunId(res.run.run_id);
      setStatusFilter("ALL");
      setNotice(
        t("supplier_portal.finance.four_way_recon.run_success", {
          runId: res.run.run_id,
          count: res.run.break_count,
        }),
      );
      await loadAll(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.finance.four_way_recon.load_failed"));
    } finally {
      setRunningRecon(false);
    }
  };

  const handleBreakAction = async (
    brk: ReconciliationBreak,
    action: "ACKNOWLEDGE" | "RESOLVE" | "WAIVE",
  ) => {
    const note = (resolutionNotes[brk.break_id] || "").trim();
    if (action === "WAIVE" && !note) {
      setError(t("supplier_portal.finance.four_way_recon.waive_note_required"));
      return;
    }
    setBusyBreakId(brk.break_id);
    setError(null);
    setNotice(null);
    try {
      const res = await financeApi.updateReconBreak(brk.break_id, {
        action,
        note: note || undefined,
        resolution_note: note || undefined,
      });
      setBreaks((prev) => prev.map((item) => (item.break_id === brk.break_id ? res.break : item)));
      setResolutionNotes((prev) => {
        const next = { ...prev };
        delete next[brk.break_id];
        return next;
      });
      await loadAll(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.finance.four_way_recon.load_failed"));
    } finally {
      setBusyBreakId(null);
    }
  };

  const activeCurrency = activeRun?.currency || breaks[0]?.currency || "UZS";

  return (
    <PageChrome
      icon="reconcile"
      title={t("supplier_portal.finance.four_way_recon.title")}
      description={t("supplier_portal.finance.four_way_recon.subtitle")}
      loading={loading && runs.length === 0 && breaks.length === 0 && !error}
      skeletonVariant="dashboard"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/trial-balance" as Route} className="portal-btn portal-btn--outline">
            <Icon name="ledger" size={16} />
            <span>{t("portal.nav.trial_balance")}</span>
          </Link>
          <button
            type="button"
            disabled={runningRecon}
            onClick={() => void handleTriggerRun()}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="reconcile" size={16} />
            <span>
              {runningRecon
                ? t("supplier_portal.finance.four_way_recon.running")
                : t("supplier_portal.finance.four_way_recon.trigger_run")}
            </span>
          </button>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadAll(true)}
            className="portal-btn portal-btn--outline"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <p className="portal-alert-text">{error}</p>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      <div className="desk-card p-4 mb-6">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-4 items-end">
          <div>
            <label htmlFor="recon-period-input" className="portal-label block mb-2">
              {t("supplier_portal.finance.four_way_recon.period_label")}
            </label>
            <PortalInput
              id="recon-period-input"
              type="month"
              value={period}
              onChange={(e) => setPeriod(e.target.value)}
            />
          </div>

          <div>
            <label htmlFor="recon-dimension-filter" className="portal-label block mb-2">
              {t("supplier_portal.finance.four_way_recon.filter_dimension")}
            </label>
            <PortalSelect
              id="recon-dimension-filter"
              value={dimensionFilter}
              onChange={(e) => setDimensionFilter(e.target.value)}
            >
              <option value="ALL">{t("supplier_portal.finance.four_way_recon.all_filter")}</option>
              {RECON_DIMENSIONS.map((dim) => (
                <option key={dim} value={dim}>
                  {t(dimensionLabelKey(dim))}
                </option>
              ))}
            </PortalSelect>
          </div>

          <div>
            <label htmlFor="recon-status-filter" className="portal-label block mb-2">
              {t("supplier_portal.finance.four_way_recon.filter_status")}
            </label>
            <PortalSelect
              id="recon-status-filter"
              value={statusFilter}
              onChange={(e) => setStatusFilter(e.target.value)}
            >
              <option value="ALL">{t("supplier_portal.finance.four_way_recon.all_filter")}</option>
              {(["OPEN", "ACKNOWLEDGED", "RESOLVED", "WAIVED"] as ReconBreakStatus[]).map((st) => (
                <option key={st} value={st}>
                  {st}
                </option>
              ))}
            </PortalSelect>
          </div>

          <div>
            <label htmlFor="recon-severity-filter" className="portal-label block mb-2">
              {t("supplier_portal.finance.four_way_recon.filter_severity")}
            </label>
            <PortalSelect
              id="recon-severity-filter"
              value={severityFilter}
              onChange={(e) => setSeverityFilter(e.target.value)}
            >
              <option value="ALL">{t("supplier_portal.finance.four_way_recon.all_filter")}</option>
              {(["CRITICAL", "HIGH", "MEDIUM", "LOW"] as ReconBreakSeverity[]).map((sev) => (
                <option key={sev} value={sev}>
                  {sev}
                </option>
              ))}
            </PortalSelect>
          </div>
        </div>
      </div>

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.latest_run")}
          value={activeRun ? activeRun.status : "—"}
          sub={activeRun ? `${activeRun.period_id} · ${activeRun.mode}` : t("supplier_portal.finance.four_way_recon.no_runs")}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.open_breaks")}
          value={activeRun ? activeRun.break_count : openBreakCount}
          sub={`${openBreakCount} / ${breaks.length}`}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.total_exposure")}
          value={formatFinanceMinor(activeRun?.exposure_minor ?? 0, activeCurrency)}
          sub={activeCurrency}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.four_way_recon.last_run_at")}
          value={activeRun ? new Date(activeRun.started_at).toLocaleString() : "—"}
          sub={activeRun?.triggered_by || "—"}
        />
      </KpiStatGrid>

      <div className="mt-6">
        <PortalSection
          icon="reconcile"
          title={t("supplier_portal.finance.four_way_recon.dimensions_title")}
          description={activeRun ? `${activeRun.run_id} · ${activeRun.period_id}` : t("supplier_portal.finance.four_way_recon.no_runs")}
        >
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
            {RECON_DIMENSIONS.map((dim) => {
              const check = activeRun?.checks?.[dim];
              const isSelected = dimensionFilter === dim;
              return (
                <button
                  key={dim}
                  type="button"
                  onClick={() => setDimensionFilter((curr) => (curr === dim ? "ALL" : dim))}
                  className="text-left p-4 rounded-xl border transition-colors"
                  style={{
                    borderColor: isSelected ? "var(--desk-accent)" : "var(--desk-border)",
                    background: isSelected ? "var(--desk-surface-subtle)" : "var(--desk-surface)",
                  }}
                >
                  <div className="flex items-center justify-between gap-2">
                    <span className="text-sm font-semibold" style={{ color: "var(--desk-text-primary)" }}>
                      {t(dimensionLabelKey(dim))}
                    </span>
                    <StatusBadge state={check?.status ?? "PENDING"} />
                  </div>
                  {check ? (
                    <div className="mt-3 space-y-1 text-xs font-mono">
                      <div className="flex justify-between">
                        <span style={{ color: "var(--desk-text-tertiary)" }}>
                          {t("supplier_portal.finance.four_way_recon.expected")}
                        </span>
                        <span>{formatFinanceMinor(check.expected_minor, check.currency || activeCurrency)}</span>
                      </div>
                      <div className="flex justify-between">
                        <span style={{ color: "var(--desk-text-tertiary)" }}>
                          {t("supplier_portal.finance.four_way_recon.actual")}
                        </span>
                        <span>{formatFinanceMinor(check.actual_minor, check.currency || activeCurrency)}</span>
                      </div>
                      <div className="flex justify-between font-semibold pt-1 border-t" style={{ borderColor: "var(--desk-border)" }}>
                        <span>{t("supplier_portal.finance.four_way_recon.delta")}</span>
                        <span>
                          {formatFinanceMinor(check.delta_minor, check.currency || activeCurrency, {
                            signDisplay: "always",
                          })}
                        </span>
                      </div>
                      <div className="flex justify-between pt-1" style={{ color: "var(--desk-text-secondary)" }}>
                        <span>{t("supplier_portal.finance.four_way_recon.breaks_count")}</span>
                        <span>{check.break_count}</span>
                      </div>
                    </div>
                  ) : (
                    <p className="mt-3 text-xs" style={{ color: "var(--desk-text-tertiary)" }}>
                      {t("supplier_portal.finance.four_way_recon.no_runs")}
                    </p>
                  )}
                </button>
              );
            })}
          </div>
        </PortalSection>
      </div>

      <div className="mt-6">
        <PortalSection
          icon="warning"
          title={t("supplier_portal.finance.four_way_recon.breaks_title")}
          description={selectedRunId ? `Run ${selectedRunId}` : undefined}
          actions={
            <div className="flex flex-wrap items-center gap-2">
              {activeRun ? (
                <button
                  type="button"
                  onClick={() =>
                    setHistoryTarget((curr) =>
                      curr?.objectId === activeRun.run_id
                        ? null
                        : {
                            objectType: "recon_run",
                            fallbackObjectTypes: ["recon"],
                            objectId: activeRun.run_id,
                          },
                    )
                  }
                  className="portal-btn portal-btn--ghost"
                >
                  {t("supplier_portal.audit.open_history")}
                </button>
              ) : null}
              {selectedRunId ? (
                <button
                  type="button"
                  onClick={() => setSelectedRunId("")}
                  className="portal-btn portal-btn--ghost"
                >
                  {t("supplier_portal.finance.four_way_recon.all_filter")}
                </button>
              ) : null}
            </div>
          }
        >
          {breaks.length === 0 ? (
            <div className="py-10 text-center text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.finance.four_way_recon.no_breaks")}
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-sm">
                <thead>
                  <tr
                    className="border-b text-xs uppercase tracking-wider"
                    style={{
                      borderColor: "var(--desk-border)",
                      color: "var(--desk-text-tertiary)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.four_way_recon.dimension")}</th>
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.four_way_recon.object_ref")}</th>
                    <th className="py-3 px-3 font-semibold text-right">{t("supplier_portal.finance.four_way_recon.expected")}</th>
                    <th className="py-3 px-3 font-semibold text-right">{t("supplier_portal.finance.four_way_recon.actual")}</th>
                    <th className="py-3 px-3 font-semibold text-right">{t("supplier_portal.finance.four_way_recon.delta")}</th>
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.four_way_recon.severity")}</th>
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.four_way_recon.status")}</th>
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.four_way_recon.sla_due")}</th>
                    <th className="py-3 px-3 font-semibold">{t("supplier_portal.finance.gl_ledger.actions")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {breaks.map((brk) => {
                    const isTerminal = brk.status === "RESOLVED" || brk.status === "WAIVED";
                    const busy = busyBreakId === brk.break_id;
                    return (
                      <tr key={brk.break_id} className="hover:bg-[var(--desk-surface-subtle)] align-top">
                        <td className="py-3 px-3 text-xs font-mono font-semibold">
                          <div>{brk.dimension}</div>
                          <div style={{ color: "var(--desk-text-tertiary)" }}>{brk.break_id}</div>
                        </td>
                        <td className="py-3 px-3 text-xs font-mono">
                          <div>{brk.object_type}</div>
                          <div style={{ color: "var(--desk-text-secondary)" }}>{brk.object_id}</div>
                          {brk.control_tower_exception_id ? (
                            <Link
                              href={"/control-tower" as Route}
                              className="inline-block mt-1 underline"
                              style={{ color: "var(--desk-text-secondary)" }}
                            >
                              {t("supplier_portal.finance.four_way_recon.control_tower_ref")}: {brk.control_tower_exception_id}
                            </Link>
                          ) : null}
                        </td>
                        <td className="py-3 px-3 font-mono text-xs text-right whitespace-nowrap">
                          {formatFinanceMinor(brk.expected_minor, brk.currency)}
                        </td>
                        <td className="py-3 px-3 font-mono text-xs text-right whitespace-nowrap">
                          {formatFinanceMinor(brk.actual_minor, brk.currency)}
                        </td>
                        <td className="py-3 px-3 font-mono text-xs font-semibold text-right whitespace-nowrap">
                          {formatFinanceMinor(brk.delta_minor, brk.currency, { signDisplay: "always" })}
                        </td>
                        <td className="py-3 px-3">
                          <StatusBadge state={brk.severity} />
                        </td>
                        <td className="py-3 px-3">
                          <StatusBadge state={brk.status} />
                          {brk.resolution_note ? (
                            <p className="mt-1 text-xs max-w-[200px]" style={{ color: "var(--desk-text-secondary)" }}>
                              {brk.resolution_note}
                            </p>
                          ) : null}
                        </td>
                        <td className="py-3 px-3 text-xs whitespace-nowrap" style={{ color: "var(--desk-text-secondary)" }}>
                          {brk.sla_due_at ? new Date(brk.sla_due_at).toLocaleString() : "—"}
                        </td>
                        <td className="py-3 px-3">
                          {!isTerminal ? (
                            <div className="space-y-2 min-w-[220px]">
                              <input
                                type="text"
                                aria-label={`Note for ${brk.break_id}`}
                                className="portal-input min-h-[34px] text-xs px-2.5"
                                placeholder={t("supplier_portal.finance.four_way_recon.resolution_note_placeholder")}
                                value={resolutionNotes[brk.break_id] ?? ""}
                                onChange={(e) =>
                                  setResolutionNotes((prev) => ({
                                    ...prev,
                                    [brk.break_id]: e.target.value,
                                  }))
                                }
                              />
                              <div className="flex flex-wrap gap-1.5">
                                {brk.status === "OPEN" ? (
                                  <button
                                    type="button"
                                    disabled={busy}
                                    onClick={() => void handleBreakAction(brk, "ACKNOWLEDGE")}
                                    className="rounded-lg border px-2 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)]"
                                    style={{ borderColor: "var(--desk-border)" }}
                                  >
                                    {t("supplier_portal.finance.four_way_recon.action_acknowledge")}
                                  </button>
                                ) : null}
                                <button
                                  type="button"
                                  disabled={busy}
                                  onClick={() => void handleBreakAction(brk, "RESOLVE")}
                                  className="rounded-lg border px-2 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)]"
                                  style={{ borderColor: "var(--desk-border-strong)" }}
                                >
                                  {t("supplier_portal.finance.four_way_recon.action_resolve")}
                                </button>
                                <button
                                  type="button"
                                  disabled={busy}
                                  onClick={() => void handleBreakAction(brk, "WAIVE")}
                                  className="rounded-lg border px-2 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)]"
                                  style={{ borderColor: "var(--desk-border)" }}
                                >
                                  {t("supplier_portal.finance.four_way_recon.action_waive")}
                                </button>
                                <button
                                  type="button"
                                  onClick={() =>
                                    setHistoryTarget((curr) =>
                                      curr?.objectId === brk.break_id
                                        ? null
                                        : {
                                            objectType: "recon",
                                            fallbackObjectTypes: ["recon_break", brk.object_type].filter(Boolean),
                                            objectId: brk.break_id,
                                          },
                                    )
                                  }
                                  className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                                  style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-secondary)" }}
                                >
                                  {t("supplier_portal.audit.open_history")}
                                </button>
                              </div>
                            </div>
                          ) : (
                            <button
                              type="button"
                              onClick={() =>
                                setHistoryTarget((curr) =>
                                  curr?.objectId === brk.break_id
                                    ? null
                                    : {
                                        objectType: "recon",
                                        fallbackObjectTypes: ["recon_break", brk.object_type].filter(Boolean),
                                        objectId: brk.break_id,
                                      },
                                )
                              }
                              className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                              style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-secondary)" }}
                            >
                              {t("supplier_portal.audit.open_history")}
                            </button>
                          )}
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

      {historyTarget ? (
        <div className="mt-6">
          <ObjectHistoryPanel
            objectType={historyTarget.objectType}
            fallbackObjectTypes={historyTarget.fallbackObjectTypes}
            objectId={historyTarget.objectId}
            compact
            autoVerifyChain
            onClose={() => setHistoryTarget(null)}
          />
        </div>
      ) : null}

      <div className="mt-6">
        <PortalSection
          icon="overview"
          title={t("supplier_portal.finance.four_way_recon.runs_history_title")}
        >
          {runs.length === 0 ? (
            <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.finance.four_way_recon.no_runs")}
            </p>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left border-collapse text-sm">
                <thead>
                  <tr
                    className="border-b text-xs uppercase tracking-wider"
                    style={{
                      borderColor: "var(--desk-border)",
                      color: "var(--desk-text-tertiary)",
                      background: "var(--desk-surface-subtle)",
                    }}
                  >
                    <th className="py-2.5 px-3">{t("supplier_portal.finance.four_way_recon.run_id")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.finance.four_way_recon.period_label")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.finance.four_way_recon.mode")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.finance.four_way_recon.status")}</th>
                    <th className="py-2.5 px-3 text-right">{t("supplier_portal.finance.four_way_recon.breaks_count")}</th>
                    <th className="py-2.5 px-3 text-right">{t("supplier_portal.finance.four_way_recon.exposure")}</th>
                    <th className="py-2.5 px-3">{t("supplier_portal.finance.four_way_recon.started_at")}</th>
                  </tr>
                </thead>
                <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                  {runs.map((run) => {
                    const selected = activeRun?.run_id === run.run_id;
                    return (
                      <tr
                        key={run.run_id}
                        onClick={() => setSelectedRunId(run.run_id)}
                        className="cursor-pointer transition-colors hover:bg-[var(--desk-surface-subtle)]"
                        style={selected ? { background: "var(--desk-surface-subtle)" } : undefined}
                      >
                        <td className="py-2.5 px-3 font-mono text-xs font-semibold">{run.run_id}</td>
                        <td className="py-2.5 px-3 font-mono text-xs">{run.period_id}</td>
                        <td className="py-2.5 px-3 font-mono text-xs">{run.mode}</td>
                        <td className="py-2.5 px-3">
                          <StatusBadge state={run.status} />
                        </td>
                        <td className="py-2.5 px-3 font-mono text-xs text-right">{run.break_count}</td>
                        <td className="py-2.5 px-3 font-mono text-xs text-right">
                          {formatFinanceMinor(run.exposure_minor, run.currency)}
                        </td>
                        <td className="py-2.5 px-3 text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                          {new Date(run.started_at).toLocaleString()}
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
