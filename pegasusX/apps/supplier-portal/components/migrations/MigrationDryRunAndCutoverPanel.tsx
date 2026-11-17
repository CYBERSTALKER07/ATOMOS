"use client";

import { useState } from "react";
import { usePortalT } from "@/lib/i18n";
import { formatUZS } from "@pegasusx/i18n";
import { exportCsv } from "@pegasusx/desktop-bridge";
import {
  migrationCutoverKey,
  migrationDryRunKey,
  migrationOpenOrdersReleaseKey,
  migrationSessionApplyKey,
  migrationSessionApproveKey,
  migrationSessionRollbackKey,
  type ApiClient,
} from "@pegasusx/api-core";
import type {
  MigrationApplyResult,
  MigrationCutoverResult,
  MigrationDryRunResponse,
  MigrationOpenOrderReleaseResponse,
  MigrationProjectTree,
  MigrationRollbackResult,
  MigrationSession,
} from "@pegasusx/types";
import { formatMinorUnits } from "@/lib/money";
import { packLabelKey } from "./pack-labels";

export interface MigrationDryRunAndCutoverPanelProps {
  api: Pick<
    ApiClient,
    | "runMigrationProjectDryRun"
    | "runMigrationSessionDryRun"
    | "getMigrationReport"
    | "approveMigrationSession"
    | "applyMigrationSession"
    | "rollbackMigrationSession"
    | "releaseMigrationProjectOpenOrders"
    | "cutoverMigrationProject"
    | "getMigrationSessionRows"
  >;
  tree: MigrationProjectTree;
  selectedSession: MigrationSession | null;
  liveApplyProgress: { processed: number; total: number } | null;
  onRefresh: () => Promise<void> | void;
  onError: (msg: string) => void;
}

function formatAmountMinor(minor: number, currency?: string): string {
  const ccy = (currency || "UZS").toUpperCase();
  if (ccy === "UZS") return formatUZS(minor);
  return `${formatMinorUnits(minor, { currency: ccy })} ${ccy}`;
}

