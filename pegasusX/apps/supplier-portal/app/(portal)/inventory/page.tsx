"use client";

import { useEffect, useState } from "react";
import { supplierFetch } from "@/lib/auth";
import { supplierScopeId } from "@/lib/supplier-scope";
import { supplierInventoryAdjustKey } from "@pegasusx/api-client";
import { downloadCsv } from "@/lib/csv";
import { usePagination } from "@/lib/use-pagination";
import { ListToolbar } from "@/components/ListToolbar";
import { PageChrome } from "@/components/PageChrome";
import { InventoryTable, type InventoryRow } from "@/components/inventory";

export default function PortalInventoryPage() {
  const [rows, setRows] = useState<InventoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deltas, setDeltas] = useState<Record<string, string>>({});
  const [adjustingSku, setAdjustingSku] = useState<string | null>(null);

  const loadInventory = () => {
    setLoading(true);
    setError(null);
    supplierFetch("/v1/supplier/inventory")
      .then(async (res) => {
        if (!res.ok) throw new Error(`inventory ${res.status}`);
        const body = (await res.json()) as { items?: InventoryRow[] };
        setRows(body.items ?? []);
      })
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load inventory"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadInventory();
  }, []);

  const adjustRow = async (row: InventoryRow) => {
    const raw = deltas[row.sku_id]?.trim() ?? "";
    const quantityDelta = Number.parseInt(raw, 10);
    if (!Number.isFinite(quantityDelta) || quantityDelta === 0) {
      setError("Enter a non-zero quantity delta");
      return;
    }
    setAdjustingSku(row.sku_id);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/inventory", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": supplierInventoryAdjustKey(
            supplierScopeId(),
            row.sku_id,
            quantityDelta,
            row.quantity,
          ),
        },
        body: JSON.stringify({
          sku_id: row.sku_id,
          quantity_delta: quantityDelta,
          quantity: row.quantity,
          reason: "portal_adjust",
        }),
      });
      if (!res.ok) throw new Error(`adjust ${res.status}`);
      setDeltas((prev) => ({ ...prev, [row.sku_id]: "" }));
      loadInventory();
    } catch (err) {
      setError(err instanceof Error ? err.message : "adjust_failed");
    } finally {
      setAdjustingSku(null);
    }
  };

  const pagination = usePagination(rows, 20);

  return (
    <PageChrome
      icon="inventory"
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
      <p className="mb-4 md-typescale-body-medium">
        <a href="/inventory/import" className="text-[var(--color-md-primary)] underline">
          Import CSV
        </a>
      </p>
      <InventoryTable
        items={pagination.pageItems}
        deltas={deltas}
        adjustingSku={adjustingSku}
        onDeltaChange={(skuId, value) => setDeltas((prev) => ({ ...prev, [skuId]: value }))}
        onApply={(row) => void adjustRow(row)}
      />
    </PageChrome>
  );
}
