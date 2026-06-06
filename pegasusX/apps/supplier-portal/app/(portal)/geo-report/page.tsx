"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PortalSurface } from "../_components/PortalSurface";

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
    <PortalSurface
      title="Geo report"
      description="H3 perimeter coverage and lane utilization from live supplier orders."
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
    >
      <div className="md-card p-6 space-y-4 md-typescale-body-medium">
        <p>
          Estimated H3 cells in service: <strong>{totalCells}</strong>
        </p>
        <ul className="space-y-2">
          {lanes.map((lane) => (
            <li key={lane.lane_id} className="border-b border-[var(--color-md-outline-variant)] pb-2">
              {lane.name} — {lane.h3_cells} cells · {lane.utilization_pct.toFixed(0)}% utilization today
            </li>
          ))}
        </ul>
        <p className="text-[var(--color-md-outline)]">
          Manage warehouse coordinates on{" "}
          <Link href="/topology" className="text-[var(--color-md-primary)] underline">
            topology
          </Link>{" "}
          or review zones on{" "}
          <Link href="/delivery-zones" className="text-[var(--color-md-primary)] underline">
            delivery zones
          </Link>
          .
        </p>
      </div>
    </PortalSurface>
  );
}
