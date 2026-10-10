'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState } from 'react';
import { useStableCallback } from '@/lib/useStableCallback';
import { ApiError } from '@pegasusx/api-core';
import { subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import type { WarehouseDispatchLock, WarehouseLiveEvent } from '@pegasusx/types';

export default function DispatchLocksPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [locks, setLocks] = useState<WarehouseDispatchLock[]>([]);
  const [loading, setLoading] = useState(true);
  const [releasing, setReleasing] = useState<string | null>(null);
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');
  const [restricted, setRestricted] = useState(false);
  const [loadError, setLoadError] = useState<string | null>(null);

  const loadLocks = useStableCallback(async () => {
    setLoading(true);
    setLoadError(null);
    try {
      const data = await warehouseApi.getWarehouseDispatchLocks();
      setLocks(data.locks || []);
      setRestricted(false);
    } catch (err) {
      if (err instanceof ApiError && err.status === 403) {
        setRestricted(true);
        setLocks([]);
      } else {
        setLoadError(err instanceof ApiError ? err.message : 'Failed to load dispatch locks');
      }
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useStableCallback((event: WarehouseLiveEvent) => {
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

  useWarehouseSessionReconcile(() => {
    void loadLocks();
  });

  async function handleAcquire() {
    const warehouseId = warehouseHomeNodeId() || 'warehouse';
    try {
      await warehouseOps.acquireDispatchLock(
        warehouseId,
        'WAREHOUSE',
        warehouseId,
        'manual_dispatch',
      );
      toast('Dispatch lock acquired', 'success');
      void loadLocks();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Failed to acquire lock', 'error');
    }
  }

  async function handleRelease(lockId: string) {
    setReleasing(lockId);
    try {
      await warehouseOps.releaseDispatchLock(lockId);
      toast('Lock released', 'success');
      void loadLocks();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Failed to release lock', 'error');
    } finally {
      setReleasing(null);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="lock"
        title={t("portal.nav.dispatch_locks")}
        description={t("warehouse_portal.residual.text.prevent_concurrent_dispatch_operations_during_loading")}
        loading={loading}
        error={restricted ? 'You do not have permission to manage dispatch locks for this scope.' : loadError}
        empty={!loading && !restricted && !loadError && locks.length === 0}
        emptyMessage={t("warehouse_portal.residual.text.no_active_dispatch_locks_dispatch_operations_are_running_freely")}
        actions={
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => void loadLocks()}
              className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
            >
              <Icon name="refresh" size={16} />
              Refresh
            </button>
            <button
              type="button"
              onClick={() => void handleAcquire()}
              className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary"
            >
              <Icon name="lock" size={16} />
              Acquire Lock
            </button>
          </div>
        }
      >
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

        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="desk-table w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.dispatch_locks.text.lock_id")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.bins.text.type")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.dispatch_locks.text.scope")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.dispatch_locks.text.locked_at")}</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.dispatch_locks.text.locked_by")}</th>
                <th className="text-right px-4 py-3 font-semibold text-[var(--muted)]">{t("warehouse_portal.dispatch_locks.text.actions")}</th>
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
                  <td className="px-4 py-3 text-xs">
                    {new Date(lock.locked_at ?? lock.created_at ?? Date.now()).toLocaleString()}
                  </td>
                  <td className="px-4 py-3 text-xs font-mono">{(lock.locked_by ?? "ops").slice(0, 8)}</td>
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
      </PageChrome>
    </PageTransition>
  );
}
