"use client";

import { usePortalT } from "@/lib/i18n";
import type { SupplierTopologyWarehouse } from "@pegasusx/types";

interface WarehouseListProps {
  warehouses: SupplierTopologyWarehouse[];
  onAddFirst: () => void;
}

export function WarehouseList({ warehouses, onAddFirst }: WarehouseListProps) {
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
        <li key={w.warehouse_id || w.name} className="p-4 md-typescale-body-medium">
          <div className="font-medium">{w.name}</div>
          <div className="text-[var(--color-md-outline)] text-sm mt-1">
            {(w.address || "Coordinates on file").toString()} · Radius {w.coverage_radius_km} km · {w.is_on_shift ? "On shift" : "Off shift"} ·{" "}
            {w.is_active ? "Active" : "Inactive"}
          </div>
        </li>
      ))}
    </ul>
  );
}
