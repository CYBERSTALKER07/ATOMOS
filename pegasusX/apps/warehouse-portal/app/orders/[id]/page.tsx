'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Link from 'next/link';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';

interface OrderDetail {
  order_id: string;
  retailer_name?: string;
  state?: string;
  status?: string;
  total_uzs?: number;
  total_minor?: number;
}

export default function OrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { toast } = useToast();
  const orderId = String(params.id ?? '');
  const [order, setOrder] = useState<OrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [reason, setReason] = useState('');

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const data = await warehouseApi.getWarehouseOrders({});
      const row = (data.orders ?? []).find((o) => o.order_id === orderId) as OrderDetail | undefined;
      if (!row) {
        toast('Order not found', 'error');
        router.replace('/orders');
        return;
      }
      setOrder(row);
    } catch {
      toast('Failed to load order', 'error');
    } finally {
      setLoading(false);
    }
  }, [orderId, router, toast]);

  useEffect(() => { load(); }, [load]);

  const state = (order?.state ?? order?.status ?? '').toUpperCase();
  const canDelay = state === 'PENDING' || state === 'LOADED';
  const canReject = state === 'PENDING' || state === 'LOADED' || state === 'IN_TRANSIT';
  const canOverflow = state === 'LOADED' || state === 'IN_TRANSIT';

  async function runMutation(
    label: string,
    fn: () => Promise<{ status?: string }>,
    requiresReason = false,
  ) {
    if (requiresReason && !reason.trim()) {
      toast('Reason is required', 'error');
      return;
    }
    setActing(true);
    try {
      const resp = await fn();
      toast(`${label} · ${resp.status ?? 'ok'}`, 'success');
      await load();
    } catch {
      toast(`${label} failed`, 'error');
    } finally {
      setActing(false);
    }
  }

  if (!loading && !order) return null;

  const total = order?.total_uzs ?? order?.total_minor ?? 0;

  return (
    <PageTransition>
      <PageChrome
        title={`Order ${orderId.slice(0, 8)}…`}
        description="Warehouse-scoped order detail and exception actions."
        loading={loading}
        actions={
          <Link href="/orders" className="button--secondary px-3 py-1.5 rounded-lg text-sm inline-flex items-center gap-1.5">
            <Icon name="arrow_back" size={16} /> Back
          </Link>
        }
      >
      {order ? (
      <div className="max-w-3xl space-y-6">
      <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 space-y-3">
        <div className="flex justify-between gap-4 flex-wrap">
          <div>
            <p className="text-sm text-[var(--muted)]">Retailer</p>
            <p className="font-medium">{order.retailer_name || '—'}</p>
          </div>
          <div>
            <p className="text-sm text-[var(--muted)]">State</p>
            <span className="status-chip">{state || '—'}</span>
          </div>
          <div>
            <p className="text-sm text-[var(--muted)]">Total (UZS)</p>
            <p className="font-mono tabular-nums">{new Intl.NumberFormat('uz-UZ').format(total)}</p>
          </div>
        </div>
      </div>

      {(canDelay || canReject || canOverflow) && (
        <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 space-y-4">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--muted)]">Warehouse actions</h2>
          <textarea
            value={reason}
            onChange={(e) => setReason(e.target.value)}
            placeholder="Reason (required for reject)"
            rows={2}
            className="w-full px-3 py-2 rounded-lg border text-sm"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
          />
          <div className="flex flex-wrap gap-2">
            {canDelay && (
              <button
                type="button"
                disabled={acting}
                className="button--secondary px-4 py-2 rounded-lg text-sm"
                onClick={() => runMutation('Delayed', () => warehouseOps.delayOrder(orderId, reason.trim() || undefined))}
              >
                Delay
              </button>
            )}
            {canOverflow && (
              <button
                type="button"
                disabled={acting}
                className="button--secondary px-4 py-2 rounded-lg text-sm"
                onClick={() => runMutation('Overflow', () => warehouseOps.overflowOrder(orderId, reason.trim() || undefined))}
              >
                Overflow
              </button>
            )}
            {canReject && (
              <button
                type="button"
                disabled={acting}
                className="button--danger px-4 py-2 rounded-lg text-sm"
                onClick={() => runMutation('Rejected', () => warehouseOps.rejectOrder(orderId, reason.trim()), true)}
              >
                Reject
              </button>
            )}
          </div>
        </div>
      )}
      </div>
      ) : null}
      </PageChrome>
    </PageTransition>
  );
}
