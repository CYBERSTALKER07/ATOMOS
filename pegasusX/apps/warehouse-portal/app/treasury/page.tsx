'use client';

import { useEffect, useState, useCallback } from 'react';
import { desktopPrint } from '@pegasusx/desktop-bridge';
import { downloadCsv } from '@/lib/csv';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { HubCard } from '@/components/portal';
import EmptyState from '@/components/EmptyState';
import { TreasuryTransactionList } from '@/components/treasury/TreasuryTransactionList';

interface TreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
}

export interface Invoice {
  invoice_id: string;
  retailer_name: string;
  amount?: number;
  amount_uzs?: number;
  currency?: string;
  status: string;
  fee_amount?: number;
  net_payout_amount?: number;
  payout_owner_type?: string;
  payout_owner_id?: string;
  fee_policy_version?: string;
  settlement_target?: string;
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
      const [ovRes, invRes, financials] = await Promise.all([
        apiFetch('/v1/warehouse/ops/treasury?view=overview'),
        apiFetch('/v1/warehouse/ops/treasury?view=invoices'),
        warehouseApi.getWarehouseOpsFinancials().catch(() => null),
      ]);
      if (ovRes.ok) {
        setOverview(await ovRes.json());
      } else if (financials) {
        setOverview({
          total_invoiced: financials.total_revenue,
          total_paid: financials.net_payout,
          total_outstanding: financials.cash_pending,
        });
      }
      if (invRes.ok) {
        const data = await invRes.json();
        setInvoices(data.invoices || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  useWarehouseSessionReconcile(() => {
    void load();
  });

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);
  const resolveAmount = (inv: Invoice) => {
    if (typeof inv.amount === 'number' && Number.isFinite(inv.amount)) return inv.amount;
    if (typeof inv.amount_uzs === 'number' && Number.isFinite(inv.amount_uzs)) return inv.amount_uzs;
    return 0;
  };
  const resolveCurrency = (inv: Invoice) => (inv.currency || 'UZS').toUpperCase();
  const formatPayoutOwner = (inv: Invoice) => {
    const ownerType = (inv.payout_owner_type || '').trim();
    const ownerID = (inv.payout_owner_id || '').trim();
    if (!ownerType && !ownerID) return 'Supplier';
    if (!ownerID) return ownerType;
    return `${ownerType}:${ownerID.slice(0, 8)}`;
  };

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

  const ov = overview || { total_invoiced: 0, total_paid: 0, total_outstanding: 0 };

  return (
    <PageTransition>
      <PageChrome
        icon="treasury"
        title="Treasury"
        description="Invoiced revenue, payouts, and outstanding liabilities for this warehouse."
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <div className="flex gap-2">
            {(['overview', 'invoices'] as const).map(v => (
              <button
                key={v}
                type="button"
                onClick={() => setView(v)}
                className={`px-3 py-1.5 rounded-lg text-sm font-medium capitalize ${v === view ? 'button--primary' : 'button--secondary'}`}
              >
                {v}
              </button>
            ))}
          </div>
        }
      >
      <div className="space-y-6">
      <div className="grid gap-4 sm:grid-cols-2">
        <HubCard
          href="/payment-config"
          icon="payment"
          title="Payment config"
          description="View checkout gateways and settlement routing for this node."
        />
      </div>
      <KpiStatGrid columns={3}>
        <KpiStatCard label="Total invoiced" value={`${fmt(ov.total_invoiced)} UZS`} />
        <KpiStatCard
          label="Paid"
          value={`${fmt(ov.total_paid)} UZS`}
          sub="Settled to date"
        />
        <KpiStatCard
          label="Outstanding"
          value={`${fmt(ov.total_outstanding)} UZS`}
          sub={ov.total_outstanding > 0 ? 'Requires collection' : 'All clear'}
        />
      </KpiStatGrid>

      {!loading && view === 'invoices' && (
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
        <TreasuryTransactionList
          invoices={invoices}
          fmt={fmt}
          resolveAmount={resolveAmount}
          resolveCurrency={resolveCurrency}
          formatPayoutOwner={formatPayoutOwner}
        />
        </section>
      )}
      </div>
      </PageChrome>
    </PageTransition>
  );
}
