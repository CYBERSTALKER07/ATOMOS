'use client';

import { useCallback, useEffect, useState } from 'react';
import type { WarehouseReplenishmentInsight } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

export default function ReplenishmentPage() {
  const [insights, setInsights] = useState<WarehouseReplenishmentInsight[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [actingId, setActingId] = useState<string | null>(null);
  const [actionError, setActionError] = useState<string | null>(null);
  const [actionSuccess, setActionSuccess] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const data = await warehouseApi.getWarehouseReplenishmentInsights();
      setInsights(data.insights || data.data || []);
    } catch (err) {
      setLoadError(err instanceof ApiError ? err.message : 'Failed to load replenishment insights');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const runAction = useCallback(async (insightId: string, action: 'approve' | 'dismiss') => {
    setActingId(insightId);
    setActionError(null);
    setActionSuccess(null);
    try {
      const result = await warehouseApi.postWarehouseReplenishmentInsightAction(insightId, action);
      setActionSuccess(
        action === 'approve'
          ? `Insight approved${result.transfer_id ? ` — transfer ${result.transfer_id.slice(0, 8)}` : ''}.`
          : 'Insight dismissed.',
      );
      setLoading(true);
      await load();
    } catch (err) {
      setActionError(err instanceof ApiError ? err.message : 'Replenishment action failed');
    } finally {
      setActingId(null);
    }
  }, [load]);

  const urgencyClass = (urgency: string) => {
    if (urgency === 'CRITICAL') return 'status-chip--critical';
    if (urgency === 'WARNING' || urgency === 'HIGH') return 'status-chip--warning';
    return 'status-chip--stable';
  };

  return (
    <PageTransition>
      <PageChrome
        title="Replenishment insights"
        description="Stock velocity alerts with approve/dismiss actions for this warehouse node."
        loading={loading}
        error={loadError}
        actions={
          <button
            type="button"
            onClick={() => {
              setLoading(true);
              void load();
            }}
            className="button--secondary flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        {actionError && (
          <p className="mb-4 text-sm" style={{ color: 'var(--error)' }}>{actionError}</p>
        )}
        {actionSuccess && (
          <p className="mb-4 text-sm" style={{ color: 'var(--success)' }}>{actionSuccess}</p>
        )}

        {insights.length === 0 && !loading && !loadError ? (
          <p className="py-8 text-center text-sm text-(--muted)">No replenishment insights at this time.</p>
        ) : (
          <div className="overflow-x-auto rounded-xl border border-(--border)" style={{ background: 'var(--background)' }}>
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="border-b border-(--border)">
                  <th className="px-4 py-3 text-left font-medium">Product</th>
                  <th className="px-4 py-3 text-left font-medium">Urgency</th>
                  <th className="px-4 py-3 text-right font-medium">Stock</th>
                  <th className="px-4 py-3 text-right font-medium">Days Left</th>
                  <th className="px-4 py-3 text-right font-medium">Reorder Qty</th>
                  <th className="px-4 py-3 text-left font-medium">Status</th>
                  <th className="px-4 py-3 text-right font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {insights.map((insight) => (
                  <tr key={insight.id} className="border-b border-(--border) last:border-0">
                    <td className="px-4 py-3">{insight.product_name}</td>
                    <td className="px-4 py-3">
                      <span className={`status-chip ${urgencyClass(insight.urgency)}`}>{insight.urgency}</span>
                    </td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{insight.current_stock}</td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{insight.days_until_stockout}</td>
                    <td className="px-4 py-3 text-right font-mono tabular-nums">{insight.reorder_quantity}</td>
                    <td className="px-4 py-3">
                      <span className={`status-chip ${insight.status === 'OPEN' ? 'status-chip--draft' : 'status-chip--approved'}`}>
                        {insight.status}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right">
                      {insight.status === 'OPEN' ? (
                        <div className="flex justify-end gap-2">
                          <button
                            type="button"
                            disabled={actingId === insight.id}
                            onClick={() => void runAction(insight.id, 'approve')}
                            className="button--primary rounded-lg px-3 py-1 text-xs disabled:opacity-50"
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            disabled={actingId === insight.id}
                            onClick={() => void runAction(insight.id, 'dismiss')}
                            className="button--secondary rounded-lg px-3 py-1 text-xs disabled:opacity-50"
                          >
                            Dismiss
                          </button>
                        </div>
                      ) : (
                        <span className="text-xs text-(--muted)">—</span>
                      )}
                    </td>
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
