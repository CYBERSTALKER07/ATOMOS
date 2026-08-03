import React from 'react';
import EmptyState from '@/components/EmptyState';
<<<<<<< HEAD
import { desktopPrint } from '@pegasusx/desktop-bridge';
import { downloadCsv } from '@/lib/csv';
=======
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
import { Invoice } from '@/app/treasury/page';

interface TreasuryTransactionListProps {
  invoices: Invoice[];
  fmt: (n: number) => string;
  resolveAmount: (inv: Invoice) => number;
  resolveCurrency: (inv: Invoice) => string;
  formatPayoutOwner: (inv: Invoice) => string;
}

<<<<<<< HEAD
export const TreasuryTransactionList: React.FC<TreasuryTransactionListProps> = ({
=======
export function TreasuryTransactionList({
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  invoices,
  fmt,
  resolveAmount,
  resolveCurrency,
<<<<<<< HEAD
  formatPayoutOwner
}) => {
  const exportCsv = () => {
    void downloadCsv(
      `treasury_export_${new Date().toISOString().slice(0, 10)}.csv`,
      ['Invoice ID', 'Retailer', 'Amount', 'Currency', 'Status', 'Due Date'],
      invoices.map((inv) => [
        inv.invoice_id,
        inv.retailer_name || '',
        String(resolveAmount(inv)),
        resolveCurrency(inv),
        inv.status,
        inv.due_date ? new Date(inv.due_date).toLocaleDateString() : '',
      ]),
    );
  };

  const exportPdf = () => {
    desktopPrint({ title: 'Warehouse Treasury' });
  };

  return (
    <section className="desk-card overflow-hidden">
      <div className="px-5 py-4 border-b flex items-center justify-between" style={{ borderColor: 'var(--desk-border)' }}>
        <div>
          <h2 className="bento-card-title">Invoices</h2>
          <p className="text-sm mt-1" style={{ color: 'var(--desk-text-secondary)' }}>Retailer billing rows for this warehouse node.</p>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={exportCsv} className="md-btn md-btn-outlined text-sm px-3 py-1.5 flex items-center gap-2">
            <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/></svg>
            Export CSV
          </button>
          <button type="button" onClick={exportPdf} className="md-btn md-btn-outlined text-sm px-3 py-1.5 flex items-center gap-2">
            <svg width="16" height="16" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M17 17h2a2 2 0 002-2v-4a2 2 0 00-2-2H5a2 2 0 00-2 2v4a2 2 0 002 2h2m2 4h6a2 2 0 002-2v-4a2 2 0 00-2-2H9a2 2 0 00-2 2v4a2 2 0 002 2zm8-12V5a2 2 0 00-2-2H9a2 2 0 00-2 2v4h10z"/></svg>
            Export PDF
          </button>
        </div>
      </div>
    {invoices.length === 0 ? (
      <EmptyState variant="no-data" headline="No invoices found" body="Invoices appear when retailers are billed for fulfilled orders." />
    ) : (
      <div className="desk-table-wrap">
        <table className="desk-table w-full text-sm">
          <thead>
            <tr className="border-b border-[var(--border)]">
              <th className="text-left py-2 px-3 font-medium">Invoice</th>
              <th className="text-left py-2 px-3 font-medium">Retailer</th>
              <th className="text-right py-2 px-3 font-medium">Amount</th>
              <th className="text-left py-2 px-3 font-medium">Status</th>
              <th className="text-right py-2 px-3 font-medium">Due</th>
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
    )}
    </section>
  );
};
=======
  formatPayoutOwner,
}: TreasuryTransactionListProps) {
  if (invoices.length === 0) {
    return (
      <EmptyState variant="no-data" headline="No invoices found" body="Invoices appear when retailers are billed for fulfilled orders." />
    );
  }

  return (
    <div className="desk-table-wrap">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]">
            <th className="text-left py-2 px-3 font-medium">Invoice</th>
            <th className="text-left py-2 px-3 font-medium">Retailer</th>
            <th className="text-right py-2 px-3 font-medium">Amount</th>
            <th className="text-left py-2 px-3 font-medium">Status</th>
            <th className="text-right py-2 px-3 font-medium">Due</th>
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
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
