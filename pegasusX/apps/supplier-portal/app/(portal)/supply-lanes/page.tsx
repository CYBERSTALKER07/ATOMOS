"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useEffect, useState } from "react";
import Link from "next/link";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { SupplyLanesList } from "@/components/supply-lanes";

const api = createSupplierApi();

export default function SupplyLanesPage() {
  const t = usePortalT();
  const [lanes, setLanes] = useState<SupplierSupplyLaneRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierSupplyLanes()
      .then((resp) => setLanes(resp.lanes))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_supply_lanes_failed")))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      icon="fleet"
      title={t("supplier_portal.supply_lanes.text.supply_lanes")}
      description={t("supplier_portal.residual.text.warehouse_lanes_derived_from_topology_and_live_order_volume_lane")}
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_active_warehouse_lanes_configure_nodes_on_topology")}
    >
      <SupplyLanesList lanes={lanes} />
    </PageChrome>
  );
}
