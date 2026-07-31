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

type InventoryApiItem = {
  product_id: string;
  warehouse_id: string;
  quantity_on_hand?: number;
  quantity_reserved?: number;
  out_of_stock_policy?: string;
  effective_policy?: string;
  accepts_backorder?: boolean;
};

function mapInventoryRow(item: InventoryApiItem): InventoryRow {
  const onHand = item.quantity_on_hand ?? 0;
  const reserved = item.quantity_reserved ?? 0;
  return {
    sku_id: item.product_id,
    warehouse_id: item.warehouse_id,
    product_name: item.product_id,
    quantity: Math.max(0, onHand - reserved),
    unit_price_minor: 0,
    currency: "UZS",
    out_of_stock_policy: item.out_of_stock_policy || "INHERIT",
    effective_policy: item.effective_policy,
    accepts_backorder: item.accepts_backorder,
  };
}

export default function PortalInventoryPage() {
  const [rows, setRows] = useState<InventoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [deltas, setDeltas] = useState<Record<string, string>>({});
  const [adjustingSku, setAdjustingSku] = useState<string | null>(null);
  const [policyUpdatingKey, setPolicyUpdatingKey] = useState<string | null>(null);

  const loadInventory = () => {
    setLoading(true);
    setError(null);
    supplierFetch("/v1/supplier/inventory")
      .then(async (res) => {
        if (!res.ok) throw new Error(`inventory ${res.status}`);
        const body = (await res.json()) as { items?: InventoryApiItem[] };
        setRows((body.items ?? []).map(mapInventoryRow));
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

  const updatePolicy = async (row: InventoryRow, policy: string) => {
    const key = `${row.warehouse_id}:${row.sku_id}`;
    setPolicyUpdatingKey(key);
    setError(null);
    try {
      const res = await supplierFetch("/v1/supplier/inventory/policy", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          warehouse_id: row.warehouse_id,
          product_id: row.sku_id,
          out_of_stock_policy: policy,
        }),
      });
      if (!res.ok) throw new Error(`policy ${res.status}`);
      loadInventory();
    } catch (err) {
      setError(err instanceof Error ? err.message : "policy_update_failed");
    } finally {
      setPolicyUpdatingKey(null);
    }
  };

  const pagination = usePagination(rows, 20);

  return (
    <PageChrome
      icon="inventory"
      title="Inventory"
      description="SKU availability, stock policy per warehouse, and quantity adjustments."
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
            ["warehouse_id", "sku_id", "quantity", "out_of_stock_policy"],
            rows.map((row) => [
              row.warehouse_id,
              row.sku_id,
              String(row.quantity),
              row.out_of_stock_policy ?? "INHERIT",
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
        policyUpdatingKey={policyUpdatingKey}
        onDeltaChange={(skuId, value) => setDeltas((prev) => ({ ...prev, [skuId]: value }))}
        onApply={(row) => void adjustRow(row)}
        onPolicyChange={(row, policy) => void updatePolicy(row, policy)}
      />
    </PageChrome>
  );
}
