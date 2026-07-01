'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';

interface InsightRow {
  id?: string;
  warehouse_id?: string;
  warehouse_name?: string;
  product_id?: string;
  product_name?: string;
  urgency?: string;
  current_stock?: number;
  avg_daily_velocity?: number;
  daily_velocity?: number;
  reorder_quantity?: number;
  reorder_qty?: number;
  days_until_stockout?: number;
  days_to_empty?: number;
  status?: string;
  created_at?: string;
  reason_code?: string;
  demand_breakdown?: Record<string, unknown> | null;
}

interface Insight {
  id: string;
  warehouse_id: string;
  warehouse_name: string;
  product_name: string;
  urgency: string;
  current_stock: number;
  daily_velocity: number;
  reorder_qty: number;
  days_to_empty: number;
  status: string;
  created_at: string;
  reason_code?: string;
  demand_breakdown?: Record<string, unknown> | null;
}

function normalizeInsight(row: InsightRow): Insight {
  return {
    id: row.id || '',
    warehouse_id: row.warehouse_id || '',
    warehouse_name: row.warehouse_name || '',
    product_name: row.product_name || '',
    urgency: row.urgency || 'NORMAL',
    current_stock: row.current_stock ?? 0,
    daily_velocity: row.avg_daily_velocity ?? row.daily_velocity ?? 0,
    reorder_qty: row.reorder_quantity ?? row.reorder_qty ?? 0,
    days_to_empty: row.days_until_stockout ?? row.days_to_empty ?? 0,
    status: row.status || 'OPEN',
    created_at: row.created_at || '',
    reason_code: row.reason_code,
    demand_breakdown: row.demand_breakdown,
  };
}

function formatDemandWhy(
  breakdown: Insight['demand_breakdown'],
  reasonCode?: string,
): string {
  if (!breakdown || typeof breakdown !== 'object') {
    return reasonCode?.replaceAll('_', ' ') ?? 'Threshold breach';
  }
  const blockedReason = typeof breakdown.blocked_reason === 'string' ? breakdown.blocked_reason : '';
  if (blockedReason) {
    return blockedReason === 'insufficient_history'
      ? 'Insufficient history — forecast blocked'
      : blockedReason.replaceAll('_', ' ');
  }
  const parts: string[] = [];
  const burn = breakdown.burn_rate_7d ?? breakdown.burn_rate;
  if (typeof burn === 'number') parts.push(`Burn ${burn.toFixed(1)}/d`);
  if (typeof breakdown.days_cover === 'number') parts.push(`${breakdown.days_cover.toFixed(1)}d cover`);
  if (typeof breakdown.confidence === 'number') parts.push(`${Math.round(breakdown.confidence * 100)}% conf`);
  if (breakdown.mei_network) parts.push('MEIO network transfer');
  if (typeof breakdown.source_warehouse === 'string') parts.push(`from ${breakdown.source_warehouse.slice(0, 8)}…`);
  if (parts.length === 0 && reasonCode) return reasonCode.replaceAll('_', ' ');
  return parts.join(' · ') || 'Demand signal';
}

export default function InsightsPage() {
  const [insights, setInsights] = useState<Insight[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [actingId, setActingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const res = await apiFetch('/v1/warehouse/replenishment/insights');
      if (res.ok) {
        const data = await res.json();
        const rows: InsightRow[] = data.insights || data.data || [];
        setInsights(rows.map(normalizeInsight));
      } else {
        setError(`Unable to load replenishment insights (${res.status}).`);
      }
    } catch {
      setError('Unable to load replenishment insights right now.');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const runInsightAction = useCallback(async (insightId: string, action: 'approve' | 'dismiss') => {
    setActingId(insightId);
    setActionError(null);
    try {
      const res = await apiFetch(`/v1/warehouse/replenishment/insights/${insightId}/${action}`, {
        method: 'POST',
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setActionError(typeof body.error === 'string' ? body.error : `Action failed (${res.status})`);
        return;
      }
      await load();
    } catch {
      setActionError('Replenishment action failed.');
    } finally {
      setActingId(null);
    }
  }, [load]);

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
      <PageChrome
        icon="insights"
        title="Replenishment insights"
        description="Stock velocity signals and reorder recommendations across connected warehouses."
        loading={loading}
        skeletonVariant="table"
        error={error && insights.length === 0 ? error : null}
        empty={!loading && !error && insights.length === 0}
        emptyMessage="No replenishment insights at this time. Insights are generated based on stock velocity."
        actions={
          <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" onClick={() => void load()}>
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        {actionError && (
          <p className="mb-4 text-sm text-[var(--destructive)]">{actionError}</p>
        )}

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="desk-table-wrap"
        >
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Warehouse</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Product</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Why</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Urgency</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Stock</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Velocity/day</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Days Left</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Reorder Qty</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Actions</th>
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
                  <td className="py-3 px-4 text-xs text-[var(--muted)] max-w-[220px]">
                    {formatDemandWhy(ins.demand_breakdown, ins.reason_code)}
                  </td>
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
                  <td className="py-3 px-4 text-right">
                    {ins.status === 'OPEN' ? (
                      <div className="flex justify-end gap-2">
                        <button
                          type="button"
                          disabled={actingId === ins.id}
                          onClick={() => void runInsightAction(ins.id, 'approve')}
                          className="portal-btn portal-btn--primary text-xs disabled:opacity-50"
                        >
                          Approve
                        </button>
                        <button
                          type="button"
                          disabled={actingId === ins.id}
                          onClick={() => void runInsightAction(ins.id, 'dismiss')}
                          className="portal-btn portal-btn--ghost text-xs disabled:opacity-50"
                        >
                          Dismiss
                        </button>
                      </div>
                    ) : (
                      <span className="text-xs text-[var(--muted)]">—</span>
                    )}
                  </td>
                </motion.tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      </PageChrome>
    </PageTransition>
  );
}
