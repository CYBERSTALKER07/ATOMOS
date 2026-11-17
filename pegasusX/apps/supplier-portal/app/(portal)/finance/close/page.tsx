"use client";

import Link from "next/link";
import type { Route } from "next";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
import { createSupplierApi } from "@/lib/api";
import { currentAccountingPeriod } from "@/lib/finance-api";
import {
  createControlAccountingApi,
  extractControlError,
  formatFinanceMinor,
  normalizeMinorBigInt,
  type AccountingPeriod,
  type PeriodChecklist,
  type PeriodSnapshotReport,
} from "@/lib/control-accounting-api";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { PageChrome } from "@/components/PageChrome";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import { ObjectHistoryPanel } from "@/components/audit/ObjectHistoryPanel";
import StatusBadge from "@/components/StatusBadge";
import Icon from "@/components/Icon";

const PERIOD_RE = /^\d{4}-(0[1-9]|1[0-2])$/;
const PERIOD_STATUSES = ["ALL", "OPEN", "SOFT_CLOSED", "CLOSED", "REOPENED"] as const;

function shortHash(hash?: string): string {
  const clean = (hash || "").trim();
  if (!clean) return "—";
  if (clean.length <= 18) return clean;
  return `${clean.slice(0, 10)}…${clean.slice(-8)}`;
}

