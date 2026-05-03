'use client';

import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseAssignVehicleResponse,
  WarehouseFleetDriver,
  WarehouseFleetDriverListResponse,
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseVehicleUnavailableReason,
} from '@pegasus/types';
import { apiFetch } from '@/lib/auth';
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

export default function DriversPage() {
  const [drivers, setDrivers] = useState<WarehouseFleetDriver[]>([]);
  const [vehicles, setVehicles] = useState<WarehouseFleetVehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ name: '', phone: '' });
  const [creating, setCreating] = useState(false);
  const [createdPin, setCreatedPin] = useState('');
  const [error, setError] = useState('');
  const [assigningDriverId, setAssigningDriverId] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const [driverRes, vehicleRes] = await Promise.all([
        apiFetch('/v1/warehouse/ops/drivers'),
        apiFetch('/v1/warehouse/ops/vehicles'),
      ]);

      if (!driverRes.ok || !vehicleRes.ok) {
        throw new Error('Unable to load warehouse fleet');
      }

      const driverData = await driverRes.json() as WarehouseFleetDriverListResponse;
      const vehicleData = await vehicleRes.json() as WarehouseFleetVehicleListResponse;
      setDrivers(driverData.drivers || []);
      setVehicles(vehicleData.vehicles || []);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load warehouse fleet');
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setError('');
    try {
      const res = await apiFetch('/v1/warehouse/ops/drivers', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      if (!res.ok) {
        throw new Error('Unable to create driver');
      }

      const data = await res.json() as { pin?: string };
      setCreatedPin(data.pin || '');
      setForm({ name: '', phone: '' });
      load();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : 'Unable to create driver');
    }
    finally { setCreating(false); }
  }

  async function handleAssignVehicle(driverId: string, vehicleId: string) {
    setAssigningDriverId(driverId);
    setError('');
    try {
      const res = await apiFetch(`/v1/warehouse/ops/drivers/${driverId}/assign-vehicle`, {
        method: 'PATCH',
        body: JSON.stringify({ vehicle_id: vehicleId }),
      });
      if (!res.ok) {
        throw new Error('Unable to update driver assignment');
      }
      await res.json() as WarehouseAssignVehicleResponse;
      await load();
    } catch (assignError) {
      setError(assignError instanceof Error ? assignError.message : 'Unable to update driver assignment');
    } finally {
      setAssigningDriverId('');
    }
  }

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

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Drivers</h1>
        <div className="flex gap-2">
          <button onClick={() => { setShowCreate(!showCreate); setCreatedPin(''); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary">
            <Icon name="plus" size={16} /> Add Driver
          </button>
          <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} />
          </button>
        </div>
      </div>

      {error && (
        <div className="rounded-xl border px-3 py-2 text-sm" style={{ borderColor: 'var(--danger)', color: 'var(--danger)', background: 'color-mix(in srgb, var(--danger) 10%, transparent)' }}>
          {error}
        </div>
      )}

      {showCreate && (
        <form onSubmit={handleCreate} className="p-4 rounded-xl border border-(--border) space-y-3" style={{ background: 'var(--surface)' }}>
          <h2 className="text-sm font-semibold">New Driver</h2>
          <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
            <input
              placeholder="Full Name"
              value={form.name}
              onChange={e => setForm({ ...form, name: e.target.value })}
              required
              className="px-3 py-2 rounded-lg border text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
            <input
              placeholder="+998..."
              value={form.phone}
              onChange={e => setForm({ ...form, phone: e.target.value })}
              required
              className="px-3 py-2 rounded-lg border text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
          </div>
          <button type="submit" disabled={creating} className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50">
            {creating ? 'Creating...' : 'Create Driver'}
          </button>
          {createdPin && (
            <div className="mt-2 p-3 rounded-lg text-sm" style={{ background: 'var(--success)', color: 'var(--success-foreground)' }}>
              Driver created. One-time PIN: <strong className="font-mono text-lg">{createdPin}</strong>
              <br /><span className="text-xs opacity-80">Share this PIN with the driver. It cannot be shown again.</span>
            </div>
          )}
        </form>
      )}

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : drivers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-(--muted)">
          <Icon name="fleet" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No drivers registered</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-(--border)">
                <th className="text-left py-2 px-3 font-medium">Name</th>
                <th className="text-left py-2 px-3 font-medium">Phone</th>
                <th className="text-left py-2 px-3 font-medium">Assigned Vehicle</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-left py-2 px-3 font-medium">Active</th>
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
                      <option value="">Unassigned</option>
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
      )}
    </div>
  );
}
