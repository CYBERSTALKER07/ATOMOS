'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';

type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';

interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}

const COLUMNS: { key: TransferState; label: string; css: string }[] = [
  { key: 'APPROVED', label: 'Ready for Loading', css: 'status-chip--approved' },
  { key: 'LOADING', label: 'Now Loading', css: 'status-chip--loading' },
  { key: 'DISPATCHED', label: 'Dispatched', css: 'status-chip--dispatched' },
];

export default function LoadingBayPage() {
  const { toast } = useToast();
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [dispatching, setDispatching] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/transfers?states=APPROVED,LOADING,DISPATCHED');
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      }
    } catch {
      // handled by empty state
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

  const grouped = useMemo(
    () =>
      COLUMNS.map((column) => ({
        ...column,
        items: transfers.filter((transfer) => transfer.state === column.key),
      })),
    [transfers],
  );
  const totalVolume = useMemo(
    () => transfers.reduce((sum, transfer) => sum + transfer.total_volume_m3, 0),
    [transfers],
  );
  const latestUpdatedAt = transfers.reduce<string | null>((latest, transfer) => {
    if (!latest) return transfer.updated_at;
    return new Date(transfer.updated_at).getTime() > new Date(latest).getTime() ? transfer.updated_at : latest;
  }, null);
  const readyCount = grouped.find((column) => column.key === 'APPROVED')?.items.length ?? 0;
  const loadingCount = grouped.find((column) => column.key === 'LOADING')?.items.length ?? 0;
  const dispatchedCount = grouped.find((column) => column.key === 'DISPATCHED')?.items.length ?? 0;

  async function handleDispatch() {
    setDispatching(true);
    try {
      const res = await apiFetch('/v1/factory/dispatch', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        toast(`Dispatched ${data.manifests_created || 0} manifest(s)`, 'success');
        load();
      } else {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Dispatch failed', 'error');
      }
    } catch {
      toast('Dispatch request failed', 'error');
    } finally {
      setDispatching(false);
    }
  }

  if (loading) {
    return (
      <div className="p-6 md:p-8">
        <div className="md-skeleton md-skeleton-title" />
        <div className="mt-4 grid gap-4 xl:grid-cols-3">
          {[1, 2, 3].map((index) => (
            <div key={index} className="md-skeleton md-skeleton-card" />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6 p-6 md:animate-in md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Dispatch workspace</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight text-[var(--foreground)]">Loading Bay</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[var(--muted)]">
            Review approved factory transfers, advance active loading, and dispatch manifests without losing warehouse context.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => load()}
            className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
          <button
            onClick={handleDispatch}
            disabled={dispatching}
            className="button--primary inline-flex h-10 items-center gap-2 rounded-full px-5 text-sm font-semibold disabled:opacity-50"
          >
            {dispatching ? 'Dispatching...' : 'Batch Dispatch'}
          </button>
        </div>
      </div>

      <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 shadow-[var(--shadow-md-elevation-1)]">
        <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Bay summary</p>
            <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">
              {readyCount > 0
                ? `${readyCount} approved transfer(s) are ready for operators.`
                : 'No approved transfers are waiting in the bay.'}
            </h2>
            <p className="mt-3 text-sm leading-6 text-[var(--muted)]">
              Use the kanban columns to keep warehouse-destination context visible while you advance transfers from approval to loading to dispatched.
            </p>
            <div className="mt-5 grid gap-3 sm:grid-cols-3">
              {[
                { label: 'Ready to load', value: readyCount, helper: 'Awaiting operator attention', tone: 'status-chip--approved' },
                { label: 'Now loading', value: loadingCount, helper: 'Active bay work', tone: 'status-chip--loading' },
                { label: 'Dispatched', value: dispatchedCount, helper: 'Already moved out', tone: 'status-chip--dispatched' },
              ].map((metric) => (
                <div key={metric.label} className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4">
                  <span className={`status-chip ${metric.tone}`}>{metric.label}</span>
                  <div className="mt-4 text-3xl font-semibold tracking-tight text-[var(--foreground)] tabular-nums">{metric.value}</div>
                  <p className="mt-1 text-sm text-[var(--muted)]">{metric.helper}</p>
                </div>
              ))}
            </div>
          </div>

          <div className="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
            {[
              { label: 'Total transfers', value: transfers.length, helper: 'Visible across the bay board' },
              { label: 'Total volume', value: `${totalVolume.toFixed(1)} m³`, helper: 'Combined payload volume in play' },
              {
                label: 'Latest update',
                value: latestUpdatedAt ? new Date(latestUpdatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) : '—',
                helper: latestUpdatedAt ? new Date(latestUpdatedAt).toLocaleDateString() : 'No transfer updates yet',
              },
            ].map((stat) => (
              <div key={stat.label} className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4">
                <p className="text-sm font-medium text-[var(--muted)]">{stat.label}</p>
                <div className="mt-3 text-2xl font-semibold tracking-tight text-[var(--foreground)]">{stat.value}</div>
                <p className="mt-2 text-sm text-[var(--muted)]">{stat.helper}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      {transfers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-[28px] border border-dashed border-[var(--border)] bg-[var(--background)] py-20 text-[var(--muted)]">
          <Icon name="loadingBay" size={48} className="mb-3 opacity-40" />
          <p className="text-base font-medium text-[var(--foreground)]">No active transfers in the loading bay</p>
          <p className="mt-2 text-sm">Approved transfers will appear here as soon as warehouse demand is accepted.</p>
        </div>
      ) : (
        <div className="grid gap-4 xl:grid-cols-3">
          {grouped.map((column) => (
            <section key={column.key} className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <span className={`status-chip ${column.css}`}>{column.label}</span>
                  <p className="mt-2 text-sm leading-6 text-[var(--muted)]">
                    {column.key === 'APPROVED' && 'Approved transfers waiting for bay operators.'}
                    {column.key === 'LOADING' && 'Transfers currently being loaded or sealed for dispatch.'}
                    {column.key === 'DISPATCHED' && 'Transfers that already left the loading bay this cycle.'}
                  </p>
                </div>
                <span className="rounded-full bg-[var(--surface)] px-3 py-1 text-xs font-semibold text-[var(--foreground)]">
                  {column.items.length}
                </span>
              </div>

              <div className="mt-4 space-y-3">
                {column.items.map((transfer) => (
                  <Link
                    key={transfer.id}
                    href={`/transfers/${transfer.id}`}
                    className="block rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)]"
                  >
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <p className="truncate text-base font-semibold text-[var(--foreground)]">{transfer.warehouse_name}</p>
                        <p className="mt-1 text-xs font-mono text-[var(--muted)]">{transfer.id}</p>
                      </div>
                      <Icon name="chevronR" size={16} className="text-[var(--muted)]" />
                    </div>

                    <div className="mt-4 grid grid-cols-2 gap-3">
                      <div className="rounded-xl bg-[var(--background)] p-3">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Items</p>
                        <p className="mt-2 text-lg font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_items}</p>
                      </div>
                      <div className="rounded-xl bg-[var(--background)] p-3">
                        <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">Volume</p>
                        <p className="mt-2 text-lg font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_volume_m3.toFixed(1)} m³</p>
                      </div>
                    </div>

                    <div className="mt-4 flex items-center justify-between gap-3 text-xs text-[var(--muted)]">
                      <span>Created {new Date(transfer.created_at).toLocaleDateString()}</span>
                      <span>Updated {new Date(transfer.updated_at).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}</span>
                    </div>
                  </Link>
                ))}

                {column.items.length === 0 && (
                  <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] px-4 py-10 text-center text-sm text-[var(--muted)]">
                    No transfers in this stage.
                  </div>
                )}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
