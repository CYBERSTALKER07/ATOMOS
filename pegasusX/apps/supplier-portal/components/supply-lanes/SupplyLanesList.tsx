"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import Link from 'next/link';
import type { SupplierSupplyLaneRow } from '@pegasusx/types';

interface SupplyLanesListProps {
  lanes: SupplierSupplyLaneRow[];
}

export function SupplyLanesList({ lanes }: SupplyLanesListProps) {
  const t = usePortalT();
  if (lanes.length === 0) return null;

  return (
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
                  <div className="text-sm text-[var(--color-md-outline)] mb-1">{t("supplier_portal.supply_lanes.supply_lanes_list.text.active_drivers")}</div>
                  <div className="md-typescale-title-medium font-medium">{lane.drivers}</div>
                </div>
                <div>
                  <div className="text-sm text-[var(--color-md-outline)] mb-1">{t("supplier_portal.supply_lanes.supply_lanes_list.text.orders_today")}</div>
                  <div className="md-typescale-title-medium font-medium">{lane.orders_today}</div>
                </div>
                <div>
                  <div className="text-sm text-[var(--color-md-outline)] mb-1">{t("supplier_portal.supply_lanes.supply_lanes_list.text.capacity_limit")}</div>
                  <div className="md-typescale-title-medium font-medium">{lane.capacity}</div>
                </div>
              </div>

              <div className="mt-2">
                <div className="flex justify-between text-xs mb-1">
                  <span className="text-[var(--color-md-outline)]">{t("supplier_portal.supply_lanes.supply_lanes_list.text.lane_utilization")}</span>
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
  );
}
