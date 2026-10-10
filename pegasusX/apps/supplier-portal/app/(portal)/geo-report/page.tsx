"use client";

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { HubCard } from "@/components/portal";
import { GeoReportLanesList } from "@/components/geo-report";

const api = createSupplierApi();

export default function GeoReportPage() {
  const t = usePortalT();
  const [lanes, setLanes] = useState<SupplierSupplyLaneRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierSupplyLanes()
      .then((resp) => setLanes(resp.lanes))
      .catch((err) => setError(err instanceof Error ? err.message : t("supplier_portal.residual.text.load_geo_report_failed")))
      .finally(() => setLoading(false));
  }, []);


  return (
    <PageChrome
      title={t("supplier_portal.geo_report.text.geo_report")}
      description={t("supplier_portal.residual.text.h3_perimeter_coverage_and_lane_utilization_from_live_supplier_or")}
      icon="hexagon"
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
    >
      <GeoReportLanesList lanes={lanes} />
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
        <HubCard
          href="/topology"
          icon="topology"
          title={t("portal.nav.topology")}
          description={t("supplier_portal.residual.text.manage_warehouse_coordinates_and_coverage_radius")}
        />
        <HubCard
          href="/delivery-zones"
          icon="pin"
          title={t("supplier_portal.delivery_zones.text.delivery_zones")}
          description={t("supplier_portal.residual.text.review_h3_perimeter_and_warehouse_coverage")}
        />
      </div>
    </PageChrome>
  );
}
