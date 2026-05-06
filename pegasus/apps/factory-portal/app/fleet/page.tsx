'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Vehicle {
  id: string;
  plate_number: string;
  capacity_m3: number;
  status: string;
  driver_name: string;
  current_route_id: string;
}

export default function FleetPage() {
  const [vehicles, setVehicles] = useState<Vehicle[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/fleet');
      if (res.ok) {
        const data = await res.json();
        setVehicles(data.vehicles || []);
      }
    } catch { /* handled */ } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        if (event.type !== 'FACTORY_TRANSFER_UPDATE' && event.type !== 'FACTORY_MANIFEST_UPDATE') {
          return;
        }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Factory Fleet</h1>
        <button onClick={() => load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 4 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : vehicles.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="fleet" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No vehicles registered</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)]">
                <th className="table__column text-left py-2 px-3 font-medium">Plate</th>
                <th className="table__column text-left py-2 px-3 font-medium">Driver</th>
                <th className="table__column text-left py-2 px-3 font-medium">Status</th>
                <th className="table__column text-right py-2 px-3 font-medium">Capacity (m³)</th>
                <th className="table__column text-left py-2 px-3 font-medium">Current Route</th>
              </tr>
            </thead>
            <tbody>
              {vehicles.map(v => (
                <tr key={v.id} className="table__row">
                  <td className="py-2.5 px-3 font-mono font-medium">{v.plate_number}</td>
                  <td className="py-2.5 px-3">{v.driver_name || '—'}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${v.status === 'AVAILABLE' ? 'status-chip--stable' : 'status-chip--loading'}`}>
                      {v.status}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{v.capacity_m3}</td>
                  <td className="py-2.5 px-3 text-[var(--muted)] font-mono text-xs">
                    {v.current_route_id ? v.current_route_id.slice(0, 8) : '—'}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
