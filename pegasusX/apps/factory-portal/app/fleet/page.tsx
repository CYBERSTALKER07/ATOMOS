'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';

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
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/fleet');
      if (res.ok) {
        const data = await res.json();
        setVehicles(data.vehicles || []);
      } else {
        setError(`Unable to load fleet (${res.status}).`);
      }
    } catch {
      setError('Unable to load fleet right now.');
    } finally {
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
    <PageTransition>
      <PageChrome
        icon="fleet"
        title="Factory fleet"
        description="Vehicle availability and route assignments for this factory node."
        loading={loading}
        skeletonVariant="table"
        error={error && vehicles.length === 0 ? error : null}
        empty={!loading && !error && vehicles.length === 0}
        emptyMessage="There are no vehicles registered in the factory fleet yet."
        actions={
          <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-1.5" onClick={() => void load()}>
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="desk-table-wrap"
        >
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Plate</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Driver</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Capacity (m³)</th>
                <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Current Route</th>
              </tr>
            </thead>
            <tbody>
              {vehicles.map((v, index) => (
                <motion.tr
                  key={v.id}
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  transition={{ delay: index * 0.05 }}
                  className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                >
                  <td className="py-3 px-4 font-mono font-medium">{v.plate_number}</td>
                  <td className="py-3 px-4">{v.driver_name || '—'}</td>
                  <td className="py-3 px-4">
                    <span className={`status-chip ${v.status === 'AVAILABLE' ? 'status-chip--stable' : 'status-chip--loading'}`}>
                      {v.status}
                    </span>
                  </td>
                  <td className="py-3 px-4 text-right tabular-nums font-mono">{v.capacity_m3}</td>
                  <td className="py-3 px-4 text-[var(--muted)] font-mono text-xs">
                    {v.current_route_id ? v.current_route_id.slice(0, 8) : '—'}
                  </td>
                </motion.tr>
              ))}
            </tbody>
          </table>
        </motion.div>
      </PageChrome>
    </PageTransition>
  );
}
