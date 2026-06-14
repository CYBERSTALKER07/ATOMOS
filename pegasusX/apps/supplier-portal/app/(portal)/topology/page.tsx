"use client";

import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { TopologyEditor } from "@/components/TopologyEditor";
import { PortalSurface } from "../_components/PortalSurface";

const api = createSupplierApi();

export default function TopologyPage() {
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTopology = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierTopology()
      .then(setTopology)
      .catch((err) => setError(err instanceof Error ? err.message : "Failed to load topology"))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadTopology();
  }, []);

  return (
    <PortalSurface
      title="Factories & warehouses"
      description="Create and edit warehouse and factory nodes. Coverage radius and co-location drive delivery zones and supply lanes."
      loading={loading}
      error={error}
      empty={!topology}
    >
      {topology ? (
        <TopologyEditor
          key={topology.updated_at}
          initial={topology}
          onSaved={(updated) => setTopology(updated)}
        />
      ) : null}
    </PortalSurface>
  );
}
