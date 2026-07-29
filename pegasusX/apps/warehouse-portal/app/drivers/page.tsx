'use client';

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

      <DriversList
        drivers={drivers}
        vehicles={vehicles}
        loading={loading}
        assigningDriverId={assigningDriverId}
        handleAssignVehicle={handleAssignVehicle}
      />
    </div>
      </PageChrome>
    </PageTransition>
  );
}
