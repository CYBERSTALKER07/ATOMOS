'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { useParams } from 'next/navigation';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';
import { MANIFEST_STATE_ORDER, manifestTransitionIdempotencyKey, nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';

interface ManifestTransfer {
  transfer_id: string;
  order_id?: string;
  state: string;
  total_vu?: number;
}

interface ManifestTransition {
  action: string;
  from_state?: string;
  to_state?: string;
  at?: string;
  reason?: string;
}

interface ManifestException {
  exception_id: string;
  transfer_id: string;
  reason: string;
  attempt_count: number;
}

interface ManifestDetailPayload {
  manifest: {
    manifest_id: string;
    state: string;
    transfer_count?: number;
    total_volume_vu?: number;
    max_volume_vu?: number;
    driver_id?: string;
    vehicle_id?: string;
    updated_at?: string;
  };
  transfers: ManifestTransfer[];
  transitions: ManifestTransition[];
  exceptions: ManifestException[];
  route_id?: string;
  stop_count?: number;
  order_count?: number;
}

function stateClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    LOADING: 'status-chip--loading',
    SEALED: 'status-chip--approved',
    DISPATCHED: 'status-chip--dispatched',
    COMPLETED: 'status-chip--received',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || '';
}

export default function ManifestDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { toast } = useToast();
  const [detail, setDetail] = useState<ManifestDetailPayload | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!id) return;
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch(`/v1/factory/manifests/${id}`);
      if (!res.ok) throw new Error(`Unable to load manifest (${res.status})`);
      setDetail(await res.json());
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unable to load manifest');
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { void load(); }, [load]);

  useFactorySessionReconcile(() => {
    setActing(false);
    void load();
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (payload) => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) return;
        if (event.type === 'FACTORY_MANIFEST_UPDATE') void load();
      },
    });
    return unsubscribe;
  }, [load]);

  const runLifecycle = async () => {
    if (!detail) return;
    const next = nextManifestLifecycleAction(detail.manifest.state);
    if (!next) return;
    setActing(true);
    try {
      const idempotencyKey = manifestTransitionIdempotencyKey(
        detail.manifest.manifest_id,
        next.path,
        factoryOperatorId(),
      );
      const res = await apiFetch(`/v1/factory/manifests/${detail.manifest.manifest_id}/${next.path}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': idempotencyKey,
        },
        body: JSON.stringify({ reason: 'factory-portal' }),
      });
      if (!res.ok) {
        const payload = await res.json().catch(() => ({}));
        throw new Error(payload.message || payload.error || `Transition failed (${res.status})`);
      }
      toast(`${next.label} applied`, 'success');
      await load();
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Transition failed', 'error');
    } finally {
      setActing(false);
    }
  };

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState kind="loading" title="Manifest detail" subtitle="Loading manifest snapshot." />
        </div>
      </PageTransition>
    );
  }

  if (error || !detail) {
    return (
      <PageTransition>
        <div className="p-6">
          <FactoryPageState
            kind="error"
            headline="Unable to load manifest"
            body={error || 'Manifest not found'}
            actionLabel="Retry"
            onAction={() => void load()}
          />
        </div>
      </PageTransition>
    );
  }

  const next = nextManifestLifecycleAction(detail.manifest.state);
  const stateIndex = MANIFEST_STATE_ORDER.indexOf(detail.manifest.state as (typeof MANIFEST_STATE_ORDER)[number]);

  return (
    <PageTransition className="space-y-6 p-6 md:p-8">
      <div>
        <Link href="/manifests" className="text-sm text-[var(--muted)] hover:text-[var(--foreground)]">← Back to manifests</Link>
        <div className="mt-2 flex flex-wrap items-center gap-3">
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--foreground)] font-mono">{detail.manifest.manifest_id}</h1>
          <span className={`status-chip ${stateClass(detail.manifest.state)}`}>{detail.manifest.state}</span>
        </div>
        <p className="mt-2 text-sm text-[var(--muted)]">
          Route {detail.route_id || '—'} · {detail.order_count ?? detail.transfers.length} orders · {detail.stop_count ?? '—'} stops
        </p>
      </div>

      <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-4">
        <h2 className="text-lg font-semibold">LEO lifecycle</h2>
        <div className="flex flex-wrap gap-2">
          {MANIFEST_STATE_ORDER.slice(0, 5).map((state, index) => (
            <span
              key={state}
              className={`rounded-full px-3 py-1 text-xs font-semibold ${
                index <= stateIndex ? 'bg-[var(--accent)] text-[var(--accent-foreground)]' : 'bg-[var(--surface)] text-[var(--muted)]'
              }`}
            >
              {state}
            </span>
          ))}
        </div>
        {next ? (
          <button
            type="button"
            disabled={acting}
            onClick={() => void runLifecycle()}
            className="md-btn md-btn-filled md-typescale-label-large inline-flex items-center gap-2 px-6 py-2 disabled:opacity-60"
          >
            <Icon name="loadingBay" size={18} />
            {acting ? 'Applying…' : next.label}
          </button>
        ) : (
          <p className="text-sm text-[var(--muted)]">No further lifecycle transitions for this state.</p>
        )}
      </section>

      <section className="grid gap-6 lg:grid-cols-2">
        <div className="md-card md-elevation-1 md-shape-md p-4">
          <h3 className="md-typescale-title-medium mb-3">Transfers</h3>
          {detail.transfers.length === 0 ? (
            <p className="text-sm text-[var(--muted)]">No transfers on this manifest.</p>
          ) : (
            <ul className="space-y-2">
              {detail.transfers.map((transfer) => (
                <li key={transfer.transfer_id} className="flex justify-between text-sm border-b border-[var(--border)] pb-2">
                  <span className="font-mono">{transfer.transfer_id}</span>
                  <span>{transfer.state}</span>
                </li>
              ))}
            </ul>
          )}
        </div>
        <div className="md-card md-elevation-1 md-shape-md p-4">
          <h3 className="md-typescale-title-medium mb-3">Transitions</h3>
          {detail.transitions.length === 0 ? (
            <p className="text-sm text-[var(--muted)]">No transitions recorded yet.</p>
          ) : (
            <ul className="space-y-2">
              {detail.transitions.map((transition, index) => (
                <li key={`${transition.action}-${index}`} className="text-sm border-b border-[var(--border)] pb-2">
                  <div className="font-medium">{transition.action}</div>
                  <div className="text-[var(--muted)]">
                    {transition.from_state || '—'} → {transition.to_state || '—'}
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      {detail.exceptions.length > 0 && (
        <section className="md-card md-elevation-1 md-shape-md p-4">
          <h3 className="md-typescale-title-medium mb-3">Exceptions</h3>
          <ul className="space-y-2">
            {detail.exceptions.map((exception) => (
              <li key={exception.exception_id} className="text-sm">
                {exception.reason} on {exception.transfer_id} ({exception.attempt_count} attempts)
              </li>
            ))}
          </ul>
        </section>
      )}
    </PageTransition>
  );
}
