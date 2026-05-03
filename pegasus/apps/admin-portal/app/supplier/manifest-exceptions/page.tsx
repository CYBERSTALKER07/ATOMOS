'use client';

import { useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Icon from '@/components/Icon';

/* ─── Types ───────────────────────────────────────────────── */

interface ManifestException {
  exception_id: string;
  order_id: string;
  manifest_id: string;
  supplier_id: string;
  reason: string;
  metadata: string;
  attempt_count: number;
  created_at: string;
}

/* ─── Helpers ─────────────────────────────────────────────── */

function shortId(id: string): string {
  return id.length > 12 ? `${id.slice(0, 8)}…` : id;
}

function reasonBadge(reason: string) {
  const colors: Record<string, string> = {
    OVERFLOW: 'var(--color-md-warning)',
    DAMAGED: 'var(--color-md-error)',
    MANUAL: 'var(--color-md-info)',
    NO_CAPACITY: 'var(--color-md-error)',
  };
  return (
    <span
      className="md-typescale-label-small md-shape-sm px-2 py-0.5 inline-block"
      style={{
        background: colors[reason] || 'var(--color-md-outline)',
        color: '#fff',
      }}
    >
      {reason}
    </span>
  );
}

/* ─── Page ────────────────────────────────────────────────── */

export default function ManifestExceptionsPage() {
  const [exceptions, setExceptions] = useState<ManifestException[]>([]);
  const [loading, setLoading] = useState(true);
  const [escalatedOnly, setEscalatedOnly] = useState(false);

  const fetchExceptions = useCallback(async (signal?: AbortSignal) => {
    try {
      const qs = escalatedOnly ? '?escalated=true' : '';
      const res = await apiFetch(`/v1/supplier/manifest-exceptions${qs}`, { signal });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setExceptions(data.exceptions || []);
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
    } finally {
      setLoading(false);
    }
  }, [escalatedOnly]);

  usePolling(fetchExceptions, 15000);

  /* ─── Render ──────────────────────────────────────────── */

  return (
    <div className="flex flex-col gap-6 p-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-small" style={{ color: 'var(--color-md-on-surface)' }}>
            Loading Gate Exceptions
          </h1>
          <p className="md-typescale-body-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            Orders removed from manifests during loading — overflow, damage, or manual pull
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            className={`md-btn ${escalatedOnly ? 'md-btn-filled' : 'md-btn-outlined'} md-typescale-label-large px-4 py-2`}
            onClick={() => setEscalatedOnly(!escalatedOnly)}
          >
            <Icon name="error" size={18} />
            {escalatedOnly ? 'Showing Escalated' : 'Show Escalated Only'}
          </button>
        </div>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex flex-col gap-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full rounded-lg" />
          ))}
        </div>
      ) : exceptions.length === 0 ? (
        <EmptyState
          icon="check_circle"
          headline={escalatedOnly ? 'No escalated exceptions' : 'No exceptions'}
          body={escalatedOnly ? 'No orders have hit the DLQ threshold (3+ overflows)' : 'All manifest loading operations completed without exceptions'}
        />
      ) : (
        <div className="md-card md-elevation-1 md-shape-md overflow-hidden">
          <table className="w-full">
            <thead>
              <tr style={{ background: 'var(--color-md-surface-container)' }}>
                {['Order', 'Manifest', 'Reason', 'Attempts', 'Time'].map((h) => (
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
                  <td className="md-typescale-body-small px-4 py-3 font-mono">
                    {shortId(ex.order_id)}
                  </td>
                  <td className="md-typescale-body-small px-4 py-3 font-mono">
                    {shortId(ex.manifest_id)}
                  </td>
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
                  <td
                    className="md-typescale-body-small px-4 py-3"
                    style={{ color: 'var(--color-md-on-surface-variant)' }}
                  >
                    {new Date(ex.created_at).toLocaleString()}
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
