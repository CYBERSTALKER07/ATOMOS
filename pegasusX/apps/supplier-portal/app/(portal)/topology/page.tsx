"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { TopologyEditor } from "@/components/TopologyEditor";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function TopologyPage() {
  const t = usePortalT();
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const loadTopology = () => {
    setLoading(true);
    setError(null);
    api
      .getSupplierTopology()
      .then(setTopology)
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.failed_to_load_topology")))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    loadTopology();
  }, []);

  return (
    <PageChrome
      icon="topology"
      title={t("supplier_portal.topology.text.factories_and_warehouses")}
      description={t("supplier_portal.residual.text.create_and_edit_warehouse_and_factory_nodes_coverage_radius_and_")}
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
    </PageChrome>
  );
}
