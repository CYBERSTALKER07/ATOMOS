'use client';

import { useEffect, useState, useCallback, useMemo } from 'react';
import type {
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchPreview,
  WarehouseUnavailableDispatchDriver,
} from '@pegasus/types';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import VuCapacityBar from '@/components/VuCapacityBar';

interface BoardOrder {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at?: string;
}

const KANBAN_COLUMNS = ['PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED'] as const;

function formatUnavailableReason(reason?: string) {
  if (!reason) return '';
  return reason
    .toLowerCase()
    .split('_')
    .map(part => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

export default function DispatchPage() {
  const [orders, setOrders] = useState<WarehouseDispatchOrder[]>([]);
  const [boardOrders, setBoardOrders] = useState<BoardOrder[]>([]);
  const [drivers, setDrivers] = useState<WarehouseDispatchDriver[]>([]);
  const [unavailableDrivers, setUnavailableDrivers] = useState<WarehouseUnavailableDispatchDriver[]>([]);
  const [loading, setLoading] = useState(true);
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);
  const [view, setView] = useState<'list' | 'kanban'>('kanban');

  const load = useCallback(async () => {
    setLoadError(null);
    try {
      const [previewRes, ordersRes] = await Promise.all([
        apiFetch('/v1/warehouse/ops/dispatch/preview'),
        apiFetch('/v1/warehouse/ops/orders'),
      ]);

      if (previewRes.ok) {
        const data = await previewRes.json() as WarehouseDispatchPreview;
        setOrders(data.undispatched_orders || data.orders || []);
        setDrivers(data.available_drivers || data.drivers || []);
        setUnavailableDrivers(data.unavailable_drivers || []);
        setRestricted(false);
      } else if (previewRes.status === 403) {
        setRestricted(true);
        setOrders([]);
        setDrivers([]);
        setUnavailableDrivers([]);
      } else {
        const data = await previewRes.json().catch(() => ({} as { error?: string }));
        setLoadError(data.error || 'Failed to load dispatch preview');
      }

      if (ordersRes.ok) {
        const data = await ordersRes.json() as { orders?: BoardOrder[] };
        setBoardOrders((data.orders || []).filter(o => KANBAN_COLUMNS.includes(o.state as typeof KANBAN_COLUMNS[number])));
      }
    } catch {
      setLoadError('Failed to load dispatch preview');
    }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const ordersByColumn = useMemo(() => {
    const map: Record<string, BoardOrder[]> = {};
    for (const col of KANBAN_COLUMNS) map[col] = [];
    for (const order of boardOrders) {
      if (map[order.state]) map[order.state].push(order);
    }
    return map;
  }, [boardOrders]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-light tracking-tight">Dispatch Board</h1>
        <div className="flex items-center gap-2">
          <div className="flex rounded-lg border border-(--border) overflow-hidden">
            {(['kanban', 'list'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => setView(mode)}
                className="px-3 py-1.5 text-sm capitalize"
                style={{
                  background: view === mode ? 'var(--accent)' : 'transparent',
                  color: view === mode ? 'var(--accent-foreground)' : 'var(--muted)',
                }}
              >
                {mode}
              </button>
            ))}
          </div>
          <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        </div>
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
        <div className="grid grid-cols-1 lg:grid-cols-4 gap-4">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton h-48 rounded-xl" />)}
        </div>
      ) : view === 'kanban' ? (
        <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
          {KANBAN_COLUMNS.map((col) => (
            <div key={col} className="rounded-xl border border-(--border) p-3 min-h-48" style={{ background: 'var(--surface)' }}>
              <div className="flex items-center justify-between mb-3">
                <h2 className="text-xs font-semibold uppercase tracking-wider text-(--muted)">{col.replace(/_/g, ' ')}</h2>
                <span className="text-xs font-mono px-2 py-0.5 rounded-full bg-(--background)">{ordersByColumn[col]?.length ?? 0}</span>
              </div>
              <div className="space-y-2 max-h-96 overflow-y-auto">
                {(ordersByColumn[col] || []).length === 0 ? (
                  <p className="text-xs text-(--muted) py-4 text-center">No orders</p>
                ) : (
                  (ordersByColumn[col] || []).map((o) => (
                    <div key={o.order_id} className="p-3 rounded-lg border border-(--border) bg-(--background)">
                      <div className="text-sm font-medium">{o.retailer_name || 'Unknown'}</div>
                      <div className="text-xs text-(--muted) font-mono mt-1">{o.order_id.slice(0, 8)}…</div>
                      <div className="text-xs font-mono mt-2">{fmt(o.total_uzs)} UZS</div>
                    </div>
                  ))
                )}
              </div>
            </div>
          ))}
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
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
                    <div key={d.driver_id} className="p-3 rounded-lg border border-(--border) space-y-2">
                      <div className="flex items-center justify-between">
                        <div>
                          <div className="text-sm font-medium">{d.name}</div>
                          <div className="text-xs text-(--muted)">{d.vehicle_label || d.phone || 'Assigned vehicle'}</div>
                        </div>
                        <span className="status-chip status-chip--stable">{d.truck_status || 'IDLE'}</span>
                      </div>
                      {d.max_volume_vu ? (
                        <VuCapacityBar used={0} max={d.max_volume_vu} label="VU capacity" compact />
                      ) : null}
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
