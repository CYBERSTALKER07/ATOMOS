"use client";

import React, { useMemo, useState } from "react";
import Link from "next/link";
import type { Route } from "next";
import { usePortalT } from "@/lib/i18n";
import { supplierFetch } from "@/lib/auth";
import StatusBadge from "@/components/StatusBadge";
import {
  defaultBatchResolveItem,
  EXCEPTION_HANDLER_GROUPS,
  exceptionRowKey,
  formatWholeSom,
  HANDLER_META,
  resolveCounterpartyLabel,
  resolveHandlerGroup,
  resolveSeverityTier,
  sortExceptionsByRisk,
  sumMoneyAtRiskMinor,
  type BatchResolveResponse,
  type ExceptionHandlerGroup,
  type ExceptionSeverityTier,
  type SupplierExceptionDeskRow,
} from "@/lib/exception-desk";

interface ExceptionsListProps {
  exceptions: SupplierExceptionDeskRow[];
  activeHandler?: ExceptionHandlerGroup | "ALL";
  onResolved?: () => void;
}

const SEVERITY_BADGE_CLASS: Record<ExceptionSeverityTier, string> = {
  CRITICAL: "bg-red-500/15 text-red-700 border-red-500/30",
  HIGH: "bg-amber-500/15 text-amber-800 border-amber-500/30",
  MEDIUM: "bg-sky-500/15 text-sky-800 border-sky-500/30",
  LOW: "bg-neutral-500/10 text-neutral-600 border-neutral-500/20",
};

