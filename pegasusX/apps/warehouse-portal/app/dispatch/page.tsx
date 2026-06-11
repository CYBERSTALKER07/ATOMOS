'use client';

import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseUnavailableDispatchDriver,
} from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

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

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const data = await warehouseApi.previewWarehouseDispatch();
      setOrders(data.undispatched_orders || data.orders || []);
      setDrivers(data.available_drivers || data.drivers || []);
      setUnavailableDrivers(data.unavailable_drivers || []);
      setProposedRoutes(data.proposed_routes || []);
      setOptimizerSource(data.optimizer_source || null);
      setOptimizerWarnings(data.optimizer_warnings || []);
      setWindowConstrainedCount(data.window_constrained_count || 0);
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

  const runAutoDispatch = useCallback(async () => {
    setExecuting(true);
    setExecuteError(null);
    setExecuteSuccess(null);
    try {
      await warehouseApi.executeWarehouseDispatch({ mode: 'AUTO' });
      setExecuteSuccess('Dispatch committed. Payloader loading gate is now active.');
      setLoading(true);
      await load();
    } catch (err) {
      setExecuteError(err instanceof ApiError ? err.message : 'Dispatch execute failed');
    } finally {
      setExecuting(false);
    }
  }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <PageTransition>
      <PageChrome
        title="Dispatch preview"
        description="Undispatched orders and driver availability for this warehouse node."
        loading={loading}
        error={restricted ? 'You do not have permission to view dispatch preview for this scope.' : loadError}
        actions={
          <div className="flex items-center gap-2">
            <button
              type="button"
              disabled={executing || restricted || orders.length === 0}
              onClick={runAutoDispatch}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--primary disabled:opacity-50"
            >
              <Icon name="dispatch" size={16} />
              {executing ? 'Dispatching…' : 'Auto dispatch'}
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
        {executeError && (
          <p className="text-sm mb-4" style={{ color: 'var(--error)' }}>{executeError}</p>
        )}
        {executeSuccess && (
          <p className="text-sm mb-4" style={{ color: 'var(--success)' }}>{executeSuccess}</p>
        )}
        {(optimizerSource || optimizerWarnings.length > 0 || windowConstrainedCount > 0) && (
          <div className="mb-4 rounded-xl border border-(--border) p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-2">Optimizer</h2>
            {optimizerSource && (
              <p className="text-xs text-(--muted)">Source: {optimizerSource}</p>
            )}
            {windowConstrainedCount > 0 && (
              <p className="text-xs" style={{ color: 'var(--warning)' }}>
                {windowConstrainedCount} order(s) constrained by receiving window
              </p>
            )}
            {optimizerWarnings.map(warning => (
              <p key={warning} className="text-xs" style={{ color: 'var(--warning)' }}>{warning}</p>
            ))}
          </div>
        )}
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6 md-animate-in">
          {/* Undispatched Orders */}
          <div className="rounded-xl border border-(--border) p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <Icon name="orders" size={16} className="text-(--muted)" />
              Undispatched Orders ({orders.length})
            </h2>
            {orders.length === 0 ? (
              <p className="text-sm text-(--muted) py-6 text-center">All orders dispatched</p>
            ) : (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {orders.map(o => (
                  <div key={o.order_id} className="flex items-center justify-between p-3 rounded-lg border border-(--border)">
                    <div>
                      <div className="text-sm font-medium">{o.retailer_name || 'Unknown'}</div>
                      <div className="text-xs text-(--muted) font-mono">{o.order_id.slice(0, 8)}...</div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-mono">{fmt(o.total_uzs)} UZS</div>
                      <div className="text-xs text-(--muted)">{o.created_at ? new Date(o.created_at).toLocaleDateString() : '—'}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Available Drivers */}
          <div className="rounded-xl border border-(--border) p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <Icon name="fleet" size={16} className="text-(--muted)" />
              Available Drivers ({drivers.length})
            </h2>
            <div className="space-y-4 max-h-80 overflow-y-auto">
              {drivers.length === 0 ? (
                <p className="text-sm text-(--muted) py-2 text-center">No drivers available</p>
              ) : (
                <div className="space-y-2">
                  {drivers.map(d => (
                    <div key={d.driver_id} className="flex items-center justify-between p-3 rounded-lg border border-(--border)">
                      <div>
                        <div className="text-sm font-medium">{d.name}</div>
                        <div className="text-xs text-(--muted)">{d.vehicle_label || d.phone || 'Assigned vehicle'}</div>
                      </div>
                      <span className="status-chip status-chip--stable">{d.truck_status || 'IDLE'}</span>
                    </div>
                  ))}
                </div>
              )}

              <div className="border-t border-(--border) pt-4">
                <h3 className="text-xs font-semibold uppercase tracking-[0.16em] text-(--muted) mb-2">
                  Vehicle Unavailable ({unavailableDrivers.length})
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
          </div>
        </div>

        {proposedRoutes.length > 0 && (
          <div className="mt-6 rounded-xl border border-(--border) p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <Icon name="dispatch" size={16} className="text-(--muted)" />
              Proposed routes ({proposedRoutes.length})
            </h2>
            <div className="space-y-3">
              {proposedRoutes.map((route, index) => (
                <div key={`${route.driver_id || 'route'}-${index}`} className="rounded-lg border border-(--border) p-3">
                  <div className="flex items-center justify-between gap-3 mb-2">
                    <div className="text-sm font-medium">{route.driver_name || route.driver_id || 'Driver'}</div>
                    <div className="text-xs text-(--muted)">
                      {(route.stop_count ?? route.order_ids?.length ?? route.stops?.length ?? 0)} stops
                    </div>
                  </div>
                  <div className="text-xs text-(--muted) font-mono">
                    {(route.order_ids || route.stops?.map(stop => stop.order_id) || []).join(' → ')}
                  </div>
                </div>
              ))}
            </div>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
