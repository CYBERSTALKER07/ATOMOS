'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Transfer {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  created_at: string;
  updated_at: string;
}

const STATE_FILTERS = ['ALL', 'DRAFT', 'APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVED', 'RECEIVED', 'CANCELLED'];

function stateClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    APPROVED: 'status-chip--approved',
    LOADING: 'status-chip--loading',
    DISPATCHED: 'status-chip--dispatched',
    IN_TRANSIT: 'status-chip--in-transit',
    ARRIVED: 'status-chip--arrived',
    RECEIVED: 'status-chip--received',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || '';
}

function priorityTone(priority: string): { background: string; color: string } {
  if (priority === 'HIGH') return { background: 'var(--danger)', color: 'var(--danger-foreground)' };
  if (priority === 'MEDIUM') return { background: 'var(--color-md-warning-container)', color: 'var(--color-md-on-warning-container)' };
  return { background: 'var(--surface)', color: 'var(--foreground)' };
}

export default function TransfersPage() {
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [stateFilter, setStateFilter] = useState('ALL');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const query = stateFilter !== 'ALL' ? `?state=${stateFilter}` : '';
      const res = await apiFetch(`/v1/factory/transfers${query}`);
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      }
    } catch {
      // empty state handled below
    } finally {
      setLoading(false);
    }
  }, [stateFilter]);

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

  const totalVolume = useMemo(
    () => transfers.reduce((sum, transfer) => sum + transfer.total_volume_m3, 0),
    [transfers],
  );
  const approvedCount = useMemo(
    () => transfers.filter((transfer) => transfer.state === 'APPROVED').length,
    [transfers],
  );
  const inFlightCount = useMemo(
    () => transfers.filter((transfer) => ['LOADING', 'DISPATCHED', 'IN_TRANSIT'].includes(transfer.state)).length,
    [transfers],
  );
  const highPriorityCount = useMemo(
    () => transfers.filter((transfer) => transfer.priority === 'HIGH').length,
    [transfers],
  );

  return (
    <div className="space-y-6 p-6 md:animate-in md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Transfer coordination</p>
          <h1 className="mt-1 text-2xl font-semibold tracking-tight text-[var(--foreground)]">Transfers</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[var(--muted)]">
            Review warehouse destination movements, filter by state, and open a transfer when it needs action from the factory floor.
          </p>
        </div>
        <button onClick={() => load()} className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {[
          { label: 'Visible transfers', value: transfers.length, helper: stateFilter === 'ALL' ? 'Across the full pipeline' : `Filtered to ${stateFilter}` },
          { label: 'Approved', value: approvedCount, helper: 'Waiting to enter loading' },
          { label: 'In flight', value: inFlightCount, helper: 'Loading, dispatched, or in transit' },
          { label: 'High priority', value: highPriorityCount, helper: 'Require close operator attention' },
        ].map((card) => (
          <div key={card.label} className="rounded-2xl border border-[var(--border)] bg-[var(--background)] p-5">
            <p className="text-sm font-medium text-[var(--muted)]">{card.label}</p>
            <div className="mt-4 text-3xl font-semibold tracking-tight text-[var(--foreground)] tabular-nums">{card.value}</div>
            <p className="mt-2 text-sm text-[var(--muted)]">{card.helper}</p>
          </div>
        ))}
      </section>

      <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
        <div className="flex flex-wrap items-center justify-between gap-4">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Transfer filter</p>
            <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Pipeline view</h2>
          </div>
          <div className="rounded-full bg-[var(--surface)] px-4 py-2 text-sm text-[var(--muted)]">
            Total volume: <span className="font-semibold text-[var(--foreground)]">{totalVolume.toFixed(1)} m³</span>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-2">
          {STATE_FILTERS.map((filter) => (
            <button
              key={filter}
              onClick={() => setStateFilter(filter)}
              className={`rounded-full border px-4 py-2 text-xs font-semibold uppercase tracking-[0.14em] transition-colors ${
                stateFilter === filter
                  ? 'border-transparent bg-[var(--accent)] text-[var(--accent-foreground)]'
                  : 'border-[var(--border)] bg-transparent text-[var(--muted)] hover:border-[var(--accent)] hover:text-[var(--foreground)]'
              }`}
            >
              {filter}
            </button>
          ))}
        </div>
      </section>

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 6 }).map((_, index) => <div key={index} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : transfers.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-[28px] border border-dashed border-[var(--border)] bg-[var(--background)] py-20 text-[var(--muted)]">
          <Icon name="transfers" size={48} className="mb-3 opacity-40" />
          <p className="text-base font-medium text-[var(--foreground)]">No transfers found</p>
          <p className="mt-2 text-sm">Adjust the state filter or wait for the next warehouse request cycle.</p>
        </div>
      ) : (
        <section className="overflow-hidden rounded-[28px] border border-[var(--border)] bg-[var(--background)]">
          <div className="overflow-x-auto">
            <table className="w-full min-w-[880px] text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)]">
                  <th className="table__column px-4 py-3 text-left font-medium">Warehouse</th>
                  <th className="table__column px-4 py-3 text-left font-medium">State</th>
                  <th className="table__column px-4 py-3 text-left font-medium">Priority</th>
                  <th className="table__column px-4 py-3 text-right font-medium">Items</th>
                  <th className="table__column px-4 py-3 text-right font-medium">Volume</th>
                  <th className="table__column px-4 py-3 text-right font-medium">Created</th>
                  <th className="table__column px-4 py-3 text-right font-medium">Action</th>
                </tr>
              </thead>
              <tbody>
                {transfers.map((transfer) => {
                  const priorityStyle = priorityTone(transfer.priority);
                  return (
                    <tr key={transfer.id} className="table__row">
                      <td className="px-4 py-4">
                        <Link href={`/transfers/${transfer.id}`} className="block">
                          <span className="block font-semibold text-[var(--foreground)] hover:underline">
                            {transfer.warehouse_name || transfer.destination_warehouse_id.slice(0, 8)}
                          </span>
                          <span className="mt-1 block text-xs font-mono text-[var(--muted)]">{transfer.id}</span>
                        </Link>
                      </td>
                      <td className="px-4 py-4">
                        <span className={`status-chip ${stateClass(transfer.state)}`}>{transfer.state}</span>
                      </td>
                      <td className="px-4 py-4">
                        <span
                          className="inline-flex rounded-full px-3 py-1 text-xs font-semibold uppercase tracking-[0.14em]"
                          style={priorityStyle}
                        >
                          {transfer.priority}
                        </span>
                      </td>
                      <td className="px-4 py-4 text-right font-semibold tabular-nums text-[var(--foreground)]">{transfer.total_items}</td>
                      <td className="px-4 py-4 text-right tabular-nums text-[var(--foreground)]">{transfer.total_volume_m3.toFixed(1)} m³</td>
                      <td className="px-4 py-4 text-right text-[var(--muted)]">{new Date(transfer.created_at).toLocaleDateString()}</td>
                      <td className="px-4 py-4 text-right">
                        <Link
                          href={`/transfers/${transfer.id}`}
                          className="inline-flex items-center gap-2 rounded-full border border-[var(--border)] px-3 py-2 text-xs font-semibold uppercase tracking-[0.14em] text-[var(--foreground)] transition-colors hover:border-[var(--accent)]"
                        >
                          Open
                          <Icon name="chevronR" size={14} />
                        </Link>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      )}
    </div>
  );
}
