'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { factoryTransferCreateKey } from '@pegasusx/api-client';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';

interface FleetDriver {
  driver_id: string;
  name: string;
  on_shift?: boolean;
}

interface FleetVehicle {
  vehicle_id: string;
  plate_no: string;
  state: string;
}

interface CreateTransferResponse {
  transfer_id: string;
  state: string;
}

export default function CreateTransferPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [loadingFleet, setLoadingFleet] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [drivers, setDrivers] = useState<FleetDriver[]>([]);
  const [vehicles, setVehicles] = useState<FleetVehicle[]>([]);
  const [orderId, setOrderId] = useState('');
  const [totalVu, setTotalVu] = useState('25');
  const [driverId, setDriverId] = useState('');
  const [vehicleId, setVehicleId] = useState('');

  const loadFleet = useCallback(async () => {
    setLoadingFleet(true);
    setError(null);
    try {
      const [driversRes, vehiclesRes] = await Promise.all([
        apiFetch('/v1/factory/fleet/drivers'),
        apiFetch('/v1/factory/fleet/vehicles'),
      ]);
      if (!driversRes.ok || !vehiclesRes.ok) {
        throw new Error('Unable to load fleet options for assignment.');
      }
      const driversData = await driversRes.json();
      const vehiclesData = await vehiclesRes.json();
      setDrivers(driversData.drivers || []);
      setVehicles(vehiclesData.vehicles || []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Failed to load fleet');
    } finally {
      setLoadingFleet(false);
    }
  }, []);

  useEffect(() => {
    void loadFleet();
  }, [loadFleet]);

  useFactorySessionReconcile(() => {
    if (submitting) {
      setSubmitting(false);
      toast('Connection restored — verify transfer was created before retrying.', 'info');
    }
    void loadFleet();
  });

  const handleSubmit = async (event: React.FormEvent) => {
    event.preventDefault();
    const parsedVu = Number(totalVu);
    if (!Number.isFinite(parsedVu) || parsedVu <= 0) {
      toast('Total VU must be a positive number', 'error');
      return;
    }

    setSubmitting(true);
    try {
      const body: Record<string, string | number> = { total_vu: Math.round(parsedVu) };
      if (orderId.trim()) body.order_id = orderId.trim();
      if (driverId) body.driver_id = driverId;
      if (vehicleId) body.vehicle_id = vehicleId;

      const res = await apiFetch('/v1/factory/transfers/create', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': factoryTransferCreateKey(
            factoryOperatorId(),
            orderId.trim(),
            Math.round(parsedVu),
            driverId,
            vehicleId,
          ),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.error || `Create failed (${res.status})`);
      }
      const created = (await res.json()) as CreateTransferResponse;
      toast('Transfer created', 'success');
      router.push(`/transfers/${created.transfer_id}`);
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Transfer create failed', 'error');
    } finally {
      setSubmitting(false);
    }
  };

  if (loadingFleet) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="loading" title="Create Transfer" subtitle="Loading fleet assignment options." />
        </div>
      </PageTransition>
    );
  }

  if (error) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState
            kind="error"
            title="Create Transfer"
            headline="Unable to prepare transfer form"
            body={error}
            actionLabel="Retry"
            onAction={() => void loadFleet()}
          />
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition className="space-y-6 p-6 md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <Link href="/transfers" className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]">
            ← Back to transfers
          </Link>
          <h1 className="mt-2 text-2xl font-semibold tracking-tight text-[var(--foreground)]">Create Transfer</h1>
          <p className="mt-2 max-w-2xl text-sm leading-6 text-[var(--muted)]">
            Stage a factory-to-warehouse movement. Volume is captured in VU; optional order and fleet assignment can be set now or during loading.
          </p>
        </div>
      </div>

      <form onSubmit={(event) => void handleSubmit(event)} className="max-w-xl space-y-5 rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
        <div>
          <label htmlFor="order_id" className="mb-1.5 block text-sm font-medium">Order ID (optional)</label>
          <input
            id="order_id"
            value={orderId}
            onChange={(e) => setOrderId(e.target.value)}
            placeholder="ord_…"
            className="md-input-outlined w-full px-4 py-3"
          />
        </div>

        <div>
          <label htmlFor="total_vu" className="mb-1.5 block text-sm font-medium">Total volume (VU)</label>
          <input
            id="total_vu"
            type="number"
            min={1}
            step={1}
            required
            value={totalVu}
            onChange={(e) => setTotalVu(e.target.value)}
            className="md-input-outlined w-full px-4 py-3"
          />
        </div>

        <div>
          <label htmlFor="driver_id" className="mb-1.5 block text-sm font-medium">Driver (optional)</label>
          <select
            id="driver_id"
            value={driverId}
            onChange={(e) => setDriverId(e.target.value)}
            className="md-input-outlined w-full px-4 py-3"
          >
            <option value="">Unassigned</option>
            {drivers.map((driver) => (
              <option key={driver.driver_id} value={driver.driver_id}>
                {driver.name} {driver.on_shift ? '(on shift)' : ''}
              </option>
            ))}
          </select>
        </div>

        <div>
          <label htmlFor="vehicle_id" className="mb-1.5 block text-sm font-medium">Vehicle (optional)</label>
          <select
            id="vehicle_id"
            value={vehicleId}
            onChange={(e) => setVehicleId(e.target.value)}
            className="md-input-outlined w-full px-4 py-3"
          >
            <option value="">Unassigned</option>
            {vehicles.map((vehicle) => (
              <option key={vehicle.vehicle_id} value={vehicle.vehicle_id}>
                {vehicle.plate_no} · {vehicle.state}
              </option>
            ))}
          </select>
        </div>

        <div className="flex flex-wrap gap-3 pt-2">
          <button
            type="submit"
            disabled={submitting}
            className="md-btn md-btn-filled md-typescale-label-large inline-flex items-center gap-2 px-6 py-2 disabled:opacity-60"
          >
            <Icon name="add" size={18} />
            {submitting ? 'Creating…' : 'Create transfer'}
          </button>
          <Link href="/transfers" className="md-btn md-btn-outlined md-typescale-label-large px-6 py-2">
            Cancel
          </Link>
        </div>
      </form>
    </PageTransition>
  );
}
