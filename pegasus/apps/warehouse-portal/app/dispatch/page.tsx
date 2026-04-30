'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface DispatchOrder {
  order_id: string;
  retailer_name: string;
  total_uzs: number;
  created_at: string;
}

interface AvailableDriver {
  driver_id: string;
  name: string;
  phone: string;
  truck_status: string;
}

export default function DispatchPage() {
  const [orders, setOrders] = useState<DispatchOrder[]>([]);
  const [drivers, setDrivers] = useState<AvailableDriver[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/dispatch/preview');
      if (res.ok) {
        const data = await res.json();
        setOrders(data.undispatched_orders || []);
        setDrivers(data.available_drivers || []);
      }
    } catch { /* handled */ }
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

      {loading ? (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div className="space-y-1">{Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}</div>
          <div className="space-y-1">{Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}</div>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          {/* Undispatched Orders */}
          <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <Icon name="orders" size={16} className="text-[var(--muted)]" />
              Undispatched Orders ({orders.length})
            </h2>
            {orders.length === 0 ? (
              <p className="text-sm text-[var(--muted)] py-6 text-center">All orders dispatched</p>
            ) : (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {orders.map(o => (
                  <div key={o.order_id} className="flex items-center justify-between p-3 rounded-lg border border-[var(--border)]">
                    <div>
                      <div className="text-sm font-medium">{o.retailer_name || 'Unknown'}</div>
                      <div className="text-xs text-[var(--muted)] font-mono">{o.order_id.slice(0, 8)}...</div>
                    </div>
                    <div className="text-right">
                      <div className="text-sm font-mono">{fmt(o.total_uzs)} UZS</div>
                      <div className="text-xs text-[var(--muted)]">{new Date(o.created_at).toLocaleDateString()}</div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>

          {/* Available Drivers */}
          <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
            <h2 className="text-sm font-semibold mb-3 flex items-center gap-2">
              <Icon name="fleet" size={16} className="text-[var(--muted)]" />
              Available Drivers ({drivers.length})
            </h2>
            {drivers.length === 0 ? (
              <p className="text-sm text-[var(--muted)] py-6 text-center">No drivers available</p>
            ) : (
              <div className="space-y-2 max-h-80 overflow-y-auto">
                {drivers.map(d => (
                  <div key={d.driver_id} className="flex items-center justify-between p-3 rounded-lg border border-[var(--border)]">
                    <div>
                      <div className="text-sm font-medium">{d.name}</div>
                      <div className="text-xs text-[var(--muted)]">{d.phone}</div>
                    </div>
                    <span className="status-chip status-chip--stable">{d.truck_status || 'IDLE'}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
