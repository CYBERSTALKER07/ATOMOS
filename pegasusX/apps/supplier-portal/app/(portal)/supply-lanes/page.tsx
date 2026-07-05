"use client";

import React, { useEffect, useState } from "react";
import Link from "next/link";
import { createSupplierApi } from "@/lib/api";
import type { SupplierSupplyLaneRow } from "@pegasusx/types";
import { PageChrome } from "@/components/PageChrome";

const api = createSupplierApi();

export default function SupplyLanesPage() {
  const [lanes, setLanes] = useState<SupplierSupplyLaneRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api
      .getSupplierSupplyLanes()
      .then((resp) => setLanes(resp.lanes))
      .catch((err) => setError(err instanceof Error ? err.message : "load_supply_lanes_failed"))
      .finally(() => setLoading(false));
  }, []);

  return (
    <PageChrome
      icon="fleet"
      title="Supply lanes"
      description="Warehouse lanes derived from topology and live order volume. Lanes are read-only — edit warehouses and coverage on Topology."
      loading={loading}
      error={error}
      empty={!loading && lanes.length === 0}
      emptyMessage="No active warehouse lanes. Configure nodes on topology."
    >
      {lanes.length > 0 ? (
        <div className="space-y-4">
          <p className="md-typescale-body-small text-[var(--color-md-outline)]">
            Supply lanes are derived from{" "}
            <Link href="/topology" className="text-[var(--color-md-primary)] underline">
              topology
            </Link>{" "}
            (warehouse nodes, coverage radius, co-location). There is no separate lane CRUD — update topology to change lane geometry and capacity signals.
          </p>
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
          {lanes.map((lane) => (
            <div
              key={lane.lane_id}
              className="md-card overflow-hidden hover:border-[var(--color-md-primary)] transition-colors cursor-pointer group"
            >
              <div className="p-5">
                <div className="flex justify-between items-start mb-4">
                  <div>
                    <h3 className="md-typescale-title-large font-light group-hover:text-[var(--color-md-primary)] transition-colors">
                      {lane.name}
                    </h3>
                    <div className="flex items-center gap-2 mt-1 text-[var(--color-md-outline)] text-sm">
                      <span>{lane.warehouse_id}</span>
                    </div>
                  </div>
                  <div className="text-right">
                    <div className="md-typescale-title-medium text-[var(--color-md-primary)]">
                      {lane.h3_cells} Cells
                    </div>
                    <div className="text-xs text-[var(--color-md-outline)] uppercase tracking-wider mt-1">
                      H3 coverage estimate
                    </div>
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-4 py-4 border-t border-[var(--color-md-outline-variant)]">
                  <div>
                    <div className="text-sm text-[var(--color-md-outline)] mb-1">Active Drivers</div>
                    <div className="md-typescale-title-medium font-medium">{lane.drivers}</div>
                  </div>
                  <div>
                    <div className="text-sm text-[var(--color-md-outline)] mb-1">Orders Today</div>
                    <div className="md-typescale-title-medium font-medium">{lane.orders_today}</div>
                  </div>
                  <div>
                    <div className="text-sm text-[var(--color-md-outline)] mb-1">Capacity limit</div>
                    <div className="md-typescale-title-medium font-medium">{lane.capacity}</div>
                  </div>
                </div>

                <div className="mt-2">
                  <div className="flex justify-between text-xs mb-1">
                    <span className="text-[var(--color-md-outline)]">Lane Utilization</span>
                    <span
                      className={`font-medium ${
                        lane.utilization_pct > 85 ? "text-[var(--color-md-error)]" : "text-[var(--color-md-on-surface)]"
                      }`}
                    >
                      {lane.utilization_pct.toFixed(0)}%
                    </span>
                  </div>
                  <div className="h-2 w-full bg-[var(--color-md-surface-container-highest)] rounded-full overflow-hidden">
                    <div
                      className={`h-full transition-all duration-500 ${
                        lane.utilization_pct > 85
                          ? "bg-[var(--color-md-error)]"
                          : lane.utilization_pct > 75
                            ? "bg-[var(--color-md-warning)]"
                            : "bg-[var(--color-md-primary)]"
                      }`}
                      style={{ width: `${Math.min(100, lane.utilization_pct)}%` }}
                    />
                  </div>
                </div>
              </div>
            </div>
          ))}
        </div>
        </div>
      ) : null}
    </PageChrome>
  );
}
