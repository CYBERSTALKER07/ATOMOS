"use client";

import { useEffect, useState } from "react";
import { ApiClient } from "@pegasusx/api-client";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyFactory } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function FactoriesPage() {
  const [factories, setFactories] = useState<SupplierTopologyFactory[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierTopology()
      .then((t) => setFactories(t.factories))
      .catch((err) => setError(err instanceof Error ? err.message : "load_factories_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PortalSurface
      title="Factories"
      description="Production nodes for manifests and warehouse replenishment."
      loading={loading}
      error={error}
      empty={factories.length === 0}
      emptyMessage="No factories configured. Add nodes on the topology surface."
    >
      <ul className="md-card divide-y divide-[var(--color-md-outline-variant)]">
        {factories.map((f) => (
          <li key={f.factory_id || f.name} className="p-4 md-typescale-body-medium">
            <div className="font-medium">{f.name}</div>
            <div className="text-[var(--color-md-outline)] text-sm mt-1">
              {f.lat.toFixed(4)}, {f.lng.toFixed(4)} · {f.is_active ? "Active" : "Inactive"}
            </div>
          </li>
        ))}
      </ul>
    </PortalSurface>
  );
}