export function ExceptionsList({
  exceptions,
  activeHandler = "ALL",
  onResolved,
}: ExceptionsListProps) {
  const t = usePortalT();
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [batchBusy, setBatchBusy] = useState(false);
  const [selectedKeys, setSelectedKeys] = useState<Set<string>>(() => new Set());
  const [batchNote, setBatchNote] = useState("");
  const [onecRefs, setOnecRefs] = useState<Record<string, string>>({});
  const [mergeTargets, setMergeTargets] = useState<Record<string, string>>({});
  const [inlineNotes, setInlineNotes] = useState<Record<string, string>>({});
  const [error, setError] = useState<string | null>(null);
  const [feedback, setFeedback] = useState<string | null>(null);

  const visibleRows = useMemo(() => {
    const filtered =
      activeHandler === "ALL"
        ? exceptions
        : exceptions.filter((row) => resolveHandlerGroup(row) === activeHandler);
    return sortExceptionsByRisk(filtered);
  }, [exceptions, activeHandler]);

  const groupedSections = useMemo(() => {
    const groups = new Map<ExceptionHandlerGroup, SupplierExceptionDeskRow[]>();
    for (const row of visibleRows) {
      const handler = resolveHandlerGroup(row);
      const bucket = groups.get(handler);
      if (bucket) {
        bucket.push(row);
      } else {
        groups.set(handler, [row]);
      }
    }
    return EXCEPTION_HANDLER_GROUPS.filter((id) => groups.has(id)).map((id) => {
      const rows = groups.get(id) ?? [];
      return {
        handler: id,
        meta: HANDLER_META[id],
        rows,
        totalRiskMinor: sumMoneyAtRiskMinor(rows),
      };
    });
  }, [visibleRows]);

  if (visibleRows.length === 0) return null;

  const allVisibleKeys = visibleRows.map((row) => exceptionRowKey(row));
  const selectedVisibleCount = allVisibleKeys.filter((key) => selectedKeys.has(key)).length;
  const allSelected = selectedVisibleCount > 0 && selectedVisibleCount === allVisibleKeys.length;

  const toggleSelect = (key: string) => {
    setSelectedKeys((prev) => {
      const next = new Set(prev);
      if (next.has(key)) next.delete(key);
      else next.add(key);
      return next;
    });
  };

  const toggleSelectAll = () => {
    setSelectedKeys((prev) => {
      if (allSelected) {
        const next = new Set(prev);
        for (const key of allVisibleKeys) next.delete(key);
        return next;
      }
      const next = new Set(prev);
      for (const key of allVisibleKeys) next.add(key);
      return next;
    });
  };

  const resolveSingle = async (
    row: SupplierExceptionDeskRow,
    payload: Record<string, string | undefined>,
  ) => {
    const key = exceptionRowKey(row);
    const entityId = row.order_id || row.posting_id || row.document_key || row.manifest_id || "";
    setBusyKey(key);
    setError(null);
    setFeedback(null);
    try {
      const cleanBody: Record<string, string> = {};
      for (const [k, v] of Object.entries(payload)) {
        if (v && v.trim()) cleanBody[k] = v.trim();
      }
      const res = await supplierFetch(
        `/v1/supplier/exceptions/${encodeURIComponent(row.kind)}/${encodeURIComponent(entityId)}/resolve`,
        {
          method: "POST",
          body: JSON.stringify(cleanBody),
        },
      );
      if (!res.ok) {
        const errBody = (await res.json().catch(() => ({}))) as { error?: string };
        throw new Error(errBody.error || `resolve_${res.status}`);
      }
      setSelectedKeys((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
      setFeedback(`Resolved ${row.kind} (${entityId})`);
      onResolved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.resolve_failed"));
    } finally {
      setBusyKey(null);
    }
  };

  const resolveBatch = async () => {
    const chosen = visibleRows.filter((row) => selectedKeys.has(exceptionRowKey(row)));
    if (chosen.length === 0) return;
    setBatchBusy(true);
    setError(null);
    setFeedback(null);
    try {
      const items = chosen.map((row) => defaultBatchResolveItem(row, batchNote));
      const res = await supplierFetch("/v1/supplier/exceptions/batch-resolve", {
        method: "POST",
        body: JSON.stringify({ items }),
      });
      if (!res.ok) {
        const errBody = (await res.json().catch(() => ({}))) as { error?: string };
        throw new Error(errBody.error || `batch_resolve_${res.status}`);
      }
      const data = (await res.json()) as BatchResolveResponse;
      const failedCount = data.failed?.length ?? 0;
      const resolvedCount = data.resolved ?? chosen.length - failedCount;
      setSelectedKeys(new Set());
      setBatchNote("");
      if (failedCount > 0) {
        const firstErr = data.failed?.[0]?.error ?? "partial_failure";
        setError(`Resolved ${resolvedCount}, failed ${failedCount} (${firstErr})`);
      } else {
        setFeedback(`Batch resolved ${resolvedCount} exception(s).`);
      }
      onResolved?.();
    } catch (err) {
      setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.resolve_failed"));
    } finally {
      setBatchBusy(false);
    }
  };

  const renderInlineActions = (row: SupplierExceptionDeskRow, key: string) => {
    const isBusy = busyKey === key || batchBusy;
    const upperKind = (row.kind ?? "").toUpperCase();
    const noteVal = inlineNotes[key] ?? "";

    if (upperKind === "EXPORT_REJECTED" || upperKind === "EXPORT_DEAD_LETTER" || upperKind === "ONEC_EXPORT_REJECTED") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "retry", note: noteVal })}
          >
            {isBusy ? "Working…" : "Retry 1C Export"}
          </button>
          <input
            type="text"
            aria-label={`1C reference for ${row.order_id}`}
            className="border border-[var(--desk-border)] rounded px-2 py-1 text-xs w-40"
            placeholder="1C doc ref (optional)"
            value={onecRefs[key] ?? ""}
            onChange={(e) => setOnecRefs((prev) => ({ ...prev, [key]: e.target.value }))}
          />
          <button
            type="button"
            className="md-btn md-btn-outlined text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "ack", onec_ref: onecRefs[key], note: noteVal })}
          >
            Acknowledge Export
          </button>
        </div>
      );
    }

    if (upperKind === "FISCAL_FAILED" || upperKind === "FISCAL_DEAD_LETTER") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "retry", note: noteVal })}
          >
            {isBusy ? "Retrying…" : "Retry Soliq Fiscal"}
          </button>
          <button
            type="button"
            className="md-btn md-btn-outlined text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "resolve", note: noteVal })}
          >
            Mark Handled
          </button>
        </div>
      );
    }

    if (upperKind === "TIN_DEDUP_CANDIDATE") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <input
            type="text"
            aria-label={`Merge target ID for ${row.order_id}`}
            className="border border-[var(--desk-border)] rounded px-2 py-1 text-xs w-44 font-mono"
            placeholder="Target retailer ID"
            value={mergeTargets[key] ?? ""}
            onChange={(e) => setMergeTargets((prev) => ({ ...prev, [key]: e.target.value }))}
          />
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy || !(mergeTargets[key] ?? "").trim()}
            onClick={() =>
              void resolveSingle(row, {
                action: "merge_into",
                merged_into_id: mergeTargets[key],
                note: noteVal,
              })
            }
          >
            Merge Duplicate
          </button>
          <button
            type="button"
            className="md-btn md-btn-outlined text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "mark_clean", note: noteVal })}
          >
            Mark Clean
          </button>
        </div>
      );
    }

    if (upperKind === "SHOP_CLOSED" || upperKind === "ARRIVED_SHOP_CLOSED") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-tonal text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "WAIT", note: noteVal })}
          >
            Wait 15m
          </button>
          <button
            type="button"
            className="md-btn md-btn-outlined text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "BYPASS", note: noteVal })}
          >
            Bypass & Deliver
          </button>
          <button
            type="button"
            className="md-btn md-btn-outlined text-xs text-red-700"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "RETURN_TO_DEPOT", note: noteVal })}
          >
            Return to Depot
          </button>
          <Link href={"/exceptions/shop-closed" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Shop-Closed Queue →
          </Link>
        </div>
      );
    }

    if (upperKind === "CASH_DISCREPANCY" || upperKind === "CASH_SHORTFALL") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <input
            type="text"
            aria-label={`Variance note for ${row.order_id}`}
            className="border border-[var(--desk-border)] rounded px-2 py-1 text-xs w-44"
            placeholder="Reconciliation note"
            value={noteVal}
            onChange={(e) => setInlineNotes((prev) => ({ ...prev, [key]: e.target.value }))}
          />
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "resolve", note: noteVal })}
          >
            Accept Variance
          </button>
          <Link href={"/treasury/cash-reconciliations" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Cash Reconciliations →
          </Link>
        </div>
      );
    }

    if (upperKind === "CREDIT_NOTE_DRAFT") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy}
            onClick={() =>
              void resolveSingle(row, {
                action: "resolve",
                credit_note_id: row.document_key || row.note || row.order_id,
                note: noteVal,
              })
            }
          >
            Issue Credit Note
          </button>
          <Link href={"/finance/credit-notes" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Credit Notes →
          </Link>
        </div>
      );
    }

    if (upperKind === "CREDIT_FREEZE" || upperKind === "CREDIT_HOLD" || upperKind === "PROMISE_BREACHED") {
      return (
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={isBusy}
            onClick={() => void resolveSingle(row, { action: "resolve", note: noteVal || "desk_unfreeze" })}
          >
            Unfreeze / Resolve
          </button>
          <Link href={"/credit/collections" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Collections Desk →
          </Link>
        </div>
      );
    }

    return (
      <div className="mt-3 flex flex-wrap items-center gap-2">
        <input
          type="text"
          aria-label={`Resolution note for ${row.order_id}`}
          className="border border-[var(--desk-border)] rounded px-2 py-1 text-xs w-44"
          placeholder="Resolution note"
          value={noteVal}
          onChange={(e) => setInlineNotes((prev) => ({ ...prev, [key]: e.target.value }))}
        />
        <button
          type="button"
          className="md-btn md-btn-outlined text-xs"
          disabled={isBusy}
          onClick={() => void resolveSingle(row, { action: "resolve", note: noteVal })}
        >
          {isBusy ? "Resolving…" : "Resolve"}
        </button>
        {upperKind.includes("CLAIM") ? (
          <Link href={"/exceptions/claims" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Claims Queue →
          </Link>
        ) : upperKind.includes("MANIFEST") ? (
          <Link href={"/manifest-exceptions" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Manifest Exceptions →
          </Link>
        ) : upperKind.includes("EARLY_COMPLETE") ? (
          <Link href={"/exceptions/early-complete" as Route} className="text-xs underline text-[var(--color-md-primary)] ml-1">
            Early Complete →
          </Link>
        ) : null}
      </div>
    );
  };

  return (
    <div className="space-y-4">
      {error ? (
        <div role="alert" className="rounded-lg border border-red-500/30 bg-red-500/10 px-3 py-2 text-sm text-red-700">
          {error}
        </div>
      ) : null}
      {feedback ? (
        <div role="status" className="rounded-lg border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-sm text-emerald-800">
          {feedback}
        </div>
      ) : null}

      <div className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-raised)] px-4 py-3">
        <label className="inline-flex items-center gap-2 text-sm font-medium cursor-pointer">
          <input
            type="checkbox"
            checked={allSelected}
            onChange={toggleSelectAll}
            aria-label="Select all visible exceptions"
          />
          <span>
            {selectedVisibleCount > 0
              ? `${selectedVisibleCount} selected`
              : `Select all (${visibleRows.length})`}
          </span>
        </label>

        <div className="flex flex-wrap items-center gap-2">
          <input
            type="text"
            aria-label="Batch resolution note"
            className="border border-[var(--desk-border)] rounded px-2.5 py-1.5 text-xs w-52 bg-[var(--desk-surface)]"
            placeholder="Batch resolution note (optional)"
            value={batchNote}
            onChange={(e) => setBatchNote(e.target.value)}
            disabled={batchBusy}
          />
          <button
            type="button"
            className="md-btn md-btn-filled text-xs"
            disabled={batchBusy || selectedVisibleCount === 0}
            onClick={() => void resolveBatch()}
          >
            {batchBusy ? "Batch Resolving…" : `Batch Resolve (${selectedVisibleCount})`}
          </button>
        </div>
      </div>

      {groupedSections.map((section) => (
        <section key={section.handler} aria-label={section.meta.label} className="space-y-2">
          <div className="flex flex-wrap items-baseline justify-between gap-2 px-1">
            <div>
              <h3 className="text-sm font-semibold text-[var(--desk-text-primary)]">
                {section.meta.label}{" "}
                <span className="font-mono text-xs text-[var(--color-md-outline)]">({section.rows.length})</span>
              </h3>
              <p className="text-xs text-[var(--color-md-outline)]">{section.meta.description}</p>
            </div>
            <div className="text-sm font-mono font-semibold text-[var(--desk-text-primary)]">
              At risk: {formatWholeSom(section.totalRiskMinor)}
            </div>
          </div>

          <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
            {section.rows.map((row) => {
              const key = exceptionRowKey(row);
              const tier = resolveSeverityTier(row);
              const entity = resolveCounterpartyLabel(row);
              const checked = selectedKeys.has(key);
              const riskMinor = row.money_at_risk_minor ?? 0;

              return (
                <li key={key} className="p-4 md-typescale-body-medium">
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="flex items-start gap-3 min-w-0">
                      <input
                        type="checkbox"
                        className="mt-1"
                        checked={checked}
                        onChange={() => toggleSelect(key)}
                        aria-label={`Select ${row.kind} ${entity.primaryName}`}
                      />
                      <div className="min-w-0">
                        <div className="flex flex-wrap items-center gap-2">
                          <span className={`inline-flex items-center rounded border px-2 py-0.5 text-[11px] font-semibold ${SEVERITY_BADGE_CLASS[tier]}`}>
                            {tier}
                          </span>
                          <span className="md-chip h-6 text-xs">{row.kind}</span>
                          <StatusBadge state={row.status} />
                          {row.top_playbook_name ? (
                            <span className="text-xs text-[var(--color-md-outline)]">
                              Playbook: {row.top_playbook_name}
                            </span>
                          ) : null}
                        </div>

                        <div className="mt-1.5 flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
                          <span className="font-semibold text-[var(--desk-text-primary)]">
                            {entity.primaryName}
                          </span>
                          {row.tin ? (
                            <span className="font-mono text-xs text-[var(--color-md-outline)]">
                              TIN {row.tin}
                            </span>
                          ) : null}
                          {row.phone ? (
                            <span className="font-mono text-xs text-[var(--color-md-outline)]">
                              {row.phone}
                            </span>
                          ) : null}
                          {entity.secondaryId ? (
                            <span className="font-mono text-xs text-[var(--color-md-outline)]">
                              ({entity.secondaryId})
                            </span>
                          ) : null}
                        </div>

                        <div className="mt-1 flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-[var(--color-md-outline)] font-mono">
                          {row.order_id ? <span>Ref: {row.order_id}</span> : null}
                          {row.posting_id ? <span>Posting: {row.posting_id}</span> : null}
                          {row.document_key ? <span>Doc: {row.document_key}</span> : null}
                          {row.dedup_cluster_id ? <span>Cluster: {row.dedup_cluster_id}</span> : null}
                          {row.manifest_id ? <span>Manifest: {row.manifest_id}</span> : null}
                        </div>

                        {row.note ? (
                          <p className="mt-1.5 text-sm text-[var(--color-md-outline)]">{row.note}</p>
                        ) : null}
                        <p className="mt-1 text-xs text-[var(--color-md-outline)]">
                          Updated {new Date(row.updated_at).toLocaleString()}
                        </p>
                      </div>
                    </div>

                    <div className="text-right shrink-0">
                      <div className="text-xs uppercase tracking-wide text-[var(--color-md-outline)]">
                        Money at risk
                      </div>
                      <div className="text-base font-mono font-bold text-[var(--desk-text-primary)]">
                        {formatWholeSom(riskMinor)}
                      </div>
                    </div>
                  </div>

                  {renderInlineActions(row, key)}
                </li>
              );
            })}
          </ul>
        </section>
      ))}
    </div>
  );
}
