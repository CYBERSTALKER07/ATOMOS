'use client';

import { useCallback, useEffect, useState } from 'react';
import { warehouseApi } from '@/lib/warehouse-api';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';

interface PreorderRow {
  order_id: string;
  status: string;
  order_source?: string;
  confirmation_status?: string;
  requested_delivery_date?: string;
  preorder_badge?: string;
  total_minor?: number;
}

export default function PreordersPage() {
  const [items, setItems] = useState<PreorderRow[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePreorders();
      const rows = (data.preorders ?? data.items ?? []) as PreorderRow[];
      setItems(rows);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <PageTransition>
      <PageChrome title="Pre-orders" subtitle="Scheduled and auto-accepted manual pre-orders">
        {loading ? (
          <p className="text-sm text-[var(--desk-text-tertiary)]">Loading…</p>
        ) : items.length === 0 ? (
          <EmptyState title="No pre-orders" message="Scheduled pre-orders appear here for T-2 edits and stock planning." />
        ) : (
          <div className="overflow-x-auto rounded-2xl border border-[var(--desk-border)]">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--desk-border)] text-left text-[var(--desk-text-tertiary)]">
                  <th className="p-3">Order</th>
                  <th className="p-3">Status</th>
                  <th className="p-3">Delivery</th>
                  <th className="p-3">Badge</th>
                </tr>
              </thead>
              <tbody>
                {items.map((row) => (
                  <tr key={row.order_id} className="border-b border-[var(--desk-border)] last:border-0">
                    <td className="p-3 font-mono">{row.order_id}</td>
                    <td className="p-3">{row.status}</td>
                    <td className="p-3">{row.requested_delivery_date ?? '—'}</td>
                    <td className="p-3">{row.preorder_badge ?? 'Pre-order'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
