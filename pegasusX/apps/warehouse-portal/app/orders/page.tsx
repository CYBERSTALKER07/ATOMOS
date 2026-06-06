'use client';

import { useEffect, useState, useCallback } from 'react';
import { warehouseApi } from '@/lib/warehouse-api';
import { downloadCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import Icon from '@/components/Icon';
import Link from 'next/link';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { ListToolbar } from '@/components/ListToolbar';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';

interface Order {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at: string;
}

const STATE_CLASSES: Record<string, string> = {
  PENDING: 'status-chip--draft',
  LOADED: 'status-chip--ready',
  IN_TRANSIT: 'status-chip--active',
  ARRIVED: 'status-chip--ready',
  COMPLETED: 'status-chip--stable',
  CANCELLED: 'status-chip--critical',
};

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');

  const load = useCallback(async () => {
    try {
      const data = await warehouseApi.getWarehouseOrders(
        filter ? { state: filter } : {},
      );
      setOrders(
        (data.orders || []).map((row) => {
          const portalRow = row as unknown as {
            order_id: string;
            retailer_name?: string;
            state?: string;
            status?: string;
            total_uzs?: number;
            total_minor?: number;
            created_at?: string;
            updated_at?: string;
          };
          return {
            order_id: portalRow.order_id,
            retailer_name: portalRow.retailer_name ?? '',
            state: portalRow.state ?? portalRow.status ?? '',
            total_uzs: portalRow.total_uzs ?? portalRow.total_minor ?? 0,
            created_at: portalRow.created_at ?? portalRow.updated_at ?? '',
          };
        }),
      );
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  const { page, pageCount, pageItems, next, prev, reset } = usePagination(orders, 25);

  useEffect(() => {
    reset();
  }, [filter, reset]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  const exportCsv = () => {
    downloadCsv(
      `warehouse-orders${filter ? `-${filter.toLowerCase()}` : ''}.csv`,
      ['order_id', 'retailer_name', 'state', 'total_uzs', 'created_at'],
      orders.map((order) => [
        order.order_id,
        order.retailer_name ?? '',
        order.state,
        String(order.total_uzs),
        order.created_at,
      ]),
    );
  };

  return (
    <PageTransition>
      <PageChrome
        title="Orders"
        description="Warehouse-scoped order queue with state filters and CSV export."
        actions={
          <div className="flex gap-2 items-center">
            <select
              value={filter}
              onChange={(e) => {
                setFilter(e.target.value);
                setLoading(true);
              }}
              className="px-3 py-1.5 rounded-lg border text-sm"
              style={{
                background: 'var(--field-background)',
                borderColor: 'var(--field-border)',
                color: 'var(--field-foreground)',
              }}
            >
              <option value="">All States</option>
              {['PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED', 'COMPLETED', 'CANCELLED'].map((s) => (
                <option key={s} value={s}>{s}</option>
              ))}
            </select>
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => {
                setLoading(true);
                load();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary active-press"
            >
              <Icon name="refresh" size={16} /> Refresh
            </motion.button>
          </div>
        }
      >
        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : orders.length === 0 ? (
          <EmptyState
            variant={filter ? 'no-results' : 'no-data'}
            headline="No orders found"
            body={filter ? `No orders found with state "${filter}".` : "There are no orders recorded in this warehouse yet."}
          />
        ) : (
          <>
          <ListToolbar
            page={page}
            pageCount={pageCount}
            totalLabel={`${orders.length} orders`}
            onPrev={prev}
            onNext={next}
            onExport={exportCsv}
          />
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
          >
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Order ID</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Retailer</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">State</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Total (UZS)</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Created</th>
                </tr>
              </thead>
              <tbody>
                {pageItems.map((o, index) => (
                  <motion.tr 
                    key={o.order_id} 
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.03 }}
                    className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                  >
                    <td className="py-3 px-4 font-mono text-xs">
                      <Link href={`/orders/${o.order_id}`} className="hover:underline text-[var(--primary)]">
                        {o.order_id.slice(0, 8)}...
                      </Link>
                    </td>
                    <td className="py-3 px-4 font-medium">{o.retailer_name || '—'}</td>
                    <td className="py-3 px-4">
                      <span className={`status-chip ${STATE_CLASSES[o.state] || ''}`}>{o.state}</span>
                    </td>
                    <td className="py-3 px-4 text-right font-mono tabular-nums">{fmt(o.total_uzs)}</td>
                    <td className="py-3 px-4 text-right text-[var(--muted)] font-mono text-xs tabular-nums">
                      {new Date(o.created_at).toLocaleDateString()}
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </motion.div>
          </>
        )}
      </PageChrome>
    </PageTransition>
  );
}
