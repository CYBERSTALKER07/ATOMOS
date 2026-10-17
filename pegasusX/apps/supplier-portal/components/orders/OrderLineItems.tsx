"use client";

import { usePortalT } from "@/lib/i18n";
import type { WarehouseOrderDetail } from '@pegasusx/types';

interface OrderLineItemsProps {
  detail: WarehouseOrderDetail | null;
}

export function OrderLineItems({ detail }: OrderLineItemsProps) {
  const t = usePortalT();
  if (!detail || (detail.line_items?.length ?? 0) === 0) return null;

  return (
    <div className="md-card p-0 overflow-hidden">
      <div className="px-5 py-3 border-b border-[var(--color-md-outline-variant)] text-sm font-semibold uppercase tracking-wider text-[var(--color-md-outline)]">
        Line items
      </div>
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-[var(--color-md-outline-variant)]">
            <th className="text-left py-2 px-4">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
            <th className="text-right py-2 px-4">{t("supplier_portal.analytics.demand.text.qty")}</th>
            <th className="text-right py-2 px-4">{t("supplier_portal.catalog.components.catalog_table.text.unit")}</th>
          </tr>
        </thead>
        <tbody>
          {detail.line_items?.map((item, idx) => (
            <tr key={`${item.product_id ?? idx}`} className="border-b border-[var(--color-md-outline-variant)] last:border-0">
              <td className="py-2 px-4">{item.product_name || item.product_id || '—'}</td>
              <td className="py-2 px-4 text-right font-mono tabular-nums">{item.quantity ?? '—'}</td>
              <td className="py-2 px-4 text-right font-mono tabular-nums">
                {item.unit_price != null ? new Intl.NumberFormat('uz-UZ').format(item.unit_price) : '—'}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
