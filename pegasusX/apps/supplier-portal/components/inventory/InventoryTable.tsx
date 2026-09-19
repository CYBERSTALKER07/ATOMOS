"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';

export type InventoryRow = {
  sku_id: string;
  warehouse_id: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  currency: string;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
};

interface InventoryTableProps {
  items: InventoryRow[];
  deltas: Record<string, string>;
  adjustingSku: string | null;
  policyUpdatingKey: string | null;
  onDeltaChange: (skuId: string, value: string) => void;
  onApply: (row: InventoryRow) => void;
  onPolicyChange: (row: InventoryRow, policy: string) => void;
}

function rowKey(row: InventoryRow) {
  return `${row.warehouse_id}:${row.sku_id}`;
}

export function InventoryTable({
  items,
  deltas,
  adjustingSku,
  policyUpdatingKey,
  onDeltaChange,
  onApply,
  onPolicyChange,
}: InventoryTableProps) {
  const t = usePortalT();
  return (
    <div className="md-card overflow-x-auto">
      <table className="min-w-full text-left">
        <thead className="border-b border-[var(--color-md-outline-variant)]">
          <tr>
            <th className="px-4 py-3 md-typescale-label-large">{t("supplier_portal.replenishment_traceability_panel.text.warehouse")}</th>
            <th className="px-4 py-3 md-typescale-label-large">SKU</th>
            <th className="px-4 py-3 md-typescale-label-large text-right">{t("supplier_portal.credit.collections.text.available")}</th>
            <th className="px-4 py-3 md-typescale-label-large">{t("supplier_portal.inventory.inventory_table.text.stock_policy")}</th>
            <th className="px-4 py-3 md-typescale-label-large text-right">{t("supplier_portal.inventory.inventory_table.text.adjust")}</th>
          </tr>
        </thead>
        <tbody>
          {items.map((row) => {
            const key = rowKey(row);
            return (
              <tr key={key} className="border-b border-[var(--color-md-outline-variant)]">
                <td className="px-4 py-3 font-mono text-xs">{row.warehouse_id}</td>
                <td className="px-4 py-3 font-mono text-sm">{row.sku_id}</td>
                <td className="px-4 py-3 text-right">{row.quantity}</td>
                <td className="px-4 py-3">
                  <select
                    className="md-input-outlined px-2 py-1 text-sm"
                    value={row.out_of_stock_policy || 'INHERIT'}
                    disabled={policyUpdatingKey === key}
                    onChange={(e) => onPolicyChange(row, e.target.value)}
                  >
                    <option value="INHERIT">{t("supplier_portal.inventory.inventory_table.text.inherit_warehouse")}</option>
                    <option value="REJECT">{t("supplier_portal.inventory.inventory_table.text.reject_when_short")}</option>
                    <option value="ACCEPT_BACKORDER">{t("supplier_portal.inventory.inventory_table.text.accept_backorder")}</option>
                  </select>
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="flex items-center justify-end gap-2">
                    <input
                      type="number"
                      className="md-input-outlined w-24 px-2 py-1 text-right"
                      placeholder={t("supplier_portal.inventory.inventory_table.text.qty")}
                      value={deltas[row.sku_id] ?? ""}
                      onChange={(e: React.ChangeEvent<HTMLInputElement>) => onDeltaChange(row.sku_id, e.target.value)}
                    />
                    <button
                      type="button"
                      className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1"
                      disabled={adjustingSku === row.sku_id}
                      onClick={() => onApply(row)}
                    >
                      Apply
                    </button>
                  </div>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
