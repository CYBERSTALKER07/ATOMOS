'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';

interface TransferItem {
  sku_id: string;
  product_name: string;
  quantity: number;
  volume_m3: number;
}

interface TransferDetail {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  notes: string;
  created_at: string;
  updated_at: string;
  items: TransferItem[];
}

const NEXT_STATE: Record<string, { label: string; targetState: string }> = {
  APPROVED: { label: 'Start Loading', targetState: 'LOADING' },
  LOADING: { label: 'Mark Dispatched', targetState: 'DISPATCHED' },
};

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

export default function TransferDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { toast } = useToast();
  const [transfer, setTransfer] = useState<TransferDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [progressing, setProgressing] = useState(false);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch(`/v1/factory/transfers/${id}`);
      if (res.ok) {
        setTransfer(await res.json());
      }
    } catch {
      // handled by empty state
    } finally {
      setLoading(false);
    }
  }, [id]);

  useEffect(() => { load(); }, [load]);

  async function handleProgress() {
    if (!transfer) return;
    const next = NEXT_STATE[transfer.state];
    if (!next) return;

    setProgressing(true);
    try {
      const res = await apiFetch(`/v1/factory/transfers/${id}/transition`, {
        method: 'POST',
        body: JSON.stringify({ target_state: next.targetState }),
      });
      if (res.ok) {
        toast(next.label, 'success');
        await load();
      } else {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Transition failed', 'error');
      }
    } catch {
      toast('Request failed', 'error');
    } finally {
      setProgressing(false);
    }
  }

  if (loading) {
    return (
      <div className="space-y-4 p-6 md:p-8">
        <div className="md-skeleton" style={{ height: 24, width: '30%' }} />
        <div className="md-skeleton md-skeleton-card" />
      </div>
    );
  }

  if (!transfer) {
    return (
      <div className="p-6 md:p-8">
        <p className="text-[var(--muted)]">Transfer not found.</p>
        <button onClick={() => router.back()} className="button--secondary mt-4 rounded-lg px-3 py-1.5 text-sm">
          <Icon name="arrowBack" size={16} /> Back
        </button>
      </div>
    );
  }

  const next = NEXT_STATE[transfer.state];
  const transferTitle = transfer.warehouse_name || transfer.destination_warehouse_id.slice(0, 8);
  const summaryCards = [
    { label: 'Priority', value: transfer.priority },
    { label: 'Total Items', value: transfer.total_items },
    { label: 'Total Volume', value: `${transfer.total_volume_m3.toFixed(1)} m³` },
    { label: 'Updated', value: new Date(transfer.updated_at).toLocaleString() },
  ];

  return (
    <div className="space-y-6 p-6 md:animate-in md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div className="flex min-w-0 items-start gap-3">
          <button onClick={() => router.back()} className="mt-1 rounded-xl border border-[var(--border)] bg-[var(--background)] p-2 transition-colors hover:bg-[var(--surface)]">
            <Icon name="arrowBack" size={18} />
          </button>
          <div className="min-w-0">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Transfer detail</p>
            <h1 className="truncate text-2xl font-semibold tracking-tight text-[var(--foreground)]">
              Transfer to {transferTitle}
            </h1>
            <p className="mt-1 text-sm font-mono text-[var(--muted)]">{transfer.id}</p>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-3">
          <span className={`status-chip ${stateClass(transfer.state)}`}>{transfer.state}</span>
          {next && (
            <button
              onClick={handleProgress}
              disabled={progressing}
              className="button--primary inline-flex h-10 items-center gap-2 rounded-full px-5 text-sm font-semibold disabled:opacity-50"
            >
              {progressing ? 'Processing...' : next.label}
            </button>
          )}
        </div>
      </div>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {summaryCards.map((metric) => (
          <div key={metric.label} className="rounded-2xl border border-[var(--border)] bg-[var(--background)] p-5">
            <p className="text-sm font-medium text-[var(--muted)]">{metric.label}</p>
            <div className="mt-4 text-xl font-semibold tracking-tight text-[var(--foreground)]">{metric.value}</div>
          </div>
        ))}
      </section>

      <div className="grid gap-6 xl:grid-cols-[1.2fr_0.8fr]">
        <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Manifest contents</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Items in this transfer</h2>
            </div>
            <div className="rounded-full bg-[var(--surface)] px-4 py-2 text-sm text-[var(--muted)]">
              {transfer.items?.length ?? 0} line item(s)
            </div>
          </div>

          {transfer.items?.length > 0 ? (
            <div className="mt-5 overflow-x-auto">
              <table className="w-full min-w-[640px] text-sm">
                <thead>
                  <tr className="table__header border-b border-[var(--border)]">
                    <th className="table__column px-3 py-3 text-left">Product</th>
                    <th className="table__column px-3 py-3 text-left">SKU</th>
                    <th className="table__column px-3 py-3 text-right">Qty</th>
                    <th className="table__column px-3 py-3 text-right">Volume</th>
                  </tr>
                </thead>
                <tbody>
                  {transfer.items.map((item) => (
                    <tr key={item.sku_id} className="table__row">
                      <td className="px-3 py-3 font-medium text-[var(--foreground)]">{item.product_name}</td>
                      <td className="px-3 py-3 font-mono text-xs text-[var(--muted)]">{item.sku_id}</td>
                      <td className="px-3 py-3 text-right font-semibold tabular-nums text-[var(--foreground)]">{item.quantity}</td>
                      <td className="px-3 py-3 text-right tabular-nums text-[var(--foreground)]">{item.volume_m3.toFixed(2)} m³</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <div className="mt-5 rounded-2xl border border-dashed border-[var(--border)] bg-[var(--surface)] px-4 py-12 text-center text-sm text-[var(--muted)]">
              No items have been loaded into this transfer yet.
            </div>
          )}
        </section>

        <aside className="space-y-4">
          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Transfer overview</p>
            <div className="mt-4 space-y-4">
              {[
                { label: 'Warehouse', value: transferTitle },
                { label: 'Source factory', value: transfer.source_factory_id },
                { label: 'Destination warehouse', value: transfer.destination_warehouse_id },
                { label: 'Created', value: new Date(transfer.created_at).toLocaleString() },
              ].map((row) => (
                <div key={row.label} className="rounded-2xl bg-[var(--surface)] p-4">
                  <p className="text-[11px] font-semibold uppercase tracking-[0.16em] text-[var(--muted)]">{row.label}</p>
                  <p className="mt-2 break-all text-sm font-medium leading-6 text-[var(--foreground)]">{row.value}</p>
                </div>
              ))}
            </div>
          </section>

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <div className="flex items-center justify-between gap-3">
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Next transition</p>
              {next ? <span className="status-chip status-chip--warning">Action available</span> : <span className="status-chip status-chip--stable">No action</span>}
            </div>
            <p className="mt-3 text-sm leading-6 text-[var(--muted)]">
              {next
                ? `${next.label} is the next allowed operation for this transfer. Use it once the factory floor confirms readiness.`
                : 'This transfer is already beyond the operator-controlled loading transitions.'}
            </p>
            {next && (
              <button
                onClick={handleProgress}
                disabled={progressing}
                className="button--primary mt-5 inline-flex h-10 items-center gap-2 rounded-full px-5 text-sm font-semibold disabled:opacity-50"
              >
                {progressing ? 'Processing...' : next.label}
              </button>
            )}
          </section>

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Notes</p>
            <p className="mt-3 text-sm leading-6 text-[var(--muted)]">
              {transfer.notes || 'No operator notes have been attached to this transfer.'}
            </p>
          </section>
        </aside>
      </div>
    </div>
  );
}