function formatIso(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

function renderSummaryPairs(summary: Record<string, unknown> | undefined): Array<{ label: string; value: string }> {
  if (!summary || typeof summary !== "object") return [];
  const currency = typeof summary.currency === "string" && summary.currency.trim() ? summary.currency.trim() : "UZS";
  const rows: Array<{ label: string; value: string }> = [];
  for (const [k, v] of Object.entries(summary)) {
    if (v === null || v === undefined || typeof v === "object") continue;
    if (k.endsWith("_minor") && (typeof v === "number" || typeof v === "bigint")) {
      rows.push({
        label: k,
        value: formatFinanceMinor(normalizeMinorBigInt(v), currency),
      });
    } else {
      rows.push({ label: k, value: String(v) });
    }
  }
  return rows.slice(0, 8);
}

export default function PeriodClosePage() {
  return (
    <Suspense fallback={null}>
      <PeriodCloseInner />
    </Suspense>
  );
}

function PeriodCloseInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const controlApi = useMemo(() => createControlAccountingApi(createSupplierApi()), []);

  const initialPeriodParam = (searchParams.get("period") || "").trim();
  const initialEntityParam = (searchParams.get("entity") || "").trim();

  const [entityParam] = useState<string>(initialEntityParam);
  const [statusFilter, setStatusFilter] = useState<string>("ALL");
  const [periods, setPeriods] = useState<AccountingPeriod[]>([]);
  const [entityTimezone, setEntityTimezone] = useState<string>("Asia/Tashkent");
  const [selectedPeriodId, setSelectedPeriodId] = useState<string>(
    PERIOD_RE.test(initialPeriodParam) ? initialPeriodParam : currentAccountingPeriod(),
  );

  // Create / open period form
  const [newPeriodId, setNewPeriodId] = useState<string>(() => currentAccountingPeriod());
  const [newTimezone, setNewTimezone] = useState<string>("");

  // Checklist & Snapshot
  const [checklist, setChecklist] = useState<PeriodChecklist | null>(null);
  const [checklistLoading, setChecklistLoading] = useState<boolean>(false);
  const [snapshot, setSnapshot] = useState<PeriodSnapshotReport | null>(null);
  const [csvPreview, setCsvPreview] = useState<string | null>(null);
  const [snapshotLoading, setSnapshotLoading] = useState<boolean>(false);

  // Reopen 2-approver / SoD state
  const [reopenReason, setReopenReason] = useState<string>("");
  const [reopenApprovalId, setReopenApprovalId] = useState<string>("");
  const [pendingReopen, setPendingReopen] = useState<{
    approval_request_id: string;
    required_approvers?: number;
    payload_hash?: string;
  } | null>(null);

  const [showHistory, setShowHistory] = useState<boolean>(false);
  const [loading, setLoading] = useState<boolean>(true);
  const [refreshing, setRefreshing] = useState<boolean>(false);
  const [busyAction, setBusyAction] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [errorCode, setErrorCode] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadPeriods = useCallback(
    async (silent = false) => {
      if (!silent) {
        setLoading(true);
      } else {
        setRefreshing(true);
      }
      setError(null);
      setErrorCode(null);
      try {
        const res = await controlApi.listPeriods({
          entity: entityParam || undefined,
          status: statusFilter !== "ALL" ? statusFilter : undefined,
        });
        const list = res.periods ?? [];
        setPeriods(list);
        if (res.timezone) {
          setEntityTimezone(res.timezone);
        }
        if (list.length > 0 && !list.some((p) => p.period_id === selectedPeriodId)) {
          setSelectedPeriodId(list[0].period_id);
        }
      } catch (err) {
        const parsed = extractControlError(err, tRef.current("supplier_portal.finance.close.load_failed"));
        setError(parsed.message);
        setErrorCode(parsed.code ?? null);
      } finally {
        setLoading(false);
        setRefreshing(false);
      }
    },
    [controlApi, entityParam, selectedPeriodId, statusFilter],
  );

  useEffect(() => {
    void loadPeriods();
  }, [loadPeriods]);

  useSupplierSessionReconcile(() => {
    void loadPeriods(true);
  });

  const selectedPeriod = useMemo<AccountingPeriod | null>(
    () => periods.find((p) => p.period_id === selectedPeriodId) ?? null,
    [periods, selectedPeriodId],
  );

  const activeTimezone = selectedPeriod?.timezone || entityTimezone || "Asia/Tashkent";

  const handleSelectPeriod = (period: AccountingPeriod) => {
    setSelectedPeriodId(period.period_id);
    setChecklist(null);
    setSnapshot(null);
    setCsvPreview(null);
    setPendingReopen(null);
  };

  const handleCreatePeriod = async (e: React.FormEvent) => {
    e.preventDefault();
    const cleanId = newPeriodId.trim();
    if (!PERIOD_RE.test(cleanId)) {
      setError(t("supplier_portal.finance.close.invalid_period_format"));
      return;
    }
    setBusyAction("create");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.createPeriod({
        entity_id: entityParam || undefined,
        period_id: cleanId,
        timezone: newTimezone.trim() || undefined,
      });
      setSelectedPeriodId(res.period.period_id);
      setNotice(
        t("supplier_portal.finance.close.period_created", { periodId: res.period.period_id }),
      );
      await loadPeriods(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleEvaluateChecklist = async (targetPeriodId = selectedPeriodId) => {
    if (!targetPeriodId) return;
    setChecklistLoading(true);
    setError(null);
    setErrorCode(null);
    try {
      const res = await controlApi.evaluatePeriodChecklist(targetPeriodId, {
        entity: entityParam || undefined,
      });
      setChecklist(res);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setChecklistLoading(false);
    }
  };

  const handleSoftClose = async (targetPeriodId = selectedPeriodId) => {
    if (!targetPeriodId) return;
    setBusyAction("soft-close");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.softClosePeriod(targetPeriodId, {
        entity_id: entityParam || undefined,
      });
      setNotice(
        t("supplier_portal.finance.close.soft_close_success", { periodId: res.period.period_id }),
      );
      await loadPeriods(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleHardClose = async (targetPeriodId = selectedPeriodId) => {
    if (!targetPeriodId) return;
    setBusyAction("close");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.closePeriod(targetPeriodId, {
        entity_id: entityParam || undefined,
      });
      if (res.checklist) setChecklist(res.checklist);
      if (res.snapshot) setSnapshot(res.snapshot);
      setNotice(
        t("supplier_portal.finance.close.hard_close_success", { periodId: res.period.period_id }),
      );
      await loadPeriods(true);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
      if (parsed.payload?.checklist && typeof parsed.payload.checklist === "object") {
        setChecklist(parsed.payload.checklist as PeriodChecklist);
      }
    } finally {
      setBusyAction(null);
    }
  };

  const handleReopen = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedPeriodId) return;
    const reason = reopenReason.trim();
    if (!reason) {
      setError(t("supplier_portal.finance.close.reopen_reason_required"));
      return;
    }
    setBusyAction("reopen");
    setError(null);
    setErrorCode(null);
    setNotice(null);
    try {
      const res = await controlApi.reopenPeriod(selectedPeriodId, {
        entity_id: entityParam || undefined,
        reason,
        approval_id: reopenApprovalId.trim() || undefined,
      });
      if (res.status === "PENDING_APPROVAL" && res.approval_request_id) {
        setPendingReopen({
          approval_request_id: res.approval_request_id,
          required_approvers: res.required_approvers,
          payload_hash: res.payload_hash,
        });
        setReopenApprovalId(res.approval_request_id);
        setNotice(
          t("supplier_portal.finance.close.reopen_pending_approval", {
            requestId: res.approval_request_id,
            approvers: res.required_approvers ?? 2,
          }),
        );
      } else {
        setPendingReopen(null);
        setNotice(
          t("supplier_portal.finance.close.reopen_success", {
            periodId: res.period?.period_id || selectedPeriodId,
          }),
        );
        await loadPeriods(true);
      }
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setBusyAction(null);
    }
  };

  const handleLoadSnapshotJson = async () => {
    if (!selectedPeriodId) return;
    setSnapshotLoading(true);
    setError(null);
    setErrorCode(null);
    try {
      const res = await controlApi.getPeriodSnapshot(selectedPeriodId, {
        entity: entityParam || undefined,
      });
      setSnapshot(res);
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setSnapshotLoading(false);
    }
  };

  const handleDownloadSnapshotCsv = async (triggerDownload: boolean) => {
    if (!selectedPeriodId) return;
    setSnapshotLoading(true);
    setError(null);
    setErrorCode(null);
    try {
      const csv = await controlApi.downloadPeriodSnapshotCsv(selectedPeriodId, {
        entity: entityParam || undefined,
      });
      setCsvPreview(csv);
      if (triggerDownload && typeof window !== "undefined" && typeof URL.createObjectURL === "function") {
        try {
          const blob = new Blob([csv], { type: "text/csv;charset=utf-8" });
          const url = URL.createObjectURL(blob);
          const a = document.createElement("a");
          a.href = url;
          a.download = `period-snapshot-${selectedPeriodId}.csv`;
          document.body.appendChild(a);
          a.click();
          a.remove();
          URL.revokeObjectURL(url);
        } catch {
          // Preview remains visible in environments without browser download support.
        }
      }
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.finance.close.load_failed"));
      setError(parsed.message);
      setErrorCode(parsed.code ?? null);
    } finally {
      setSnapshotLoading(false);
    }
  };

  const activeSnapshotHash = snapshot?.snapshot_hash || selectedPeriod?.snapshot_hash || "";
  const activeSnapshotUri = snapshot?.snapshot_uri || selectedPeriod?.snapshot_uri || "";

  return (
    <PageChrome
      icon="calendar"
      title={t("supplier_portal.finance.close.title")}
      description={t("supplier_portal.finance.close.subtitle")}
      loading={loading && periods.length === 0 && !error}
      skeletonVariant="table"
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/finance/trial-balance" as Route} className="portal-btn portal-btn--outline">
            <Icon name="ledger" size={16} />
            <span>{t("portal.nav.trial_balance")}</span>
          </Link>
          <Link href={"/finance/reconciliation" as Route} className="portal-btn portal-btn--outline">
            <Icon name="reconcile" size={16} />
            <span>{t("portal.nav.four_way_reconciliation")}</span>
          </Link>
          <Link href={"/approvals" as Route} className="portal-btn portal-btn--outline">
            <Icon name="verified" size={16} />
            <span>{t("portal.nav.approvals")}</span>
          </Link>
          <button
            type="button"
            disabled={loading || refreshing}
            onClick={() => void loadPeriods(true)}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="refresh" size={16} className={refreshing ? "animate-spin" : ""} />
            <span>{refreshing ? t("common.status.loading") : t("common.action.refresh")}</span>
          </button>
        </div>
      }
    >
      {error ? (
        <div className="portal-alert portal-alert--error mb-6" role="alert">
          <div>
            <p className="portal-alert-text font-semibold">{error}</p>
            {errorCode ? (
              <p className="text-xs font-mono mt-1" style={{ color: "var(--desk-danger)" }}>
                {errorCode}
              </p>
            ) : null}
          </div>
        </div>
      ) : null}

      {notice ? (
        <div className="portal-alert portal-alert--info mb-6">
          <p className="portal-alert-text">{notice}</p>
        </div>
      ) : null}

      {pendingReopen ? (
        <div
          className="desk-card p-4 mb-6 flex flex-wrap items-center justify-between gap-3"
          style={{ borderColor: "var(--desk-warning)", background: "var(--desk-warning-soft)" }}
        >
          <div className="space-y-1 text-sm">
            <div className="font-semibold" style={{ color: "var(--desk-text-primary)" }}>
              {t("supplier_portal.finance.close.two_approver_banner_title")}
            </div>
            <div className="text-xs font-mono" style={{ color: "var(--desk-text-secondary)" }}>
              {pendingReopen.approval_request_id} · {t("supplier_portal.finance.close.required_approvers_label")}:{" "}
              {pendingReopen.required_approvers ?? 2}
              {pendingReopen.payload_hash ? ` · ${shortHash(pendingReopen.payload_hash)}` : ""}
            </div>
          </div>
          <Link
            href={`/approvals?requestId=${encodeURIComponent(pendingReopen.approval_request_id)}` as Route}
            className="portal-btn portal-btn--primary"
          >
            <Icon name="verified" size={16} />
            <span>{t("supplier_portal.finance.close.open_in_approvals")}</span>
          </Link>
        </div>
      ) : null}

      <KpiStatGrid columns={4}>
        <KpiStatCard
          label={t("supplier_portal.finance.close.kpi_selected_period")}
          value={selectedPeriodId || "—"}
          sub={
            selectedPeriod ? (
              <span className="inline-flex items-center gap-2">
                <StatusBadge state={selectedPeriod.status} />
              </span>
            ) : (
              t("supplier_portal.finance.close.period_not_registered")
            )
          }
        />
        <KpiStatCard
          label={t("supplier_portal.finance.close.kpi_timezone_cutoff")}
          value={activeTimezone}
          sub={
            selectedPeriod?.cutoff_local ||
            (selectedPeriod?.period_end_utc
              ? `${formatIso(selectedPeriod.period_start_utc)} → ${formatIso(selectedPeriod.period_end_utc)}`
              : t("supplier_portal.finance.close.cutoff_default_hint"))
          }
        />
        <KpiStatCard
          label={t("supplier_portal.finance.close.kpi_checklist")}
          value={
            checklist
              ? checklist.ready_to_close
                ? t("supplier_portal.finance.close.checklist_ready")
                : t("supplier_portal.finance.close.checklist_blocked", { count: checklist.blocking_count })
              : "—"
          }
          sub={t("supplier_portal.finance.close.checklist_sub")}
        />
        <KpiStatCard
          label={t("supplier_portal.finance.close.kpi_snapshot_hash")}
          value={<span className="font-mono text-base">{shortHash(activeSnapshotHash)}</span>}
          sub={activeSnapshotUri || t("supplier_portal.finance.close.snapshot_pending")}
        />
      </KpiStatGrid>

      <div className="mt-6 grid grid-cols-1 lg:grid-cols-3 gap-6">
        <div className="lg:col-span-2">
          <PortalSection
            icon="calendar"
            title={t("supplier_portal.finance.close.periods_table_title")}
            description={t("supplier_portal.finance.close.periods_table_desc")}
            actions={
              <div className="flex items-center gap-2">
                <label htmlFor="close-status-filter" className="sr-only">
                  {t("supplier_portal.finance.close.status_filter_label")}
                </label>
                <PortalSelect
                  id="close-status-filter"
                  value={statusFilter}
                  onChange={(e) => setStatusFilter(e.target.value)}
                >
                  {PERIOD_STATUSES.map((st) => (
                    <option key={st} value={st}>
                      {st === "ALL" ? t("supplier_portal.finance.close.all_statuses") : st}
                    </option>
                  ))}
                </PortalSelect>
              </div>
            }
          >
            {periods.length === 0 ? (
              <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
                {t("supplier_portal.finance.close.no_periods")}
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
                      <th className="py-2.5 px-3">{t("supplier_portal.finance.close.col_period")}</th>
                      <th className="py-2.5 px-3">{t("supplier_portal.finance.close.col_status")}</th>
                      <th className="py-2.5 px-3">{t("supplier_portal.finance.close.col_timezone_cutoff")}</th>
                      <th className="py-2.5 px-3">{t("supplier_portal.finance.close.col_close_audit")}</th>
                      <th className="py-2.5 px-3">{t("supplier_portal.finance.close.col_snapshot_hash")}</th>
                      <th className="py-2.5 px-3 text-right">{t("supplier_portal.finance.close.col_actions")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {periods.map((p) => {
                      const isSelected = p.period_id === selectedPeriodId;
                      return (
                        <tr
                          key={p.period_id}
                          onClick={() => handleSelectPeriod(p)}
                          className="cursor-pointer transition-colors hover:bg-[var(--desk-surface-subtle)]"
                          style={isSelected ? { background: "var(--desk-surface-subtle)" } : undefined}
                        >
                          <td className="py-2.5 px-3 font-mono font-semibold">{p.period_id}</td>
                          <td className="py-2.5 px-3">
                            <StatusBadge state={p.status} />
                          </td>
                          <td className="py-2.5 px-3 text-xs">
                            <div className="font-mono font-medium">{p.timezone}</div>
                            <div style={{ color: "var(--desk-text-tertiary)" }}>
                              {p.cutoff_local || (p.period_end_utc ? formatIso(p.period_end_utc) : "—")}
                            </div>
                          </td>
                          <td className="py-2.5 px-3 text-xs">
                            {p.closed_by ? (
                              <div>
                                <span className="font-mono">{p.closed_by}</span>
                                <div style={{ color: "var(--desk-text-tertiary)" }}>{formatIso(p.closed_at)}</div>
                              </div>
                            ) : p.soft_closed_by ? (
                              <div>
                                <span className="font-mono">{p.soft_closed_by}</span>
                                <div style={{ color: "var(--desk-text-tertiary)" }}>{formatIso(p.soft_closed_at)}</div>
                              </div>
                            ) : (
                              "—"
                            )}
                          </td>
                          <td className="py-2.5 px-3 font-mono text-xs">{shortHash(p.snapshot_hash)}</td>
                          <td className="py-2.5 px-3 text-right">
                            <div className="flex flex-wrap justify-end gap-1.5" onClick={(e) => e.stopPropagation()}>
                              <button
                                type="button"
                                onClick={() => {
                                  handleSelectPeriod(p);
                                  void handleEvaluateChecklist(p.period_id);
                                }}
                                className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)]"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {t("supplier_portal.finance.close.btn_checklist")}
                              </button>
                              <button
                                type="button"
                                disabled={busyAction !== null || p.status === "CLOSED"}
                                onClick={() => {
                                  handleSelectPeriod(p);
                                  void handleSoftClose(p.period_id);
                                }}
                                className="rounded-lg border px-2 py-1 text-xs font-medium hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                style={{ borderColor: "var(--desk-border)" }}
                              >
                                {t("supplier_portal.finance.close.btn_soft_close")}
                              </button>
                              <button
                                type="button"
                                disabled={busyAction !== null || p.status === "CLOSED"}
                                onClick={() => {
                                  handleSelectPeriod(p);
                                  void handleHardClose(p.period_id);
                                }}
                                className="rounded-lg border px-2 py-1 text-xs font-semibold hover:bg-[var(--desk-surface-subtle)] disabled:opacity-50"
                                style={{ borderColor: "var(--desk-border-strong)" }}
                              >
                                {t("supplier_portal.finance.close.btn_hard_close")}
                              </button>
                            </div>
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

        <div className="space-y-6">
          <PortalSection
            icon="plus"
            title={t("supplier_portal.finance.close.open_period_title")}
            description={t("supplier_portal.finance.close.open_period_desc")}
          >
            <form onSubmit={(e) => void handleCreatePeriod(e)} className="space-y-3">
              <div>
                <label htmlFor="new-period-id" className="portal-label block mb-1.5">
                  {t("supplier_portal.finance.close.period_id_label")}
                </label>
                <PortalInput
                  id="new-period-id"
                  type="month"
                  value={newPeriodId}
                  onChange={(e) => setNewPeriodId(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="new-period-tz" className="portal-label block mb-1.5">
                  {t("supplier_portal.finance.close.timezone_label")}
                </label>
                <PortalInput
                  id="new-period-tz"
                  type="text"
                  placeholder={entityTimezone || "Asia/Tashkent"}
                  value={newTimezone}
                  onChange={(e) => setNewTimezone(e.target.value)}
                />
              </div>
              <button
                type="submit"
                disabled={busyAction !== null}
                className="portal-btn portal-btn--primary w-full"
              >
                <Icon name="plus" size={16} />
                <span>{t("supplier_portal.finance.close.create_period_button")}</span>
              </button>
            </form>
          </PortalSection>

          <PortalSection
            icon="warning"
            title={t("supplier_portal.finance.close.reopen_section_title")}
            description={t("supplier_portal.finance.close.reopen_section_desc")}
          >
            <form onSubmit={(e) => void handleReopen(e)} className="space-y-3">
              <div>
                <label htmlFor="reopen-reason-input" className="portal-label block mb-1.5">
                  {t("supplier_portal.finance.close.reopen_reason_label")}
                </label>
                <PortalInput
                  id="reopen-reason-input"
                  type="text"
                  placeholder={t("supplier_portal.finance.close.reopen_reason_placeholder")}
                  value={reopenReason}
                  onChange={(e) => setReopenReason(e.target.value)}
                />
              </div>
              <div>
                <label htmlFor="reopen-approval-id" className="portal-label block mb-1.5">
                  {t("supplier_portal.finance.close.reopen_approval_id_label")}
                </label>
                <PortalInput
                  id="reopen-approval-id"
                  type="text"
                  placeholder="apr-..."
                  value={reopenApprovalId}
                  onChange={(e) => setReopenApprovalId(e.target.value)}
                />
              </div>
              <button
                type="submit"
                disabled={busyAction !== null || !selectedPeriodId}
                className="portal-btn portal-btn--outline w-full"
              >
                <Icon name="returns" size={16} />
                <span>
                  {busyAction === "reopen"
                    ? t("common.status.loading")
                    : t("supplier_portal.finance.close.btn_reopen_period", { periodId: selectedPeriodId })}
                </span>
              </button>
            </form>
          </PortalSection>
        </div>
      </div>

      <div className="mt-6 grid grid-cols-1 lg:grid-cols-2 gap-6">
        <PortalSection
          icon="reconcile"
          title={t("supplier_portal.finance.close.checklist_title")}
          description={t("supplier_portal.finance.close.checklist_desc")}
          actions={
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                disabled={checklistLoading || !selectedPeriodId}
                onClick={() => void handleEvaluateChecklist()}
                className="portal-btn portal-btn--outline"
              >
                <Icon name="reconcile" size={16} />
                <span>
                  {checklistLoading
                    ? t("common.status.loading")
                    : t("supplier_portal.finance.close.evaluate_checklist_button")}
                </span>
              </button>
              <button
                type="button"
                disabled={busyAction !== null || !selectedPeriodId}
                onClick={() => void handleHardClose()}
                className="portal-btn portal-btn--primary"
              >
                <Icon name="verified" size={16} />
                <span>{t("supplier_portal.finance.close.btn_hard_close")}</span>
              </button>
            </div>
          }
        >
          {!checklist ? (
            <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.finance.close.checklist_empty")}
            </p>
          ) : (
            <div className="space-y-4">
              <div
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border p-3 text-xs"
                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
              >
                <div className="space-y-0.5">
                  <div className="font-mono font-semibold">
                    {checklist.period_id} · {checklist.timezone}
                  </div>
                  <div style={{ color: "var(--desk-text-secondary)" }}>
                    {formatIso(checklist.period_start_utc)} → {formatIso(checklist.period_end_utc)}
                  </div>
                </div>
                <StatusBadge state={checklist.ready_to_close ? "PASS" : "FAIL"} />
              </div>

              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse text-xs">
                  <thead>
                    <tr className="border-b uppercase" style={{ borderColor: "var(--desk-border)", color: "var(--desk-text-tertiary)" }}>
                      <th className="py-2 px-2.5">{t("supplier_portal.finance.close.checklist_col_item")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.finance.close.col_status")}</th>
                      <th className="py-2 px-2.5 text-right">{t("supplier_portal.finance.close.checklist_col_open")}</th>
                      <th className="py-2 px-2.5">{t("supplier_portal.finance.close.checklist_col_detail")}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y" style={{ borderColor: "var(--desk-border)" }}>
                    {(checklist.items ?? []).map((item) => (
                      <tr key={item.key}>
                        <td className="py-2 px-2.5">
                          <div className="font-medium">{item.label}</div>
                          <div className="font-mono text-[11px]" style={{ color: "var(--desk-text-tertiary)" }}>
                            {item.key} {item.blocking ? "· BLOCKING" : "· ADVISORY"}
                          </div>
                        </td>
                        <td className="py-2 px-2.5">
                          <StatusBadge state={item.passed ? "PASS" : item.blocking ? "FAIL" : "WARN"} />
                        </td>
                        <td className="py-2 px-2.5 font-mono text-right">{item.open_count}</td>
                        <td className="py-2 px-2.5" style={{ color: "var(--desk-text-secondary)" }}>
                          {item.detail || "—"}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}
        </PortalSection>

        <PortalSection
          icon="ledger"
          title={t("supplier_portal.finance.close.snapshot_title")}
          description={t("supplier_portal.finance.close.snapshot_desc")}
          actions={
            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                disabled={snapshotLoading || !selectedPeriodId}
                onClick={() => void handleLoadSnapshotJson()}
                className="portal-btn portal-btn--outline"
              >
                <span>{t("supplier_portal.finance.close.load_snapshot_json")}</span>
              </button>
              <button
                type="button"
                disabled={snapshotLoading || !selectedPeriodId}
                onClick={() => void handleDownloadSnapshotCsv(true)}
                className="portal-btn portal-btn--primary"
              >
                <span>{t("supplier_portal.finance.close.download_snapshot_csv")}</span>
              </button>
              <button
                type="button"
                onClick={() => setShowHistory((v) => !v)}
                className="portal-btn portal-btn--ghost"
              >
                <span>{t("supplier_portal.audit.open_history")}</span>
              </button>
            </div>
          }
        >
          <div
            className="rounded-xl border p-3 space-y-1 text-xs"
            style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
          >
            <div>
              <span style={{ color: "var(--desk-text-tertiary)" }}>SHA-256: </span>
              <span className="font-mono break-all">{activeSnapshotHash || "—"}</span>
            </div>
            <div>
              <span style={{ color: "var(--desk-text-tertiary)" }}>URI: </span>
              <span className="font-mono break-all">{activeSnapshotUri || "—"}</span>
            </div>
          </div>

          {snapshot ? (
            <div className="space-y-3">
              <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {(
                  [
                    ["trial_balance", snapshot.trial_balance],
                    ["ar_summary", snapshot.ar_summary],
                    ["valuation_summary", snapshot.valuation_summary],
                    ["vat_summary", snapshot.vat_summary],
                  ] as const
                ).map(([sectionName, sectionObj]) => {
                  const pairs = renderSummaryPairs(sectionObj);
                  if (pairs.length === 0) return null;
                  return (
                    <div
                      key={sectionName}
                      className="rounded-xl border p-3 text-xs space-y-1.5"
                      style={{ borderColor: "var(--desk-border)" }}
                    >
                      <div className="font-mono font-semibold uppercase">{sectionName}</div>
                      {pairs.map((pair) => (
                        <div key={pair.label} className="flex justify-between gap-2">
                          <span style={{ color: "var(--desk-text-tertiary)" }}>{pair.label}</span>
                          <span className="font-mono">{pair.value}</span>
                        </div>
                      ))}
                    </div>
                  );
                })}
              </div>
              <pre
                className="max-h-56 overflow-auto rounded-xl border p-3 font-mono text-[11px]"
                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
              >
                {JSON.stringify(snapshot, null, 2)}
              </pre>
            </div>
          ) : null}

          {csvPreview !== null ? (
            <div>
              <div className="text-xs font-semibold mb-1.5">
                {t("supplier_portal.finance.close.csv_preview_title")}
              </div>
              <pre
                className="max-h-56 overflow-auto rounded-xl border p-3 font-mono text-[11px]"
                style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface-subtle)" }}
              >
                {csvPreview}
              </pre>
            </div>
          ) : null}

          {!snapshot && csvPreview === null ? (
            <p className="text-sm" style={{ color: "var(--desk-text-tertiary)" }}>
              {t("supplier_portal.finance.close.snapshot_empty")}
            </p>
          ) : null}
        </PortalSection>
      </div>

      {showHistory && selectedPeriodId ? (
        <div className="mt-6">
          <ObjectHistoryPanel
            objectType="period"
            objectId={selectedPeriodId}
            compact
            autoVerifyChain
            onClose={() => setShowHistory(false)}
          />
        </div>
      ) : null}
    </PageChrome>
  );
}
