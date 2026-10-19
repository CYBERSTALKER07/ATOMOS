"use client";

import Link from "next/link";
import { usePortalT } from "@/lib/i18n";
import { coverageModeLabel } from "@/lib/coverage";
import type { SupplierTopologyWarehouse } from "@pegasusx/types";

interface WarehouseListProps {
  warehouses: SupplierTopologyWarehouse[];
  modes?: Record<string, string>;
  onAddFirst: () => void;
}

export function WarehouseList({ warehouses, modes = {}, onAddFirst }: WarehouseListProps) {
  const t = usePortalT();
  if (warehouses.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-16 gap-4">
        <p className="md-typescale-body-medium text-[var(--color-md-outline)]">{t("supplier_portal.warehouses.components.warehouse_list.text.add_your_first_warehouse_to_start_fulfilling_orders")}</p>
        <button type="button" className="md-btn md-btn-filled px-6 py-3" onClick={onAddFirst}>
          Add first warehouse
        </button>
      </div>
    );
  }

  return (
    <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
      {warehouses.map((w) => (
        <li key={w.warehouse_id || w.name} className="p-4 md-typescale-body-medium flex items-start justify-between gap-4">
          <div>
            <div className="font-medium">{w.name}</div>
            <div className="text-[var(--color-md-outline)] text-sm mt-1">
              {(w.address || "Coordinates on file").toString()} · {coverageModeLabel(modes[w.warehouse_id])} ·{" "}
              {w.is_on_shift ? "On shift" : "Off shift"} · {w.is_active ? "Active" : "Inactive"}
            </div>
          </div>
          {w.warehouse_id ? (
            <Link href={`/warehouses/${w.warehouse_id}/coverage` as any} className="md-btn md-btn-outlined px-3 py-1 text-sm shrink-0">
              Coverage & pins
            </Link>
          ) : null}
        </li>
      ))}
    </ul>
  );
}
