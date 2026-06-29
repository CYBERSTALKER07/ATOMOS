"use client";

import { ApiClient, supplierChargebackKey, supplierChargebackReversalKey } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type { PaymentChargebackRequest, PaymentChargebackReversalRequest, PaymentLedgerEntry } from "@pegasusx/types";
import { useEffect, useMemo, useState } from "react";
import type { FinanceAuthoritySnapshot } from "../payments/_shared/finance";
import {
  createIdempotencyKey,
  errorToMessage,
  formatDateTime,
  formatMinor,
  loadFinanceAuthoritySnapshot,
  parseDate,
  useSupplierFinanceLiveRefresh,
} from "../payments/_shared/finance";
import { PageChrome } from "@/components/PageChrome";
import { PortalSection } from "@/components/portal";

type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };

type ActionState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "success"; message: string }
  | { status: "error"; message: string };

const gatewayOptions = ["ADYEN", "GLOBAL_PAY", "STRIPE", "PAYME", "CLICK", "CASH"];

export default function EarningsPage() {
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const [refreshTick, setRefreshTick] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [chargebackForm, setChargebackForm] = useState<PaymentChargebackRequest>({
    order_id: "",
    retailer_id: "",
    gateway: gatewayOptions[0],
    amount: 0,
    currency: "UZS",
  });
  const [reversalForm, setReversalForm] = useState<PaymentChargebackReversalRequest>({
    session_id: "",
  });
  const [chargebackState, setChargebackState] = useState<ActionState>({ status: "idle" });
  const [reversalState, setReversalState] = useState<ActionState>({ status: "idle" });
  const liveState = useSupplierFinanceLiveRefresh(() => {
    setRefreshTick((value) => value + 1);
  });

  useSupplierSessionReconcile(() => {
    if (chargebackState.status === "submitting") {
      setChargebackState({
        status: "error",
        message: "Connection restored — verify chargeback status before retrying.",
      });
    }
    if (reversalState.status === "submitting") {
      setReversalState({
        status: "error",
        message: "Connection restored — verify reversal status before retrying.",
      });
    }
    setRefreshTick((value) => value + 1);
  });

  useEffect(() => {
    let cancelled = false;

    async function load() {
      if (state.status === "ready") {
        setIsRefreshing(true);
      } else {
        setState({ status: "loading" });
      }

      try {
        const snapshot = await loadFinanceAuthoritySnapshot(api);
        if (!cancelled) {
          setState({ status: "ready", snapshot });
        }
      } catch (error) {
        if (!cancelled) {
          setState({ status: "error", message: errorToMessage(error) });
        }
      } finally {
        if (!cancelled) {
          setIsRefreshing(false);
        }
      }
    }

    void load();
    return () => {
      cancelled = true;
    };
  }, [api, refreshTick, state.status]);

  const disputeEntries = state.status === "ready" ? sortLedgerEntries(state.snapshot.ledger.items).filter(isDisputeEntry) : [];
  const recentEntries = state.status === "ready" ? sortLedgerEntries(state.snapshot.ledger.items).slice(0, 12) : [];

  async function submitChargeback(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setChargebackState({ status: "submitting" });
    try {
      await api.recordPaymentChargeback(
        chargebackForm,
        supplierChargebackKey(chargebackForm.order_id, chargebackForm.order_id),
      );
      setChargebackState({ status: "success", message: "Chargeback recorded. Live finance refresh queued." });
      setRefreshTick((value) => value + 1);
    } catch (error) {
      setChargebackState({ status: "error", message: errorToMessage(error) });
    }
  }

  async function submitReversal(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setReversalState({ status: "submitting" });
    try {
      await api.recordPaymentChargebackReversal(
        reversalForm,
        supplierChargebackReversalKey(reversalForm.session_id, reversalForm.session_id),
      );
      setReversalState({ status: "success", message: "Chargeback reversal recorded. Live finance refresh queued." });
      setRefreshTick((value) => value + 1);
    } catch (error) {
      setReversalState({ status: "error", message: errorToMessage(error) });
    }
  }

  return (
    <PageChrome
      icon="treasury"
      title="Earnings & Treasury"
      description="Supplier treasury authority, disputes, and reconciliation operations sourced directly from payment ledger state."
      loading={state.status === "loading"}
      skeletonVariant="table"
      error={state.status === "error" ? state.message : null}
      actions={
        <button className="portal-btn portal-btn--outline" type="button" onClick={() => setRefreshTick((value) => value + 1)}>
          {isRefreshing ? "Refreshing…" : "Refresh now"}
        </button>
      }
    >
      <PortalSection title="Live treasury stream" icon="treasury">
        <p className="md-typescale-body-medium">{liveState.message}</p>
        {(liveState.lastEventType || liveState.lastEventAt) && (
          <p className="md-typescale-body-small mt-2" style={{ color: "var(--desk-text-secondary)" }}>
            {liveState.lastEventType ? `Last event: ${liveState.lastEventType}. ` : ""}
            {liveState.lastEventAt ? `Updated ${formatDateTime(liveState.lastEventAt)}.` : ""}
          </p>
        )}
      </PortalSection>

      {state.status === "ready" && (
        <>
          {state.snapshot.source === "ledger_fallback" && (
            <section className="md-card md-shape-md p-4 mb-4">
              <p className="md-typescale-body-medium" style={{ color: "var(--color-md-outline)" }}>
                Settlement authority endpoint unavailable. Treasury values are derived from the durable ledger fallback.
              </p>
            </section>
          )}

          <section className="grid gap-4 md:grid-cols-2 xl:grid-cols-4 mb-6">
            <MetricCard label="Supplier scope" value={state.snapshot.authority.supplier_id || "(global)"} />
            <MetricCard label="Ledger rows" value={String(state.snapshot.ledger.count)} />
            <MetricCard label="Dispute rows" value={String(disputeEntries.length)} />
            <MetricCard label="Mismatch groups" value={String(state.snapshot.mismatches.length)} />
          </section>

          <section className="grid gap-6 xl:grid-cols-[2fr,1fr] mb-6">
            <article className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">Totals by currency</h2>
              {state.snapshot.authority.totals_by_currency.length === 0 ? (
                <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                  No currency totals available for the active finance window.
                </p>
              ) : (
                <div className="mt-4 grid gap-4 md:grid-cols-2">
                  {state.snapshot.authority.totals_by_currency.map((row) => (
                    <div key={row.currency} className="md-card md-shape-sm p-4" style={{ background: "var(--color-md-surface-container)" }}>
                      <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                        {row.currency}
                      </p>
                      <p className="md-typescale-title-large mt-2">{formatMinor(row.amount_minor_total, row.currency)}</p>
                      <p className="md-typescale-body-small mt-2" style={{ color: "var(--color-md-outline)" }}>
                        {row.entry_count} authority entries
                      </p>
                    </div>
                  ))}
                </div>
              )}
            </article>

            <aside className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">Finance actions</h2>
              <p className="md-typescale-body-small mt-2" style={{ color: "var(--color-md-outline)" }}>
                Use only for verified dispute handling and gateway-authorized reversals.
              </p>

              <form className="mt-5 grid gap-3" onSubmit={submitChargeback}>
                <h3 className="md-typescale-title-medium">Record chargeback</h3>
                <input
                  className="md-input-outlined"
                  placeholder="Order ID"
                  value={chargebackForm.order_id}
                  onChange={(event) => setChargebackForm((current) => ({ ...current, order_id: event.target.value }))}
                />
                <input
                  className="md-input-outlined"
                  placeholder="Retailer ID"
                  value={chargebackForm.retailer_id}
                  onChange={(event) => setChargebackForm((current) => ({ ...current, retailer_id: event.target.value }))}
                />
                <select
                  className="md-input-outlined"
                  value={chargebackForm.gateway}
                  onChange={(event) => setChargebackForm((current) => ({ ...current, gateway: event.target.value }))}
                >
                  {gatewayOptions.map((gateway) => (
                    <option key={gateway} value={gateway}>
                      {gateway}
                    </option>
                  ))}
                </select>
                <div className="grid gap-3 md:grid-cols-2">
                  <input
                    className="md-input-outlined"
                    placeholder="Amount (minor units)"
                    inputMode="numeric"
                    value={chargebackForm.amount === 0 ? "" : String(chargebackForm.amount)}
                    onChange={(event) =>
                      setChargebackForm((current) => ({
                        ...current,
                        amount: Number.parseInt(event.target.value || "0", 10) || 0,
                      }))
                    }
                  />
                  <input
                    className="md-input-outlined"
                    placeholder="Currency"
                    value={chargebackForm.currency ?? "UZS"}
                    onChange={(event) => setChargebackForm((current) => ({ ...current, currency: event.target.value || "UZS" }))}
                  />
                </div>
                <button className="md-btn md-btn-filled" type="submit" disabled={chargebackState.status === "submitting"}>
                  {chargebackState.status === "submitting" ? "Recording..." : "Record chargeback"}
                </button>
                <ActionMessage state={chargebackState} />
              </form>

              <form className="mt-6 grid gap-3" onSubmit={submitReversal}>
                <h3 className="md-typescale-title-medium">Record reversal</h3>
                <input
                  className="md-input-outlined"
                  placeholder="Session ID"
                  value={reversalForm.session_id}
                  onChange={(event) => setReversalForm({ session_id: event.target.value })}
                />
                <button className="md-btn md-btn-tonal" type="submit" disabled={reversalState.status === "submitting"}>
                  {reversalState.status === "submitting" ? "Recording..." : "Record reversal"}
                </button>
                <ActionMessage state={reversalState} />
              </form>
            </aside>
          </section>

          <section className="md-card md-shape-md p-6 mb-6">
            <h2 className="md-typescale-title-large">Recent dispute history</h2>
            {disputeEntries.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No recorded chargebacks or reversals for the current finance window.
              </p>
            ) : (
              <FinanceEntryTable entries={disputeEntries.slice(0, 12)} emptyLabel="No dispute entries found." />
            )}
          </section>

          <section className="md-card md-shape-md p-6 mb-6">
            <h2 className="md-typescale-title-large">Recent ledger activity</h2>
            <FinanceEntryTable entries={recentEntries} emptyLabel="No recent ledger activity." />
          </section>

          <section className="md-card md-shape-md p-6">
            <h2 className="md-typescale-title-large">Reconciliation mismatches</h2>
            {state.snapshot.mismatches.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No mismatch groups detected for the current supplier scope.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Gateway</th>
                      <th className="py-2 pr-4">Currency</th>
                      <th className="py-2 pr-4">Net</th>
                      <th className="py-2 pr-4">Credit total</th>
                      <th className="py-2 pr-4">Debit total</th>
                      <th className="py-2 pr-4">Entries</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.snapshot.mismatches.map((row) => (
                      <tr key={`${row.gateway}:${row.currency}`} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{row.gateway}</td>
                        <td className="py-2 pr-4">{row.currency}</td>
                        <td className="py-2 pr-4">{formatMinor(row.net_amount_minor, row.currency)}</td>
                        <td className="py-2 pr-4">{formatMinor(row.credit_amount_minor_total, row.currency)}</td>
                        <td className="py-2 pr-4">{formatMinor(row.debit_amount_minor_total, row.currency)}</td>
                        <td className="py-2 pr-4">{row.entry_count_total}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>
        </>
      )}
    </PageChrome>
  );
}

function sortLedgerEntries(entries: PaymentLedgerEntry[]): PaymentLedgerEntry[] {
  return [...entries].sort((left, right) => parseDate(right.occurred_at) - parseDate(left.occurred_at));
}

function isDisputeEntry(entry: PaymentLedgerEntry): boolean {
  return entry.entry_type === "CHARGEBACK_RECORDED" || entry.entry_type === "CHARGEBACK_REVERSAL_RECORDED";
}

function MetricCard({ label, value }: { label: string; value: string }) {
  return (
    <article className="md-card md-shape-md p-4">
      <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
        {label}
      </p>
      <p className="md-typescale-title-large mt-2">{value}</p>
    </article>
  );
}

function ActionMessage({ state }: { state: ActionState }) {
  if (state.status === "idle" || state.status === "submitting") {
    return null;
  }
  return (
    <p
      className="md-typescale-body-small"
      style={{ color: state.status === "error" ? "var(--color-md-error)" : "var(--color-md-outline)" }}
    >
      {state.message}
    </p>
  );
}

function FinanceEntryTable({ entries, emptyLabel }: { entries: PaymentLedgerEntry[]; emptyLabel: string }) {
  if (entries.length === 0) {
    return (
      <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
        {emptyLabel}
      </p>
    );
  }

  return (
    <div className="mt-3 overflow-x-auto">
      <table className="w-full text-left">
        <thead>
          <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
            <th className="py-2 pr-4">Occurred</th>
            <th className="py-2 pr-4">Gateway</th>
            <th className="py-2 pr-4">Entry type</th>
            <th className="py-2 pr-4">Reference</th>
            <th className="py-2 pr-4">Amount</th>
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => (
            <tr key={entry.ledger_entry_id} className="md-typescale-body-medium">
              <td className="py-2 pr-4">{formatDateTime(entry.occurred_at)}</td>
              <td className="py-2 pr-4">{entry.gateway}</td>
              <td className="py-2 pr-4">{entry.entry_type}</td>
              <td className="py-2 pr-4">{entry.reference_id || entry.session_id || entry.order_id || "-"}</td>
              <td className="py-2 pr-4">{formatMinor(entry.amount_minor, entry.currency)}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
