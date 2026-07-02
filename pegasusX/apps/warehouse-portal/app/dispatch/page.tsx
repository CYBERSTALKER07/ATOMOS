'use client';

import { useEffect, useState, useCallback, useMemo } from 'react';
import { useRouter } from 'next/navigation';
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
import { ExplainStatusBanner, explainFromApiError } from '@pegasusx/explain-ui';
import type { StatusExplain } from '@pegasusx/types';
import { isTauri } from '@pegasusx/desktop-bridge';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import { apiFetch } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import {
  dispatchPreviewCacheKey,
  getDispatchPreviewCache,
  setDispatchPreviewCache,
  snapshotFromPreviewResponse,
  type DispatchPreviewSnapshot,
} from '@/lib/dispatch-preview-cache';
import {
  dispatchRunsCacheKey,
  getDispatchRunsCache,
  setDispatchRunsCache,
  type DispatchRunRow,
} from '@/lib/dispatch-runs-cache';
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
import HandoffTimelinePanel from '@/components/HandoffTimelinePanel';
import EmptyState from '@/components/EmptyState';
import { OrderActionDialog, OrderOpsCard, OrderProposeDateDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { warehouseOps } from '@/lib/warehouse-ops';

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
type DispatchMode = 'manual' | 'smart';

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

function isoDeliveryDate(dateInput: string): string {
  const dateOnly = dateInput.slice(0, 10);
  return `${dateOnly}T12:00:00+05:00`;
}

export default function DispatchPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [orders, setOrders] = useState<WarehouseDispatchOrder[]>([]);
  const [drivers, setDrivers] = useState<WarehouseDispatchDriver[]>([]);
  const [unavailableDrivers, setUnavailableDrivers] = useState<WarehouseUnavailableDispatchDriver[]>([]);
  const [loading, setLoading] = useState(true);
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [executing, setExecuting] = useState(false);
  const [executeError, setExecuteError] = useState<string | null>(null);
  const [executeExplain, setExecuteExplain] = useState<StatusExplain | null>(null);
  const [executeSuccess, setExecuteSuccess] = useState<string | null>(null);
  const [dispatchRuns, setDispatchRuns] = useState<DispatchRunRow[]>([]);
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
  const [dispatchMode, setDispatchMode] = useState<DispatchMode>('smart');
  const [showSmartConfirm, setShowSmartConfirm] = useState(false);
  const [warehouseId, setWarehouseId] = useState(() => warehouseHomeNodeId() || 'warehouse');
  const [vehicles, setVehicles] = useState<WarehouseFleetVehicle[]>([]);
  const [vehicleReasons, setVehicleReasons] = useState<Record<string, WarehouseVehicleUnavailableReason>>({});
  const [vehicleNotes, setVehicleNotes] = useState<Record<string, string>>({});
  const [mutatingVehicleId, setMutatingVehicleId] = useState('');
  const [fleetAlert, setFleetAlert] = useState<string | null>(null);
  const [opsDialog, setOpsDialog] = useState<{ orderId: string; kind: 'propose' | 'reject' } | null>(null);
  const [opsReason, setOpsReason] = useState('');
  const [opsProposedDate, setOpsProposedDate] = useState(() => new Date().toISOString().slice(0, 10));
  const [opsActingId, setOpsActingId] = useState<string | null>(null);

  const selectedOrderList = useMemo(() => [...selectedOrderIds].sort(), [selectedOrderIds]);

  const closeOpsDialog = () => {
    setOpsDialog(null);
    setOpsReason('');
    setOpsProposedDate(new Date().toISOString().slice(0, 10));
  };

  async function submitOpsDialog() {
    if (!opsDialog) return;
    const trimmed = opsReason.trim();
    setOpsActingId(opsDialog.orderId);
    try {
      if (opsDialog.kind === 'reject') {
        if (!trimmed) {
          toast('Reason is required', 'error');
          return;
        }
        const resp = await warehouseOps.rejectOrder(opsDialog.orderId, trimmed);
        toast(`Order cancelled · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      } else {
        if (!opsProposedDate || !trimmed) {
          toast('Proposed date and reason are required', 'error');
          return;
        }
        const resp = await warehouseOps.proposeOrderDelivery(
          opsDialog.orderId,
          isoDeliveryDate(opsProposedDate),
          trimmed,
        );
        toast(`New delivery date proposed · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      }
      closeOpsDialog();
      await load();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Action failed', 'error');
    } finally {
      setOpsActingId(null);
    }
  }

  const applyDispatchSnapshot = useCallback((snapshot: DispatchPreviewSnapshot) => {
    const nextOrders = snapshot.orders;
    setOrders(nextOrders);
    setDrivers(snapshot.drivers);
    setUnavailableDrivers(snapshot.unavailableDrivers);
    setProposedRoutes(snapshot.proposedRoutes);
    setOptimizerSource(snapshot.optimizerSource);
    setOptimizerWarnings(snapshot.optimizerWarnings);
    setWindowConstrainedCount(snapshot.windowConstrainedCount);
    setFleetEffectiveCapacityVU(snapshot.fleetEffectiveCapacityVU);
    setPlanFingerprint(snapshot.planFingerprint);
    setSelectedOrderIds(prev => {
      const valid = new Set(nextOrders.map(order => order.order_id));
      return new Set([...prev].filter(id => valid.has(id)));
    });
  }, []);

  const load = useCallback(async (orderFilter?: string[]) => {
    const scopedWarehouseId = warehouseHomeNodeId();
    if (scopedWarehouseId) {
      setWarehouseId(scopedWarehouseId);
    }
    const filter = orderFilter ?? selectedOrderList;
    const cacheKey = dispatchPreviewCacheKey(
      scopedWarehouseId || warehouseId,
      filter,
    );
    let hydratedFromCache = false;
    if (isTauri()) {
      const cached = await getDispatchPreviewCache(cacheKey);
      if (cached) {
        applyDispatchSnapshot(cached);
        setLoading(false);
        hydratedFromCache = true;
      }
    }
    setLoadError(null);
    try {
      const body = filter.length > 0 ? { order_ids: filter } : {};
      const data = await warehouseApi.previewWarehouseDispatch({}, body);
      const snapshot = snapshotFromPreviewResponse(data);
      applyDispatchSnapshot(snapshot);
      if (isTauri()) {
        void setDispatchPreviewCache(cacheKey, snapshot);
      }
      setRestricted(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setRestricted(true);
        if (!hydratedFromCache) {
          setOrders([]);
          setDrivers([]);
          setUnavailableDrivers([]);
        }
      } else if (!hydratedFromCache) {
        setLoadError(err instanceof ApiError ? err.message : 'Failed to load dispatch preview');
      }
    } finally {
      setLoading(false);
    }
  }, [applyDispatchSnapshot, selectedOrderList, warehouseId]);

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

  const loadDispatchRuns = useCallback(async () => {
    const scopedWarehouseId = warehouseHomeNodeId() || warehouseId;
    const cacheKey = dispatchRunsCacheKey(scopedWarehouseId);
    let hydratedFromCache = false;
    if (isTauri()) {
      const cached = await getDispatchRunsCache(cacheKey);
      if (cached) {
        setDispatchRuns(cached);
        hydratedFromCache = true;
      }
    }
    try {
      const data = await apiFetch<{ runs: DispatchRunRow[] }>(
        '/v1/warehouse/ops/dispatch/runs',
      );
      const runs = data.runs ?? [];
      setDispatchRuns(runs);
      if (isTauri()) {
        void setDispatchRunsCache(cacheKey, runs);
      }
    } catch {
      if (!hydratedFromCache) {
        setDispatchRuns([]);
      }
    }
  }, [warehouseId]);

  const loadAll = useCallback(async (orderFilter?: string[]) => {
    await Promise.all([load(orderFilter), loadVehicles(), loadDispatchRuns()]);
  }, [load, loadVehicles, loadDispatchRuns]);

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
      setExecuteExplain(null);
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
  const canSmartDispatch = dispatchMode === 'smart' && !restricted && orders.length > 0 && drivers.length > 0;
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
      if (result.status === 'plan_stale') {
        setExecuteError('Dispatch plan is stale — refresh preview and try again.');
        setLoading(true);
        await loadAll();
        return;
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
      if (err instanceof ApiError) {
        setExecuteError(err.message);
        setExecuteExplain(explainFromApiError(err.body));
      } else {
        setExecuteExplain(null);
        setExecuteError(err instanceof Error ? err.message : 'Smart dispatch failed');
      }
    } finally {
      setExecuting(false);
    }
  }, [loadAll, orders, planFingerprint, selectedOrderList, warehouseId]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);
  const canDispatch = dispatchMode === 'manual' && Boolean(selectedDriverId) && selectedOrderIds.size > 0 && !restricted;
  const suggestedUnselectIds = useMemo(
    () => new Set((capacityPrompt ?? []).flatMap(w => w.suggested_unselect_order_ids ?? [])),
    [capacityPrompt],
  );

  return (
    <PageTransition>
      <PageChrome
        icon="dispatch"
        title="Dispatch"
        description="Manual truck assignment or smart dispatch across the fleet. Capacity uses product VU × quantity."
        loading={loading}
        skeletonVariant="table"
        error={restricted ? 'You do not have permission to view dispatch for this scope.' : loadError}
        actions={
          <div className="flex items-center gap-2 flex-wrap">
            <div className="inline-flex rounded-lg border p-0.5" style={{ borderColor: 'var(--field-border)', background: 'var(--field-background)' }}>
              <button
                type="button"
                onClick={() => setDispatchMode('smart')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${dispatchMode === 'smart' ? 'button--primary' : 'button--ghost'}`}
              >
                Smart fleet
              </button>
              <button
                type="button"
                onClick={() => setDispatchMode('manual')}
                className={`px-3 py-1.5 rounded-md text-sm font-medium transition-colors ${dispatchMode === 'manual' ? 'button--primary' : 'button--ghost'}`}
              >
                Manual truck
              </button>
            </div>
            <select
              value={selectedDriverId}
              disabled={restricted || drivers.length === 0 || dispatchMode === 'smart'}
              onChange={event => setSelectedDriverId(event.target.value)}
              className="px-3 py-1.5 rounded-lg border text-sm min-w-48 disabled:opacity-50"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            >
              <option value="">{dispatchMode === 'smart' ? 'Fleet auto-assign' : 'Select truck / driver'}</option>
              {drivers.map(driver => (
                <option key={driver.driver_id} value={driver.driver_id}>
                  {(driver.vehicle_label || driver.name)}
                  {driver.free_volume_vu != null && driver.free_volume_vu > 0
                    ? ` — ${formatVU(driver.free_volume_vu)} VU free`
                    : ` — ${formatVU(driver.max_volume_vu ?? 0)} VU max`}
                </option>
              ))}
            </select>
            {dispatchMode === 'manual' ? (
              <button
                type="button"
                disabled={executing || !canDispatch}
                onClick={() => runManualDispatch(false)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary disabled:opacity-50"
              >
                <Icon name="dispatch" size={16} />
                {executing ? 'Dispatching…' : `Manual (${selectedOrderIds.size})`}
              </button>
            ) : (
              <button
                type="button"
                disabled={executing || !canSmartDispatch}
                onClick={() => setShowSmartConfirm(true)}
                className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary disabled:opacity-50"
              >
                <Icon name="dispatch" size={16} />
                Smart Dispatch
              </button>
            )}
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
        {dispatchMode === 'manual' && selectedDriver && (
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
          <div className="mb-4">
            <ExplainStatusBanner explain={executeExplain} className="mb-2" />
            <p className="text-sm" style={{ color: 'var(--error)' }}>{executeError}</p>
          </div>
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
          title="Recent dispatch commits"
          description="Replay the last committed smart-dispatch runs for this node."
          className="mt-6"
        >
          {dispatchRuns.length === 0 ? (
            <p className="text-sm text-(--muted)">No dispatch runs recorded yet.</p>
          ) : (
            <div className="space-y-2">
              {dispatchRuns.slice(0, 8).map((run) => (
                <div key={run.run_id} className="rounded-lg border border-(--border) p-3 text-sm">
                  <div className="font-medium">{run.status}</div>
                  <div className="text-(--muted)">
                    {run.orders_assigned} orders · {run.manifest_count} manifests · {new Date(run.created_at).toLocaleString()}
                  </div>
                </div>
              ))}
            </div>
          )}
        </PageSection>

        <PageSection
          title="Handoff timeline"
          description="Preorder → accept → dispatch → seal events from the ops pulse feed."
          className="mt-6"
        >
          <HandoffTimelinePanel />
        </PageSection>

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
            description="Select for dispatch. Double-click a card for order detail."
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
              <VirtualScrollList
                className="-mx-5 px-5"
                height="28rem"
                items={orders}
                itemKey={(order) => order.order_id}
                renderItem={(order, index) => (
                  <div className="flex items-start gap-2 pb-3">
                    <label className="flex items-center pt-4 shrink-0 cursor-pointer">
                      <input
                        type="checkbox"
                        checked={selectedOrderIds.has(order.order_id)}
                        onChange={() => toggleOrder(order.order_id)}
                      />
                    </label>
                    <div className="flex-1 min-w-0">
                      <OrderOpsCard
                        orderId={order.order_id}
                        retailerName={order.retailer_name || 'Unknown'}
                        state="PENDING"
                        amountLabel={`${fmt(order.total_uzs)} UZS · ${formatVU(order.volume_vu ?? 0)} VU`}
                        index={index}
                        disabled={opsActingId === order.order_id}
                        detailOpenMode="double"
                        onOpenDetail={() => router.push(`/orders/${order.order_id}?from=dispatch`)}
                        onProposeDate={() => {
                          setOpsDialog({ orderId: order.order_id, kind: 'propose' });
                          setOpsReason('');
                          setOpsProposedDate(new Date().toISOString().slice(0, 10));
                        }}
                        onReject={() => {
                          setOpsDialog({ orderId: order.order_id, kind: 'reject' });
                          setOpsReason('');
                        }}
                      />
                    </div>
                  </div>
                )}
              />
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
                    <div className="text-sm font-medium">
                      {route.driver_name || route.driver_id || 'Driver'}
                      {(() => {
                        const driver = drivers.find(d => d.driver_id === route.driver_id);
                        if (!driver) return null;
                        const capacity = driver.free_volume_vu != null && driver.free_volume_vu > 0 ? driver.free_volume_vu : driver.max_volume_vu ?? 0;
                        if (capacity > 0 && routeVolumeVU(route) > capacity * TETRIS_BUFFER) {
                          return (
                            <span className="ml-2 inline-flex items-center px-1.5 py-0.5 rounded text-[10px] font-medium bg-[color-mix(in_srgb,var(--warning)_15%,transparent)] text-[var(--warning)] border border-[color-mix(in_srgb,var(--warning)_30%,transparent)]">
                              Over capacity
                            </span>
                          );
                        }
                        return null;
                      })()}
                    </div>
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

      {opsDialog ? (
        <>
          <OrderActionDialog
            open={opsDialog.kind === 'reject'}
            title="Cancel order"
            description="Cancels the order and notifies the retailer immediately."
            confirmLabel="Cancel order"
            destructive
            reason={opsReason}
            onReasonChange={setOpsReason}
            reasonRequired
            submitting={opsActingId === opsDialog.orderId}
            onConfirm={() => void submitOpsDialog()}
            onClose={closeOpsDialog}
          />
          <OrderProposeDateDialog
            open={opsDialog.kind === 'propose'}
            proposedDate={opsProposedDate}
            onProposedDateChange={setOpsProposedDate}
            reason={opsReason}
            onReasonChange={setOpsReason}
            submitting={opsActingId === opsDialog.orderId}
            onConfirm={() => void submitOpsDialog()}
            onClose={closeOpsDialog}
          />
        </>
      ) : null}
    </PageTransition>
  );
}
