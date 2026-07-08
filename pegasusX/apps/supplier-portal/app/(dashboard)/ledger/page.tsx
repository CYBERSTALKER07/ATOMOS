"use client";

import { useCallback } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw, Receipt } from "lucide-react";

type PaymentLedgerEntry = {
  id: string;
  entry_type: string;
  amount_minor: number;
  currency: string;
  order_id?: string;
  occurred_at: string;
};

type PaymentLedgerResponse = {
  items: PaymentLedgerEntry[];
};

export default function LedgerPage() {
  const {
    data,
    loading,
    error,
    isRefreshing,
    mutate,
  } = useLiveData<PaymentLedgerResponse>("/v1/payment/ledger");

  const rows = data?.items || [];

  const refreshAll = useCallback(() => {
    void mutate();
  }, [mutate]);

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: "var(--desk-canvas)" }}>
      <PageChrome
        icon="orders"
        title="Payment Ledger"
        description="View your durable finance ledger and payment movements."
        loading={loading}
        skeletonVariant="table"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={loading || isRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
              />
              {isRefreshing ? "Syncing" : "Sync"}
            </button>
          </div>
        }
      >
        {error && (
          <div className="mb-6 p-4 rounded-xl border bg-[var(--desk-danger)]/10 text-[var(--desk-danger)] border-[var(--desk-danger)]/30">
            {error.message || "Failed to load ledger."}
          </div>
        )}

        <PageSection title="Ledger Entries" description="Recent financial transactions.">
          <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)] overflow-hidden">
            {rows.length === 0 ? (
              <div className="p-12 text-center text-[var(--desk-text-tertiary)] flex flex-col items-center">
                <Receipt size={48} className="opacity-20 mb-4" />
                <p className="md-typescale-body-large">No ledger entries.</p>
                <p className="md-typescale-body-small mt-1">Payment movements will appear here.</p>
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left border-collapse">
                  <thead>
                    <tr className="border-b border-[var(--desk-border)] bg-[var(--desk-surface-subtle)]">
                      <th className="py-3 px-4 md-typescale-label-small uppercase text-[var(--desk-text-tertiary)] font-medium">Date</th>
                      <th className="py-3 px-4 md-typescale-label-small uppercase text-[var(--desk-text-tertiary)] font-medium">Type</th>
                      <th className="py-3 px-4 md-typescale-label-small uppercase text-[var(--desk-text-tertiary)] font-medium">Order ID</th>
                      <th className="py-3 px-4 md-typescale-label-small uppercase text-[var(--desk-text-tertiary)] font-medium text-right">Amount</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-[var(--desk-border)]">
                    {rows.map((row) => (
                      <tr key={row.id} className="hover:bg-[var(--desk-surface-hover)] transition-colors">
                        <td className="py-3 px-4 md-typescale-body-small text-[var(--desk-text-secondary)]">
                          {new Date(row.occurred_at).toLocaleString()}
                        </td>
                        <td className="py-3 px-4 md-typescale-body-medium font-medium text-[var(--desk-text-primary)]">
                          {row.entry_type}
                        </td>
                        <td className="py-3 px-4 md-typescale-body-small text-[var(--desk-text-secondary)]">
                          {row.order_id || "—"}
                        </td>
                        <td className="py-3 px-4 text-right">
                          <span className={`md-typescale-body-medium font-medium ${row.amount_minor < 0 ? "text-[var(--desk-danger)]" : "text-[var(--desk-success)]"}`}>
                            {row.amount_minor > 0 ? "+" : ""}{(row.amount_minor / 100).toFixed(2)} {row.currency}
                          </span>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>
        </PageSection>
      </PageChrome>
    </div>
  );
}
