"use client";

import { usePortalT } from "@/lib/i18n";
import type {
  WarehouseFleetDriver,
  WarehouseFleetVehicle,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import Icon from '@/components/Icon';

function formatUnavailableReason(reason?: string) {
  if (!reason) {
    return '';
  }

  return reason
    .toLowerCase()
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export interface DriversListProps {
  drivers: WarehouseFleetDriver[];
  vehicles: WarehouseFleetVehicle[];
  loading: boolean;
  assigningDriverId: string;
  handleAssignVehicle: (driverId: string, vehicleId: string) => void;
}

export function DriversList({
  drivers,
  vehicles,
  loading,
  assigningDriverId,
  handleAssignVehicle,
}: DriversListProps) {
  const t = usePortalT();
  function assignedVehicleLabel(driver: WarehouseFleetDriver) {
    if (!driver.vehicle_id) {
      return 'Unassigned';
    }
    const vehicle = vehicles.find(item => item.vehicle_id === driver.vehicle_id);
    if (!vehicle) {
      return 'Assigned vehicle unavailable';
    }
    return [vehicle.label || vehicle.license_plate, vehicle.vehicle_class].filter(Boolean).join(' · ');
  }

  function assignedVehicleReason(driver: WarehouseFleetDriver) {
    if (!driver.vehicle_id) {
      return '';
    }

    const directReason = driver.vehicle_unavailable_reason as WarehouseVehicleUnavailableReason | undefined;
    if (directReason && driver.vehicle_is_active === false) {
      return `Vehicle unavailable: ${formatUnavailableReason(directReason)}`;
    }

    const vehicle = vehicles.find(item => item.vehicle_id === driver.vehicle_id);
    if (vehicle && !vehicle.is_active && vehicle.unavailable_reason) {
      return `Vehicle unavailable: ${formatUnavailableReason(vehicle.unavailable_reason)}`;
    }

    return '';
  }

  function vehicleOptionLabel(vehicle: WarehouseFleetVehicle) {
    return [vehicle.label || vehicle.license_plate, vehicle.vehicle_class, `${vehicle.capacity_vu} VU`]
      .filter(Boolean)
      .join(' · ');
  }

  if (loading) {
    return (
      <div className="space-y-1">
        {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
      </div>
    );
  }

  if (drivers.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-20 text-(--muted)">
        <Icon name="fleet" size={48} className="mb-3 opacity-40" />
        <p className="text-sm">{t("warehouse_portal.drivers.drivers_list.text.no_drivers_registered")}</p>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="desk-table w-full text-sm">
        <thead>
          <tr className="border-b border-(--border)">
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.drivers.text.name")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("common.field.phone")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.drivers.drivers_list.text.assigned_vehicle")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.bins.text.status")}</th>
            <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.bins.text.active")}</th>
          </tr>
        </thead>
        <tbody>
          {drivers.map(d => (
            <tr key={d.driver_id} className="border-b border-(--border) hover:bg-(--surface) transition-colors">
              <td className="py-2.5 px-3 font-medium">{d.name}</td>
              <td className="py-2.5 px-3 text-(--muted)">{d.phone}</td>
              <td className="py-2.5 px-3">
                <label className="sr-only" htmlFor={`driver-vehicle-${d.driver_id}`}>
                  Assign vehicle for {d.name}
                </label>
                <select
                  id={`driver-vehicle-${d.driver_id}`}
                  value={d.vehicle_id || ''}
                  onChange={event => handleAssignVehicle(d.driver_id, event.target.value)}
                  disabled={assigningDriverId === d.driver_id}
                  className="w-full rounded-lg border px-3 py-2 text-sm"
                  style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                >
                  <option value="">{t("warehouse_portal.drivers.drivers_list.text.unassigned")}</option>
                  {vehicles
                    .filter(vehicle => vehicle.is_active || vehicle.vehicle_id === d.vehicle_id)
                    .map(vehicle => (
                      <option key={vehicle.vehicle_id} value={vehicle.vehicle_id}>
                        {vehicleOptionLabel(vehicle)}
                      </option>
                    ))}
                </select>
                <p className="mt-1 text-xs text-(--muted)">{assignedVehicleLabel(d)}</p>
                {assignedVehicleReason(d) && (
                  <p className="mt-1 text-xs" style={{ color: 'var(--warning)' }}>{assignedVehicleReason(d)}</p>
                )}
              </td>
              <td className="py-2.5 px-3">
                <span className={`status-chip ${['IN_TRANSIT', 'RETURNING'].includes(d.truck_status) ? 'status-chip--active' : 'status-chip--stable'}`}>
                  {d.truck_status || 'IDLE'}
                </span>
              </td>
              <td className="py-2.5 px-3">
                <span className={`status-chip ${d.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                  {d.is_active ? 'Active' : 'Inactive'}
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
