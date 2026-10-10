"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import EmptyState from '@/components/EmptyState';
import { Invoice } from '@/app/treasury/page';

interface TreasuryTransactionListProps {
  invoices: Invoice[];
  fmt: (n: number) => string;
  resolveAmount: (inv: Invoice) => number;
  resolveCurrency: (inv: Invoice) => string;
  formatPayoutOwner: (inv: Invoice) => string;
}

export function TreasuryTransactionList({
  invoices,
  fmt,
  resolveAmount,
  resolveCurrency,
  formatPayoutOwner,
}: TreasuryTransactionListProps) {
  const t = usePortalT();
  if (invoices.length === 0) {
    return (
      <EmptyState variant="no-data" headline={t("warehouse_portal.residual.text.no_invoices_found")} body={t("warehouse_portal.residual.text.invoices_appear_when_retailers_are_billed_for_fulfilled_orders")} />
    );
  }

  return (
    <div className="desk-table-wrap">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]">
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.treasury.treasury_transaction_list.text.invoice")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.orders._id_.text.retailer")}</th>
            <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.treasury.treasury_transaction_list.text.amount")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.bins.text.status")}</th>
            <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.treasury.treasury_transaction_list.text.due")}</th>
          </tr>
        </thead>
        <tbody>
          {invoices.map(inv => (
            <tr key={inv.invoice_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
              <td className="py-2.5 px-3 font-mono text-xs">{inv.invoice_id.slice(0, 8)}...</td>
              <td className="py-2.5 px-3">{inv.retailer_name || '—'}</td>
              <td className="py-2.5 px-3 text-right font-mono">{fmt(resolveAmount(inv))} {resolveCurrency(inv)}</td>
              <td className="py-2.5 px-3">
                <span className={`status-chip ${inv.status === 'PAID' ? 'status-chip--stable' : inv.status === 'OVERDUE' ? 'status-chip--critical' : 'status-chip--draft'}`}>
                  {inv.status}
                </span>
                <div className="text-[11px] mt-1 text-[var(--muted)]">
                  Owner {formatPayoutOwner(inv)}
                  {typeof inv.fee_amount === 'number' ? ` · Fee ${fmt(inv.fee_amount)}` : ''}
                  {typeof inv.net_payout_amount === 'number' ? ` · Net ${fmt(inv.net_payout_amount)}` : ''}
                </div>
              </td>
              <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                {inv.due_date ? new Date(inv.due_date).toLocaleDateString() : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
