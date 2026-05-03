'use client';

import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseVehicleUnavailableReason,
  WarehouseVehicleMutationResponse,
} from '@pegasus/types';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

const VEHICLE_UNAVAILABLE_REASONS: WarehouseVehicleUnavailableReason[] = [
  'MAINTENANCE',
  'TRUCK_DAMAGED',
  'REGULATORY_HOLD',
  'MANUAL_HOLD',
];

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

export default function VehiclesPage() {
  const [vehicles, setVehicles] = useState<WarehouseFleetVehicle[]>([]);
  const [vehicleReasons, setVehicleReasons] = useState<Record<string, WarehouseVehicleUnavailableReason>>({});
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
  const [creating, setCreating] = useState(false);
  const [error, setError] = useState('');
  const [mutatingVehicleId, setMutatingVehicleId] = useState('');

  const load = useCallback(async () => {
    setError('');
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles');
      if (!res.ok) {
        throw new Error('Unable to load vehicles');
      }

      const data = await res.json() as WarehouseFleetVehicleListResponse;
      const nextVehicles = data.vehicles || [];
      setVehicles(nextVehicles);
      setVehicleReasons(current => {
        const next = { ...current };
        for (const vehicle of nextVehicles) {
          if (!next[vehicle.vehicle_id]) {
            next[vehicle.vehicle_id] = vehicle.unavailable_reason || 'MANUAL_HOLD';
          }
        }
        return next;
      });
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load vehicles');
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setError('');
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles', {
        method: 'POST',
        body: JSON.stringify(form),
      });
      if (!res.ok) {
        throw new Error('Unable to create vehicle');
      }

      setForm({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
      setShowCreate(false);
      load();
    } catch (createError) {
      setError(createError instanceof Error ? createError.message : 'Unable to create vehicle');
    }
    finally { setCreating(false); }
  }

	async function handleToggleAvailability(vehicle: WarehouseFleetVehicle, nextActive: boolean) {
    setMutatingVehicleId(vehicle.vehicle_id);
    setError('');
    try {
	    const unavailableReason = vehicleReasons[vehicle.vehicle_id] || vehicle.unavailable_reason || 'MANUAL_HOLD';
      const res = await apiFetch(`/v1/warehouse/ops/vehicles/${vehicle.vehicle_id}`, {
        method: 'PATCH',
		  body: JSON.stringify(nextActive ? { is_active: true } : { is_active: false, unavailable_reason: unavailableReason }),
      });
      if (!res.ok) {
        throw new Error('Unable to update vehicle availability');
      }
      await res.json() as WarehouseVehicleMutationResponse;
      await load();
    } catch (updateError) {
      setError(updateError instanceof Error ? updateError.message : 'Unable to update vehicle availability');
    } finally {
      setMutatingVehicleId('');
    }
  }

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Vehicles</h1>
        <div className="flex gap-2">
          <button onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary">
            <Icon name="plus" size={16} /> Add Vehicle
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
          <h2 className="text-sm font-semibold">New Vehicle</h2>
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
            <input
              placeholder="Label (e.g. Truck-01)"
              value={form.label}
              onChange={e => setForm({ ...form, label: e.target.value })}
              required
              className="px-3 py-2 rounded-lg border text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
            <input
              placeholder="License Plate"
              value={form.license_plate}
              onChange={e => setForm({ ...form, license_plate: e.target.value })}
              required
              className="px-3 py-2 rounded-lg border text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
            <select
              value={form.vehicle_class}
              onChange={e => setForm({ ...form, vehicle_class: e.target.value })}
              className="px-3 py-2 rounded-lg border text-sm"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            >
              <option value="CLASS_A">Class A (50 VU)</option>
              <option value="CLASS_B">Class B (150 VU)</option>
              <option value="CLASS_C">Class C (400 VU)</option>
            </select>
          </div>
          <button type="submit" disabled={creating} className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50">
            {creating ? 'Creating...' : 'Create Vehicle'}
          </button>
        </form>
      )}

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : vehicles.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-(--muted)">
          <Icon name="fleet" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No vehicles registered</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-(--border)">
                <th className="text-left py-2 px-3 font-medium">Label</th>
                <th className="text-left py-2 px-3 font-medium">Plate</th>
                <th className="text-left py-2 px-3 font-medium">Class</th>
                <th className="text-left py-2 px-3 font-medium">Assigned Driver</th>
                <th className="text-right py-2 px-3 font-medium">Capacity</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-left py-2 px-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {vehicles.map(v => (
                <tr key={v.vehicle_id} className="border-b border-(--border) hover:bg-(--surface) transition-colors">
                  <td className="py-2.5 px-3 font-medium">{v.label || '—'}</td>
                  <td className="py-2.5 px-3">{v.license_plate}</td>
                  <td className="py-2.5 px-3">{v.vehicle_class}</td>
                  <td className="py-2.5 px-3 text-(--muted)">{v.assigned_driver_name || 'Unassigned'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{v.capacity_vu} VU</td>
                  <td className="py-2.5 px-3">
                    <div className="space-y-1">
                      <span className={`status-chip ${v.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                        {v.is_active ? v.status || 'Active' : 'Unavailable'}
                      </span>
                      {!v.is_active && v.unavailable_reason && (
                        <div className="text-xs text-(--muted)">{formatUnavailableReason(v.unavailable_reason)}</div>
                      )}
                    </div>
                  </td>
                  <td className="py-2.5 px-3">
                    <div className="flex flex-wrap items-center gap-2">
                      {v.is_active && (
                        <select
                          value={vehicleReasons[v.vehicle_id] || v.unavailable_reason || 'MANUAL_HOLD'}
                          onChange={event => setVehicleReasons(current => ({
                            ...current,
                            [v.vehicle_id]: event.target.value as WarehouseVehicleUnavailableReason,
                          }))}
                          disabled={mutatingVehicleId === v.vehicle_id}
                          className="rounded-lg border px-3 py-1.5 text-sm"
                          style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                        >
                          {VEHICLE_UNAVAILABLE_REASONS.map(reason => (
                            <option key={reason} value={reason}>{formatUnavailableReason(reason)}</option>
                          ))}
                        </select>
                      )}
                      <button
                        onClick={() => handleToggleAvailability(v, !v.is_active)}
                        disabled={mutatingVehicleId === v.vehicle_id}
                        className="rounded-lg px-3 py-1.5 text-sm font-medium disabled:opacity-50"
                        style={{
                          background: v.is_active ? 'color-mix(in srgb, var(--warning) 15%, transparent)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
                          color: v.is_active ? 'var(--warning)' : 'var(--success)',
                        }}
                      >
                        {mutatingVehicleId === v.vehicle_id ? 'Updating...' : v.is_active ? 'Set Unavailable' : 'Restore Vehicle'}
                      </button>
                    </div>
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
