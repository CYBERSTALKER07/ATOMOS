'use client';

import { usePortalT } from "@/lib/i18n";
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
  const t = usePortalT();
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
        title={t("warehouse_portal.replenishment.text.replenishment_insights")}
        description={t("warehouse_portal.residual.text.stock_velocity_alerts_with_approve_dismiss_actions_for_this_ware")}
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
          <KpiStatCard label={t("warehouse_portal.residual.text.open_insights")} value={openInsights.length} sub="Awaiting warehouse action" />
          <KpiStatCard
            label={t("warehouse_portal.residual.text.critical_urgency")}
            value={criticalCount}
            sub={criticalCount > 0 ? 'Auto-transfer may apply' : 'No critical SKUs'}
          />
          <KpiStatCard label={t("warehouse_portal.residual.text.warning_high")} value={warningCount} sub="Monitor burn rate" />
        </KpiStatGrid>

        {insights.length === 0 ? (
          <EmptyState
            variant="no-data"
            headline={t("warehouse_portal.residual.text.no_replenishment_insights")}
            body={t("warehouse_portal.residual.text.the_replenishment_engine_will_surface_stock_velocity_alerts_when")}
          />
        ) : (
          <PageSection title={t("warehouse_portal.replenishment.text.insight_queue")} description={t("warehouse_portal.residual.text.approve_to_create_factory_transfer_rows_dismiss_to_clear")} className="mt-6">
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
