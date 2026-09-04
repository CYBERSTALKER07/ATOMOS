"use client";

import { usePortalT } from "@/lib/i18n";
import { ApiClient, supplierChargebackKey, supplierChargebackReversalKey } from "@pegasusx/api-core";
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
import { useSupplierPaymentCatalog } from "@/lib/use-payment-catalog";

type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };

type ActionState =
  | { status: "idle" }
  | { status: "submitting" }
  | { status: "success"; message: string }
  | { status: "error"; message: string };

export default function EarningsPage() {
  const t = usePortalT();
  const { gateways, currency: packCurrencyCode } = useSupplierPaymentCatalog();
  const gatewayOptions = gateways.length ? gateways : ["CASH"];
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const [refreshTick, setRefreshTick] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const [chargebackForm, setChargebackForm] = useState<PaymentChargebackRequest>({
    order_id: "",
    retailer_id: "",
    gateway: "CASH",
    amount: 0,
    currency: "",
  });
  useEffect(() => {
    setChargebackForm((current) => ({
      ...current,
      gateway: gatewayOptions.includes(current.gateway) ? current.gateway : gatewayOptions[0],
      currency: packCurrencyCode || current.currency,
    }));
  }, [gatewayOptions.join("|"), packCurrencyCode]);
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
        message: t("supplier_portal.residual.text.connection_restored_verify_chargeback_status_before_retrying"),
      });
    }
    if (reversalState.status === "submitting") {
      setReversalState({
        status: "error",
        message: t("supplier_portal.residual.text.connection_restored_verify_reversal_status_before_retrying"),
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
      setChargebackState({ status: "success", message: t("supplier_portal.residual.text.chargeback_recorded_live_finance_refresh_queued") });
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
      setReversalState({ status: "success", message: t("supplier_portal.residual.text.chargeback_reversal_recorded_live_finance_refresh_queued") });
      setRefreshTick((value) => value + 1);
    } catch (error) {
      setReversalState({ status: "error", message: errorToMessage(error) });
    }
  }

  return (
    <PageChrome
      icon="treasury"
      title={t("supplier_portal.earnings.text.earnings_and_treasury")}
      description={t("supplier_portal.residual.text.supplier_treasury_authority_disputes_and_reconciliation_operatio")}
      loading={state.status === "loading"}
      skeletonVariant="table"
      error={state.status === "error" ? state.message : null}
      actions={
        <button className="portal-btn portal-btn--outline" type="button" onClick={() => setRefreshTick((value) => value + 1)}>
          {isRefreshing ? "Refreshing…" : "Refresh now"}
        </button>
      }
    >
      <PortalSection title={t("supplier_portal.earnings.text.live_treasury_stream")} icon="treasury">
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
            <MetricCard label={t("supplier_portal.residual.text.supplier_scope")} value={state.snapshot.authority.supplier_id || "(global)"} />
            <MetricCard label={t("supplier_portal.residual.text.ledger_rows")} value={String(state.snapshot.ledger.count)} />
            <MetricCard label={t("supplier_portal.residual.text.dispute_rows")} value={String(disputeEntries.length)} />
            <MetricCard label={t("supplier_portal.residual.text.mismatch_groups")} value={String(state.snapshot.mismatches.length)} />
          </section>

          <section className="grid gap-6 xl:grid-cols-[2fr,1fr] mb-6">
            <article className="md-card md-shape-md p-6">
              <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.totals_by_currency")}</h2>
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
              <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.finance_actions")}</h2>
              <p className="md-typescale-body-small mt-2" style={{ color: "var(--color-md-outline)" }}>
                Use only for verified dispute handling and gateway-authorized reversals.
              </p>

              <form className="mt-5 grid gap-3" onSubmit={submitChargeback}>
                <h3 className="md-typescale-title-medium">{t("supplier_portal.earnings.text.record_chargeback")}</h3>
                <input
                  className="md-input-outlined"
                  placeholder={t("supplier_portal.admin.control_center.field.order_id")}
                  value={chargebackForm.order_id}
                  onChange={(event) => setChargebackForm((current) => ({ ...current, order_id: event.target.value }))}
                />
                <input
                  className="md-input-outlined"
                  placeholder={t("supplier_portal.chargebacks.text.retailer_id")}
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
                    placeholder={t("supplier_portal.chargebacks.text.amount_minor_units")}
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
                    placeholder={t("supplier_portal.chargebacks.text.currency")}
                    value={packCurrencyCode || chargebackForm.currency}
                    readOnly
                  />
                </div>
                <button className="md-btn md-btn-filled" type="submit" disabled={chargebackState.status === "submitting"}>
                  {chargebackState.status === "submitting" ? "Recording..." : "Record chargeback"}
                </button>
                <ActionMessage state={chargebackState} />
              </form>

              <form className="mt-6 grid gap-3" onSubmit={submitReversal}>
                <h3 className="md-typescale-title-medium">{t("supplier_portal.earnings.text.record_reversal")}</h3>
                <input
                  className="md-input-outlined"
                  placeholder={t("supplier_portal.admin.control_center.field.session_id")}
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
            <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.recent_dispute_history")}</h2>
            {disputeEntries.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No recorded chargebacks or reversals for the current finance window.
              </p>
            ) : (
              <FinanceEntryTable entries={disputeEntries.slice(0, 12)} emptyLabel="No dispute entries found." />
            )}
          </section>

          <section className="md-card md-shape-md p-6 mb-6">
            <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.recent_ledger_activity")}</h2>
            <FinanceEntryTable entries={recentEntries} emptyLabel="No recent ledger activity." />
          </section>

          <section className="md-card md-shape-md p-6">
            <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.reconciliation_mismatches")}</h2>
            {state.snapshot.mismatches.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No mismatch groups detected for the current supplier scope.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.gateway")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.currency")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.net")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.credit_total")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.debit_total")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.entries")}</th>
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
  const t = usePortalT();
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
            <th className="py-2 pr-4">{t("supplier_portal.earnings.text.occurred")}</th>
            <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.gateway")}</th>
            <th className="py-2 pr-4">{t("supplier_portal.earnings.text.entry_type")}</th>
            <th className="py-2 pr-4">{t("supplier_portal.earnings.text.reference")}</th>
            <th className="py-2 pr-4">{t("supplier_portal.ledger.text.amount")}</th>
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
