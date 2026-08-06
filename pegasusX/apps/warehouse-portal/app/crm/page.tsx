'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';

interface Retailer {
  retailer_id: string;
  business_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date: string;
}

export default function CRMPage() {
  const t = usePortalT();
  const [retailers, setRetailers] = useState<Retailer[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/crm');
      if (res.ok) {
        const data = await res.json();
        setRetailers(data.retailers || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <PageTransition>
      <PageChrome
        icon="crm"
        title={t("warehouse_portal.crm.text.retailer_crm")}
        description={t("warehouse_portal.residual.text.retailer_relationships_order_volume_and_revenue_for_this_warehou")}
        actions={
          <button type="button" onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <div className="space-y-4">
      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 5 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : retailers.length === 0 ? (
        <EmptyState 
          variant="no-data" 
          headline={t("warehouse_portal.residual.text.no_buyer_data_to_analyze_yet")} 
          body={t("warehouse_portal.residual.text.retailer_relationships_and_purchase_histories_will_appear_here_o")} 
        />
      ) : (
        <div className="overflow-x-auto">
          <table className="desk-table w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.crm.text.business_name")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.crm.text.total_orders")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.analytics.text.revenue_uzs")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.crm.text.last_order")}</th>
              </tr>
            </thead>
            <tbody>
              {retailers.map(r => (
                <tr key={r.retailer_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-medium">{r.business_name || '—'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_orders)}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_revenue)}</td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {r.last_order_date ? new Date(r.last_order_date).toLocaleDateString() : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
        </div>
      </PageChrome>
    </PageTransition>
  );
}
