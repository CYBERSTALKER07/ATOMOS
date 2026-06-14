"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyWarehouse } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function WarehousesPage() {
  const [warehouses, setWarehouses] = useState<SupplierTopologyWarehouse[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierTopology()
      .then((t) => setWarehouses(t.warehouses))
      .catch((err) => setError(err instanceof Error ? err.message : "load_warehouses_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Warehouses"
      description="Distribution nodes and coverage for retailer serviceability."
      loading={loading}
      error={error}
      empty={warehouses.length === 0}
      emptyMessage="No warehouses configured. Add nodes on the topology surface."
    >
      <div className="mb-4">
        <Link href="/topology" className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2 inline-flex">
          Edit topology
        </Link>
      </div>
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {warehouses.map((w) => (
          <li key={w.warehouse_id || w.name} className="p-4 md-typescale-body-medium">
            <div className="font-medium">{w.name}</div>
            <div className="text-[var(--color-md-outline)] text-sm mt-1">
              Radius {w.coverage_radius_km} km · {w.is_on_shift ? "On shift" : "Off shift"} ·{" "}
              {w.is_active ? "Active" : "Inactive"}
            </div>
          </li>
        ))}
      </ul>
    </PortalSurface>
  );
}
