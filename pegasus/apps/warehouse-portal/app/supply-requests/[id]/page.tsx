'use client';

import { useEffect, useEffectEvent, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { apiFetch, connectWarehouseWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import type { WarehouseLiveEvent, WarehouseSupplyRequestDetail } from '@pegasus/types';

const ACTIONS: Record<string, { label: string; action: string; variant: string }[]> = {
  DRAFT: [
    { label: 'Submit', action: 'submit', variant: 'button--primary' },
    { label: 'Cancel', action: 'cancel', variant: 'button--danger' },
  ],
  SUBMITTED: [
    { label: 'Cancel', action: 'cancel', variant: 'button--danger' },
  ],
  ACKNOWLEDGED: [],
  IN_PRODUCTION: [],
  READY: [],
  FULFILLED: [],
  CANCELLED: [],
};

function chipClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    SUBMITTED: 'status-chip--submitted',
    ACKNOWLEDGED: 'status-chip--acknowledged',
    IN_PRODUCTION: 'status-chip--in-production',
    READY: 'status-chip--ready',
    FULFILLED: 'status-chip--fulfilled',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || 'status-chip--draft';
}

export default function SupplyRequestDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const [detail, setDetail] = useState<WarehouseSupplyRequestDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);

  const id = params.id as string;

  const loadDetail = useEffectEvent(async () => {
    setLoading(true);
    try {
      const res = await apiFetch(`/v1/warehouse/supply-requests/${id}`);
      if (res.ok) {
        setDetail(await res.json() as WarehouseSupplyRequestDetail);
      } else {
        toast('Request not found', 'error');
        router.replace('/supply-requests');
      }
    } catch {
      toast('Failed to load request', 'error');
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useEffectEvent((event: WarehouseLiveEvent) => {
    if (event.type !== 'SUPPLY_REQUEST_UPDATE' || event.request_id !== id) {
      return;
    }
    void loadDetail();
  });

  useEffect(() => {
    void loadDetail();
  }, [id, loadDetail]);

  useEffect(() => {
    const socket = connectWarehouseWS();
    socket.onmessage = message => {
      try {
        handleWarehouseLiveEvent(JSON.parse(message.data) as WarehouseLiveEvent);
      } catch {
        // Ignore unrelated frames.
      }
    };
    return () => socket.close();
  }, [handleWarehouseLiveEvent]);

  async function handleAction(action: string) {
    setActing(true);
    try {
      const res = await apiFetch(`/v1/warehouse/supply-requests/${id}`, {
        method: 'PATCH',
        body: JSON.stringify({ action }),
      });
      if (res.ok) {
        toast(`Request ${action}ed successfully`, 'success');
        void loadDetail();
      } else {
        const data = await res.json().catch(() => ({}));
        toast(data.error || `Failed to ${action}`, 'error');
      }
    } catch {
      toast('Network error', 'error');
    } finally {
      setActing(false);
    }
  }

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div className="md-skeleton md-skeleton-title" />
        <div className="md-skeleton md-skeleton-card" style={{ height: 200 }} />
      </div>
    );
  }

  if (!detail) return null;

  const actions = ACTIONS[detail.state] || [];

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center gap-3">
        <button onClick={() => router.back()} className="p-1 rounded-lg hover:bg-[var(--surface)]">
          <Icon name="left" size={20} />
        </button>
        <div className="flex-1">
          <h1 className="text-xl font-bold tracking-tight">Supply Request</h1>
          <p className="text-xs text-[var(--muted)] font-mono">{detail.request_id}</p>
        </div>
        <span className={`status-chip ${chipClass(detail.state)}`}>{detail.state}</span>
      </div>

      {/* Metadata grid */}
      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Priority</div>
          <div className={`text-sm font-semibold ${
            detail.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
            detail.priority === 'URGENT' ? 'text-[var(--warning)]' : ''
          }`}>{detail.priority}</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Delivery Date</div>
          <div className="text-sm font-semibold">
            {detail.requested_delivery_date
              ? new Date(detail.requested_delivery_date).toLocaleDateString()
              : 'Not set'}
          </div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Total Volume</div>
          <div className="text-sm font-semibold">{detail.total_volume_vu} VU</div>
        </div>
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-1">Items</div>
          <div className="text-sm font-semibold">{detail.items?.length || 0}</div>
        </div>
      </div>

      {/* Notes */}
      {detail.notes && (
        <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--surface)' }}>
          <div className="text-xs text-[var(--muted)] mb-2">Notes</div>
          <p className="text-sm whitespace-pre-wrap">{detail.notes}</p>
        </div>
      )}

      {/* Items table */}
      {detail.items && detail.items.length > 0 && (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Product</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Requested</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Recommended</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Unit Volume</th>
              </tr>
            </thead>
            <tbody>
              {detail.items.map(item => (
                <tr key={item.item_id} className="border-b border-[var(--border)] last:border-b-0">
                  <td className="px-4 py-3 font-mono text-xs">{item.product_id}</td>
                  <td className="px-4 py-3 font-mono">{item.requested_quantity}</td>
                  <td className="px-4 py-3 font-mono">{item.recommended_qty}</td>
                  <td className="px-4 py-3 text-[var(--muted)]">{item.unit_volume_vu} VU</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {/* Actions */}
      {actions.length > 0 && (
        <div className="flex gap-2">
          {actions.map(a => (
            <button
              key={a.action}
              onClick={() => handleAction(a.action)}
              disabled={acting}
              className={`px-4 py-2 rounded-lg text-sm font-semibold ${a.variant} disabled:opacity-50`}
            >
              {acting ? '...' : a.label}
            </button>
          ))}
        </div>
      )}

      {/* Transfer order link */}
      {detail.transfer_order_id && (
        <div className="text-xs text-[var(--muted)]">
          Linked Transfer Order: <span className="font-mono">{detail.transfer_order_id}</span>
        </div>
      )}
    </div>
  );
}
