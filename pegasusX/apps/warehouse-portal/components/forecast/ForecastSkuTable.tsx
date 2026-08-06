"use client";

import { usePortalT } from "@/lib/i18n";
import { ForecastConfidenceView } from '@/components/ForecastConfidenceView';

export function ForecastSkuTable({ products }: { products: any[] }) {
  const t = usePortalT();
  return (
    <div className="border border-[var(--border)] rounded-xl overflow-hidden">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("portal.nav.stock")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.recommended")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.stockout")}</th>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.supply_requests._id_.text.priority")}</th>
            <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.confidence")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.incoming")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.ai_pred")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.pre_orders")}</th>
            <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.forecast.forecast_sku_table.text.burn_day")}</th>
          </tr>
        </thead>
        <tbody>
          {products.map(p => (
            <tr key={p.product_id} className="border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--surface)] transition-colors">
              <td className="px-4 py-3">{p.product_name || p.product_id.slice(0, 8)}</td>
              <td className="px-4 py-3 text-right font-mono">{p.current_stock}</td>
              <td className="px-4 py-3 text-right font-mono font-semibold">{p.recommended_qty}</td>
              <td className="px-4 py-3 text-right">
                <span className={
                  p.days_until_stockout < 2 ? 'text-[var(--danger)] font-semibold' :
                  p.days_until_stockout < 5 ? 'text-[var(--warning)]' : ''
                }>
                  {p.days_until_stockout.toFixed(1)}d
                </span>
              </td>
              <td className="px-4 py-3">
                <span className={`text-xs font-semibold ${
                  p.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
                  p.priority === 'URGENT' ? 'text-[var(--warning)]' : 'text-[var(--muted)]'
                }`}>{p.priority}</span>
              </td>
              <td className="px-4 py-3">
                {p.confidence ? (
                  <ForecastConfidenceView confidence={p.confidence} compact />
                ) : (
                  <span className="text-xs text-[var(--muted)]">—</span>
                )}
              </td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.incoming_orders || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.ai_prediction || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.pre_orders || 0}</td>
              <td className="px-4 py-3 text-right font-mono text-xs">{p.sources?.burn_rate?.toFixed(1) || '0.0'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
