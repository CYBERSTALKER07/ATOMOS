"use client";

import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";
import { DataList, DataListRow, HubCard } from "@/components/portal";

const api = createSupplierApi();

export default function GeoReportPage() {
  const [lanes, setLanes] = useState<SupplierSupplyLaneRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierSupplyLanes()
      .then((resp) => setLanes(resp.lanes))
      .catch((err) => setError(err instanceof Error ? err.message : "load_geo_report_failed"))
      .finally(() => setLoading(false));
  }, []);

  const totalCells = lanes.reduce((sum, lane) => sum + lane.h3_cells, 0);

  return (
    <PageChrome
      title="Geo report"
      description="H3 perimeter coverage and lane utilization from live supplier orders."
      icon="hexagon"
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
    >
      <p className="md-typescale-body-medium mb-4">
        Estimated H3 cells in service: <strong>{totalCells}</strong>
      </p>
      <DataList>
        {lanes.map((lane) => (
          <DataListRow key={lane.lane_id}>
            <div className="min-w-0 md-typescale-body-medium">
              <div className="font-medium">{lane.name}</div>
              <div className="text-[var(--color-md-outline)] text-sm mt-1">
                {lane.h3_cells} cells · {lane.utilization_pct.toFixed(0)}% utilization today
              </div>
            </div>
          </DataListRow>
        ))}
      </DataList>
      <div className="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
        <HubCard
          href="/topology"
          icon="topology"
          title="Topology"
          description="Manage warehouse coordinates and coverage radius."
        />
        <HubCard
          href="/delivery-zones"
          icon="pin"
          title="Delivery zones"
          description="Review H3 perimeter and warehouse coverage."
        />
      </div>
    </PageChrome>
  );
}
