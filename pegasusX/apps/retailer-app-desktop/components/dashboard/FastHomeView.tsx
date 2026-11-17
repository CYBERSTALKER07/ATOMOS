"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import {
  Truck,
  Wallet,
  Sparkles,
  Clock,
  CheckCircle2,
  Pencil,
  PauseCircle,
  AlertTriangle,
  ArrowRight,
  Layers,
  ShieldAlert,
  RefreshCw,
} from "lucide-react";
import { usePortalLocale } from "@/lib/i18n";
import {
  FAST_HOME_COPY,
  computeCutoffCountdown,
  computeServerClockOffsetMs,
  formatFastWholeSom,
  type RetailerFastHomePayload,
} from "@/lib/fast-home";
import {
  confirmAiOrder,
  confirmPreorder,
  editPreorder,
  rejectAiOrder,
  rejectPreorder,
} from "@/lib/api";

interface FastHomeViewProps {
  data: RetailerFastHomePayload | null;
  loading: boolean;
  error?: string | null;
  onRefresh: () => void | Promise<void>;
  onSwitchToFullErp: () => void;
}

export function FastHomeView({
  data,
  loading,
  error,
  onRefresh,
  onSwitchToFullErp,
}: FastHomeViewProps) {
  const locale = usePortalLocale();
  const copy = FAST_HOME_COPY[locale] ?? FAST_HOME_COPY.uz;

  const [nowMs, setNowMs] = useState<number>(() => Date.now());
  const [busyAction, setBusyAction] = useState<"confirm" | "hold" | "edit" | null>(null);
  const [actionMessage, setActionMessage] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [editingOpen, setEditingOpen] = useState(false);
  const [editDate, setEditDate] = useState("");
  const [editQuantities, setEditQuantities] = useState<Record<string, number>>({});

  const serverOffsetMs = useMemo(
    () => computeServerClockOffsetMs(data?.server_now),
    [data?.server_now],
  );

  useEffect(() => {
    const id = window.setInterval(() => {
      setNowMs(Date.now());
    }, 1000);
    return () => window.clearInterval(id);
  }, []);

  const countdown = useMemo(
    () => computeCutoffCountdown(data?.cutoff_at, serverOffsetMs, nowMs),
    [data?.cutoff_at, serverOffsetMs, nowMs],
  );

  const proposal = data?.pending_proposal ?? null;
  const tomorrow = data?.tomorrow_delivery ?? null;
  const currency = data?.currency || tomorrow?.currency || proposal?.currency || "UZS";
  const balanceOwedMinor = data?.balance_owed_minor ?? 0;
  const overdueMinor = data?.overdue_minor ?? 0;
  const daysOverdue = data?.days_overdue ?? 0;
  const headroomMinor = data?.credit_headroom_minor ?? 0;

  const openEditModal = () => {
    if (!proposal) return;
    setEditDate((proposal.proposed_delivery_date || data?.next_delivery_date || "").slice(0, 10));
    const nextQty: Record<string, number> = {};
    for (const item of proposal.items ?? []) {
      nextQty[item.product_id] = item.quantity;
    }
    setEditQuantities(nextQty);
    setActionError(null);
    setEditingOpen(true);
  };

  const handleConfirm = async () => {
    if (!proposal || busyAction) return;
    setBusyAction("confirm");
    setActionError(null);
    setActionMessage(null);
    try {
      const isAi = proposal.source === "AI_PREDICTED";
      const res = isAi
        ? await confirmAiOrder(proposal.order_id)
        : await confirmPreorder(proposal.order_id);
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { detail?: string; message?: string; error?: string } | null;
        throw new Error(body?.detail || body?.message || body?.error || `HTTP ${res.status}`);
      }
      setActionMessage(copy.actionConfirmed);
      await onRefresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to confirm order");
    } finally {
      setBusyAction(null);
    }
  };

  const handleHold = async () => {
    if (!proposal || busyAction) return;
    setBusyAction("hold");
    setActionError(null);
    setActionMessage(null);
    try {
      const isAi = proposal.source === "AI_PREDICTED";
      const res = isAi
        ? await rejectAiOrder(proposal.order_id, copy.holdReasonPrompt)
        : await rejectPreorder(proposal.order_id, copy.holdReasonPrompt);
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { detail?: string; message?: string; error?: string } | null;
        throw new Error(body?.detail || body?.message || body?.error || `HTTP ${res.status}`);
      }
      setActionMessage(copy.actionHeld);
      await onRefresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to hold order");
    } finally {
      setBusyAction(null);
    }
  };

  const handleSaveEdit = async () => {
    if (!proposal || busyAction) return;
    setBusyAction("edit");
    setActionError(null);
    setActionMessage(null);
    try {
      const lineItems = (proposal.items ?? [])
        .map((item) => ({
          sku: item.sku || item.product_id,
          name: item.product_name,
          quantity: Math.max(0, Math.round(editQuantities[item.product_id] ?? item.quantity)),
          unit_price_minor: item.unit_price_minor,
        }))
        .filter((item) => item.quantity > 0);
      const res = await editPreorder(proposal.order_id, editDate, lineItems);
      if (!res.ok) {
        const body = await res.json().catch(() => null) as { detail?: string; message?: string; error?: string } | null;
        throw new Error(body?.detail || body?.message || body?.error || `HTTP ${res.status}`);
      }
      setEditingOpen(false);
      setActionMessage(copy.actionUpdated);
      await onRefresh();
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Failed to update proposal");
    } finally {
      setBusyAction(null);
    }
  };

  return (
    <div className="space-y-6" data-testid="fast-home-view">
      {/* Top Fast Mode Header Strip */}
      <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 shadow-[var(--shadow-sm)]">
        <div>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-[var(--desk-accent-soft)] px-2.5 py-0.5 text-[11px] font-medium text-[var(--desk-accent)]">
            <Sparkles size={12} />
            {copy.fastModeLabel}
          </span>
          <h1 className="mt-2 text-2xl font-light tracking-tight text-[var(--desk-text-primary)]">
            {copy.fastModeTitle}
          </h1>
          <p className="mt-1 text-sm text-[var(--desk-text-secondary)]">
            {copy.fastModeSubtitle}
          </p>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          {(data?.pending_approvals_count ?? 0) > 0 && (
            <Link
              href="/approvals"
              className="inline-flex items-center gap-2 rounded-xl border border-amber-500/40 bg-amber-500/10 px-3.5 py-2 text-xs font-medium text-amber-300 hover:bg-amber-500/20"
            >
              <ShieldAlert size={14} />
              {copy.pendingApprovalsCta(data?.pending_approvals_count ?? 0)}
            </Link>
          )}
          <button
            type="button"
            onClick={() => void onRefresh()}
            disabled={loading}
            className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl text-xs"
          >
            <RefreshCw size={14} className={loading ? "animate-spin mr-1.5" : "mr-1.5"} />
            Sync
          </button>
          <button
            type="button"
            onClick={onSwitchToFullErp}
            className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl text-xs"
            data-testid="fast-home-switch-full"
          >
            <Layers size={14} className="mr-1.5" />
            {copy.switchToFull}
          </button>
        </div>
      </div>

      {data?.degraded_fallback && (
        <div className="flex items-center gap-2 rounded-xl border border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 px-4 py-2.5 text-xs text-[var(--desk-warning)]">
          <AlertTriangle size={14} />
          <span>{copy.degradedNotice}</span>
        </div>
      )}

      {error && (
        <div className="flex items-center justify-between gap-3 rounded-xl border border-red-500/30 bg-red-500/10 px-4 py-2.5 text-xs text-red-300">
          <span>{error}</span>
          <button
            type="button"
            onClick={() => void onRefresh()}
            className="rounded-lg border border-current/30 px-2.5 py-1 hover:bg-current/10"
          >
            Retry
          </button>
        </div>
      )}

      {(actionMessage || actionError) && (
        <div
          className={`rounded-xl border px-4 py-2.5 text-xs ${
            actionError
              ? "border-red-500/30 bg-red-500/10 text-red-300"
              : "border-emerald-500/30 bg-emerald-500/10 text-emerald-300"
          }`}
        >
          {actionError || actionMessage}
        </div>
      )}

      {/* Core 3-Card Fast Grid */}
      <div className="grid gap-5 lg:grid-cols-3">
        {/* 1. Tomorrow's Delivery */}
        <section
          className="flex flex-col justify-between rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 shadow-[var(--shadow-sm)]"
          data-testid="fast-home-tomorrow-card"
        >
          <div>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-[var(--desk-text-tertiary)]">
                <Truck size={15} style={{ color: "var(--desk-accent)" }} />
                <span>{copy.tomorrowDeliveryTitle}</span>
              </div>
              {tomorrow?.state && (
                <span className="rounded-full bg-[var(--desk-accent-soft)] px-2.5 py-0.5 text-[11px] font-medium text-[var(--desk-accent)]">
                  {tomorrow.state}
                </span>
              )}
            </div>

            {tomorrow ? (
              <div className="mt-4 space-y-2">
                <p className="text-xl font-medium text-[var(--desk-text-primary)]">
                  {tomorrow.supplier_name || tomorrow.supplier_id || `#${tomorrow.order_id.slice(0, 8)}`}
                </p>
                <p className="text-2xl font-light text-[var(--desk-text-primary)] tabular-nums">
                  {formatFastWholeSom(tomorrow.amount_minor, tomorrow.currency || currency)}
                </p>
                <div className="flex flex-wrap items-center gap-3 text-xs text-[var(--desk-text-secondary)]">
                  <span>{copy.tomorrowDeliveryItems(tomorrow.item_count)}</span>
                  {(tomorrow.delivery_date || data?.next_delivery_date) && (
                    <span>· {(tomorrow.delivery_date || data?.next_delivery_date || "").slice(0, 10)}</span>
                  )}
                  {tomorrow.window_label && <span>· {tomorrow.window_label}</span>}
                </div>
              </div>
            ) : (
              <p className="mt-4 text-sm text-[var(--desk-text-secondary)]">
                {copy.tomorrowDeliveryEmpty}
              </p>
            )}
          </div>

          <div className="mt-6">
            <Link
              href={tomorrow ? `/tracking` : `/catalog`}
              className="portal-btn portal-btn--ghost w-full justify-between rounded-xl px-4 py-2.5 text-xs"
            >
              <span>{tomorrow ? copy.trackDelivery : copy.tomorrowDeliveryBrowse}</span>
              <ArrowRight size={14} />
            </Link>
          </div>
        </section>

        {/* 2. What You Owe (Zero raw tiyin) */}
        <section
          className="flex flex-col justify-between rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 shadow-[var(--shadow-sm)]"
          data-testid="fast-home-owe-card"
        >
          <div>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-[var(--desk-text-tertiary)]">
                <Wallet size={15} style={{ color: "var(--desk-warning)" }} />
                <span>{copy.whatYouOweTitle}</span>
              </div>
              {overdueMinor > 0 ? (
                <span className="rounded-full bg-red-500/15 px-2.5 py-0.5 text-[11px] font-medium text-red-300">
                  {copy.whatYouOweOverdue(formatFastWholeSom(overdueMinor, currency), daysOverdue)}
                </span>
              ) : (
                <span className="rounded-full bg-emerald-500/15 px-2.5 py-0.5 text-[11px] font-medium text-emerald-300">
                  {copy.whatYouOweClean}
                </span>
              )}
            </div>

            <div className="mt-4 space-y-2">
              <p className="text-3xl font-light text-[var(--desk-text-primary)] tabular-nums">
                {formatFastWholeSom(balanceOwedMinor, currency)}
              </p>
              <p className="text-xs text-[var(--desk-text-secondary)] tabular-nums">
                {copy.creditHeadroom(formatFastWholeSom(headroomMinor, currency))}
              </p>
            </div>
          </div>

          <div className="mt-6">
            <Link
              href="/credit"
              className="portal-btn portal-btn--primary w-full justify-between rounded-xl px-4 py-2.5 text-xs"
            >
              <span>{copy.payNowCta}</span>
              <ArrowRight size={14} />
            </Link>
          </div>
        </section>

        {/* 3. Today's Proposed Order + Server-Synced Cutoff */}
        <section
          className="flex flex-col justify-between rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-5 shadow-[var(--shadow-sm)]"
          data-testid="fast-home-proposal-card"
        >
          <div>
            <div className="flex items-center justify-between gap-2">
              <div className="flex items-center gap-2 text-xs uppercase tracking-wider text-[var(--desk-text-tertiary)]">
                <Sparkles size={15} style={{ color: "var(--desk-info)" }} />
                <span>{copy.proposedOrderTitle}</span>
              </div>
              <div
                className={`inline-flex items-center gap-1 rounded-full px-2.5 py-0.5 text-[11px] font-mono ${
                  countdown.expired
                    ? "bg-red-500/15 text-red-300"
                    : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-primary)]"
                }`}
                title={copy.cutoffLabel}
                data-testid="fast-home-cutoff-timer"
              >
                <Clock size={11} />
                <span>{countdown.expired ? copy.cutoffExpired : countdown.formatted}</span>
              </div>
            </div>

            {proposal ? (
              <div className="mt-4 space-y-2">
                <p className="text-xs text-[var(--desk-text-tertiary)]">{copy.cutoffLabel}</p>
                <p className="text-2xl font-light text-[var(--desk-text-primary)] tabular-nums">
                  {formatFastWholeSom(proposal.amount_minor, proposal.currency || currency)}
                </p>
                <p className="text-xs text-[var(--desk-text-secondary)]">
                  {proposal.supplier_name || proposal.source} ·{" "}
                  {copy.tomorrowDeliveryItems(proposal.item_count)}
                </p>
                {proposal.items && proposal.items.length > 0 && (
                  <ul className="mt-2 space-y-1 rounded-xl bg-[var(--desk-surface-subtle)] p-2.5 text-xs text-[var(--desk-text-secondary)]">
                    {proposal.items.slice(0, 3).map((item) => (
                      <li key={item.product_id} className="flex items-center justify-between gap-2">
                        <span className="truncate">{item.product_name}</span>
                        <span className="font-mono shrink-0">x{item.quantity}</span>
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            ) : (
              <div className="mt-4 space-y-2">
                <p className="text-xs text-[var(--desk-text-tertiary)]">{copy.cutoffLabel}</p>
                <p className="text-sm text-[var(--desk-text-secondary)]">
                  {copy.proposedOrderEmpty}
                </p>
              </div>
            )}
          </div>

          {proposal ? (
            <div className="mt-6 flex flex-wrap gap-2">
              <button
                type="button"
                disabled={busyAction !== null}
                onClick={() => void handleConfirm()}
                className="portal-btn portal-btn--primary flex-1 justify-center rounded-xl px-3 py-2.5 text-xs"
                data-testid="fast-home-confirm-btn"
              >
                <CheckCircle2 size={14} className="mr-1.5" />
                {copy.confirmCta}
              </button>
              <button
                type="button"
                disabled={busyAction !== null}
                onClick={openEditModal}
                className="portal-btn portal-btn--ghost justify-center rounded-xl px-3 py-2.5 text-xs"
                data-testid="fast-home-edit-btn"
              >
                <Pencil size={14} className="mr-1" />
                {copy.editCta}
              </button>
              <button
                type="button"
                disabled={busyAction !== null}
                onClick={() => void handleHold()}
                className="portal-btn portal-btn--ghost justify-center rounded-xl px-3 py-2.5 text-xs text-amber-300"
                data-testid="fast-home-hold-btn"
              >
                <PauseCircle size={14} className="mr-1" />
                {copy.holdCta}
              </button>
            </div>
          ) : (
            <div className="mt-6">
              <Link
                href="/catalog"
                className="portal-btn portal-btn--ghost w-full justify-between rounded-xl px-4 py-2.5 text-xs"
              >
                <span>{copy.proposedOrderNewOrder}</span>
                <ArrowRight size={14} />
              </Link>
            </div>
          )}
        </section>
      </div>

      {/* Inline Proposal Edit Dialog */}
      {editingOpen && proposal && (
        <div
          className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 p-4"
          role="dialog"
          aria-modal="true"
        >
          <div className="w-full max-w-md rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-6 shadow-2xl">
            <h2 className="text-lg font-medium text-[var(--desk-text-primary)]">
              {copy.editTitle}
            </h2>
            <div className="mt-4 space-y-4">
              <label className="block text-xs text-[var(--desk-text-secondary)]">
                <span className="mb-1 block">{copy.editDeliveryDate}</span>
                <input
                  type="date"
                  value={editDate}
                  onChange={(e) => setEditDate(e.target.value)}
                  className="w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] px-3 py-2 text-sm text-[var(--desk-text-primary)]"
                />
              </label>

              {(proposal.items ?? []).length > 0 && (
                <div className="max-h-56 space-y-2 overflow-y-auto">
                  {(proposal.items ?? []).map((item) => (
                    <div
                      key={item.product_id}
                      className="flex items-center justify-between gap-3 rounded-xl border border-[var(--desk-border)] px-3 py-2"
                    >
                      <div className="min-w-0">
                        <p className="truncate text-sm text-[var(--desk-text-primary)]">
                          {item.product_name}
                        </p>
                        <p className="text-[11px] text-[var(--desk-text-tertiary)]">
                          {formatFastWholeSom(item.unit_price_minor, currency)}
                        </p>
                      </div>
                      <input
                        type="number"
                        min={0}
                        value={editQuantities[item.product_id] ?? item.quantity}
                        onChange={(e) =>
                          setEditQuantities((prev) => ({
                            ...prev,
                            [item.product_id]: Number(e.target.value) || 0,
                          }))
                        }
                        className="w-20 rounded-lg border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] px-2 py-1 text-right text-sm text-[var(--desk-text-primary)]"
                      />
                    </div>
                  ))}
                </div>
              )}
            </div>

            <div className="mt-6 flex justify-end gap-2">
              <button
                type="button"
                onClick={() => setEditingOpen(false)}
                className="portal-btn portal-btn--ghost rounded-xl px-4 py-2 text-xs"
              >
                {copy.cancelCta}
              </button>
              <button
                type="button"
                disabled={busyAction === "edit"}
                onClick={() => void handleSaveEdit()}
                className="portal-btn portal-btn--primary rounded-xl px-4 py-2 text-xs"
              >
                {copy.saveEditCta}
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
