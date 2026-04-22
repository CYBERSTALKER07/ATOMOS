'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface TreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
}

interface Invoice {
  invoice_id: string;
  retailer_name: string;
  amount_uzs: number;
  status: string;
  due_date: string;
  created_at: string;
}

export default function TreasuryPage() {
  const [overview, setOverview] = useState<TreasuryOverview | null>(null);
  const [invoices, setInvoices] = useState<Invoice[]>([]);
  const [loading, setLoading] = useState(true);
  const [view, setView] = useState<'overview' | 'invoices'>('overview');

  const load = useCallback(async () => {
    try {
      const [ovRes, invRes] = await Promise.all([
        apiFetch('/v1/warehouse/ops/treasury?view=overview'),
        apiFetch('/v1/warehouse/ops/treasury?view=invoices'),
      ]);
      if (ovRes.ok) setOverview(await ovRes.json());
      if (invRes.ok) {
        const data = await invRes.json();
        setInvoices(data.invoices || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div className="md-skeleton md-skeleton-title" />
        <div className="grid grid-cols-3 gap-4">
          {Array.from({ length: 3 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-card" />)}
        </div>
      </div>
    );
  }

  const ov = overview || { total_invoiced: 0, total_paid: 0, total_outstanding: 0 };

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Treasury</h1>
        <div className="flex gap-2">
          {(['overview', 'invoices'] as const).map(v => (
            <button
              key={v}
              onClick={() => setView(v)}
              className={`px-3 py-1.5 rounded-lg text-sm font-medium capitalize ${v === view ? 'button--primary' : 'button--secondary'}`}
            >
              {v}
            </button>
          ))}
        </div>
      </div>

      {/* Overview KPIs */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Total Invoiced</div>
          <div className="text-2xl font-bold">{fmt(ov.total_invoiced)} UZS</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Paid</div>
          <div className="text-2xl font-bold text-[var(--success)]">{fmt(ov.total_paid)} UZS</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Outstanding</div>
          <div className="text-2xl font-bold" style={{ color: ov.total_outstanding > 0 ? 'var(--danger)' : 'var(--foreground)' }}>
            {fmt(ov.total_outstanding)} UZS
          </div>
        </div>
      </div>

      {/* Invoices Table */}
      {view === 'invoices' && (
        invoices.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 text-[var(--muted)]">
            <Icon name="treasury" size={48} className="mb-3 opacity-40" />
            <p className="text-sm">No invoices found</p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--border)]">
                  <th className="text-left py-2 px-3 font-medium">Invoice</th>
                  <th className="text-left py-2 px-3 font-medium">Retailer</th>
                  <th className="text-right py-2 px-3 font-medium">Amount (UZS)</th>
                  <th className="text-left py-2 px-3 font-medium">Status</th>
                  <th className="text-right py-2 px-3 font-medium">Due</th>
                </tr>
              </thead>
              <tbody>
                {invoices.map(inv => (
                  <tr key={inv.invoice_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                    <td className="py-2.5 px-3 font-mono text-xs">{inv.invoice_id.slice(0, 8)}...</td>
                    <td className="py-2.5 px-3">{inv.retailer_name || '—'}</td>
                    <td className="py-2.5 px-3 text-right font-mono">{fmt(inv.amount_uzs)}</td>
                    <td className="py-2.5 px-3">
                      <span className={`status-chip ${inv.status === 'PAID' ? 'status-chip--stable' : inv.status === 'OVERDUE' ? 'status-chip--critical' : 'status-chip--draft'}`}>
                        {inv.status}
                      </span>
                    </td>
                    <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                      {inv.due_date ? new Date(inv.due_date).toLocaleDateString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )
      )}
    </div>
  );
}
