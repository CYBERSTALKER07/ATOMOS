import React from 'react';

export type InventoryRow = {
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  currency: string;
};

interface InventoryTableProps {
  items: InventoryRow[];
  deltas: Record<string, string>;
  adjustingSku: string | null;
  onDeltaChange: (skuId: string, value: string) => void;
  onApply: (row: InventoryRow) => void;
}

export function InventoryTable({
  items,
  deltas,
  adjustingSku,
  onDeltaChange,
  onApply,
}: InventoryTableProps) {
  return (
    <div className="md-card overflow-x-auto">
      <table className="min-w-full text-left">
        <thead className="border-b border-[var(--color-md-outline-variant)]">
          <tr>
            <th className="px-4 py-3 md-typescale-label-large">SKU</th>
            <th className="px-4 py-3 md-typescale-label-large">Product</th>
            <th className="px-4 py-3 md-typescale-label-large text-right">Qty</th>
            <th className="px-4 py-3 md-typescale-label-large text-right">Unit (minor)</th>
            <th className="px-4 py-3 md-typescale-label-large text-right">Adjust</th>
          </tr>
        </thead>
        <tbody>
          {items.map((row) => (
            <tr key={row.sku_id} className="border-b border-[var(--color-md-outline-variant)]">
              <td className="px-4 py-3 font-mono text-sm">{row.sku_id}</td>
              <td className="px-4 py-3">{row.product_name}</td>
              <td className="px-4 py-3 text-right">{row.quantity}</td>
              <td className="px-4 py-3 text-right">{row.unit_price_minor}</td>
              <td className="px-4 py-3 text-right">
                <div className="flex items-center justify-end gap-2">
                  <input
                    type="number"
                    className="md-input-outlined w-24 px-2 py-1 text-right"
                    placeholder="±qty"
                    value={deltas[row.sku_id] ?? ""}
                    onChange={(e) => onDeltaChange(row.sku_id, e.target.value)}
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
          ))}
        </tbody>
      </table>
    </div>
  );
}
