'use client';
import { usePolling } from '@pegasusx/api-react';


import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { PayloadList } from '../../components/payload-override/PayloadList';
import { PayloadOverrideForm } from '../../components/payload-override/PayloadOverrideForm';

export interface Transfer {
  transfer_id: string;
  product_name: string;
  quantity: number;
  volume_vu: number;
  state: string;
}

export interface Manifest {
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
  const t = usePortalT();
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

  useFactorySessionReconcile(() => {
    void fetchManifests({ background: true, silent: true });
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        if (!event.type.startsWith('TRANSFER_') && !event.type.startsWith('MANIFEST_') && !event.type.startsWith('WAREHOUSE_TRANSFER_') && !event.type.startsWith('FACTORY_SUPPLY_')) { return; }
        void fetchManifests({ background: true, silent: true });
      },
    });

    return () => {
      unsubscribe();
    };
  }, [fetchManifests]);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await fetchManifests({ background: true, silent: true });
    },
    LIVE_REFRESH_MS,
    [fetchManifests],
  );

  useEffect(() => {
    const handleOnline = () => {
      setIsOffline(false);
      toast('Connection restored. Refreshing loading manifests.', 'info');
      void fetchManifests({ background: true, silent: true });
    };

    const handleOffline = () => {
      setIsOffline(true);
      toast('Offline. Showing the last synced loading manifests.', 'warning');
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
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
        <PageChrome
          icon="manifests"
          title={t("factory_portal.payload_override.text.payload_override")}
          description={t("factory_portal.residual.text.rebalance_or_cancel_transfers_on_manifests_currently_in_loading_")}
          loading
          skeletonVariant="table"
        >
          <span />
        </PageChrome>
      </PageTransition>
    );
  }

  if (error && manifests.length === 0) {
    return (
      <PageTransition>
        <PageChrome
          icon="manifests"
          title={t("factory_portal.payload_override.text.payload_override")}
          description={t("factory_portal.residual.text.rebalance_or_cancel_transfers_on_manifests_currently_in_loading_")}
          error={error}
          actions={
            <button type="button" className="portal-btn portal-btn--ghost inline-flex items-center gap-2" onClick={() => void fetchManifests()}>
              <Icon name="refresh" size={16} /> Retry
            </button>
          }
        >
          <span />
        </PageChrome>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <PageChrome
        icon="manifests"
        title={t("factory_portal.payload_override.text.payload_override")}
        description={t("factory_portal.residual.text.rebalance_or_cancel_transfers_on_manifests_currently_in_loading_")}
        actions={
          <div className="flex items-center gap-3">
            <span className="text-sm text-[var(--muted)]">
              {loadingManifests.length} loading manifest{loadingManifests.length !== 1 ? 's' : ''}
            </span>
            <button
              type="button"
              onClick={() => void fetchManifests({ background: manifests.length > 0 })}
              className="portal-btn portal-btn--ghost inline-flex items-center gap-2"
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
          </div>
        }
      >

        <div
          className={`mb-6 rounded-2xl border px-4 py-3 text-sm ${
            isOffline || error
              ? 'border-[var(--warning)] bg-[var(--surface-muted)]'
              : 'border-[var(--border)] bg-[var(--surface)]'
          } text-[var(--muted)]`}
        >
          {runtimeMessage}
        </div>

        {loadingManifests.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-production-line.png"
            headline={t("factory_portal.residual.text.no_manifests_currently_in_loading_state")}
            body={t("factory_portal.residual.text.payload_override_is_only_available_during_the_loading_phase_all_")}
          />
        ) : (
          <PayloadList
            loadingManifests={loadingManifests}
            acting={acting}
            onMove={(transfer, sourceManifest) => setRebalanceModal({ transfer, sourceManifest })}
            onCancelTransfer={(transferId, manifestId) => void handleCancelTransfer(transferId, manifestId)}
            onCancelManifest={(manifestId) => void handleCancelManifest(manifestId)}
          />
        )}

        <PayloadOverrideForm
          rebalanceModal={rebalanceModal}
          targetManifestId={targetManifestId}
          setTargetManifestId={setTargetManifestId}
          loadingManifests={loadingManifests}
          acting={acting}
          onClose={() => { setRebalanceModal(null); setTargetManifestId(''); }}
          onSubmit={() => void handleRebalance()}
        />
      </PageChrome>
    </PageTransition>
  );
}
