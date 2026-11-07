"use client";

import { usePortalT } from "@/lib/i18n";
import type { WarehouseOrderDetail } from '@pegasusx/types';

export interface OrderLineItemsProps {
  order: WarehouseOrderDetail;
}

export function OrderLineItems({ order }: OrderLineItemsProps) {
  const t = usePortalT();
  if (!order.line_items || order.line_items.length === 0) {
    return null;
  }

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <section className="wh-bay-panel wh-bay--inventory wh-order-bento-lines">
      <div className="wh-section-head">
        <div>
          <h2 className="wh-section-title">{t("warehouse_portal.orders.order_line_items.text.line_items")}</h2>
          <p className="wh-section-desc">{order.line_items.length} products in this order.</p>
        </div>
      </div>
      <div className="desk-table-wrap">
        <table className="desk-table">
          <thead>
            <tr>
              <th>{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
              <th className="text-right">{t("warehouse_portal.pick_waves.text.qty")}</th>
              <th className="text-right">{t("warehouse_portal.orders.order_line_items.text.unit_uzs")}</th>
            </tr>
          </thead>
          <tbody>
            {order.line_items.map((item, idx) => (
              <tr key={`${item.product_id ?? idx}`}>
                <td>{item.product_name || item.product_id || '—'}</td>
                <td className="text-right font-mono tabular-nums">{item.quantity ?? '—'}</td>
                <td className="text-right font-mono tabular-nums">
                  {item.unit_price != null ? fmt(item.unit_price) : '—'}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}
