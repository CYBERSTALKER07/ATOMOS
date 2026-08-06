'use client';

import { usePortalT } from "@/lib/i18n";
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
import { PageChrome } from '@/components/PageChrome';
import { PortalField, PortalInput, PortalSelect } from '@/components/portal';

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
  const t = usePortalT();
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
        <PageChrome icon="transfers" title={t("factory_portal.transfers.create.text.create_transfer")} description={t("factory_portal.residual.text.loading_fleet_assignment_options")} loading skeletonVariant="form" />
      </PageTransition>
    );
  }

  if (error) {
    return (
      <PageTransition>
        <PageChrome
          icon="transfers"
          title={t("factory_portal.transfers.create.text.create_transfer")}
          description={t("factory_portal.residual.text.stage_a_factory_to_warehouse_movement")}
          error={error}
          actions={
            <button type="button" className="portal-btn portal-btn--outline" onClick={() => void loadFleet()}>
              Retry
            </button>
          }
        />
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <PageChrome
        icon="transfers"
        title={t("factory_portal.transfers.create.text.create_transfer")}
        description={t("factory_portal.residual.text.stage_a_factory_to_warehouse_movement_volume_is_captured_in_vu_o")}
        actions={
          <Link href="/transfers" className="portal-btn portal-btn--ghost text-sm">
            ← Back to transfers
          </Link>
        }
      >
      <form onSubmit={(event) => void handleSubmit(event)} className="max-w-xl space-y-4">
        <PortalField id="order_id" label={t("supplier_portal.admin.control_center.field.order_id")} optional>
          <PortalInput id="order_id" value={orderId} onChange={(e) => setOrderId(e.target.value)} placeholder={t("factory_portal.transfers.create.text.ord")} />
        </PortalField>
        <PortalField id="total_vu" label={t("factory_portal.residual.text.total_volume_vu")}>
          <PortalInput id="total_vu" type="number" min={1} step={1} required value={totalVu} onChange={(e) => setTotalVu(e.target.value)} />
        </PortalField>
        <PortalField id="driver_id" label={t("supplier_portal.analytics.route_performance.text.driver")} optional>
          <PortalSelect id="driver_id" value={driverId} onChange={(e) => setDriverId(e.target.value)}>
            <option value="">{t("factory_portal.transfers._id_.text.unassigned")}</option>
            {drivers.map((driver) => (
              <option key={driver.driver_id} value={driver.driver_id}>
                {driver.name} {driver.on_shift ? '(on shift)' : ''}
              </option>
            ))}
          </PortalSelect>
        </PortalField>
        <PortalField id="vehicle_id" label={t("supplier_portal.org_fleet.components.driver_table.text.vehicle")} optional>
          <PortalSelect id="vehicle_id" value={vehicleId} onChange={(e) => setVehicleId(e.target.value)}>
            <option value="">{t("factory_portal.transfers._id_.text.unassigned")}</option>
            {vehicles.map((vehicle) => (
              <option key={vehicle.vehicle_id} value={vehicle.vehicle_id}>
                {vehicle.plate_no} · {vehicle.state}
              </option>
            ))}
          </PortalSelect>
        </PortalField>
        <div className="flex flex-wrap gap-3 pt-2">
          <button type="submit" disabled={submitting} className="portal-btn portal-btn--primary inline-flex items-center gap-2">
            <Icon name="add" size={18} />
            {submitting ? 'Creating…' : 'Create transfer'}
          </button>
          <Link href="/transfers" className="portal-btn portal-btn--outline">
            Cancel
          </Link>
        </div>
      </form>
      </PageChrome>
    </PageTransition>
  );
}
