'use client';

import { useEffect, useState } from 'react';
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

const NEXT_STATE: Record<string, { label: string; action: string }> = {
  APPROVED: { label: 'Start Loading', action: 'start_loading' },
  LOADING: { label: 'Mark Dispatched', action: 'dispatch' },
};

export default function TransferDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const { toast } = useToast();
  const [transfer, setTransfer] = useState<TransferDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [progressing, setProgressing] = useState(false);

  useEffect(() => {
    async function load() {
      try {
        const res = await apiFetch(`/v1/factory/transfers/${id}`);
        if (res.ok) setTransfer(await res.json());
      } catch { /* handled */ } finally {
        setLoading(false);
      }
    }
    load();
  }, [id]);

  async function handleProgress() {
    if (!transfer) return;
    const next = NEXT_STATE[transfer.state];
    if (!next) return;

    setProgressing(true);
    try {
      const res = await apiFetch(`/v1/factory/transfers/${id}/transition`, {
        method: 'POST',
        body: JSON.stringify({ action: next.action }),
      });
      if (res.ok) {
        toast(`Transfer ${next.action.replace('_', ' ')}`, 'success');
        const updated = await apiFetch(`/v1/factory/transfers/${id}`);
        if (updated.ok) setTransfer(await updated.json());
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
      <div className="p-6 space-y-4">
        <div className="md-skeleton" style={{ height: 24, width: '30%' }} />
        <div className="md-skeleton md-skeleton-card" />
      </div>
    );
  }

  if (!transfer) {
    return (
      <div className="p-6">
        <p className="text-[var(--muted)]">Transfer not found.</p>
        <button onClick={() => router.back()} className="mt-4 text-sm button--secondary px-3 py-1.5 rounded-lg">
          <Icon name="arrowBack" size={16} /> Back
        </button>
      </div>
    );
  }

  const stateClass = (s: string) => {
    const map: Record<string, string> = {
      DRAFT: 'status-chip--draft', APPROVED: 'status-chip--approved',
      LOADING: 'status-chip--loading', DISPATCHED: 'status-chip--dispatched',
      IN_TRANSIT: 'status-chip--in-transit', ARRIVED: 'status-chip--arrived',
      RECEIVED: 'status-chip--received', CANCELLED: 'status-chip--cancelled',
    };
    return map[s] || '';
  };

  const next = NEXT_STATE[transfer.state];

  return (
    <div className="p-6 space-y-6 md-animate-in">
      {/* Header */}
      <div className="flex items-center gap-3">
        <button onClick={() => router.back()} className="p-1.5 rounded-lg hover:bg-[var(--surface)]">
          <Icon name="arrowBack" size={20} />
        </button>
        <div className="flex-1">
          <h1 className="text-xl font-bold tracking-tight">
            Transfer to {transfer.warehouse_name}
          </h1>
          <p className="text-xs text-[var(--muted)] font-mono">{transfer.id}</p>
        </div>
        <span className={`status-chip ${stateClass(transfer.state)}`}>{transfer.state}</span>
      </div>

      {/* Summary card */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
        {[
          { label: 'Priority', value: transfer.priority },
          { label: 'Total Items', value: transfer.total_items },
          { label: 'Total Volume', value: `${transfer.total_volume_m3.toFixed(1)} m³` },
          { label: 'Updated', value: new Date(transfer.updated_at).toLocaleString() },
        ].map(m => (
          <div key={m.label} className="rounded-lg border border-[var(--border)] p-3">
            <div className="text-xs text-[var(--muted)] mb-0.5">{m.label}</div>
            <div className="text-sm font-semibold">{m.value}</div>
          </div>
        ))}
      </div>

      {/* Action */}
      {next && (
        <button
          onClick={handleProgress}
          disabled={progressing}
          className="px-5 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
        >
          {progressing ? 'Processing...' : next.label}
        </button>
      )}

      {/* Items table */}
      <div>
        <h2 className="text-sm font-semibold mb-2">Items</h2>
        {transfer.items?.length > 0 ? (
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)]">
                <th className="table__column text-left py-2 px-3">Product</th>
                <th className="table__column text-left py-2 px-3">SKU</th>
                <th className="table__column text-right py-2 px-3">Qty</th>
                <th className="table__column text-right py-2 px-3">Volume</th>
              </tr>
            </thead>
            <tbody>
              {transfer.items.map((item, i) => (
                <tr key={i} className="table__row">
                  <td className="py-2 px-3 font-medium">{item.product_name}</td>
                  <td className="py-2 px-3 text-[var(--muted)] font-mono text-xs">{item.sku_id.slice(0, 8)}</td>
                  <td className="py-2 px-3 text-right tabular-nums">{item.quantity}</td>
                  <td className="py-2 px-3 text-right tabular-nums">{item.volume_m3.toFixed(2)} m³</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <p className="text-sm text-[var(--muted)]">No items loaded.</p>
        )}
      </div>

      {/* Notes */}
      {transfer.notes && (
        <div>
          <h2 className="text-sm font-semibold mb-1">Notes</h2>
          <p className="text-sm text-[var(--muted)]">{transfer.notes}</p>
        </div>
      )}
    </div>
  );
}
