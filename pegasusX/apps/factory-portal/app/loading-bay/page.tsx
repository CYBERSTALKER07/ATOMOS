'use client';

import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import { motion } from 'framer-motion';

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

  return (
    <PageTransition>
      <PageChrome
        title="Loading bay"
        description="Review approved factory transfers, advance active loading, and dispatch manifests without losing warehouse context."
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              onClick={() => void load()}
              className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
            >
              <Icon name="refresh" size={16} /> Refresh
            </button>
            <button
              type="button"
              onClick={handleDispatch}
              disabled={dispatching}
              className="button--primary inline-flex h-10 items-center gap-2 rounded-full px-5 text-sm font-semibold disabled:opacity-50"
            >
              {dispatching ? 'Dispatching...' : 'Batch dispatch'}
            </button>
          </div>
        }
      >
        <KpiStatGrid columns={4}>
          <KpiStatCard label="Ready to load" value={readyCount} sub="Awaiting operator attention" />
          <KpiStatCard label="Now loading" value={loadingCount} sub="Active bay work" />
          <KpiStatCard label="Dispatched" value={dispatchedCount} sub="Already moved out" />
          <KpiStatCard
            label="Total volume"
            value={`${totalVolume.toFixed(1)} m³`}
            sub={latestUpdatedAt ? `Updated ${new Date(latestUpdatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : 'No updates yet'}
          />
        </KpiStatGrid>

        {transfers.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-production-line.png"
            headline="No active transfers in the loading bay"
            body="Approved transfers will appear here as soon as warehouse demand is accepted."
          />
        ) : (
          <div className="mt-6 grid gap-4 xl:grid-cols-3">
            {grouped.map((column) => (
              <PageSection
                key={column.key}
                title={column.label}
                description={
                  column.key === 'APPROVED'
                    ? 'Approved transfers waiting for bay operators.'
                    : column.key === 'LOADING'
                      ? 'Transfers currently being loaded or sealed for dispatch.'
                      : 'Transfers that already left the loading bay this cycle.'
                }
                actions={<span className={`status-chip ${column.css}`}>{column.items.length}</span>}
              >
                <motion.div
                  className="space-y-3"
                  initial="hidden"
                  animate="show"
                  variants={{
                    hidden: { opacity: 0 },
                    show: { opacity: 1, transition: { staggerChildren: 0.05 } },
                  }}
                >
                  {column.items.map((transfer) => (
                    <Link key={transfer.id} href={`/transfers/${transfer.id}`}>
                      <motion.div
                        variants={{
                          hidden: { opacity: 0, y: 10 },
                          show: { opacity: 1, y: 0 },
                        }}
                        whileTap={{ scale: 0.98 }}
                        className="block rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift active-press"
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
                      </motion.div>
                    </Link>
                  ))}

                  {column.items.length === 0 && (
                    <div className="rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] px-4 py-10 text-center text-sm text-[var(--muted)]">
                      No transfers in this stage.
                    </div>
                  )}
                </motion.div>
              </PageSection>
            ))}
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}
