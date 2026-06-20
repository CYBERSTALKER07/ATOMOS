'use client';

import { useState } from 'react';
import Link from 'next/link';
import type { WarehouseFleetVehicle } from '@pegasusx/types';
import { warehouseCreateVehicleKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseVehiclesLive } from '@/lib/use-warehouse-vehicles-live';
import { formatUnavailableReason, vehicleStatusLabel } from '@/lib/vehicle-fleet';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';

export default function VehiclesPage() {
  const { vehicles, loading, error, liveMessage, reload } = useWarehouseVehiclesLive();
  const [showCreate, setShowCreate] = useState(false);
  const [form, setForm] = useState({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState('');

  useWarehouseSessionReconcile(() => {
    void reload();
  });

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault();
    setCreating(true);
    setCreateError('');
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles', {
        method: 'POST',
        body: JSON.stringify(form),
        headers: {
          'Idempotency-Key': warehouseCreateVehicleKey(warehouseId, form.license_plate),
        },
      });
      if (!res.ok) {
        throw new Error('Unable to create truck');
      }
      setForm({ label: '', license_plate: '', vehicle_class: 'CLASS_A' });
      setShowCreate(false);
      await reload();
    } catch (err) {
      setCreateError(err instanceof Error ? err.message : 'Unable to create truck');
    } finally {
      setCreating(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="fleet"
        title="Trucks"
        description="Fleet trucks with capacity, driver assignment, and live availability."
        loading={loading}
        skeletonVariant="table"
        error={error}
        actions={
          <div className="flex gap-2">
            <button type="button" onClick={() => setShowCreate(!showCreate)} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary">
              <Icon name="plus" size={16} /> Add truck
            </button>
            <button type="button" onClick={() => { void reload(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
              <Icon name="refresh" size={16} />
            </button>
          </div>
        }
      >
        {liveMessage && (
          <p className="text-sm mb-4" style={{ color: 'var(--warning)' }}>{liveMessage}</p>
        )}

        {showCreate && (
          <form onSubmit={handleCreate} className="p-4 rounded-xl border border-(--border) space-y-3 mb-4" style={{ background: 'var(--surface)' }}>
            <h2 className="text-sm font-semibold">New truck</h2>
            {createError && <p className="text-sm" style={{ color: 'var(--danger)' }}>{createError}</p>}
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
                placeholder="License plate"
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
              {creating ? 'Creating…' : 'Create truck'}
            </button>
          </form>
        )}

        {!loading && vehicles.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-20 text-(--muted)">
            <Icon name="fleet" size={48} className="mb-3 opacity-40" />
            <p className="text-sm">No trucks registered</p>
          </div>
        ) : (
          <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
            {vehicles.map(vehicle => (
              <TruckCard key={vehicle.vehicle_id} vehicle={vehicle} />
            ))}
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}

function TruckCard({ vehicle }: { vehicle: WarehouseFleetVehicle }) {
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
          <div className="text-xs text-(--muted)">Capacity</div>
          <div className="font-mono">{capacity} VU</div>
        </div>
        <div>
          <div className="text-xs text-(--muted)">Driver</div>
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
