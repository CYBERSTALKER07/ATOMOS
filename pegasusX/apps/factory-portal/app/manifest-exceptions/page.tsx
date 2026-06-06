'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';
import { PageSkeleton } from '@/components/Skeleton';

interface ManifestException {
  exception_id: string;
  manifest_id: string;
  transfer_id: string;
  reason: string;
  metadata?: string;
  attempt_count: number;
  escalated: boolean;
  created_at: string;
  correlation_id?: string;
}

const LIVE_REFRESH_MS = 30_000;

const REASON_COLORS: Record<string, string> = {
  OVERFLOW: 'var(--color-md-warning)',
  DAMAGED: 'var(--color-md-error)',
  MANUAL: 'var(--color-md-info)',
  NO_CAPACITY: 'var(--color-md-error)',
};

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

function exceptionSignature(items: ManifestException[]) {
  return items
    .map((item) => `${item.exception_id}:${item.attempt_count}:${item.escalated}`)
    .join('|');
}

function formatSyncTime(value: number | null) {
  if (!value) return 'Waiting for first sync';
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

function reasonBadge(reason: string) {
  return (
    <span
      className="md-typescale-label-small md-shape-sm px-2 py-0.5 inline-block"
      style={{
        background: REASON_COLORS[reason] || 'var(--color-md-outline)',
        color: '#fff',
      }}
    >
      {reason}
    </span>
  );
}

export default function ManifestExceptionsPage() {
  const [exceptions, setExceptions] = useState<ManifestException[]>([]);
  const [loading, setLoading] = useState(true);
  const [escalatedOnly, setEscalatedOnly] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState<number | null>(null);
  const [isOffline, setIsOffline] = useState(() => (typeof navigator === 'undefined' ? false : !navigator.onLine));
  const previousSignatureRef = useRef('');

  const fetchExceptions = useCallback(async (options?: { background?: boolean; silent?: boolean }) => {
    const background = options?.background ?? false;
    const silent = options?.silent ?? false;

    if (background) {
      setRefreshing(true);
    } else if (exceptions.length === 0) {
      setLoading(true);
    }

    try {
      const qs = escalatedOnly ? '?escalated=true' : '';
      const res = await apiFetch(`/v1/factory/manifest-exceptions${qs}`);
      if (!res.ok) {
        throw new Error(`Factory API responded with ${res.status}`);
      }
      const data = await res.json();
      const next: ManifestException[] = data.exceptions || [];
      const nextSignature = exceptionSignature(next);

      if (background && previousSignatureRef.current && previousSignatureRef.current !== nextSignature && !silent) {
        // signature drift only — no toast spam for exception inbox
      }
      previousSignatureRef.current = nextSignature;
      setExceptions(next);
      setError(null);
      setLastSyncedAt(Date.now());
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      setError(e instanceof Error ? e.message : 'Failed to load exceptions');
    } finally {
      if (background) {
        setRefreshing(false);
      } else {
        setLoading(false);
      }
    }
  }, [escalatedOnly, exceptions.length]);

  useEffect(() => {
    fetchExceptions();
  }, [fetchExceptions]);

  useEffect(() => {
    const onOnline = () => setIsOffline(false);
    const onOffline = () => setIsOffline(true);
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, []);

  useEffect(() => {
    const interval = window.setInterval(() => {
      if (!document.hidden && navigator.onLine) {
        fetchExceptions({ background: true, silent: true });
      }
    }, LIVE_REFRESH_MS);
    return () => window.clearInterval(interval);
  }, [fetchExceptions]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (raw) => {
        const event = parseFactoryLiveEvent(raw);
        if (!event?.type) return;
        if (event.type === 'FACTORY_MANIFEST_UPDATE' || event.type === 'FACTORY_TRANSFER_UPDATE') {
          fetchExceptions({ background: true, silent: true });
        }
      },
    });
    return unsubscribe;
  }, [fetchExceptions]);

  const runtimeMessage = isOffline
    ? `Offline — showing last sync from ${formatSyncTime(lastSyncedAt)}`
    : error && exceptions.length > 0
      ? `${error} Last sync ${formatSyncTime(lastSyncedAt)}`
      : refreshing
        ? `Refreshing exception inbox — last sync ${formatSyncTime(lastSyncedAt)}`
        : `Live sync active — last sync ${formatSyncTime(lastSyncedAt)}`;

  const runtimeTone = isOffline
    ? 'offline'
    : error && exceptions.length > 0
      ? 'warning'
      : refreshing
        ? 'refreshing'
        : 'live';

  return (
    <PageTransition>
      <div className="flex flex-col gap-6 p-6">
        <FactoryRuntimeBanner tone={runtimeTone} message={runtimeMessage} />

        <div className="flex items-center justify-between gap-4 flex-wrap">
          <div>
            <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
              Loading Gate Exceptions
            </h1>
            <p className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              Transfers removed from manifests during loading — overflow, damage, or manual pull
            </p>
          </div>
          <button
            type="button"
            className={`md-btn ${escalatedOnly ? 'md-btn-filled' : 'md-btn-outlined'} md-typescale-label-large px-4 py-2`}
            onClick={() => setEscalatedOnly((value) => !value)}
          >
            <Icon name="error" size={18} />
            {escalatedOnly ? 'Showing Escalated' : 'Show Escalated Only'}
          </button>
        </div>

        {error && !loading ? (
          <FactoryPageState
            kind="error"
            title="Unable to load exceptions"
            body={error}
            actionLabel="Retry"
            onAction={() => fetchExceptions()}
          />
        ) : loading ? (
          <PageSkeleton />
        ) : exceptions.length === 0 ? (
          <EmptyState
            icon="check_circle"
            headline={escalatedOnly ? 'No escalated exceptions' : 'No exceptions'}
            body={
              escalatedOnly
                ? 'No transfers have hit the DLQ threshold (3+ overflows)'
                : 'All manifest loading operations completed without exceptions'
            }
          />
        ) : (
          <div className="md-card md-elevation-1 md-shape-md overflow-hidden">
            <table className="w-full">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container)' }}>
                  {['Transfer', 'Manifest', 'Reason', 'Attempts', 'Escalated', 'Time'].map((h) => (
                    <th
                      key={h}
                      className="md-typescale-label-small px-4 py-3 text-left"
                      style={{ color: 'var(--color-md-on-surface-variant)' }}
                    >
                      {h}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {exceptions.map((ex) => (
                  <tr
                    key={ex.exception_id}
                    className="border-t"
                    style={{
                      borderColor: 'var(--color-md-outline-variant)',
                      background: ex.attempt_count >= 3 ? 'var(--color-md-error-container)' : undefined,
                    }}
                  >
                    <td className="md-typescale-body-small px-4 py-3 font-mono">{shortId(ex.transfer_id)}</td>
                    <td className="md-typescale-body-small px-4 py-3 font-mono">{shortId(ex.manifest_id)}</td>
                    <td className="px-4 py-3">{reasonBadge(ex.reason)}</td>
                    <td className="md-typescale-body-small px-4 py-3">
                      <span
                        className={ex.attempt_count >= 3 ? 'font-bold' : ''}
                        style={{ color: ex.attempt_count >= 3 ? 'var(--color-md-error)' : 'var(--color-md-on-surface)' }}
                      >
                        {ex.attempt_count}
                        {ex.attempt_count >= 3 && ' — DLQ'}
                      </span>
                    </td>
                    <td className="md-typescale-body-small px-4 py-3">
                      {ex.escalated ? 'Yes' : 'No'}
                    </td>
                    <td
                      className="md-typescale-body-small px-4 py-3"
                      style={{ color: 'var(--color-md-on-surface-variant)' }}
                    >
                      {ex.created_at ? new Date(ex.created_at).toLocaleString() : '—'}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </PageTransition>
  );
}
