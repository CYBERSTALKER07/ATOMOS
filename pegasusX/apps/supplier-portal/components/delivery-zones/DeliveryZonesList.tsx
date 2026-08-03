import React from 'react';
import type { SupplierTopologyWarehouse } from '@pegasusx/types';
import { DataList, DataListRow } from "@/components/portal";

interface DeliveryZonesListProps {
  warehouses: SupplierTopologyWarehouse[];
}

export function DeliveryZonesList({ warehouses }: DeliveryZonesListProps) {
  return (
    <DataList>
      {warehouses.map((warehouse) => (
        <DataListRow key={warehouse.warehouse_id || warehouse.name}>
          <div className="min-w-0 md-typescale-body-medium">
            <div className="font-medium">{warehouse.name}</div>
            <div className="text-[var(--color-md-outline)] text-sm mt-1">
              Radius {warehouse.coverage_radius_km} km · {warehouse.lat.toFixed(4)}, {warehouse.lng.toFixed(4)} ·{" "}
              {warehouse.is_active ? "Active" : "Inactive"}
            </div>
          </div>
        </DataListRow>
      ))}
    </DataList>
  );
}
