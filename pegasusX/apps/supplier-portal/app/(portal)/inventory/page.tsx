"use client";

import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import { PortalSurface } from "../_components/PortalSurface";

type InventoryRow = {
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  currency: string;
};

export default function PortalInventoryPage() {
  const [rows, setRows] = useState<InventoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    supplierFetch("/v1/supplier/inventory")
      .then(async (res) => {
        if (!res.ok) throw new Error(`inventory ${res.status}`);
        const body = (await res.json()) as { items?: InventoryRow[] };
        setRows(body.items ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load inventory"))
      .finally(() => setLoading(false));
  }, []);

  const pagination = usePagination(rows, 20);

  return (
    <PortalSurface
      title="Inventory"
      description="SKU availability and audit trail for supplier operations."
      loading={loading}
      error={error}
      empty={rows.length === 0}
    >
      <ListToolbar
        page={pagination.page}
        pageCount={pagination.pageCount}
        totalLabel={`${rows.length} SKUs`}
        onPrev={pagination.prev}
        onNext={pagination.next}
        onExport={() =>
          downloadCsv(
            "supplier-inventory.csv",
            ["sku_id", "product_name", "quantity", "unit_price_minor", "currency"],
            rows.map((row) => [
              row.sku_id,
              row.product_name,
              String(row.quantity),
              String(row.unit_price_minor),
              row.currency,
            ]),
          )
        }
      />
      <div className="md-card overflow-x-auto">
        <table className="min-w-full text-left">
          <thead className="border-b border-[var(--color-md-outline-variant)]">
            <tr>
              <th className="px-4 py-3 md-typescale-label-large">SKU</th>
              <th className="px-4 py-3 md-typescale-label-large">Product</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Qty</th>
              <th className="px-4 py-3 md-typescale-label-large text-right">Unit (minor)</th>
            </tr>
          </thead>
          <tbody>
            {pagination.pageItems.map((row) => (
              <tr key={row.sku_id} className="border-b border-[var(--color-md-outline-variant)]">
                <td className="px-4 py-3 font-mono text-sm">{row.sku_id}</td>
                <td className="px-4 py-3">{row.product_name}</td>
                <td className="px-4 py-3 text-right">{row.quantity}</td>
                <td className="px-4 py-3 text-right">{row.unit_price_minor}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </PortalSurface>
  );
}
