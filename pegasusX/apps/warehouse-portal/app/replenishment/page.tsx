'use client';

import { useCallback, useEffect, useState } from 'react';
import type { WarehouseReplenishmentInsight } from '@pegasusx/types';
import { ApiError, warehouseReplenishmentInsightActionKey } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { ReplenishmentList } from '@/components/replenishment/ReplenishmentList';

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
      const result = await warehouseApi.postWarehouseReplenishmentInsightAction(
        insightId,
        action,
        {},
        warehouseReplenishmentInsightActionKey(insightId, action),
      );
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

  const openInsights = insights.filter((i) => i.status === 'OPEN');
  const criticalCount = insights.filter((i) => i.urgency === 'CRITICAL').length;
  const warningCount = insights.filter((i) => i.urgency === 'WARNING' || i.urgency === 'HIGH').length;

  return (
    <PageTransition>
      <PageChrome
        icon="forecast"
        title="Replenishment insights"
        description="Stock velocity alerts with approve/dismiss actions for this warehouse node."
        loading={loading}
        skeletonVariant="table"
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

        <KpiStatGrid columns={3}>
          <KpiStatCard label="Open insights" value={openInsights.length} sub="Awaiting warehouse action" />
          <KpiStatCard
            label="Critical urgency"
            value={criticalCount}
            sub={criticalCount > 0 ? 'Auto-transfer may apply' : 'No critical SKUs'}
          />
          <KpiStatCard label="Warning / high" value={warningCount} sub="Monitor burn rate" />
        </KpiStatGrid>

        {insights.length === 0 ? (
          <EmptyState
            variant="no-data"
            headline="No replenishment insights"
            body="The replenishment engine will surface stock velocity alerts when burn thresholds are crossed."
          />
        ) : (
          <PageSection title="Insight queue" description="Approve to create factory transfer rows; dismiss to clear." className="mt-6">
            <ReplenishmentList
              insights={insights}
              actingId={actingId}
              onApprove={(id) => void runAction(id, 'approve')}
              onDismiss={(id) => void runAction(id, 'dismiss')}
            />
          </PageSection>
        )}
      </PageChrome>
    </PageTransition>
  );
}
