'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { HubCard } from '@/components/portal';
import EmptyState from '@/components/EmptyState';

interface TreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
}

import { TreasuryTransactionList } from '@/components/treasury/TreasuryTransactionList';

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
        <TreasuryTransactionList
          invoices={invoices}
          fmt={fmt}
          resolveAmount={resolveAmount}
          resolveCurrency={resolveCurrency}
          formatPayoutOwner={formatPayoutOwner}
        />
      )}
      </div>
      </PageChrome>
    </PageTransition>
  );
}
