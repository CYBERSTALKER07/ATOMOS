"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function DeliveryZonesPage() {
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierTopology()
      .then(setTopology)
      .catch((err) => setError(err instanceof Error ? err.message : "load_topology_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Delivery zones"
      description="H3 perimeter and warehouse coverage configured via topology."
      loading={loading}
      error={error}
      empty={!topology || topology.warehouses.length === 0}
      emptyMessage="No warehouse coverage configured."
    >
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {topology?.warehouses.map((warehouse) => (
          <li key={warehouse.warehouse_id || warehouse.name} className="p-4 md-typescale-body-medium">
            <div className="font-medium">{warehouse.name}</div>
            <div className="text-[var(--color-md-outline)] text-sm mt-1">
              Radius {warehouse.coverage_radius_km} km · {warehouse.lat.toFixed(4)}, {warehouse.lng.toFixed(4)} ·{" "}
              {warehouse.is_active ? "Active" : "Inactive"}
            </div>
          </li>
        ))}
      </ul>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Edit coverage on{" "}
        <Link href="/topology" className="text-[var(--color-md-primary)] underline">
          topology
        </Link>
        .
      </p>
    </PortalSurface>
  );
}
