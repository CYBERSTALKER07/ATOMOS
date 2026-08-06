'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseAssignVehicleResponse,
  WarehouseFleetDriver,
  WarehouseFleetDriverListResponse,
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { warehouseAssignDriverVehicleKey, warehouseCreateDriverKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { DriversList } from '@/components/drivers/DriversList';

export default function DriversPage() {
  const t = usePortalT();
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
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const res = await apiFetch('/v1/warehouse/ops/drivers', {
        method: 'POST',
        body: JSON.stringify(form),
        headers: {
          'Idempotency-Key': warehouseCreateDriverKey(warehouseId, form.phone),
        },
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
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      const res = await apiFetch(`/v1/warehouse/ops/drivers/${driverId}/assign-vehicle`, {
        method: 'PATCH',
        body: JSON.stringify({ vehicle_id: vehicleId }),
        headers: {
          'Idempotency-Key': warehouseAssignDriverVehicleKey(warehouseId, driverId, vehicleId),
        },
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

  return (
    <PageTransition>
      <PageChrome
        icon="fleet"
        title={t("portal.nav.drivers")}
        description={t("warehouse_portal.residual.text.fleet_drivers_with_vehicle_assignment_and_live_truck_status")}
        loading={loading}
        skeletonVariant="table"
        error={error}
        actions={
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setShowCreate(!showCreate)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary"
            >
              <Icon name="plus" size={16} /> Add driver
            </button>
            <button
              type="button"
              onClick={() => { void load(); }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary"
            >
              <Icon name="refresh" size={16} />
            </button>
          </div>
        }
      >
        {showCreate && (
          <form
            onSubmit={handleCreate}
            className="p-4 rounded-xl border border-(--border) space-y-3 mb-4"
            style={{ background: 'var(--surface)' }}
          >
            <h2 className="text-sm font-semibold">{t("warehouse_portal.drivers.text.new_driver")}</h2>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
              <input
                placeholder={t("warehouse_portal.drivers.text.name")}
                value={form.name}
                onChange={e => setForm({ ...form, name: e.target.value })}
                required
                className="px-3 py-2 rounded-lg border text-sm"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
              />
              <input
                placeholder={t("common.field.phone")}
                value={form.phone}
                onChange={e => setForm({ ...form, phone: e.target.value })}
                required
                className="px-3 py-2 rounded-lg border text-sm"
                style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
              />
            </div>
            <button
              type="submit"
              disabled={creating}
              className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
            >
              {creating ? 'Creating…' : 'Create driver'}
            </button>
          </form>
        )}

        {createdPin && (
          <div
            className="p-4 rounded-xl border border-(--border) mb-4"
            style={{ background: 'var(--surface)' }}
          >
            <p className="text-sm font-semibold">{t("warehouse_portal.drivers.text.driver_pin")}</p>
            <p className="text-sm text-(--muted) mt-1">{t("warehouse_portal.drivers.text.share_this_one_time_pin_with_the_driver")}</p>
            <p className="mt-2 font-mono text-lg tracking-widest">{createdPin}</p>
          </div>
        )}

        <DriversList
          drivers={drivers}
          vehicles={vehicles}
          loading={loading}
          assigningDriverId={assigningDriverId}
          handleAssignVehicle={handleAssignVehicle}
        />
      </PageChrome>
    </PageTransition>
  );
}
