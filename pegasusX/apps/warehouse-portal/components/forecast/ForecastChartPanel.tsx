"use client";

import { usePortalT } from "@/lib/i18n";
import type { ForecastProduct } from '@/app/demand-forecast/page'; // Need to export ForecastProduct

export function ForecastChartPanel({ products }: { products: any[] }) {
  const t = usePortalT();
  const critical = products.filter(p => p.priority === 'CRITICAL');
  const urgent = products.filter(p => p.priority === 'URGENT');
  const normal = products.filter(p => p.priority === 'NORMAL');

  return (
    <div className="grid grid-cols-3 gap-4">
      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
        <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.forecast.forecast_chart_panel.text.critical_items")}</div>
        <div className="text-2xl font-light text-[var(--danger)]">{critical.length}</div>
        <div className="text-xs text-[var(--muted)]">&lt; 2 days to stockout</div>
      </div>
      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
        <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.forecast.forecast_chart_panel.text.urgent_items")}</div>
        <div className="text-2xl font-light text-[var(--warning)]">{urgent.length}</div>
        <div className="text-xs text-[var(--muted)]">&lt; 5 days to stockout</div>
      </div>
      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
        <div className="text-xs text-[var(--muted)] mb-1">{t("warehouse_portal.forecast.forecast_chart_panel.text.healthy_items")}</div>
        <div className="text-2xl font-light text-[var(--success)]">{normal.length}</div>
        <div className="text-xs text-[var(--muted)]">5+ days of stock</div>
      </div>
    </div>
  );
}
