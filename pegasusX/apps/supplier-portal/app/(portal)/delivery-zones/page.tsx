"use client";

import { usePortalT } from "@/lib/i18n";
import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierTopologyResponse } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { DeliveryZonesList } from "@/components/delivery-zones";

const api = createSupplierApi();

export default function DeliveryZonesPage() {
  const t = usePortalT();
  const [topology, setTopology] = useState<SupplierTopologyResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierTopology()
      .then(setTopology)
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_topology_failed")))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      title={t("supplier_portal.delivery_zones.text.delivery_zones")}
      description={t("supplier_portal.residual.text.h3_perimeter_and_warehouse_coverage_configured_via_topology")}
      icon="pin"
      loading={loading}
      error={error}
      empty={!topology || topology.warehouses.length === 0}
      emptyMessage={t("supplier_portal.residual.text.no_warehouse_coverage_configured")}
    >
      <DeliveryZonesList warehouses={topology?.warehouses ?? []} />
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
