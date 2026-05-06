'use client';

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageSkeleton } from '@/components/Skeleton';

interface Transfer {
  transfer_id: string;
  product_name: string;
  quantity: number;
  volume_vu: number;
  state: string;
}

interface Manifest {
  manifest_id: string;
  truck_id: string;
  truck_plate?: string;
  state: string;
  total_volume_vu: number;
  max_capacity_vu: number;
  transfers: Transfer[];
  created_at: string;
}

const LIVE_REFRESH_MS = 30_000;

function manifestSignature(items: Manifest[]) {
  return items
    .map((manifest) => `${manifest.manifest_id}:${manifest.state}:${manifest.transfers.length}:${manifest.total_volume_vu}`)
    .join('|');
}

function formatSyncTime(value: number | null) {
  if (!value) return 'Waiting for first sync';
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export default function PayloadOverridePage() {
  const { toast } = useToast();
  const [manifests, setManifests] = useState<Manifest[]>([]);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState<number | null>(null);
  const [isOffline, setIsOffline] = useState(() => (typeof navigator === 'undefined' ? false : !navigator.onLine));
  const [rebalanceModal, setRebalanceModal] = useState<{
    transfer: Transfer;
    sourceManifest: string;
  } | null>(null);
  const [targetManifestId, setTargetManifestId] = useState('');
  const previousSignatureRef = useRef('');

  const fetchManifests = useCallback(async (options?: { background?: boolean; silent?: boolean }) => {
    const background = options?.background ?? false;
    const silent = options?.silent ?? false;

    if (background) {
      setRefreshing(true);
    } else if (manifests.length === 0) {
      setLoading(true);
    }

    try {
      const res = await apiFetch('/v1/factory/manifests?state=LOADING');
      if (!res.ok) {
        throw new Error(`Factory API responded with ${res.status}`);
      }

      const data = await res.json();
      const next = (data.manifests || data.data || []).filter((manifest: Manifest) => manifest.state === 'LOADING');
      const nextSignature = manifestSignature(next);

      if (background && previousSignatureRef.current && previousSignatureRef.current !== nextSignature && !silent) {
        toast('Loading manifests updated', 'info');
      }

      previousSignatureRef.current = nextSignature;
      setManifests(next);
      setLastSyncedAt(Date.now());
      setError(null);
      setIsOffline(false);
    } catch {
      const message = isOffline || (typeof navigator !== 'undefined' && !navigator.onLine)
        ? 'Offline. Showing the last synced loading manifests.'
        : 'Live refresh failed. Showing the last synced loading manifests.';

      if (manifests.length === 0) {
        setError(message);
      } else {
        setError(message);
        if (!silent) {
          toast(message, 'warning');
        }
      }

      if (typeof navigator !== 'undefined') {
        setIsOffline(!navigator.onLine);
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [isOffline, manifests.length, toast]);

  useEffect(() => {
    void fetchManifests();
  }, [fetchManifests]);

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
        void fetchManifests({ background: true, silent: true });
      },
    });

    return () => {
      unsubscribe();
    };
  }, [fetchManifests]);

  useEffect(() => {
    const refreshLiveData = () => {
      if (document.visibilityState === 'visible') {
        void fetchManifests({ background: true, silent: true });
      }
    };

    const handleOnline = () => {
      setIsOffline(false);
      toast('Connection restored. Refreshing loading manifests.', 'info');
      void fetchManifests({ background: true, silent: true });
    };

    const handleOffline = () => {
      setIsOffline(true);
      toast('Offline. Showing the last synced loading manifests.', 'warning');
    };

    const interval = window.setInterval(refreshLiveData, LIVE_REFRESH_MS);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    document.addEventListener('visibilitychange', refreshLiveData);

    return () => {
      window.clearInterval(interval);
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      document.removeEventListener('visibilitychange', refreshLiveData);
    };
  }, [fetchManifests, toast]);

  const loadingManifests = useMemo(
    () => manifests.filter((manifest) => manifest.state === 'LOADING'),
    [manifests],
  );

  const runtimeMessage = isOffline
    ? `Offline — showing last sync from ${formatSyncTime(lastSyncedAt)}`
    : error && manifests.length > 0
      ? `${error} Last sync ${formatSyncTime(lastSyncedAt)}`
      : refreshing
        ? `Refreshing live manifests — last sync ${formatSyncTime(lastSyncedAt)}`
        : `Live sync active — last sync ${formatSyncTime(lastSyncedAt)}`;

  const handleRebalance = async () => {
    if (!rebalanceModal || !targetManifestId) return;
    setActing(rebalanceModal.transfer.transfer_id);
    try {
      const res = await apiFetch('/v1/factory/manifests/rebalance', {
        method: 'POST',
        body: JSON.stringify({
          transfer_ids: [rebalanceModal.transfer.transfer_id],
          source_manifest_id: rebalanceModal.sourceManifest,
          target_manifest_id: targetManifestId,
        }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Rebalance failed', 'error');
        return;
      }

      setRebalanceModal(null);
      setTargetManifestId('');
      toast('Transfer moved to the selected manifest', 'success');
      await fetchManifests({ background: true, silent: true });
    } catch {
      toast('Rebalance failed', 'error');
    } finally {
      setActing(null);
    }
  };

  const handleCancelTransfer = async (transferId: string, manifestId: string) => {
    if (!confirm('Remove this transfer from the manifest? It will return to APPROVED state.')) return;
    setActing(transferId);
    try {
      const res = await apiFetch('/v1/factory/manifests/cancel-transfer', {
        method: 'POST',
        body: JSON.stringify({ transfer_id: transferId, manifest_id: manifestId }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Remove transfer failed', 'error');
        return;
      }

      toast('Transfer released back to APPROVED', 'success');
      await fetchManifests({ background: true, silent: true });
    } catch {
      toast('Remove transfer failed', 'error');
    } finally {
      setActing(null);
    }
  };

  const handleCancelManifest = async (manifestId: string) => {
    if (!confirm('Cancel this entire manifest? All transfers will return to APPROVED state.')) return;
    setActing(manifestId);
    try {
      const res = await apiFetch('/v1/factory/manifests/cancel', {
        method: 'POST',
        body: JSON.stringify({ manifest_id: manifestId }),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Cancel manifest failed', 'error');
        return;
      }

      toast('Manifest cancelled', 'success');
      await fetchManifests({ background: true, silent: true });
    } catch {
      toast('Cancel manifest failed', 'error');
    } finally {
      setActing(null);
    }
  };

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6 space-y-4">
          <h1 className="text-xl font-semibold">Payload Override</h1>
          <PageSkeleton />
        </div>
      </PageTransition>
    );
  }

  if (error && manifests.length === 0) {
    return (
      <PageTransition>
        <div className="p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h1 className="text-xl font-semibold">Payload Override</h1>
            <button
              onClick={() => void fetchManifests()}
              className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
            >
              <Icon name="refresh" size={16} /> Retry
            </button>
          </div>
          <div
            className="rounded-2xl border p-6 text-sm"
            style={{
              borderColor: 'var(--color-md-outline-variant)',
              background: 'var(--color-md-surface-container-lowest)',
              color: 'var(--color-md-on-surface-variant)',
            }}
          >
            {error}
          </div>
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Payload Override</h1>
            <p className="text-sm mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Rebalance or cancel transfers on manifests currently in LOADING state
            </p>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {loadingManifests.length} loading manifest{loadingManifests.length !== 1 ? 's' : ''}
            </span>
            <button
              onClick={() => void fetchManifests({ background: manifests.length > 0 })}
              className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
          </div>
        </div>

        <div
          className="rounded-2xl border px-4 py-3 text-sm"
          style={{
            borderColor: isOffline || error ? 'var(--color-md-warning)' : 'var(--color-md-outline-variant)',
            background: isOffline || error ? 'var(--color-md-surface-container-high)' : 'var(--color-md-surface-container-low)',
            color: 'var(--color-md-on-surface-variant)',
          }}
        >
          {runtimeMessage}
        </div>

        {loadingManifests.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            <Icon name="loadingBay" size={40} />
            <p className="text-sm">No manifests currently in LOADING state</p>
            <p className="text-xs">Payload override is only available during the loading phase</p>
          </div>
        ) : (
          <div className="space-y-6">
            {loadingManifests.map((manifest) => (
              <div
                key={manifest.manifest_id}
                className="rounded-xl border overflow-hidden"
                style={{ borderColor: 'var(--color-md-outline-variant)', background: 'var(--color-md-surface-container-lowest)' }}
              >
                <div
                  className="flex items-center justify-between px-4 py-3 border-b"
                  style={{ background: 'var(--color-md-surface-container)', borderColor: 'var(--color-md-outline-variant)' }}
                >
                  <div className="flex items-center gap-3">
                    <Icon name="fleet" size={18} />
                    <div>
                      <span className="font-medium text-sm">{manifest.truck_plate || manifest.truck_id.slice(0, 8)}</span>
                      <span className="text-xs ml-2" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {manifest.manifest_id.slice(0, 8)}
                      </span>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    <div className="text-xs">
                      <span style={{ color: 'var(--color-md-on-surface-variant)' }}>Capacity: </span>
                      <span className="font-medium tabular-nums">
                        {manifest.total_volume_vu.toLocaleString()} / {manifest.max_capacity_vu.toLocaleString()} VU
                      </span>
                    </div>
                    <div className="h-2 w-24 rounded-full overflow-hidden" style={{ background: 'var(--color-md-surface-container-high)' }}>
                      <div
                        className="h-full rounded-full transition-all"
                        style={{
                          width: `${Math.min(100, (manifest.total_volume_vu / manifest.max_capacity_vu) * 100)}%`,
                          background: manifest.total_volume_vu > manifest.max_capacity_vu * 0.9
                            ? 'var(--color-md-error)'
                            : 'var(--color-md-primary)',
                        }}
                      />
                    </div>
                    <button
                      onClick={() => void handleCancelManifest(manifest.manifest_id)}
                      disabled={acting === manifest.manifest_id}
                      className="px-3 py-1.5 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                      style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}
                    >
                      {acting === manifest.manifest_id ? '...' : 'Cancel Manifest'}
                    </button>
                  </div>
                </div>

                <table className="w-full text-sm">
                  <thead>
                    <tr style={{ background: 'var(--color-md-surface-container)' }}>
                      <th className="text-left px-4 py-2 font-medium text-xs">Transfer</th>
                      <th className="text-left px-4 py-2 font-medium text-xs">Product</th>
                      <th className="text-right px-4 py-2 font-medium text-xs">Qty</th>
                      <th className="text-right px-4 py-2 font-medium text-xs">Volume (VU)</th>
                      <th className="text-right px-4 py-2 font-medium text-xs">Actions</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(manifest.transfers || []).map((transfer) => (
                      <tr key={transfer.transfer_id} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                        <td className="px-4 py-2.5">
                          <span className="font-mono text-xs">{transfer.transfer_id.slice(0, 8)}</span>
                        </td>
                        <td className="px-4 py-2.5">{transfer.product_name || '—'}</td>
                        <td className="px-4 py-2.5 text-right tabular-nums">{transfer.quantity}</td>
                        <td className="px-4 py-2.5 text-right tabular-nums">{transfer.volume_vu.toLocaleString()}</td>
                        <td className="px-4 py-2.5 text-right">
                          <div className="flex gap-2 justify-end">
                            <button
                              onClick={() => setRebalanceModal({ transfer, sourceManifest: manifest.manifest_id })}
                              disabled={acting === transfer.transfer_id}
                              className="px-2.5 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                              style={{ background: 'var(--color-md-primary-container)', color: 'var(--color-md-on-primary-container)' }}
                            >
                              Move
                            </button>
                            <button
                              onClick={() => void handleCancelTransfer(transfer.transfer_id, manifest.manifest_id)}
                              disabled={acting === transfer.transfer_id}
                              className="px-2.5 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                              style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}
                            >
                              Remove
                            </button>
                          </div>
                        </td>
                      </tr>
                    ))}
                    {(!manifest.transfers || manifest.transfers.length === 0) && (
                      <tr>
                        <td colSpan={5} className="px-4 py-8 text-center text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          No transfers in this manifest
                        </td>
                      </tr>
                    )}
                  </tbody>
                </table>
              </div>
            ))}
          </div>
        )}

        {rebalanceModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setRebalanceModal(null)}>
            <div
              className="rounded-2xl p-6 w-full max-w-md space-y-4"
              style={{ background: 'var(--color-md-surface-container-high)' }}
              onClick={(event) => event.stopPropagation()}
            >
              <h2 className="text-lg font-semibold">Move Transfer</h2>
              <p className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                Moving <span className="font-mono">{rebalanceModal.transfer.transfer_id.slice(0, 8)}</span>
                {' '}({rebalanceModal.transfer.volume_vu} VU) to another manifest
              </p>

              <div>
                <label className="text-xs font-medium block mb-1">Target Manifest</label>
                <select
                  value={targetManifestId}
                  onChange={(event) => setTargetManifestId(event.target.value)}
                  className="w-full px-3 py-2 rounded-lg text-sm border"
                  style={{
                    background: 'var(--color-md-surface)',
                    borderColor: 'var(--color-md-outline)',
                    color: 'var(--color-md-on-surface)',
                  }}
                >
                  <option value="">Select a manifest...</option>
                  {loadingManifests
                    .filter((manifest) => manifest.manifest_id !== rebalanceModal.sourceManifest)
                    .map((manifest) => (
                      <option key={manifest.manifest_id} value={manifest.manifest_id}>
                        {manifest.truck_plate || manifest.truck_id.slice(0, 8)} — {manifest.total_volume_vu}/{manifest.max_capacity_vu} VU
                      </option>
                    ))}
                </select>
              </div>

              <div className="flex gap-3 justify-end">
                <button
                  onClick={() => { setRebalanceModal(null); setTargetManifestId(''); }}
                  className="px-4 py-2 rounded-lg text-sm font-medium"
                  style={{ color: 'var(--color-md-on-surface-variant)' }}
                >
                  Cancel
                </button>
                <button
                  onClick={() => void handleRebalance()}
                  disabled={!targetManifestId || acting === rebalanceModal.transfer.transfer_id}
                  className="px-4 py-2 rounded-lg text-sm font-medium text-white disabled:opacity-50"
                  style={{ background: 'var(--color-md-primary)' }}
                >
                  {acting ? 'Moving...' : 'Move Transfer'}
                </button>
              </div>
            </div>
          </div>
        )}
      </div>
    </PageTransition>
  );
}