export function MigrationDryRunAndCutoverPanel({
  api,
  tree,
  selectedSession,
  liveApplyProgress,
  onRefresh,
  onError,
}: MigrationDryRunAndCutoverPanelProps) {
  const t = usePortalT();
  const [applyMode, setApplyMode] = useState<"MERGE" | "SET_ON_HAND">("MERGE");
  const [setOnHandConfirmed, setSetOnHandConfirmed] = useState(false);
  const [busy, setBusy] = useState(false);
  const [dryRun, setDryRun] = useState<MigrationDryRunResponse | null>(null);
  const [applyResult, setApplyResult] = useState<MigrationApplyResult | null>(null);
  const [rollbackResult, setRollbackResult] = useState<MigrationRollbackResult | null>(null);
  const [openOrderIdsText, setOpenOrderIdsText] = useState("");
  const [releaseTargetStatus, setReleaseTargetStatus] = useState<"PENDING" | "SCHEDULED">("PENDING");
  const [releaseResult, setReleaseResult] = useState<MigrationOpenOrderReleaseResponse | null>(null);
  const [cutoverResult, setCutoverResult] = useState<MigrationCutoverResult | null>(null);

  const projectId = tree.project.project_id;
  const sessionId = selectedSession?.session_id;

  async function handleDryRun(scope: "project" | "session") {
    if (busy) return;
    setBusy(true);
    try {
      const res =
        scope === "session" && sessionId
          ? await api.runMigrationSessionDryRun(sessionId, migrationDryRunKey(sessionId), applyMode)
          : await api.runMigrationProjectDryRun(projectId, migrationDryRunKey(projectId), applyMode);
      setDryRun(res);
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleDownloadErrors() {
    if (!sessionId) return;
    try {
      const res = await api.getMigrationSessionRows(sessionId, {
        limit: 500,
        offset: 0,
        invalid_only: true,
      });
      const rows = (res.rows ?? []).map((r) => [
        String(r.row_index),
        (r.validation_errors ?? []).join("; "),
        JSON.stringify(r.raw_data ?? {}),
      ]);
      await exportCsv(
        `migration-${sessionId}-errors.csv`,
        ["row_index", "validation_errors", "raw_data"],
        rows,
      );
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    }
  }

  async function handleApproveSession() {
    if (!sessionId || busy) return;
    setBusy(true);
    try {
      await api.approveMigrationSession(
        sessionId,
        {
          apply_mode: applyMode,
          set_on_hand_confirmed: setOnHandConfirmed,
          expected_report_id: dryRun?.report.report_id || selectedSession?.dry_run_report_id,
        },
        migrationSessionApproveKey(sessionId, dryRun?.report.report_id),
      );
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleApplySession() {
    if (!sessionId || busy) return;
    setBusy(true);
    try {
      const res = await api.applyMigrationSession(
        sessionId,
        { apply_mode: applyMode, chunk_size: 500 },
        migrationSessionApplyKey(sessionId, selectedSession?.dry_run_fingerprint),
      );
      setApplyResult(res);
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleRollback() {
    if (!sessionId || busy) return;
    setBusy(true);
    try {
      const res = await api.rollbackMigrationSession(
        sessionId,
        migrationSessionRollbackKey(sessionId),
      );
      setRollbackResult(res);
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleReleaseOpenOrders() {
    const ids = openOrderIdsText
      .split(/[\s,]+/)
      .map((s) => s.trim())
      .filter(Boolean);
    if (ids.length === 0 || busy) return;
    setBusy(true);
    try {
      const res = await api.releaseMigrationProjectOpenOrders(
        projectId,
        { order_ids: ids, target_status: releaseTargetStatus },
        migrationOpenOrdersReleaseKey(projectId, ids.join(",")),
      );
      setReleaseResult(res);
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleCutover() {
    if (busy) return;
    setBusy(true);
    try {
      const res = await api.cutoverMigrationProject(
        projectId,
        migrationCutoverKey(projectId),
      );
      setCutoverResult(res);
      await onRefresh();
    } catch (err) {
      onError(err instanceof Error ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  const rollbackHours = selectedSession?.rollback_until
    ? Math.max(
        0,
        Math.round((new Date(selectedSession.rollback_until).getTime() - Date.now()) / 3600_000),
      )
    : 72;

  return (
    <div className="flex flex-col gap-4">
      <section className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col gap-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <div>
            <h3 className="md-typescale-title-medium font-semibold">
              {t("supplier_portal.migrations.dry_run_title")}
            </h3>
            {selectedSession && (
              <p className="text-xs text-[var(--muted)]">
                {t(packLabelKey(selectedSession.pack ?? ""))} · {selectedSession.status}
              </p>
            )}
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <select
              className="md-input-outlined px-2 py-1.5 text-xs"
              value={applyMode}
              onChange={(e) => setApplyMode(e.target.value as "MERGE" | "SET_ON_HAND")}
            >
              <option value="MERGE">MERGE</option>
              <option value="SET_ON_HAND">SET_ON_HAND</option>
            </select>
            {sessionId && (
              <button
                type="button"
                disabled={busy}
                onClick={() => void handleDryRun("session")}
                className="md-btn md-btn-outlined px-3 py-1.5 text-xs"
              >
                {t("supplier_portal.migrations.run_dry_run")}
              </button>
            )}
            <button
              type="button"
              disabled={busy}
              onClick={() => void handleDryRun("project")}
              className="md-btn md-btn-filled px-3 py-1.5 text-xs"
            >
              {t("supplier_portal.migrations.run_dry_run")} ({tree.sessions.length})
            </button>
          </div>
        </div>

        {dryRun && (
          <div className="flex flex-col gap-3">
            <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-xs">
              <div className="rounded-lg border border-[var(--border)] p-3">
                <div className="text-[var(--muted)]">Create / Update / Skip</div>
                <div className="text-base font-semibold mt-1">
                  {dryRun.report.would_create} / {dryRun.report.would_update} / {dryRun.report.would_skip}
                </div>
              </div>
              <div className="rounded-lg border border-[var(--border)] p-3">
                <div className="text-[var(--muted)]">
                  {t("supplier_portal.migrations.nasiya_total")}
                </div>
                <div className="text-base font-semibold mt-1">
                  {formatAmountMinor(dryRun.report.money_impact_minor, dryRun.report.currency)}
                </div>
              </div>
              <div className="rounded-lg border border-[var(--border)] p-3">
                <div className="text-[var(--muted)]">
                  {t("supplier_portal.migrations.errors_count")}
                </div>
                <div className="text-base font-semibold mt-1">
                  {dryRun.report.rows_invalid} ({dryRun.report.blocking_count} blocking)
                </div>
              </div>
              <div className="rounded-lg border border-[var(--border)] p-3">
                <div className="text-[var(--muted)]">
                  {dryRun.summary.balanced_journal_check
                    ? t("supplier_portal.migrations.balance_status_balanced")
                    : t("supplier_portal.migrations.balance_status_unbalanced")}
                </div>
                <div className="text-xs font-mono mt-1">
                  Dr {formatAmountMinor(dryRun.summary.journal_debit_minor, dryRun.report.currency)} / Cr{" "}
                  {formatAmountMinor(dryRun.summary.journal_credit_minor, dryRun.report.currency)}
                </div>
              </div>
            </div>

            {dryRun.summary.cold_lot_migrated_without_live_baseline > 0 && (
              <div className="rounded border border-amber-500/40 bg-amber-500/10 p-2 text-xs">
                {t("warehouse_portal.cold_lot.warning_banner")} (
                {dryRun.summary.cold_lot_migrated_without_live_baseline})
              </div>
            )}

            {dryRun.report.rows_invalid > 0 && sessionId && (
              <div>
                <button
                  type="button"
                  onClick={() => void handleDownloadErrors()}
                  className="md-btn md-btn-outlined px-3 py-1.5 text-xs"
                >
                  {t("supplier_portal.migrations.download_errors_csv")}
                </button>
              </div>
            )}
          </div>
        )}

        {sessionId && (
          <div className="border-t border-[var(--border)] pt-3 flex flex-col gap-3">
            <label className="flex items-center gap-2 text-xs cursor-pointer">
              <input
                type="checkbox"
                checked={setOnHandConfirmed}
                onChange={(e) => setSetOnHandConfirmed(e.target.checked)}
              />
              <span>{t("supplier_portal.migrations.set_on_hand_ack")}</span>
            </label>

            <div className="flex flex-wrap gap-2">
              <button
                type="button"
                disabled={busy}
                onClick={() => void handleApproveSession()}
                className="md-btn md-btn-outlined px-4 py-2 text-xs"
              >
                {t("supplier_portal.migrations.approve_button")}
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void handleApplySession()}
                className="md-btn md-btn-filled px-4 py-2 text-xs"
              >
                {t("supplier_portal.migrations.apply_button")}
              </button>
              {selectedSession?.status === "APPLIED" && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => void handleRollback()}
                  className="md-btn md-btn-outlined px-4 py-2 text-xs text-rose-600"
                >
                  {t("supplier_portal.migrations.rollback_button")}
                </button>
              )}
            </div>

            {liveApplyProgress && (
              <p className="text-xs font-medium text-[var(--color-md-primary)]">
                {t("supplier_portal.migrations.apply_progress", {
                  done: String(liveApplyProgress.processed),
                  total: String(liveApplyProgress.total),
                })}
              </p>
            )}

            {applyResult && (
              <p className="text-xs text-emerald-700 dark:text-emerald-300">
                Created {applyResult.created} · Updated {applyResult.updated} · Skipped{" "}
                {applyResult.skipped} · Chunks {applyResult.chunks_applied}
              </p>
            )}

            {selectedSession?.status === "APPLIED" && (
              <p className="text-xs text-[var(--muted)]">
                {t("supplier_portal.migrations.rollback_window_hours", {
                  hours: String(rollbackHours),
                })}
              </p>
            )}

            {rollbackResult && (
              <p className="text-xs text-amber-700 dark:text-amber-300">
                Rolled back {rollbackResult.reverted_snapshots} snapshots ·{" "}
                {rollbackResult.reversed_invoices} invoices ·{" "}
                {rollbackResult.superseded_orders} orders
              </p>
            )}
          </div>
        )}
      </section>

      <section className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col gap-3">
        <h3 className="md-typescale-title-medium font-semibold">
          {t("supplier_portal.migrations.open_orders_title")}
        </h3>
        <div className="flex flex-wrap items-center gap-2">
          <input
            type="text"
            className="md-input-outlined flex-1 min-w-56 px-3 py-1.5 text-xs font-mono"
            placeholder="order-id-1, order-id-2"
            value={openOrderIdsText}
            onChange={(e) => setOpenOrderIdsText(e.target.value)}
          />
          <select
            className="md-input-outlined px-2 py-1.5 text-xs"
            value={releaseTargetStatus}
            onChange={(e) => setReleaseTargetStatus(e.target.value as "PENDING" | "SCHEDULED")}
          >
            <option value="PENDING">PENDING</option>
            <option value="SCHEDULED">SCHEDULED</option>
          </select>
          <button
            type="button"
            disabled={busy || !openOrderIdsText.trim()}
            onClick={() => void handleReleaseOpenOrders()}
            className="md-btn md-btn-outlined px-3 py-1.5 text-xs"
          >
            {t("supplier_portal.migrations.release_orders_button")}
          </button>
        </div>
        {releaseResult && (
          <p className="text-xs text-[var(--muted)]">
            Released: {releaseResult.released_order_ids?.length ?? 0} · Skipped:{" "}
            {releaseResult.skipped_order_ids?.length ?? 0}
          </p>
        )}
      </section>

      <section className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-4 flex flex-col gap-3">
        <div className="flex items-center justify-between gap-2">
          <div>
            <h3 className="md-typescale-title-medium font-semibold">
              {t("supplier_portal.migrations.cutover_title")}
            </h3>
            <p className="text-xs text-[var(--muted)]">
              DAG Missing: {(tree.dag.missing_packs ?? []).join(", ") || "None"} · Status:{" "}
              {tree.project.status}
            </p>
          </div>
          <button
            type="button"
            disabled={busy || tree.project.status === "CUTOVER_COMPLETE"}
            onClick={() => void handleCutover()}
            className="md-btn md-btn-filled px-4 py-2 text-xs"
          >
            {tree.project.status === "CUTOVER_COMPLETE"
              ? t("supplier_portal.migrations.status_live")
              : t("supplier_portal.migrations.cutover_button")}
          </button>
        </div>

        {cutoverResult?.checklist && (
          <ul className="flex flex-col gap-1 text-xs">
            {cutoverResult.checklist.map((item) => (
              <li key={item.name} className="flex items-center gap-2">
                <span>{item.passed ? "✅" : "⚠️"}</span>
                <span className="font-medium">{item.name}</span>
                <span className="text-[var(--muted)]">{item.detail}</span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}
