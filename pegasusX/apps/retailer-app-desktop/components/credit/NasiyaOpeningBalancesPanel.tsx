"use client";

import { useMemo, useState } from "react";
import { CheckCircle2, FileWarning, ShieldAlert } from "lucide-react";
import { createTranslator } from "@pegasusx/i18n";
import { usePortalLocale } from "@/lib/i18n";
import {
  acknowledgeRetailerNasiyaOpeningBalance,
  disputeRetailerNasiyaOpeningBalance,
} from "@/lib/api";
import { formatWholeSom } from "@/lib/retailer-w2";
import type { MigrationOpeningBalanceInvoice } from "@pegasusx/types";

function formatDueDate(value?: string): string {
  if (!value) return "—";
  const ms = Date.parse(value);
  return Number.isFinite(ms) ? new Date(ms).toLocaleDateString() : value;
}

export function NasiyaOpeningBalancesPanel({
  invoices,
  loading,
  error,
  onChanged,
}: {
  invoices: MigrationOpeningBalanceInvoice[];
  loading?: boolean;
  error?: string | null;
  onChanged: () => void | Promise<void>;
}) {
  const locale = usePortalLocale();
  const t = useMemo(() => createTranslator(locale), [locale]);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [disputingId, setDisputingId] = useState<string | null>(null);
  const [disputeReason, setDisputeReason] = useState("");
  const [actionError, setActionError] = useState<string | null>(null);

  const hasDunningFrozen = invoices.some(
    (inv) => inv.dunning_frozen || (!inv.acknowledged_at && !inv.disputed_at),
  );

  async function handleAcknowledge(invoiceId: string) {
    setBusyId(invoiceId);
    setActionError(null);
    try {
      await acknowledgeRetailerNasiyaOpeningBalance(invoiceId);
      setDisputingId(null);
      setDisputeReason("");
      await onChanged();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to confirm opening balance",
      );
    } finally {
      setBusyId(null);
    }
  }

  async function handleSubmitDispute(invoiceId: string) {
    const trimmed = disputeReason.trim();
    if (!trimmed) {
      setActionError(t("retailer_desktop.nasiya_opening.dispute_reason_placeholder"));
      return;
    }
    setBusyId(invoiceId);
    setActionError(null);
    try {
      await disputeRetailerNasiyaOpeningBalance(invoiceId, trimmed);
      setDisputingId(null);
      setDisputeReason("");
      await onChanged();
    } catch (err) {
      setActionError(
        err instanceof Error ? err.message : "Failed to submit dispute",
      );
    } finally {
      setBusyId(null);
    }
  }

  if (!loading && invoices.length === 0 && !error) {
    return null;
  }

  return (
    <section className="mb-6 rounded-xl border border-border bg-card p-5 space-y-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div>
          <div className="flex items-center gap-2">
            <span className="rounded-full bg-amber-500/15 px-2.5 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-amber-700 dark:text-amber-300">
              {t("retailer_desktop.nasiya_opening.migration_badge")}
            </span>
            <h2 className="text-sm font-semibold text-foreground">
              {t("retailer_desktop.nasiya_opening.title")}
            </h2>
          </div>
          <p className="mt-1 text-xs text-muted-foreground">
            {t("retailer_desktop.nasiya_opening.subtitle")}
          </p>
        </div>
      </div>

      {hasDunningFrozen ? (
        <div className="flex items-center gap-2 rounded-lg border border-amber-500/30 bg-amber-500/10 px-3.5 py-2.5 text-xs text-amber-800 dark:text-amber-200">
          <ShieldAlert className="h-4 w-4 shrink-0" />
          <span>{t("retailer_desktop.nasiya_opening.dunning_frozen_banner")}</span>
        </div>
      ) : null}

      {error || actionError ? (
        <div
          role="alert"
          className="rounded-lg border border-red-500/30 bg-red-500/10 px-3.5 py-2.5 text-xs text-red-700 dark:text-red-300"
        >
          {actionError || error}
        </div>
      ) : null}

      {invoices.length === 0 ? (
        <div className="text-xs text-muted-foreground">
          {loading ? "..." : t("retailer_desktop.nasiya_opening.empty_state")}
        </div>
      ) : (
        <div className="space-y-3">
          {invoices.map((inv) => {
            const isAcked = Boolean(inv.acknowledged_at) || inv.status === "ACKNOWLEDGED";
            const isDisputed = Boolean(inv.disputed_at) || inv.status === "DISPUTED";
            const isPending = !isAcked && !isDisputed;
            const isBusy = busyId === inv.invoice_id;
            return (
              <div
                key={inv.invoice_id}
                className="rounded-lg border border-border bg-background/60 p-4 space-y-3"
              >
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="space-y-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="rounded-md bg-amber-500/15 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wider text-amber-700 dark:text-amber-300">
                        {t("retailer_desktop.nasiya_opening.migration_badge")}
                      </span>
                      <span className="font-mono text-xs font-semibold text-foreground">
                        {inv.external_id || inv.order_id || inv.invoice_id}
                      </span>
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase ${
                          isAcked
                            ? "bg-emerald-500/15 text-emerald-700 dark:text-emerald-300"
                            : isDisputed
                              ? "bg-red-500/15 text-red-700 dark:text-red-300"
                              : "bg-amber-500/15 text-amber-700 dark:text-amber-300"
                        }`}
                      >
                        {isAcked ? (
                          <>
                            <CheckCircle2 className="h-3 w-3" />
                            {t("retailer_desktop.nasiya_opening.acknowledged_status")}
                          </>
                        ) : isDisputed ? (
                          <>
                            <FileWarning className="h-3 w-3" />
                            {t("retailer_desktop.nasiya_opening.disputed_status")}
                          </>
                        ) : (
                          t("retailer_desktop.nasiya_opening.pending_status")
                        )}
                      </span>
                    </div>
                    <div className="flex flex-wrap items-center gap-4 text-xs text-muted-foreground">
                      <span>{inv.supplier_name || inv.supplier_id}</span>
                      <span>
                        {t("retailer_desktop.nasiya_opening.due_date")}:{" "}
                        <strong className="text-foreground">{formatDueDate(inv.due_at)}</strong>
                      </span>
                    </div>
                    {isDisputed && inv.dispute_reason ? (
                      <p className="text-xs text-red-700 dark:text-red-300">
                        {t("retailer_desktop.nasiya_opening.dispute_reason_label")}:{" "}
                        {inv.dispute_reason}
                      </p>
                    ) : null}
                  </div>

                  <div className="flex flex-wrap items-center gap-3">
                    <div className="text-right">
                      <div className="text-[11px] text-muted-foreground">
                        {t("retailer_desktop.nasiya_opening.amount_due")}
                      </div>
                      <div className="font-mono text-sm font-semibold tabular-nums">
                        {formatWholeSom(inv.balance_minor || inv.principal_minor)}
                      </div>
                    </div>

                    {isPending ? (
                      <div className="flex flex-wrap gap-2">
                        <button
                          type="button"
                          disabled={isBusy}
                          onClick={() => void handleAcknowledge(inv.invoice_id)}
                          className="rounded-lg bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:opacity-90 disabled:opacity-50"
                        >
                          {t("retailer_desktop.nasiya_opening.acknowledge_button")}
                        </button>
                        <button
                          type="button"
                          disabled={isBusy}
                          onClick={() => {
                            setDisputingId(
                              disputingId === inv.invoice_id ? null : inv.invoice_id,
                            );
                            setDisputeReason("");
                            setActionError(null);
                          }}
                          className="rounded-lg border border-border bg-card px-3 py-1.5 text-xs font-medium text-foreground hover:bg-muted/60 disabled:opacity-50"
                        >
                          {t("retailer_desktop.nasiya_opening.dispute_button")}
                        </button>
                      </div>
                    ) : null}
                  </div>
                </div>

                {disputingId === inv.invoice_id ? (
                  <div className="flex flex-wrap items-end gap-2 border-t border-border pt-3">
                    <div className="flex-1 min-w-[220px]">
                      <label
                        htmlFor={`dispute-reason-${inv.invoice_id}`}
                        className="mb-1 block text-xs font-medium text-foreground"
                      >
                        {t("retailer_desktop.nasiya_opening.dispute_reason_label")}
                      </label>
                      <input
                        id={`dispute-reason-${inv.invoice_id}`}
                        value={disputeReason}
                        onChange={(e) => setDisputeReason(e.target.value)}
                        placeholder={t(
                          "retailer_desktop.nasiya_opening.dispute_reason_placeholder",
                        )}
                        className="w-full rounded-lg border border-border bg-card px-3 py-1.5 text-xs text-foreground outline-none focus:border-primary"
                      />
                    </div>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => void handleSubmitDispute(inv.invoice_id)}
                      className="rounded-lg bg-red-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-red-700 disabled:opacity-50"
                    >
                      {t("retailer_desktop.nasiya_opening.submit_dispute")}
                    </button>
                    <button
                      type="button"
                      disabled={isBusy}
                      onClick={() => {
                        setDisputingId(null);
                        setDisputeReason("");
                      }}
                      className="rounded-lg border border-border px-3 py-1.5 text-xs font-medium text-muted-foreground hover:text-foreground"
                    >
                      {t("common.action.cancel")}
                    </button>
                  </div>
                ) : null}
              </div>
            );
          })}
        </div>
      )}
    </section>
  );
}
