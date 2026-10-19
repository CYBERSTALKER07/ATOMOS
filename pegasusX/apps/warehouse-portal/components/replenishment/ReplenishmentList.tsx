"use client";

import { usePortalT } from "@/lib/i18n";
import type { WarehouseReplenishmentInsight } from '@pegasusx/types';
import { parseForecastConfidence } from '@/lib/forecast-confidence';
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';

interface ReplenishmentListProps {
  insights: WarehouseReplenishmentInsight[];
  actingId: string | null;
  onApprove: (id: string) => void;
  onDismiss: (id: string) => void;
}

export function ReplenishmentList({ insights, actingId, onApprove, onDismiss }: ReplenishmentListProps) {
  const t = usePortalT();
  const urgencyClass = (urgency: string) => {
    if (urgency === 'PROACTIVE') return 'status-chip--proactive';
    if (urgency === 'CRITICAL') return 'status-chip--critical';
    if (urgency === 'WARNING' || urgency === 'HIGH') return 'status-chip--warning';
    return 'status-chip--stable';
  };

  return (
    <div className="overflow-x-auto -mx-5 px-5">
      <table className="desk-table w-full text-sm">
      <thead>
        <tr className="border-b border-(--border)">
          <th className="px-4 py-3 text-left font-medium">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
          <th className="px-4 py-3 text-left font-medium">{t("warehouse_portal.forecast.forecast_sku_table.text.confidence")}</th>
          <th className="px-4 py-3 text-left font-medium">{t("warehouse_portal.replenishment.replenishment_list.text.why")}</th>
          <th className="px-4 py-3 text-left font-medium">{t("warehouse_portal.replenishment.replenishment_list.text.urgency")}</th>
          <th className="px-4 py-3 text-right font-medium">{t("portal.nav.stock")}</th>
          <th className="px-4 py-3 text-right font-medium">{t("warehouse_portal.replenishment.replenishment_list.text.days_left")}</th>
          <th className="px-4 py-3 text-right font-medium">{t("warehouse_portal.replenishment.replenishment_list.text.reorder_qty")}</th>
          <th className="px-4 py-3 text-left font-medium">{t("warehouse_portal.bins.text.status")}</th>
          <th className="px-4 py-3 text-right font-medium">{t("warehouse_portal.dispatch_locks.text.actions")}</th>
        </tr>
      </thead>
      <tbody>
        {insights.map((insight) => {
          const confidence = parseForecastConfidence(insight.demand_breakdown) ?? undefined;
          return (
          <tr key={insight.id} className="border-b border-(--border) last:border-0">
            <td className="px-4 py-3">
              <div className="flex items-center gap-2">
                <span>{insight.product_name}</span>
                {insight.reason_code === 'PREDICTIVE_PUSH' && (
                  <span className="flex items-center gap-1 text-[10px] font-bold uppercase tracking-wider text-(--desk-surface) bg-(--desk-text-primary) px-1.5 py-0.5 rounded-full">
                    AI Push
                  </span>
                )}
              </div>
            </td>
            <td className="px-4 py-3">
              {confidence ? (
                <ForecastConfidenceView confidence={confidence} compact />
              ) : (
                <span className="text-xs text-(--muted)">—</span>
              )}
            </td>
            <td className="px-4 py-3 text-xs text-(--desk-text-secondary) max-w-[220px]">
              {formatDemandWhy(insight.demand_breakdown, insight.reason_code)}
            </td>
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
                    onClick={() => onApprove(insight.id)}
                    className="button--primary rounded-lg px-3 py-1 text-xs disabled:opacity-50"
                  >
                    Approve
                  </button>
                  <button
                    type="button"
                    disabled={actingId === insight.id}
                    onClick={() => onDismiss(insight.id)}
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
          );
        })}
      </tbody>
    </table>
    </div>
  );
}

export function formatDemandWhy(
  breakdown: WarehouseReplenishmentInsight['demand_breakdown'],
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
