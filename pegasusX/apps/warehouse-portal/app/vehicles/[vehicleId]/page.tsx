'use client';

import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import type { WarehouseFleetVehicle } from '@pegasusx/types';
import { fetchWarehouseVehicle } from '@/lib/use-warehouse-vehicles-live';
import { formatUnavailableReason, vehicleStatusLabel } from '@/lib/vehicle-fleet';
import { subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from '@/lib/fleet-ws-events';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import VehicleAvailabilityPanel from '@/components/VehicleAvailabilityPanel';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';

export default function TruckDetailPage() {
  const params = useParams();
  const router = useRouter();
  const vehicleId = String(params.vehicleId ?? '');
  const [vehicle, setVehicle] = useState<WarehouseFleetVehicle | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [liveMessage, setLiveMessage] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!vehicleId) {
      return;
    }
    setError(null);
    try {
      const row = await fetchWarehouseVehicle(vehicleId);
      if (!row) {
        router.replace('/vehicles');
        return;
      }
      setVehicle(row);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Failed to load truck');
    } finally {
      setLoading(false);
    }
  }, [router, vehicleId]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    let timer: number | undefined;
    const unsubscribe = subscribeWarehouseWS({
      onMessage: (payload) => {
        const eventType = parseWarehouseWsEventType(payload);
        if (!eventType || !WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS.has(eventType)) {
          return;
        }
        let eventVehicleId = '';
        try {
          const parsed = JSON.parse(payload) as { vehicle_id?: string; body?: string; title?: string };
          eventVehicleId = parsed.vehicle_id || '';
          if ((eventType === 'VEHICLE_AVAILABILITY_CHANGED' || eventType === 'DRIVER_AVAILABILITY_CHANGED')
            && (parsed.body || parsed.title)) {
            setLiveMessage(parsed.body || parsed.title || 'Fleet updated');
          }
        } catch {
          // ignore malformed payloads
        }
        if (eventType === 'VEHICLE_AVAILABILITY_CHANGED' && eventVehicleId && eventVehicleId !== vehicleId) {
          return;
        }
        if (timer !== undefined) {
          window.clearTimeout(timer);
        }
        timer = window.setTimeout(() => {
          void load();
        }, eventType === 'VEHICLE_AVAILABILITY_CHANGED' ? 0 : 250);
      },
    });
    return () => {
      if (timer !== undefined) {
        window.clearTimeout(timer);
      }
      unsubscribe();
    };
  }, [load, vehicleId]);

  useWarehouseSessionReconcile(() => {
    void load();
  });

  if (!loading && !vehicle && !error) {
    return null;
  }

  const capacity = vehicle?.capacity_vu ?? vehicle?.max_volume_vu ?? 0;
  const title = vehicle?.label || vehicle?.license_plate || 'Truck';

  return (
    <PageTransition>
      <PageChrome
        title={title}
        description={vehicle ? `${vehicle.license_plate} · ${vehicle.vehicle_class}` : 'Truck detail'}
        loading={loading}
        error={error}
        actions={
          <Link href="/vehicles" className="button--secondary px-3 py-1.5 rounded-lg text-sm inline-flex items-center gap-1.5">
            <Icon name="arrow_back" size={16} /> All trucks
          </Link>
        }
      >
        {liveMessage && (
          <p className="text-sm mb-4" style={{ color: 'var(--warning)' }}>{liveMessage}</p>
        )}

        {vehicle && (
          <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
            <div className="lg:col-span-2 space-y-6">
              <PageSection title="Overview">
                <dl className="grid grid-cols-1 sm:grid-cols-2 gap-4 text-sm">
                  <div>
                    <dt className="text-(--muted)">Status</dt>
                    <dd className="mt-1">
                      <span className={`status-chip ${vehicle.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                        {vehicleStatusLabel(vehicle.is_active)}
                      </span>
                    </dd>
                  </div>
                  <div>
                    <dt className="text-(--muted)">Capacity</dt>
                    <dd className="mt-1 font-mono">{capacity} VU</dd>
                  </div>
                  <div>
                    <dt className="text-(--muted)">License plate</dt>
                    <dd className="mt-1">{vehicle.license_plate}</dd>
                  </div>
                  <div>
                    <dt className="text-(--muted)">Class</dt>
                    <dd className="mt-1">{vehicle.vehicle_class}</dd>
                  </div>
                  <div className="sm:col-span-2">
                    <dt className="text-(--muted)">Assigned driver</dt>
                    <dd className="mt-1">
                      {vehicle.assigned_driver_id ? (
                        <Link href="/drivers" className="underline underline-offset-2">
                          {vehicle.assigned_driver_name || vehicle.assigned_driver_id}
                        </Link>
                      ) : (
                        'Unassigned'
                      )}
                    </dd>
                  </div>
                  {!vehicle.is_active && (vehicle.unavailable_reason || vehicle.unavailable_note) && (
                    <div className="sm:col-span-2">
                      <dt className="text-(--muted)">Hold reason</dt>
                      <dd className="mt-1" style={{ color: 'var(--warning)' }}>
                        {formatUnavailableReason(vehicle.unavailable_reason, vehicle.unavailable_note)}
                      </dd>
                    </div>
                  )}
                </dl>
              </PageSection>

              <PageSection title="Dispatch impact">
                <p className="text-sm text-(--muted)">
                  {vehicle.is_active
                    ? 'This truck is eligible for manual and smart dispatch when its assigned driver is on shift and not in transit.'
                    : 'This truck is excluded from dispatch fleet calculations until restored. Assigned drivers appear as vehicle-unavailable on the dispatch board.'}
                </p>
                <Link href="/dispatch" className="inline-flex items-center gap-1.5 mt-3 text-sm underline underline-offset-2">
                  Open dispatch board <Icon name="dispatch" size={14} />
                </Link>
              </PageSection>
            </div>

            <div>
              <VehicleAvailabilityPanel
                vehicle={vehicle}
                onUpdated={updated => {
                  setVehicle(updated);
                  setLiveMessage(updated.is_active ? 'Truck restored for dispatch' : 'Truck marked unavailable');
                }}
              />
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
