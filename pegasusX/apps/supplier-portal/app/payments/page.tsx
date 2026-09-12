"use client";

import { usePortalT } from "@/lib/i18n";
import { ApiClient } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useEffect, useMemo, useState } from "react";
import type { FinanceAuthoritySnapshot } from "./_shared/finance";
import {
  errorToMessage,
  formatDateTime,
  formatMinor,
  loadFinanceAuthoritySnapshot,
  useSupplierFinanceLiveRefresh,
} from "./_shared/finance";
import { PageChrome } from "@/components/PageChrome";
import { FormAlert, PortalSection } from "@/components/portal";

type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };

export default function PaymentsAuthorityPage() {
  const t = usePortalT();
  const api = useMemo(() => createSupplierApi(), []);
  const [state, setState] = useState<LoadState>({ status: "loading" });
  const [refreshTick, setRefreshTick] = useState(0);
  const [isRefreshing, setIsRefreshing] = useState(false);
  const liveState = useSupplierFinanceLiveRefresh(() => {
    setRefreshTick((value) => value + 1);
  });

  useSupplierSessionReconcile(() => {
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

  return (
    <PageChrome
      icon="payment"
      title={t("supplier_portal.payments.text.payment_settlement_authority")}
      description={t("supplier_portal.residual.text.immutable_ledger_derived_settlement_and_reconciliation_view_for_")}
      loading={state.status === "loading"}
      skeletonVariant="table"
      error={state.status === "error" ? state.message : null}
      actions={
        <button
          className="portal-btn portal-btn--outline"
          type="button"
          onClick={() => setRefreshTick((value) => value + 1)}
        >
          {isRefreshing ? "Refreshing…" : "Refresh now"}
        </button>
      }
    >
      <PortalSection title={t("supplier_portal.payments.text.live_finance_stream")} icon="treasury">
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
            <FormAlert>{t("supplier_portal.payments.text.settlement_summary_endpoint_unavailable_showing_derived_fallback")}</FormAlert>
          )}

          <section className="grid gap-4 md:grid-cols-4 mb-6">
            <article className="md-card md-shape-md p-4">
              <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                Supplier scope
              </p>
              <p className="md-typescale-title-large mt-2">{state.snapshot.authority.supplier_id || "(global)"}</p>
            </article>
            <article className="md-card md-shape-md p-4">
              <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                Grouped rows
              </p>
              <p className="md-typescale-title-large mt-2">{state.snapshot.authority.count}</p>
            </article>
            <article className="md-card md-shape-md p-4">
              <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                Total entries
              </p>
              <p className="md-typescale-title-large mt-2">{state.snapshot.authority.entry_count_total}</p>
            </article>
            <article className="md-card md-shape-md p-4">
              <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                Reconciliation groups
              </p>
              <p className="md-typescale-title-large mt-2">{state.snapshot.mismatches.length}</p>
            </article>
          </section>

          <section className="md-card md-shape-md p-6 mb-6">
            <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.totals_by_currency")}</h2>
            {state.snapshot.authority.totals_by_currency.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No totals available for the current filter window.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.currency")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.payments.text.entry_count")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.amount_minor_units")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.snapshot.authority.totals_by_currency.map((row) => (
                      <tr key={row.currency} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{row.currency}</td>
                        <td className="py-2 pr-4">{row.entry_count}</td>
                        <td className="py-2 pr-4">{formatMinor(row.amount_minor_total, row.currency)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="md-card md-shape-md p-6 mb-6">
            <h2 className="md-typescale-title-large">{t("supplier_portal.earnings.text.reconciliation_mismatches")}</h2>
            {state.snapshot.mismatches.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No non-zero mismatches detected for current ledger authority scope.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.gateway")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.currency")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.payments.text.net_minor_units")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.credit_total")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.debit_total")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.entries")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.payments.text.last_occurred")}</th>
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
                        <td className="py-2 pr-4">{formatDateTime(row.last_occurred_at)}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="md-card md-shape-md p-6">
            <h2 className="md-typescale-title-large">{t("supplier_portal.payments.text.settlement_groups")}</h2>
            {state.snapshot.authority.items.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No settlement groups found for the active filters.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.gateway")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.entry_type")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.currency")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.earnings.text.entries")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.chargebacks.text.amount_minor_units")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.payments.text.first_occurred")}</th>
                      <th className="py-2 pr-4">{t("supplier_portal.payments.text.last_occurred")}</th>
                    </tr>
                  </thead>
                  <tbody>
                    {state.snapshot.authority.items.map((row) => (
                      <tr key={`${row.gateway}:${row.entry_type}:${row.currency}`} className="md-typescale-body-medium">
                        <td className="py-2 pr-4">{row.gateway}</td>
                        <td className="py-2 pr-4">{row.entry_type}</td>
                        <td className="py-2 pr-4">{row.currency}</td>
                        <td className="py-2 pr-4">{row.entry_count}</td>
                        <td className="py-2 pr-4">{formatMinor(row.amount_minor_total, row.currency)}</td>
                        <td className="py-2 pr-4">{formatDateTime(row.first_occurred_at)}</td>
                        <td className="py-2 pr-4">{formatDateTime(row.last_occurred_at)}</td>
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
