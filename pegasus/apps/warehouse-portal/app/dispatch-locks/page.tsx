'use client';

import { useEffect, useEffectEvent, useState } from 'react';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import type { WarehouseDispatchLock, WarehouseLiveEvent } from '@pegasus/types';

export default function DispatchLocksPage() {
  const { toast } = useToast();
  const [locks, setLocks] = useState<WarehouseDispatchLock[]>([]);
  const [loading, setLoading] = useState(true);
  const [releasing, setReleasing] = useState<string | null>(null);
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');

  const loadLocks = useEffectEvent(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/warehouse/dispatch-locks');
      if (res.ok) {
        const data = await res.json() as WarehouseDispatchLock[] | { locks?: WarehouseDispatchLock[] };
        setLocks(Array.isArray(data) ? data : (data.locks || []));
      }
    } catch {
      toast('Failed to load locks', 'error');
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useEffectEvent((event: WarehouseLiveEvent) => {
    if (event.type !== 'DISPATCH_LOCK_CHANGE') {
      return;
    }
    void loadLocks();
  });

  useEffect(() => {
    void loadLocks();
  }, [loadLocks]);

  useEffect(() => {
    return subscribeWarehouseWS({
      onStatusChange: setSocketStatus,
      onMessage: payload => {
      try {
        handleWarehouseLiveEvent(JSON.parse(payload) as WarehouseLiveEvent);
      } catch {
        // Ignore unrelated frames.
      }
      },
    });
  }, [handleWarehouseLiveEvent]);

  async function handleAcquire() {
    try {
      const res = await apiFetch('/v1/warehouse/dispatch-lock', {
        method: 'POST',
        body: JSON.stringify({ lock_type: 'MANUAL_DISPATCH' }),
      });
      if (res.ok) {
        toast('Dispatch lock acquired', 'success');
        void loadLocks();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to acquire lock', 'error');
      }
    } catch {
      toast('Network error', 'error');
    }
  }

  async function handleRelease(lockId: string) {
    setReleasing(lockId);
    try {
      const res = await apiFetch(`/v1/warehouse/dispatch-lock?lock_id=${lockId}`, {
        method: 'DELETE',
      });
      if (res.ok) {
        toast('Lock released', 'success');
        void loadLocks();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || 'Failed to release lock', 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setReleasing(null);
    }
  }

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-xl font-bold tracking-tight">Dispatch Locks</h1>
          <p className="text-xs text-[var(--muted)] mt-0.5">
            Prevent concurrent dispatch operations during loading
          </p>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={loadLocks}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
          >
            <Icon name="refresh" size={16} />
            Refresh
          </button>
          <button
            onClick={handleAcquire}
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary"
          >
            <Icon name="lock" size={16} />
            Acquire Lock
          </button>
        </div>
      </div>

      {socketStatus !== 'idle' && socketStatus !== 'live' && (
        <div className={`rounded-xl border px-4 py-3 text-sm ${socketStatus === 'offline'
          ? 'border-[var(--danger)]/30 bg-[var(--danger)]/8 text-[var(--danger)]'
          : 'border-[var(--warning)]/30 bg-[var(--warning)]/8 text-[var(--warning)]'}`}>
          {socketStatus === 'offline'
            ? 'Offline. Live dispatch-lock updates are paused until the network returns.'
            : socketStatus === 'reconnecting'
              ? 'Live dispatch-lock updates are reconnecting. Current lock state may be stale.'
              : 'Connecting live dispatch-lock updates…'}
        </div>
      )}

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <div key={i} className="md-skeleton md-skeleton-row" />
          ))}
        </div>
      ) : locks.length === 0 ? (
        <div className="text-center py-20 text-[var(--muted)]">
          <Icon name="lock" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No active dispatch locks</p>
          <p className="text-xs mt-1">Dispatch operations are running freely</p>
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Lock ID</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Type</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Scope</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Locked At</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Locked By</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">Actions</th>
              </tr>
            </thead>
            <tbody>
              {locks.map(lock => (
                <tr key={lock.lock_id} className="border-b border-[var(--border)] last:border-b-0">
                  <td className="px-4 py-3 font-mono text-xs">{lock.lock_id.slice(0, 8)}...</td>
                  <td className="px-4 py-3">
                    <span className="status-chip status-chip--submitted">{lock.lock_type}</span>
                  </td>
                  <td className="px-4 py-3 text-xs text-[var(--muted)]">
                    {lock.warehouse_id ? `WH: ${lock.warehouse_id.slice(0, 8)}` :
                     lock.factory_id ? `Factory: ${lock.factory_id.slice(0, 8)}` :
                     'Global'}
                  </td>
                  <td className="px-4 py-3 text-xs">{new Date(lock.locked_at).toLocaleString()}</td>
                  <td className="px-4 py-3 text-xs font-mono">{lock.locked_by.slice(0, 8)}</td>
                  <td className="px-4 py-3 text-right">
                    <button
                      onClick={() => handleRelease(lock.lock_id)}
                      disabled={releasing === lock.lock_id}
                      className="px-3 py-1 rounded-lg text-xs font-semibold button--danger disabled:opacity-50"
                    >
                      {releasing === lock.lock_id ? '...' : 'Release'}
                    </button>
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
