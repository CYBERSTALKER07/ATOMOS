'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';

interface Insight {
  id: string;
  warehouse_id: string;
  warehouse_name: string;
  sku_id: string;
  product_name: string;
  urgency: string;
  current_stock: number;
  daily_velocity: number;
  reorder_qty: number;
  days_to_empty: number;
  lead_time_days: number;
  status: string;
  created_at: string;
}

export default function InsightsPage() {
  const [insights, setInsights] = useState<Insight[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/replenishment/insights');
      if (res.ok) {
        const data = await res.json();
        setInsights(data.insights || []);
      }
    } catch { /* handled */ } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        if (event.type !== 'FACTORY_SUPPLY_REQUEST_UPDATE' && event.type !== 'FACTORY_TRANSFER_UPDATE') {
          return;
        }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  const urgencyClass = (u: string) => {
    if (u === 'CRITICAL') return 'status-chip--critical';
    if (u === 'WARNING') return 'status-chip--warning';
    return 'status-chip--stable';
  };

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Replenishment Insights</h1>
          <motion.button 
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => load()} 
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary hover-lift active-press"
          >
            <Icon name="refresh" size={16} /> Refresh
          </motion.button>
        </div>

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : insights.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-predictions.png"
            headline="No replenishment insights"
            body="No replenishment insights at this time. Insights are generated based on stock velocity."
          />
        ) : (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
          >
            <table className="w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Warehouse</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Product</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Urgency</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Stock</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Velocity/day</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Days Left</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Reorder Qty</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                </tr>
              </thead>
              <tbody>
                {insights.map((ins, index) => (
                  <motion.tr 
                    key={ins.id} 
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.05 }}
                    className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                  >
                    <td className="py-3 px-4 font-medium">{ins.warehouse_name}</td>
                    <td className="py-3 px-4">{ins.product_name}</td>
                    <td className="py-3 px-4">
                      <span className={`status-chip ${urgencyClass(ins.urgency)}`}>{ins.urgency}</span>
                    </td>
                    <td className="py-3 px-4 text-right tabular-nums font-mono">{ins.current_stock}</td>
                    <td className="py-3 px-4 text-right tabular-nums font-mono">{ins.daily_velocity.toFixed(1)}</td>
                    <td className="py-3 px-4 text-right tabular-nums font-mono">{ins.days_to_empty.toFixed(1)}</td>
                    <td className="py-3 px-4 text-right tabular-nums font-mono">{ins.reorder_qty}</td>
                    <td className="py-3 px-4">
                      <span className={`status-chip ${ins.status === 'ACTIVE' ? 'status-chip--approved' : 'status-chip--draft'}`}>
                        {ins.status}
                      </span>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </motion.div>
        )}
      </div>
    </PageTransition>
  );
}
