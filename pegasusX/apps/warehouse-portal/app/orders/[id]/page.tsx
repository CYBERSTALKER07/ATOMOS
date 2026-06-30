'use client';

import { useCallback, useEffect, useState } from 'react';
import { useParams, useRouter, useSearchParams } from 'next/navigation';
import Link from 'next/link';
import type { WarehouseOrderDetail } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import Icon from '@/components/Icon';
import { OrderTimelinePanel } from '@/components/OrderTimelinePanel';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { orderActionFlags } from '@/lib/order-actions';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';

function isoDeliveryDate(dateInput: string): string {
  return `${dateInput.slice(0, 10)}T12:00:00+05:00`;
}

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
  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

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
      setReason('');
      await load();
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        toast('Order was updated elsewhere. Refreshing...', 'error');
        await load();
      } else {
        toast(err instanceof ApiError ? err.message : `${label} failed`, 'error');
      }
    } finally {
      setActing(false);
    }
  }

  const backHref = fromDispatch ? '/dispatch' : fromPreorders ? '/orders?tab=preorders' : '/orders';
  const showOps = flags.canDelay || flags.canReject || flags.canOverflow;

  if (!loading && !order) return null;

  return (
    <PageTransition>
      <PageChrome
        icon="orders"
        title={order?.retailer_name || `Order ${orderId.slice(0, 8)}…`}
        description="Full order detail and warehouse exception actions."
        loading={loading}
        actions={
          <Link href={backHref} className="portal-btn portal-btn--outline text-sm inline-flex items-center gap-1.5">
            <Icon name="arrow_back" size={16} /> Back
          </Link>
        }
      >
        {order ? (
          <div className="wh-order-bento">
            <section className="wh-bay-panel wh-bay--ops wh-order-bento-summary">
              <div className="wh-section-head">
                <div>
                  <h2 className="wh-section-title">Order summary</h2>
                  <p className="wh-section-desc">Retailer, state, and total for this fulfillment.</p>
                </div>
                <OrderStateChip state={state} />
              </div>
              <div className="p-5 grid gap-4 sm:grid-cols-2">
                <div>
                  <p className="text-xs uppercase tracking-wider text-[var(--muted)]">Retailer</p>
                  <p className="font-medium mt-1">{order.retailer_name || '—'}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wider text-[var(--muted)]">Total (UZS)</p>
                  <p className="wh-kpi-value text-xl mt-1">{fmt(total)}</p>
                </div>
                <div className="sm:col-span-2">
                  <p className="text-xs uppercase tracking-wider text-[var(--muted)]">Order ID</p>
                  <p className="wh-ops-card-id mt-1">{order.order_id}</p>
                </div>
              </div>
            </section>

            {showOps ? (
              <aside className="wh-bay-panel wh-bay--ops wh-order-bento-ops">
                <div className="wh-section-head">
                  <div>
                    <h2 className="wh-section-title">Quick actions</h2>
                    <p className="wh-section-desc">Propose a new date or cancel. Retailer is notified.</p>
                  </div>
                </div>
                <div className="p-5 space-y-4">
                  {flags.canDelay ? (
                    <label className="portal-field">
                      <span className="portal-label">Proposed delivery date</span>
                      <input
                        type="date"
                        value={proposedDate}
                        onChange={(e) => setProposedDate(e.target.value)}
                        className="portal-input"
                      />
                    </label>
                  ) : null}
                  <label className="portal-field">
                    <span className="portal-label">Reason</span>
                    <textarea
                      value={reason}
                      onChange={(e) => setReason(e.target.value)}
                      placeholder="Required for propose date and cancel"
                      rows={3}
                      className="portal-input min-h-[88px] py-2"
                    />
                  </label>
                  <div className="flex flex-col gap-2">
                    {flags.canDelay ? (
                      <button
                        type="button"
                        disabled={acting || !proposedDate || !reason.trim()}
                        className="portal-btn portal-btn--primary w-full"
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
                        Propose new date
                      </button>
                    ) : null}
                    {flags.canOverflow ? (
                      <button
                        type="button"
                        disabled={acting}
                        className="portal-btn portal-btn--outline w-full"
                        onClick={() =>
                          runMutation('Returned to dispatch pool', () =>
                            warehouseOps.overflowOrder(orderId, reason.trim() || undefined),
                          )
                        }
                      >
                        Return to dispatch pool
                      </button>
                    ) : null}
                    {flags.canReject ? (
                      <button
                        type="button"
                        disabled={acting || !reason.trim()}
                        className="portal-btn portal-btn--outline w-full"
                        style={{ color: 'var(--danger)' }}
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
                    ) : null}
                  </div>
                </div>
              </aside>
            ) : null}

            <section className="wh-bay-panel wh-bay--inventory">
              <div className="wh-section-head">
                <div>
                  <h2 className="wh-section-title">Status history</h2>
                  <p className="wh-section-desc">Delays, promotions, and lifecycle changes.</p>
                </div>
              </div>
              <OrderTimelinePanel orderId={orderId} />
            </section>

            {(order.line_items?.length ?? 0) > 0 ? (
              <section className="wh-bay-panel wh-bay--inventory wh-order-bento-lines">
                <div className="wh-section-head">
                  <div>
                    <h2 className="wh-section-title">Line items</h2>
                    <p className="wh-section-desc">{order.line_items?.length ?? 0} products in this order.</p>
                  </div>
                </div>
                <div className="desk-table-wrap">
                  <table className="desk-table">
                    <thead>
                      <tr>
                        <th>Product</th>
                        <th className="text-right">Qty</th>
                        <th className="text-right">Unit (UZS)</th>
                      </tr>
                    </thead>
                    <tbody>
                      {order.line_items?.map((item, idx) => (
                        <tr key={`${item.product_id ?? idx}`}>
                          <td>{item.product_name || item.product_id || '—'}</td>
                          <td className="text-right font-mono tabular-nums">{item.quantity ?? '—'}</td>
                          <td className="text-right font-mono tabular-nums">
                            {item.unit_price != null ? fmt(item.unit_price) : '—'}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </section>
            ) : null}
          </div>
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}
