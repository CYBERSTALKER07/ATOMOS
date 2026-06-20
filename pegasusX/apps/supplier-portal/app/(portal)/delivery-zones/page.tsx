"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow } from "@/components/portal";

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
    <PageChrome
      title="Delivery zones"
      description="H3 perimeter and warehouse coverage configured via topology."
      icon="pin"
      loading={loading}
      error={error}
      empty={!topology || topology.warehouses.length === 0}
      emptyMessage="No warehouse coverage configured."
    >
      <DataList>
        {topology?.warehouses.map((warehouse) => (
          <DataListRow key={warehouse.warehouse_id || warehouse.name}>
            <div className="min-w-0 md-typescale-body-medium">
              <div className="font-medium">{warehouse.name}</div>
              <div className="text-[var(--color-md-outline)] text-sm mt-1">
                Radius {warehouse.coverage_radius_km} km · {warehouse.lat.toFixed(4)}, {warehouse.lng.toFixed(4)} ·{" "}
                {warehouse.is_active ? "Active" : "Inactive"}
              </div>
            </div>
          </DataListRow>
        ))}
      </DataList>
      <p className="md-typescale-body-medium text-[var(--color-md-outline)]">
        Edit coverage on{" "}
        <Link href="/topology" className="text-[var(--color-md-primary)] underline">
          topology
        </Link>
        .
      </p>
    </PageChrome>
  );
}
