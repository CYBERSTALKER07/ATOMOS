'use client';

import { useEffect, useState, useCallback, useMemo } from 'react';
import type {
  WarehouseDispatchCapacityWarning,
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseFleetVehicle,
  WarehouseFleetVehicleListResponse,
  WarehouseUnavailableDispatchDriver,
  WarehouseVehicleUnavailableReason,
} from '@pegasusx/types';
import { ApiError, warehouseDispatchKey, warehouseUpdateVehicleKey } from '@pegasusx/api-client';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { WAREHOUSE_DISPATCH_REFRESH_EVENTS, parseWarehouseWsEventType } from '@/lib/fleet-ws-events';
import { subscribeWarehouseWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import DispatchPreviewMap from '@/components/DispatchPreviewMap';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';

const TETRIS_BUFFER = 0.95;

const VEHICLE_UNAVAILABLE_REASONS: WarehouseVehicleUnavailableReason[] = [
  'MAINTENANCE',
  'TRUCK_DAMAGED',
  'REGULATORY_HOLD',
  'MANUAL_HOLD',
  'OTHER',
];

const FLEET_AVAILABILITY_EVENTS = new Set([
  'DRIVER_AVAILABILITY_CHANGED',
  'VEHICLE_AVAILABILITY_CHANGED',
]);

type CapacityPromptMode = 'manual' | 'auto';

function formatUnavailableReason(reason?: string, note?: string) {
  if (!reason) {
    return note || '';
  }
  if (reason.toUpperCase() === 'OTHER' && note?.trim()) {
    return note.trim();
  }
  return reason
    .toLowerCase()
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function formatVU(value: number) {
  return value.toFixed(1);
}

function routeVolumeVU(route: WarehouseDispatchProposedRoute) {
  return route.volume_vu ?? route.loaded_volume ?? 0;
}

export default function DispatchPage() {
  const [orders, setOrders] = useState<WarehouseDispatchOrder[]>([]);
  const [drivers, setDrivers] = useState<WarehouseDispatchDriver[]>([]);
  const [unavailableDrivers, setUnavailableDrivers] = useState<WarehouseUnavailableDispatchDriver[]>([]);
  const [loading, setLoading] = useState(true);
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [executing, setExecuting] = useState(false);
  const [executeError, setExecuteError] = useState<string | null>(null);
  const [executeSuccess, setExecuteSuccess] = useState<string | null>(null);
  const [proposedRoutes, setProposedRoutes] = useState<WarehouseDispatchProposedRoute[]>([]);
  const [optimizerSource, setOptimizerSource] = useState<string | null>(null);
  const [optimizerWarnings, setOptimizerWarnings] = useState<string[]>([]);
  const [windowConstrainedCount, setWindowConstrainedCount] = useState(0);
  const [fleetEffectiveCapacityVU, setFleetEffectiveCapacityVU] = useState(0);
  const [planFingerprint, setPlanFingerprint] = useState<string | null>(null);
  const [selectedDriverId, setSelectedDriverId] = useState('');
  const [selectedOrderIds, setSelectedOrderIds] = useState<Set<string>>(new Set());
  const [capacityPrompt, setCapacityPrompt] = useState<WarehouseDispatchCapacityWarning[] | null>(null);
  const [capacityPromptMode, setCapacityPromptMode] = useState<CapacityPromptMode>('manual');
  const [showSmartConfirm, setShowSmartConfirm] = useState(false);
  const [warehouseId, setWarehouseId] = useState(() => warehouseHomeNodeId() || 'warehouse');
  const [vehicles, setVehicles] = useState<WarehouseFleetVehicle[]>([]);
  const [vehicleReasons, setVehicleReasons] = useState<Record<string, WarehouseVehicleUnavailableReason>>({});
  const [vehicleNotes, setVehicleNotes] = useState<Record<string, string>>({});
  const [mutatingVehicleId, setMutatingVehicleId] = useState('');
  const [fleetAlert, setFleetAlert] = useState<string | null>(null);

  const selectedOrderList = useMemo(() => [...selectedOrderIds].sort(), [selectedOrderIds]);

  const load = useCallback(async (orderFilter?: string[]) => {
    const scopedWarehouseId = warehouseHomeNodeId();
    if (scopedWarehouseId) {
      setWarehouseId(scopedWarehouseId);
    }
    setLoadError(null);
    try {
      const filter = orderFilter ?? selectedOrderList;
      const body = filter.length > 0 ? { order_ids: filter } : {};
      const data = await warehouseApi.previewWarehouseDispatch({}, body);
      const nextOrders = data.undispatched_orders || data.orders || [];
      setOrders(nextOrders);
      setDrivers(data.available_drivers || data.drivers || []);
      setUnavailableDrivers(data.unavailable_drivers || []);
      setProposedRoutes(data.proposed_routes || []);
      setOptimizerSource(data.optimizer_source || null);
      setOptimizerWarnings(data.optimizer_warnings || []);
      setWindowConstrainedCount(data.window_constrained_count || 0);
      setFleetEffectiveCapacityVU(data.fleet_effective_capacity_vu ?? 0);
      setPlanFingerprint(data.plan_fingerprint ?? null);
      setSelectedOrderIds(prev => {
        const valid = new Set(nextOrders.map(order => order.order_id));
        return new Set([...prev].filter(id => valid.has(id)));
      });
      setRestricted(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setRestricted(true);
        setOrders([]);
        setDrivers([]);
        setUnavailableDrivers([]);
      } else {
        setLoadError(err instanceof ApiError ? err.message : 'Failed to load dispatch preview');
      }
    } finally {
      setLoading(false);
    }
  }, [selectedOrderList]);

  const loadVehicles = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles');
      if (!res.ok) {
        return;
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
      setVehicleNotes(current => {
        const next = { ...current };
        for (const vehicle of nextVehicles) {
          if (!next[vehicle.vehicle_id] && vehicle.unavailable_note) {
            next[vehicle.vehicle_id] = vehicle.unavailable_note;
          }
        }
        return next;
      });
    } catch {
      // Non-fatal: dispatch preview still works without vehicle list.
    }
  }, []);

  const loadAll = useCallback(async (orderFilter?: string[]) => {
    await Promise.all([load(orderFilter), loadVehicles()]);
  }, [load, loadVehicles]);

  useEffect(() => { void loadAll(); }, [loadAll]);

  useEffect(() => {
    let signalTimer: number | undefined;
    const unsubscribe = subscribeWarehouseWS({
      onMessage: (payload) => {
        const eventType = parseWarehouseWsEventType(payload);
        if (!eventType || !WAREHOUSE_DISPATCH_REFRESH_EVENTS.has(eventType)) {
          return;
        }
        if (FLEET_AVAILABILITY_EVENTS.has(eventType)) {
          try {
            const parsed = JSON.parse(payload) as { body?: string; title?: string };
            if (parsed.body || parsed.title) {
              setFleetAlert(parsed.body || parsed.title || 'Fleet availability updated');
            }
          } catch {
            setFleetAlert('Fleet availability updated');
          }
        }
        if (signalTimer !== undefined) {
          window.clearTimeout(signalTimer);
        }
        const delay = FLEET_AVAILABILITY_EVENTS.has(eventType) ? 0 : 500;
        signalTimer = window.setTimeout(() => {
          void loadAll();
        }, delay);
      },
    });
    return () => {
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
      }
      unsubscribe();
    };
  }, [loadAll]);

  useWarehouseSessionReconcile(() => {
    void loadAll();
    if (executing) {
      setExecuting(false);
      setExecuteError(null);
      setExecuteSuccess('Connection restored — dispatch board refreshed from server.');
    }
  });

  const selectedDriver = useMemo(
    () => drivers.find(driver => driver.driver_id === selectedDriverId),
    [drivers, selectedDriverId],
  );

  const selectedVolumeVU = useMemo(() => {
    let total = 0;
    for (const order of orders) {
      if (selectedOrderIds.has(order.order_id)) {
        total += order.volume_vu ?? 0;
      }
    }
    return total;
  }, [orders, selectedOrderIds]);

  const effectiveCapacityVU = useMemo(() => {
    if (!selectedDriver) {
      return 0;
    }
    if (selectedDriver.free_volume_vu != null && selectedDriver.free_volume_vu > 0) {
      return selectedDriver.free_volume_vu * TETRIS_BUFFER;
    }
    const max = selectedDriver.max_volume_vu ?? 0;
    return max > 0 ? max * TETRIS_BUFFER : 0;
  }, [selectedDriver]);

  const smartTargetVolumeVU = useMemo(() => {
    if (selectedOrderIds.size > 0) {
      return selectedVolumeVU;
    }
    return orders.reduce((sum, order) => sum + (order.volume_vu ?? 0), 0);
  }, [orders, selectedOrderIds.size, selectedVolumeVU]);

  const allSelected = orders.length > 0 && selectedOrderIds.size === orders.length;
  const canSmartDispatch = !restricted && orders.length > 0 && drivers.length > 0;
  const smartStopCount = proposedRoutes.reduce(
    (sum, route) => sum + (route.stop_count ?? route.order_ids?.length ?? route.stops?.length ?? 0),
    0,
  );

  const toggleOrder = (orderId: string) => {
    setSelectedOrderIds(prev => {
      const next = new Set(prev);
      if (next.has(orderId)) {
        next.delete(orderId);
      } else {
        next.add(orderId);
      }
      return next;
    });
  };

  const toggleSelectAll = () => {
    if (allSelected) {
      setSelectedOrderIds(new Set());
      return;
    }
    setSelectedOrderIds(new Set(orders.map(order => order.order_id)));
  };

  const applyCapacitySuggestion = () => {
    const suggested = new Set(
      (capacityPrompt ?? []).flatMap(w => w.suggested_unselect_order_ids ?? []),
    );
    if (suggested.size > 0) {
      setSelectedOrderIds(prev => new Set([...prev].filter(id => !suggested.has(id))));
    }
    setCapacityPrompt(null);
  };

  const runManualDispatch = useCallback(async (forceCapacity = false) => {
    if (!selectedDriverId || selectedOrderIds.size === 0) {
      return;
    }
    setExecuting(true);
    setExecuteError(null);
    setExecuteSuccess(null);
    try {
      const routeFingerprint = JSON.stringify({
        mode: 'MANUAL',
        force_capacity: forceCapacity,
        routes: [{
          driver_id: selectedDriverId,
          order_ids: selectedOrderList,
        }],
      });
      const idempotencyKey = warehouseDispatchKey(warehouseId, selectedDriverId, routeFingerprint);
      const result = await warehouseApi.executeWarehouseDispatch({
        mode: 'MANUAL',
        force_capacity: forceCapacity,
        order_ids: selectedOrderList,
        routes: [{
          driver_id: selectedDriverId,
          order_ids: selectedOrderList,
        }],
      }, {}, idempotencyKey);
      if (result.warehouse_id) {
        setWarehouseId(result.warehouse_id);
      }
      if (result.status === 'capacity_exceeded' && result.capacity_warnings?.length) {
        setCapacityPromptMode('manual');
        setCapacityPrompt(result.capacity_warnings);
        return;
      }
      if (result.status === 'dispatched') {
        setExecuteSuccess(`Dispatched ${result.orders_assigned ?? selectedOrderIds.size} order(s). Payloader loading gate is active.`);
        setSelectedOrderIds(new Set());
        setCapacityPrompt(null);
        setLoading(true);
        await loadAll();
        return;
      }
      if (result.warnings?.length) {
        setExecuteError(result.warnings.join(', '));
      } else {
        setExecuteError('Dispatch did not commit. Refresh and try again.');
      }
    } catch (err) {
      setExecuteError(err instanceof ApiError ? err.message : 'Dispatch execute failed');
    } finally {
      setExecuting(false);
    }
  }, [loadAll, selectedDriverId, selectedOrderIds.size, selectedOrderList, warehouseId]);

  async function handleToggleVehicleAvailability(vehicle: WarehouseFleetVehicle, nextActive: boolean) {
    setMutatingVehicleId(vehicle.vehicle_id);
    setExecuteError(null);
    try {
      const unavailableReason = vehicleReasons[vehicle.vehicle_id] || vehicle.unavailable_reason || 'MANUAL_HOLD';
      const unavailableNote = vehicleNotes[vehicle.vehicle_id]?.trim() || '';
      const body: Record<string, unknown> = nextActive
        ? { is_active: true }
        : {
            is_active: false,
            unavailable_reason: unavailableReason,
            ...(unavailableReason === 'OTHER' && unavailableNote ? { unavailable_note: unavailableNote } : {}),
          };
      const res = await apiFetch(`/v1/warehouse/ops/vehicles/${vehicle.vehicle_id}`, {
        method: 'PATCH',
        body: JSON.stringify(body),
        headers: {
          'Idempotency-Key': warehouseUpdateVehicleKey(
            warehouseId,
            vehicle.vehicle_id,
            nextActive,
            nextActive ? undefined : unavailableReason,
          ),
        },
      });
      if (!res.ok) {
        throw new Error('Unable to update vehicle availability');
      }
      setFleetAlert(nextActive
        ? `${vehicle.label || vehicle.license_plate} restored for dispatch`
        : `${vehicle.label || vehicle.license_plate} marked unavailable`);
      await loadAll();
    } catch (err) {
      setExecuteError(err instanceof Error ? err.message : 'Vehicle update failed');
    } finally {
      setMutatingVehicleId('');
    }
  }

  const runSmartDispatch = useCallback(async (opts: { forceCapacity?: boolean; acceptPartial?: boolean } = {}) => {
    const orderIds = selectedOrderList.length > 0 ? selectedOrderList : orders.map(o => o.order_id);
    if (orderIds.length === 0) {
      return;
    }
    setExecuting(true);
    setExecuteError(null);
    setExecuteSuccess(null);
    setShowSmartConfirm(false);
    try {
      const routeFingerprint = JSON.stringify({
        mode: 'AUTO',
        order_ids: orderIds,
        force_capacity: Boolean(opts.forceCapacity),
        accept_partial: Boolean(opts.acceptPartial),
        plan_fingerprint: planFingerprint,
      });
      const idempotencyKey = warehouseDispatchKey(warehouseId, 'smart-dispatch', routeFingerprint);
      const result = await warehouseApi.executeWarehouseDispatch({
        mode: 'AUTO',
        order_ids: orderIds,
        force_capacity: opts.forceCapacity,
        accept_partial: opts.acceptPartial,
        plan_fingerprint: planFingerprint ?? undefined,
      }, {}, idempotencyKey);
      if (result.warehouse_id) {
        setWarehouseId(result.warehouse_id);
      }
      if (result.status === 'capacity_exceeded' && (result.capacity_warnings?.length || result.orphan_order_ids?.length)) {
        setCapacityPromptMode('auto');
        setCapacityPrompt(result.capacity_warnings ?? []);
        return;
      }
      if (result.status === 'dispatched') {
        const parts = [`Smart dispatch assigned ${result.orders_assigned ?? 0} order(s).`];
        if (result.orphan_order_ids?.length) {
          parts.push(`${result.orphan_order_ids.length} order(s) could not be assigned.`);
        }
        setExecuteSuccess(parts.join(' '));
        setSelectedOrderIds(new Set());
        setCapacityPrompt(null);
        setLoading(true);
        await loadAll();
        return;
      }
      if (result.warnings?.length) {
        setExecuteError(result.warnings.join(', '));
      } else {
        setExecuteError('Smart dispatch did not commit. Refresh and try again.');
      }
    } catch (err) {
      setExecuteError(err instanceof ApiError ? err.message : 'Smart dispatch failed');
    } finally {
      setExecuting(false);
    }
  }, [loadAll, orders, planFingerprint, selectedOrderList, warehouseId]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);
  const canDispatch = Boolean(selectedDriverId) && selectedOrderIds.size > 0 && !restricted;
  const suggestedUnselectIds = useMemo(
    () => new Set((capacityPrompt ?? []).flatMap(w => w.suggested_unselect_order_ids ?? [])),
    [capacityPrompt],
  );

  return (
    <PageTransition>
      <PageChrome
        title="Dispatch"
        description="Manual truck assignment or smart dispatch across the fleet. Capacity uses product VU × quantity."
        loading={loading}
        skeletonVariant="table"
        error={restricted ? 'You do not have permission to view dispatch for this scope.' : loadError}
        actions={
          <div className="flex items-center gap-2 flex-wrap">
            <select
              value={selectedDriverId}
              disabled={restricted || drivers.length === 0}
              onChange={event => setSelectedDriverId(event.target.value)}
              className="px-3 py-1.5 rounded-lg border text-sm min-w-48"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            >
              <option value="">Select truck / driver</option>
              {drivers.map(driver => (
                <option key={driver.driver_id} value={driver.driver_id}>
                  {(driver.vehicle_label || driver.name)}
                  {driver.free_volume_vu != null && driver.free_volume_vu > 0
                    ? ` — ${formatVU(driver.free_volume_vu)} VU free`
                    : ` — ${formatVU(driver.max_volume_vu ?? 0)} VU max`}
                </option>
              ))}
            </select>
            <button
              type="button"
              disabled={executing || !canDispatch}
              onClick={() => runManualDispatch(false)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary disabled:opacity-50"
            >
              <Icon name="dispatch" size={16} />
              {executing ? 'Dispatching…' : `Manual (${selectedOrderIds.size})`}
            </button>
            <button
              type="button"
              disabled={executing || !canSmartDispatch}
              onClick={() => setShowSmartConfirm(true)}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary disabled:opacity-50"
            >
              <Icon name="dispatch" size={16} />
              Smart Dispatch
            </button>
            <button
              type="button"
              onClick={() => {
                setLoading(true);
                void loadAll();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary"
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
          </div>
        }
      >
        {selectedDriver && (
          <div className="mb-4 rounded-xl border border-(--border) p-3 text-sm" style={{ background: 'var(--background)' }}>
            <span className="font-medium">{selectedDriver.name}</span>
            <span className="text-(--muted)"> — selected </span>
            <span className="font-mono">{formatVU(selectedVolumeVU)}</span>
            <span className="text-(--muted)"> / </span>
            <span className="font-mono">{formatVU(effectiveCapacityVU)}</span>
            <span className="text-(--muted)"> VU effective (95% buffer)</span>
            {selectedDriver.used_volume_vu != null && selectedDriver.used_volume_vu > 0 && (
              <span className="text-(--muted)"> · {formatVU(selectedDriver.used_volume_vu)} VU already on manifest</span>
            )}
          </div>
        )}

        {fleetEffectiveCapacityVU > 0 && (
          <div className="mb-4 rounded-xl border border-(--border) p-3 text-sm" style={{ background: 'var(--background)' }}>
            <span className="text-(--muted)">Fleet effective capacity </span>
            <span className="font-mono">{formatVU(fleetEffectiveCapacityVU)}</span>
            <span className="text-(--muted)"> VU · smart target </span>
            <span className="font-mono">{formatVU(smartTargetVolumeVU)}</span>
            <span className="text-(--muted)"> VU</span>
          </div>
        )}

        {executeError && (
          <p className="text-sm mb-4" style={{ color: 'var(--error)' }}>{executeError}</p>
        )}
        {executeSuccess && (
          <p className="text-sm mb-4" style={{ color: 'var(--success)' }}>{executeSuccess}</p>
        )}
        {fleetAlert && (
          <p className="text-sm mb-4" style={{ color: 'var(--warning)' }}>{fleetAlert}</p>
        )}

        <PageSection
          title={`Fleet trucks (${vehicles.length})`}
          description="Mark trucks unavailable in real time — dispatch and smart suggest exclude them immediately."
          className="mb-6"
        >
          {vehicles.length === 0 ? (
            <p className="text-sm text-(--muted)">No vehicles registered for this warehouse.</p>
          ) : (
            <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
              {vehicles.map(vehicle => (
                <div key={vehicle.vehicle_id} className="rounded-lg border border-(--border) p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div>
                      <div className="text-sm font-medium">{vehicle.label || vehicle.license_plate}</div>
                      <div className="text-xs text-(--muted)">{vehicle.license_plate} · {vehicle.vehicle_class}</div>
                      {!vehicle.is_active && (
                        <div className="text-xs mt-1" style={{ color: 'var(--warning)' }}>
                          {formatUnavailableReason(vehicle.unavailable_reason, vehicle.unavailable_note)}
                        </div>
                      )}
                    </div>
                    <span className={`status-chip ${vehicle.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                      {vehicle.is_active ? 'Active' : 'Unavailable'}
                    </span>
                  </div>
                  <div className="mt-3 flex flex-wrap items-center gap-2">
                    {vehicle.is_active && (
                      <>
                        <select
                          value={vehicleReasons[vehicle.vehicle_id] || vehicle.unavailable_reason || 'MANUAL_HOLD'}
                          onChange={event => setVehicleReasons(current => ({
                            ...current,
                            [vehicle.vehicle_id]: event.target.value as WarehouseVehicleUnavailableReason,
                          }))}
                          disabled={mutatingVehicleId === vehicle.vehicle_id}
                          className="rounded-lg border px-2 py-1 text-xs"
                          style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                        >
                          {VEHICLE_UNAVAILABLE_REASONS.map(reason => (
                            <option key={reason} value={reason}>{formatUnavailableReason(reason)}</option>
                          ))}
                        </select>
                        {(vehicleReasons[vehicle.vehicle_id] || 'MANUAL_HOLD') === 'OTHER' && (
                          <input
                            type="text"
                            placeholder="Custom reason"
                            value={vehicleNotes[vehicle.vehicle_id] || ''}
                            onChange={event => setVehicleNotes(current => ({
                              ...current,
                              [vehicle.vehicle_id]: event.target.value,
                            }))}
                            disabled={mutatingVehicleId === vehicle.vehicle_id}
                            className="rounded-lg border px-2 py-1 text-xs min-w-32"
                            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
                          />
                        )}
                      </>
                    )}
                    <button
                      type="button"
                      disabled={mutatingVehicleId === vehicle.vehicle_id}
                      onClick={() => void handleToggleVehicleAvailability(vehicle, !vehicle.is_active)}
                      className="rounded-lg px-2 py-1 text-xs font-medium disabled:opacity-50"
                      style={{
                        background: vehicle.is_active ? 'color-mix(in srgb, var(--warning) 15%, transparent)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
                        color: vehicle.is_active ? 'var(--warning)' : 'var(--success)',
                      }}
                    >
                      {mutatingVehicleId === vehicle.vehicle_id ? 'Updating…' : vehicle.is_active ? 'Mark unavailable' : 'Restore'}
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}
        </PageSection>

        <KpiStatGrid columns={4}>
          <KpiStatCard label="Undispatched orders" value={orders.length} sub="Awaiting assignment" />
          <KpiStatCard label="Available drivers" value={drivers.length} sub="Ready for dispatch" />
          <KpiStatCard
            label="Unavailable"
            value={unavailableDrivers.length}
            sub={unavailableDrivers.length > 0 ? 'Vehicle or status blocked' : 'All assigned drivers clear'}
          />
          <KpiStatCard
            label="Smart suggest routes"
            value={proposedRoutes.length}
            sub={optimizerSource ? `Source: ${optimizerSource}` : 'Optimizer preview'}
          />
        </KpiStatGrid>

        <PageSection
          title="Live fleet map"
          description="Sealed manifest polylines and driver GPS for this warehouse node."
          className="mt-6 overflow-hidden"
        >
          <FleetLiveMapPanel className="h-80 w-full -mx-5 -mb-5" />
        </PageSection>

        {(optimizerWarnings.length > 0 || windowConstrainedCount > 0) && (
          <PageSection title="Smart suggest preview" className="mt-6">
            {windowConstrainedCount > 0 && (
              <p className="text-xs" style={{ color: 'var(--warning)' }}>
                {windowConstrainedCount} order(s) constrained by receiving window
              </p>
            )}
            {optimizerWarnings.map(warning => (
              <p key={warning} className="text-xs" style={{ color: 'var(--warning)' }}>{warning}</p>
            ))}
          </PageSection>
        )}

        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 md-animate-in mt-6">
          <PageSection
            title={`Undispatched orders (${orders.length})`}
            actions={
              orders.length > 0 ? (
                <label className="flex items-center gap-2 text-xs text-(--muted) cursor-pointer">
                  <input type="checkbox" checked={allSelected} onChange={toggleSelectAll} />
                  Select all
                </label>
              ) : undefined
            }
          >
            {orders.length === 0 ? (
              <EmptyState variant="no-data" headline="All orders dispatched" body="No pending orders need assignment right now." />
            ) : (
              <div className="space-y-2 max-h-80 overflow-y-auto -mx-5 px-5">
                {orders.map(order => (
                  <label
                    key={order.order_id}
                    className="flex items-center gap-3 p-3 rounded-lg border border-(--border) cursor-pointer"
                  >
                    <input
                      type="checkbox"
                      checked={selectedOrderIds.has(order.order_id)}
                      onChange={() => toggleOrder(order.order_id)}
                    />
                    <div className="flex-1 flex items-center justify-between gap-3">
                      <div>
                        <div className="text-sm font-medium">{order.retailer_name || 'Unknown'}</div>
                        <div className="text-xs text-(--muted) font-mono">{order.order_id.slice(0, 8)}...</div>
                      </div>
                      <div className="text-right">
                        <div className="text-sm font-mono">{fmt(order.total_uzs)} UZS</div>
                        <div className="text-xs text-(--muted)">{formatVU(order.volume_vu ?? 0)} VU</div>
                      </div>
                    </div>
                  </label>
                ))}
              </div>
            )}
          </PageSection>

          <PageSection title={`Available drivers (${drivers.length})`}>
            <div className="space-y-4 max-h-80 overflow-y-auto">
              {drivers.length === 0 ? (
                <EmptyState variant="no-data" headline="No drivers available" body="All drivers are on route or blocked." />
              ) : (
                <div className="space-y-2">
                  {drivers.map(driver => (
                    <div key={driver.driver_id} className="flex items-center justify-between p-3 rounded-lg border border-(--border)">
                      <div>
                        <div className="text-sm font-medium">{driver.name}</div>
                        <div className="text-xs text-(--muted)">{driver.vehicle_label || driver.phone || 'Assigned vehicle'}</div>
                      </div>
                      <div className="text-right">
                        <span className="status-chip status-chip--stable">{driver.truck_status || 'IDLE'}</span>
                        <div className="text-xs text-(--muted) mt-1 font-mono">
                          {driver.free_volume_vu != null && driver.free_volume_vu > 0
                            ? `${formatVU(driver.free_volume_vu)} VU free`
                            : `${formatVU(driver.max_volume_vu ?? 0)} VU max`}
                        </div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <div className="border-t border-(--border) pt-4">
                <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-(--muted) mb-2">
                  Unavailable ({unavailableDrivers.length})
                </h3>
                {unavailableDrivers.length === 0 ? (
                  <p className="text-sm text-(--muted) py-2 text-center">No drivers blocked — all assigned trucks eligible or off-shift reasons shown here in real time.</p>
                ) : (
                  <div className="space-y-2">
                    {unavailableDrivers.map(driver => (
                      <div key={driver.driver_id} className="rounded-lg border border-(--border) p-3">
                        <div className="flex items-center justify-between gap-3">
                          <div>
                            <div className="text-sm font-medium">{driver.name}</div>
                            <div className="text-xs text-(--muted)">{driver.vehicle_label || driver.phone || 'Assigned vehicle unavailable'}</div>
                          </div>
                          <span className="status-chip status-chip--draft">{driver.truck_status || 'IDLE'}</span>
                        </div>
                        {driver.unavailable_reason && (
                          <div className="mt-2 text-xs" style={{ color: 'var(--warning)' }}>
                            {formatUnavailableReason(driver.unavailable_reason)}
                          </div>
                        )}
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          </PageSection>
        </div>

        {proposedRoutes.length > 0 && (
          <PageSection title={`Smart suggest routes (${proposedRoutes.length})`} className="mt-6">
            <DispatchPreviewMap
              routes={proposedRoutes}
              className="h-80 w-full rounded-lg border border-(--border) overflow-hidden"
            />
            <div className="space-y-3">
              {proposedRoutes.map((route, index) => (
                <div key={`${route.driver_id || 'route'}-${index}`} className="rounded-lg border border-(--border) p-3">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div className="text-sm font-medium">{route.driver_name || route.driver_id || 'Driver'}</div>
                    <div className="text-xs text-(--muted)">
                      {(route.stop_count ?? route.order_ids?.length ?? route.stops?.length ?? 0)} stops
                      {routeVolumeVU(route) > 0 && ` · ${formatVU(routeVolumeVU(route))} VU`}
                    </div>
                  </div>
                  <div className="text-xs text-(--muted) font-mono">
                    {(route.order_ids || route.stops?.map(stop => stop.order_id) || []).join(' → ')}
                  </div>
                </div>
              ))}
            </div>
          </PageSection>
        )}

        {showSmartConfirm && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
            <div className="w-full max-w-md rounded-xl border border-(--border) p-5" style={{ background: 'var(--background)' }}>
              <h3 className="text-base font-semibold mb-2">Run smart dispatch?</h3>
              <p className="text-sm text-(--muted) mb-4">
                {selectedOrderIds.size > 0
                  ? `Assign ${selectedOrderIds.size} selected order(s) using the optimizer.`
                  : `Assign all ${orders.length} undispatched order(s) using the optimizer.`}
                {proposedRoutes.length > 0 && ` ${proposedRoutes.length} route(s), ${smartStopCount} stop(s) in preview.`}
              </p>
              <div className="flex justify-end gap-2">
                <button type="button" className="px-3 py-1.5 rounded-lg text-sm button--secondary" onClick={() => setShowSmartConfirm(false)}>
                  Cancel
                </button>
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--primary"
                  disabled={executing}
                  onClick={() => void runSmartDispatch()}
                >
                  Smart Dispatch
                </button>
              </div>
            </div>
          </div>
        )}

        {capacityPrompt && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
            <div className="w-full max-w-md rounded-xl border border-(--border) p-5" style={{ background: 'var(--background)' }}>
              <h3 className="text-base font-semibold mb-2">Capacity exceeded</h3>
              <p className="text-sm text-(--muted) mb-4">
                {capacityPromptMode === 'auto'
                  ? 'Smart dispatch cannot fit all orders on available trucks. Accept to dispatch feasible routes and leave the rest undispatched, or force to override capacity.'
                  : 'Selected orders exceed the truck effective capacity (95% Tetris buffer). Apply the suggestion, force dispatch, or cancel.'}
              </p>
              <ul className="text-sm space-y-2 mb-4">
                {capacityPrompt.map(warning => (
                  <li key={warning.driver_id} className="font-mono text-xs">
                    {formatVU(warning.loaded_vu)} VU loaded / {formatVU(warning.effective_max_vu)} VU effective max
                    {warning.suggested_unselect_order_ids?.length ? (
                      <div className="mt-1 text-(--muted)">
                        Suggested unselect: {warning.suggested_unselect_order_ids.join(', ')}
                      </div>
                    ) : null}
                    {warning.suggested_defer_order_ids?.length ? (
                      <div className="mt-1 text-(--muted)">
                        Suggested defer: {warning.suggested_defer_order_ids.join(', ')}
                      </div>
                    ) : null}
                  </li>
                ))}
              </ul>
              <div className="flex justify-end gap-2 flex-wrap">
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                  onClick={() => setCapacityPrompt(null)}
                >
                  Cancel
                </button>
                {capacityPromptMode === 'manual' && suggestedUnselectIds.size > 0 && (
                  <button
                    type="button"
                    className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                    onClick={applyCapacitySuggestion}
                  >
                    Apply suggestion
                  </button>
                )}
                {capacityPromptMode === 'auto' && (
                  <button
                    type="button"
                    className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                    disabled={executing}
                    onClick={() => {
                      setCapacityPrompt(null);
                      void runSmartDispatch({ acceptPartial: true });
                    }}
                  >
                    Accept partial
                  </button>
                )}
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--primary"
                  disabled={executing}
                  onClick={() => {
                    setCapacityPrompt(null);
                    if (capacityPromptMode === 'auto') {
                      void runSmartDispatch({ forceCapacity: true });
                    } else {
                      void runManualDispatch(true);
                    }
                  }}
                >
                  Force dispatch
                </button>
              </div>
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
