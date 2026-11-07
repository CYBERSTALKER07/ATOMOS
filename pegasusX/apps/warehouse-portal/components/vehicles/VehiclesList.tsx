"use client";

import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import type { WarehouseFleetVehicle } from '@pegasusx/types';
import { formatUnavailableReason, vehicleStatusLabel } from '@/lib/vehicle-fleet';
import Icon from '@/components/Icon';

interface VehiclesListProps {
  vehicles: WarehouseFleetVehicle[];
  loading: boolean;
}

export function VehiclesList({ vehicles, loading }: VehiclesListProps) {
  const t = usePortalT();
  if (!loading && vehicles.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-(--muted)">
        <Icon name="fleet" size={48} className="mb-3 opacity-40" />
        <p className="text-sm">{t("warehouse_portal.vehicles.vehicles_list.text.no_trucks_registered")}</p>
      </div>
    );
  }

  return (
    <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
      {vehicles.map(vehicle => (
        <TruckCard key={vehicle.vehicle_id} vehicle={vehicle} />
      ))}
    </div>
  );
}

function TruckCard({ vehicle }: { vehicle: WarehouseFleetVehicle }) {
  const t = usePortalT();
  const capacity = vehicle.capacity_vu ?? vehicle.max_volume_vu ?? 0;
  return (
    <Link
      href={`/vehicles/${vehicle.vehicle_id}`}
      className="block rounded-xl border border-(--border) p-4 hover:border-(--foreground)/20 transition-colors"
      style={{ background: 'var(--surface)' }}
    >
      <div className="flex items-start justify-between gap-3">
        <div>
          <div className="text-base font-semibold">{vehicle.label || vehicle.license_plate}</div>
          <div className="text-xs text-(--muted) mt-1">{vehicle.license_plate} · {vehicle.vehicle_class}</div>
        </div>
        <span className={`status-chip ${vehicle.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
          {vehicleStatusLabel(vehicle.is_active)}
        </span>
      </div>
      <div className="mt-3 grid grid-cols-2 gap-2 text-sm">
        <div>
          <div className="text-xs text-(--muted)">{t("warehouse_portal.vehicles._vehicle_id_.text.capacity")}</div>
          <div className="font-mono">{capacity} VU</div>
        </div>
        <div>
          <div className="text-xs text-(--muted)">{t("warehouse_portal.manifests.text.driver")}</div>
          <div>{vehicle.assigned_driver_name || 'Unassigned'}</div>
        </div>
      </div>
      {!vehicle.is_active && (vehicle.unavailable_reason || vehicle.unavailable_note) && (
        <p className="text-xs mt-3" style={{ color: 'var(--warning)' }}>
          {formatUnavailableReason(vehicle.unavailable_reason, vehicle.unavailable_note)}
        </p>
      )}
      <div className="mt-3 text-xs text-(--muted) inline-flex items-center gap-1">
        View details <Icon name="chevron_right" size={14} />
      </div>
    </Link>
  );
}
