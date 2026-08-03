import React from 'react';
import type { SupplierSupplyLaneRow } from '@pegasusx/types';
import { DataList, DataListRow } from "@/components/portal";

interface GeoReportLanesListProps {
  lanes: SupplierSupplyLaneRow[];
}

export function GeoReportLanesList({ lanes }: GeoReportLanesListProps) {
  const totalCells = lanes.reduce((sum, lane) => sum + lane.h3_cells, 0);

  return (
    <>
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
    </>
  );
}
