'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import type { WarehouseOrderDetail } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { orderActionFlags } from '@/lib/order-actions';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';

export default function OrderDetailPage() {
  const params = useParams();
  const searchParams = useSearchParams();
  const router = useRouter();
  const { toast } = useToast();
  const orderId = String(params.id ?? '');
  const fromDispatch = searchParams.get('from') === 'dispatch';
  const fromPreorders = searchParams.get('from') === 'preorders';
  const [order, setOrder] = useState<WarehouseOrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState(() => new Date().toISOString().slice(0, 10));

  function isoDeliveryDate(dateInput: string): string {
    return `${dateInput.slice(0, 10)}T12:00:00+05:00`;
  }

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const row = await warehouseApi.getWarehouseOrder(orderId);
      setOrder(row);
    } catch (err) {
      if (err instanceof ApiError && err.status === 404) {
        toast('Order not found', 'error');
        router.replace('/orders');
        return;
      }
      toast('Failed to load order', 'error');
    } finally {
      setLoading(false);
    }
  }, [orderId, router, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  useWarehouseWsRefresh(() => {
    void load();
  });

  useWarehouseSessionReconcile(() => {
    void load();
    if (acting) {
      setActing(false);
      toast('Connection restored — verify order status before retrying.', 'info');
    }
  });

  const state = (order?.state ?? order?.status ?? '').toUpperCase();
  const flags = orderActionFlags(state);
  const total = order?.total_uzs ?? order?.total_minor ?? 0;

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
    } catch (err) {
      toast(err instanceof ApiError ? err.message : `${label} failed`, 'error');
    } finally {
      setActing(false);
    }
  }

  const backHref = fromDispatch ? '/dispatch' : fromPreorders ? '/orders?tab=preorders' : '/orders';

  if (!loading && !order) return null;

  return (
    <PageTransition>
      <PageChrome
        icon="orders"
        title={`Order ${orderId.slice(0, 8)}…`}
        description="Warehouse-scoped order detail and exception actions."
        loading={loading}
        actions={
          <Link href={backHref} className="button--secondary px-3 py-1.5 rounded-lg text-sm inline-flex items-center gap-1.5">
            <Icon name="arrow_back" size={16} /> Back
          </Link>
        }
      >
        {order ? (
          <div className="max-w-4xl space-y-6">
            <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 space-y-4">
              <div className="flex justify-between gap-4 flex-wrap">
                <div>
                  <p className="text-sm text-[var(--muted)]">Retailer</p>
                  <p className="font-medium">{order.retailer_name || '—'}</p>
                </div>
                <div>
                  <p className="text-sm text-[var(--muted)]">State</p>
                  <OrderStateChip state={state} />
                </div>
                <div>
                  <p className="text-sm text-[var(--muted)]">Total (UZS)</p>
                  <p className="font-mono tabular-nums">{new Intl.NumberFormat('uz-UZ').format(total)}</p>
                </div>
              </div>
              <p className="text-xs font-mono text-[var(--muted)]">{order.order_id}</p>
            </div>

            {(order.line_items?.length ?? 0) > 0 ? (
              <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] overflow-hidden">
                <div className="px-5 py-3 border-b border-[var(--border)] text-sm font-semibold uppercase tracking-wider text-[var(--muted)]">
                  Line items
                </div>
                <table className="desk-table w-full text-sm">
                  <thead>
                    <tr className="border-b border-[var(--border)]">
                      <th className="text-left py-2 px-4">Product</th>
                      <th className="text-right py-2 px-4">Qty</th>
                      <th className="text-right py-2 px-4">Unit (UZS)</th>
                    </tr>
                  </thead>
                  <tbody>
                    {order.line_items?.map((item, idx) => (
                      <tr key={`${item.product_id ?? idx}`} className="border-b border-[var(--border)] last:border-0">
                        <td className="py-2 px-4">{item.product_name || item.product_id || '—'}</td>
                        <td className="py-2 px-4 text-right font-mono tabular-nums">{item.quantity ?? '—'}</td>
                        <td className="py-2 px-4 text-right font-mono tabular-nums">
                          {item.unit_price != null ? new Intl.NumberFormat('uz-UZ').format(item.unit_price) : '—'}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            ) : null}

            {(flags.canDelay || flags.canReject || flags.canOverflow) && (
              <div className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 space-y-4">
                <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--muted)]">Warehouse actions</h2>
                {flags.canDelay ? (
                  <label className="block text-sm">
                    <span className="text-[var(--muted)]">New delivery date</span>
                    <input
                      type="date"
                      value={proposedDate}
                      onChange={(e) => setProposedDate(e.target.value)}
                      className="mt-1 w-full px-3 py-2 rounded-lg border text-sm"
                      style={{
                        background: 'var(--field-background)',
                        borderColor: 'var(--field-border)',
                        color: 'var(--field-foreground)',
                      }}
                    />
                  </label>
                ) : null}
                <textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Reason (required for delay and cancel)"
                  rows={2}
                  className="w-full px-3 py-2 rounded-lg border text-sm"
                  style={{
                    background: 'var(--field-background)',
                    borderColor: 'var(--field-border)',
                    color: 'var(--field-foreground)',
                  }}
                />
                <div className="flex flex-wrap gap-2">
                  {flags.canDelay && (
                    <button
                      type="button"
                      disabled={acting || !proposedDate || !reason.trim()}
                      className="button--secondary px-4 py-2 rounded-lg text-sm"
                      onClick={() =>
                        runMutation(
                          'Delivery date proposed · retailer notified',
                          () =>
                            warehouseOps.proposeOrderDelivery(
                              orderId,
                              isoDeliveryDate(proposedDate),
                              reason.trim(),
                            ),
                          true,
                        )
                      }
                    >
                      Delay delivery
                    </button>
                  )}
                  {flags.canOverflow && (
                    <button
                      type="button"
                      disabled={acting}
                      className="button--secondary px-4 py-2 rounded-lg text-sm"
                      onClick={() => runMutation('Overflow', () => warehouseOps.overflowOrder(orderId, reason.trim() || undefined))}
                    >
                      Overflow
                    </button>
                  )}
                  {flags.canReject && (
                    <button
                      type="button"
                      disabled={acting}
                      className="button--danger px-4 py-2 rounded-lg text-sm"
                      onClick={() =>
                        runMutation(
                          'Order cancelled · retailer notified',
                          () => warehouseOps.rejectOrder(orderId, reason.trim()),
                          true,
                        )
                      }
                    >
                      Cancel order
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
