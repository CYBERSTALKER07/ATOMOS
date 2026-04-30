'use client';

import { useState, useEffect, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
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

export default function PayloadOverridePage() {
  const [manifests, setManifests] = useState<Manifest[]>([]);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | null>(null);
  const [rebalanceModal, setRebalanceModal] = useState<{
    transfer: Transfer;
    sourceManifest: string;
  } | null>(null);
  const [targetManifestId, setTargetManifestId] = useState('');

  const fetchManifests = useCallback(async () => {
    try {
      // Fetch manifests in LOADING state
      const res = await apiFetch('/v1/factory/manifests?state=LOADING');
      if (res.ok) {
        const data = await res.json();
        setManifests(data.manifests || data.data || []);
      }
    } catch (e) {
      console.error('[PAYLOAD OVERRIDE]', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchManifests(); }, [fetchManifests]);

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
      if (res.ok) {
        setRebalanceModal(null);
        setTargetManifestId('');
        fetchManifests();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || 'Rebalance failed');
      }
    } catch (e) {
      console.error('[REBALANCE]', e);
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
      if (res.ok) {
        fetchManifests();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || 'Cancel transfer failed');
      }
    } catch (e) {
      console.error('[CANCEL TRANSFER]', e);
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
      if (res.ok) {
        fetchManifests();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || 'Cancel manifest failed');
      }
    } catch (e) {
      console.error('[CANCEL MANIFEST]', e);
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

  const loadingManifests = manifests.filter(m => m.state === 'LOADING');

  return (
    <PageTransition>
      <div className="p-6 space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-xl font-semibold">Payload Override</h1>
            <p className="text-sm mt-1" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Rebalance or cancel transfers on manifests currently in LOADING state
            </p>
          </div>
          <span className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            {loadingManifests.length} loading manifest{loadingManifests.length !== 1 ? 's' : ''}
          </span>
        </div>

        {loadingManifests.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3"
               style={{ color: 'var(--color-md-on-surface-variant)' }}>
            <Icon name="loadingBay" size={40} />
            <p className="text-sm">No manifests currently in LOADING state</p>
            <p className="text-xs">Payload override is only available during the loading phase</p>
          </div>
        ) : (
          <div className="space-y-6">
            {loadingManifests.map(manifest => (
              <div key={manifest.manifest_id}
                   className="rounded-xl border overflow-hidden"
                   style={{ borderColor: 'var(--color-md-outline-variant)', background: 'var(--color-md-surface-container-lowest)' }}>
                {/* Manifest header */}
                <div className="flex items-center justify-between px-4 py-3 border-b"
                     style={{ background: 'var(--color-md-surface-container)', borderColor: 'var(--color-md-outline-variant)' }}>
                  <div className="flex items-center gap-3">
                    <Icon name="fleet" size={18} />
                    <div>
                      <span className="font-medium text-sm">
                        {manifest.truck_plate || manifest.truck_id.slice(0, 8)}
                      </span>
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
                    <div className="h-2 w-24 rounded-full overflow-hidden"
                         style={{ background: 'var(--color-md-surface-container-high)' }}>
                      <div className="h-full rounded-full transition-all"
                           style={{
                             width: `${Math.min(100, (manifest.total_volume_vu / manifest.max_capacity_vu) * 100)}%`,
                             background: manifest.total_volume_vu > manifest.max_capacity_vu * 0.9
                               ? 'var(--color-md-error)'
                               : 'var(--color-md-primary)',
                           }} />
                    </div>
                    <button
                      onClick={() => handleCancelManifest(manifest.manifest_id)}
                      disabled={acting === manifest.manifest_id}
                      className="px-3 py-1.5 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                      style={{ background: 'var(--color-md-error-container)', color: 'var(--color-md-on-error-container)' }}
                    >
                      {acting === manifest.manifest_id ? '...' : 'Cancel Manifest'}
                    </button>
                  </div>
                </div>

                {/* Transfer list */}
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
                    {(manifest.transfers || []).map(transfer => (
                      <tr key={transfer.transfer_id}
                          className="border-t"
                          style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                        <td className="px-4 py-2.5">
                          <span className="font-mono text-xs">{transfer.transfer_id.slice(0, 8)}</span>
                        </td>
                        <td className="px-4 py-2.5">{transfer.product_name || '—'}</td>
                        <td className="px-4 py-2.5 text-right tabular-nums">{transfer.quantity}</td>
                        <td className="px-4 py-2.5 text-right tabular-nums">{transfer.volume_vu.toLocaleString()}</td>
                        <td className="px-4 py-2.5 text-right">
                          <div className="flex gap-2 justify-end">
                            <button
                              onClick={() => setRebalanceModal({
                                transfer,
                                sourceManifest: manifest.manifest_id,
                              })}
                              disabled={acting === transfer.transfer_id}
                              className="px-2.5 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                              style={{ background: 'var(--color-md-primary-container)', color: 'var(--color-md-on-primary-container)' }}
                            >
                              Move
                            </button>
                            <button
                              onClick={() => handleCancelTransfer(transfer.transfer_id, manifest.manifest_id)}
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
                        <td colSpan={5} className="px-4 py-8 text-center text-xs"
                            style={{ color: 'var(--color-md-on-surface-variant)' }}>
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

        {/* Rebalance Modal */}
        {rebalanceModal && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40" onClick={() => setRebalanceModal(null)}>
            <div className="rounded-2xl p-6 w-full max-w-md space-y-4"
                 style={{ background: 'var(--color-md-surface-container-high)' }}
                 onClick={e => e.stopPropagation()}>
              <h2 className="text-lg font-semibold">Move Transfer</h2>
              <p className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                Moving <span className="font-mono">{rebalanceModal.transfer.transfer_id.slice(0, 8)}</span>
                {' '}({rebalanceModal.transfer.volume_vu} VU) to another manifest
              </p>

              <div>
                <label className="text-xs font-medium block mb-1">Target Manifest</label>
                <select
                  value={targetManifestId}
                  onChange={e => setTargetManifestId(e.target.value)}
                  className="w-full px-3 py-2 rounded-lg text-sm border"
                  style={{
                    background: 'var(--color-md-surface)',
                    borderColor: 'var(--color-md-outline)',
                    color: 'var(--color-md-on-surface)',
                  }}
                >
                  <option value="">Select a manifest...</option>
                  {loadingManifests
                    .filter(m => m.manifest_id !== rebalanceModal.sourceManifest)
                    .map(m => (
                      <option key={m.manifest_id} value={m.manifest_id}>
                        {m.truck_plate || m.truck_id.slice(0, 8)} — {m.total_volume_vu}/{m.max_capacity_vu} VU
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
                  onClick={handleRebalance}
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
