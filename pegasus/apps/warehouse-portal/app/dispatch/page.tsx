'use client';

import { useEffect, useState, useCallback } from 'react';
import type {
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchPreview,
  WarehouseUnavailableDispatchDriver,
} from '@pegasus/types';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

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

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const res = await apiFetch('/v1/warehouse/ops/dispatch/preview');
      if (res.ok) {
        const data = await res.json() as WarehouseDispatchPreview;
        setOrders(data.undispatched_orders || data.orders || []);
        setDrivers(data.available_drivers || data.drivers || []);
        setUnavailableDrivers(data.unavailable_drivers || []);
        setRestricted(false);
      } else if (res.status === 403) {
        setRestricted(true);
        setOrders([]);
        setDrivers([]);
        setUnavailableDrivers([]);
      } else {
        const data = await res.json().catch(() => ({} as { error?: string }));
        setLoadError(data.error || 'Failed to load dispatch preview');
      }
    } catch {
      setLoadError('Failed to load dispatch preview');
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Dispatch Preview</h1>
        <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {restricted ? (
        <div className="rounded-xl border border-[var(--danger)]/30 bg-[var(--danger)]/8 p-4 text-sm text-[var(--danger)]">
          You do not have permission to view dispatch preview for this scope.
        </div>
      ) : null}
      {loadError ? (
        <div className="rounded-xl border border-[var(--warning)]/30 bg-[var(--warning)]/8 p-4 text-sm text-[var(--warning)]">
          {loadError}
        </div>
      ) : null}

      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="space-y-1">{Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}</div>
          <div className="space-y-1">{Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}</div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
      )}
    </div>
  );
}
