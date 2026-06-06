"use client";

import { useEffect, useState } from "react";
import { ApiClient } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function TopologyPage() {
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierTopology()
      .then(setTopology)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load topology"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Factories & warehouses"
      description="Node topology for dispatch, manifests, and serviceability."
      loading={loading}
      error={error}
      empty={!topology}
    >
      {topology ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          <NodeList title="Warehouses" items={topology.warehouses.map((w) => `${w.name} (${w.lat}, ${w.lng})`)} />
          <NodeList title="Factories" items={topology.factories.map((f) => `${f.name} (${f.lat}, ${f.lng})`)} />
        </div>
      ) : null}
    </PortalSurface>
  );
}

function NodeList({ title, items }: { title: string; items: string[] }) {
  return (
    <div className="md-card p-4">
      <h2 className="md-typescale-title-medium mb-3">{title}</h2>
      <ul className="space-y-2 md-typescale-body-medium">
        {items.map((item) => (
          <li key={item} className="border-b border-[var(--color-md-outline-variant)] pb-2">
            {item}
          </li>
        ))}
      </ul>
    </div>
  );
}
