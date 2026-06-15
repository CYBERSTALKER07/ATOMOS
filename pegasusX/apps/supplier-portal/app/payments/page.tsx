"use client";

import { ApiClient } from "@pegasusx/api-client";
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

type LoadState =
  | { status: "loading" }
  | { status: "ready"; snapshot: FinanceAuthoritySnapshot }
  | { status: "error"; message: string };

export default function PaymentsAuthorityPage() {
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
    <div className="desk-page">
      <div className="desk-page-header">
        <div>
          <h1 className="desk-page-title">Payment settlement authority</h1>
          <p className="desk-page-subtitle">
            Immutable ledger-derived settlement and reconciliation view for supplier finance and support.
          </p>
        </div>
        <div className="desk-toolbar">
          <button className="md-btn md-btn-outlined" type="button" onClick={() => setRefreshTick((value) => value + 1)}>
            {isRefreshing ? "Refreshing..." : "Refresh now"}
          </button>
        </div>
      </div>

      <section className="md-card md-shape-md p-4 mb-6">
        <p className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
          Live finance stream
        </p>
        <p className="md-typescale-body-medium mt-2">{liveState.message}</p>
        {(liveState.lastEventType || liveState.lastEventAt) && (
          <p className="md-typescale-body-small mt-2" style={{ color: "var(--color-md-outline)" }}>
            {liveState.lastEventType ? `Last event: ${liveState.lastEventType}. ` : ""}
            {liveState.lastEventAt ? `Updated ${formatDateTime(liveState.lastEventAt)}.` : ""}
          </p>
        )}
      </section>

      {state.status === "loading" && (
        <section className="md-card md-shape-md p-6">
          <p className="md-typescale-body-large">Loading settlement authority view...</p>
        </section>
      )}

      {state.status === "error" && (
        <section className="md-card md-shape-md p-6" role="alert">
          <h2 className="md-typescale-title-large">Payment authority unavailable</h2>
          <p className="md-typescale-body-medium mt-2" style={{ color: "var(--color-md-error)" }}>
            {state.message}
          </p>
        </section>
      )}

      {state.status === "ready" && (
        <>
          {state.snapshot.source === "ledger_fallback" && (
            <section className="md-card md-shape-md p-4 mb-4">
              <p className="md-typescale-body-medium" style={{ color: "var(--color-md-outline)" }}>
                Settlement summary endpoint unavailable. Showing derived fallback from payment ledger entries.
              </p>
            </section>
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
            <h2 className="md-typescale-title-large">Totals by currency</h2>
            {state.snapshot.authority.totals_by_currency.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No totals available for the current filter window.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Currency</th>
                      <th className="py-2 pr-4">Entry count</th>
                      <th className="py-2 pr-4">Amount (minor units)</th>
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
            <h2 className="md-typescale-title-large">Reconciliation mismatches</h2>
            {state.snapshot.mismatches.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No non-zero mismatches detected for current ledger authority scope.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Gateway</th>
                      <th className="py-2 pr-4">Currency</th>
                      <th className="py-2 pr-4">Net (minor units)</th>
                      <th className="py-2 pr-4">Credit total</th>
                      <th className="py-2 pr-4">Debit total</th>
                      <th className="py-2 pr-4">Entries</th>
                      <th className="py-2 pr-4">Last occurred</th>
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
            <h2 className="md-typescale-title-large">Settlement groups</h2>
            {state.snapshot.authority.items.length === 0 ? (
              <p className="md-typescale-body-medium mt-3" style={{ color: "var(--color-md-outline)" }}>
                No settlement groups found for the active filters.
              </p>
            ) : (
              <div className="mt-3 overflow-x-auto">
                <table className="w-full text-left">
                  <thead>
                    <tr className="md-typescale-label-medium" style={{ color: "var(--color-md-outline)" }}>
                      <th className="py-2 pr-4">Gateway</th>
                      <th className="py-2 pr-4">Entry type</th>
                      <th className="py-2 pr-4">Currency</th>
                      <th className="py-2 pr-4">Entries</th>
                      <th className="py-2 pr-4">Amount (minor units)</th>
                      <th className="py-2 pr-4">First occurred</th>
                      <th className="py-2 pr-4">Last occurred</th>
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
    </div>
  );
}
