'use client';

import { useEffect, useState, useCallback, useMemo } from 'react';
import type {
  WarehouseDispatchCapacityWarning,
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseUnavailableDispatchDriver,
} from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import DispatchPreviewMap from '@/components/DispatchPreviewMap';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';

const TETRIS_BUFFER = 0.95;

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

function formatVU(value: number) {
  return value.toFixed(1);
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
  const [selectedDriverId, setSelectedDriverId] = useState('');
  const [selectedOrderIds, setSelectedOrderIds] = useState<Set<string>>(new Set());
  const [capacityPrompt, setCapacityPrompt] = useState<WarehouseDispatchCapacityWarning[] | null>(null);

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const data = await warehouseApi.previewWarehouseDispatch();
      const nextOrders = data.undispatched_orders || data.orders || [];
      setOrders(nextOrders);
      setDrivers(data.available_drivers || data.drivers || []);
      setUnavailableDrivers(data.unavailable_drivers || []);
      setProposedRoutes(data.proposed_routes || []);
      setOptimizerSource(data.optimizer_source || null);
      setOptimizerWarnings(data.optimizer_warnings || []);
      setWindowConstrainedCount(data.window_constrained_count || 0);
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
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

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
    const max = selectedDriver?.max_volume_vu ?? 0;
    return max > 0 ? max * TETRIS_BUFFER : 0;
  }, [selectedDriver]);

  const allSelected = orders.length > 0 && selectedOrderIds.size === orders.length;

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

  const runManualDispatch = useCallback(async (forceCapacity = false) => {
    if (!selectedDriverId || selectedOrderIds.size === 0) {
      return;
    }
    setExecuting(true);
    setExecuteError(null);
    setExecuteSuccess(null);
    try {
      const result = await warehouseApi.executeWarehouseDispatch({
        mode: 'MANUAL',
        force_capacity: forceCapacity,
        routes: [{
          driver_id: selectedDriverId,
          order_ids: [...selectedOrderIds],
        }],
      }, {}, crypto.randomUUID());
      if (result.status === 'capacity_exceeded' && result.capacity_warnings?.length) {
        setCapacityPrompt(result.capacity_warnings);
        return;
      }
      if (result.status === 'dispatched') {
        setExecuteSuccess(`Dispatched ${result.orders_assigned ?? selectedOrderIds.size} order(s). Payloader loading gate is active.`);
        setSelectedOrderIds(new Set());
        setCapacityPrompt(null);
        setLoading(true);
        await load();
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
  }, [load, selectedDriverId, selectedOrderIds]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);
  const canDispatch = Boolean(selectedDriverId) && selectedOrderIds.size > 0 && !restricted;

  return (
    <PageTransition>
      <PageChrome
        title="Dispatch"
        description="Assign undispatched orders to an available truck. Capacity uses product VU × quantity."
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
                  {(driver.vehicle_label || driver.name)} — {formatVU(driver.max_volume_vu ?? 0)} VU max
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
              {executing ? 'Dispatching…' : `Dispatch (${selectedOrderIds.size})`}
            </button>
            <button
              type="button"
              onClick={() => {
                setLoading(true);
                load();
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
            <span className="text-(--muted)"> — loaded </span>
            <span className="font-mono">{formatVU(selectedVolumeVU)}</span>
            <span className="text-(--muted)"> / </span>
            <span className="font-mono">{formatVU(effectiveCapacityVU)}</span>
            <span className="text-(--muted)"> VU effective (95% buffer)</span>
          </div>
        )}
        {executeError && (
          <p className="text-sm mb-4" style={{ color: 'var(--error)' }}>{executeError}</p>
        )}
        {executeSuccess && (
          <p className="text-sm mb-4" style={{ color: 'var(--success)' }}>{executeSuccess}</p>
        )}

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
                        <div className="text-xs text-(--muted) mt-1 font-mono">{formatVU(driver.max_volume_vu ?? 0)} VU</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}

              <div className="border-t border-(--border) pt-4">
                <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-(--muted) mb-2">
                  Vehicle unavailable ({unavailableDrivers.length})
                </h3>
                {unavailableDrivers.length === 0 ? (
                  <p className="text-sm text-(--muted) py-2 text-center">No assigned drivers blocked by vehicle availability</p>
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
                      {route.volume_vu != null && ` · ${formatVU(route.volume_vu)} VU`}
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

        {capacityPrompt && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4" style={{ background: 'rgba(0,0,0,0.45)' }}>
            <div className="w-full max-w-md rounded-xl border border-(--border) p-5" style={{ background: 'var(--background)' }}>
              <h3 className="text-base font-semibold mb-2">Capacity exceeded</h3>
              <p className="text-sm text-(--muted) mb-4">
                Selected orders exceed the truck&apos;s effective capacity (95% Tetris buffer). Confirm only if you intend to overload this manifest.
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
                  </li>
                ))}
              </ul>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--secondary"
                  onClick={() => setCapacityPrompt(null)}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="px-3 py-1.5 rounded-lg text-sm button--primary"
                  disabled={executing}
                  onClick={() => {
                    setCapacityPrompt(null);
                    void runManualDispatch(true);
                  }}
                >
                  Dispatch anyway
                </button>
              </div>
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
