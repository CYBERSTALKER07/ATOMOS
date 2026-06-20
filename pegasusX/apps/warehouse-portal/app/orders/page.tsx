'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { downloadCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { ListToolbar } from '@/components/ListToolbar';
import { PageChrome } from '@/components/PageChrome';
import { OrderActionDialog, OrderOpsCard } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';

type OrdersTab = 'active' | 'preorders';

interface OrderRow {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at: string;
}

function isoDeliveryDate(dateInput: string): string {
  const dateOnly = dateInput.slice(0, 10);
  return `${dateOnly}T12:00:00+05:00`;
}

function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean {
  return String(row.confirmation_status) === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

export default function OrdersPage() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { toast } = useToast();
  const tab: OrdersTab = searchParams.get('tab') === 'preorders' ? 'preorders' : 'active';

  const [orders, setOrders] = useState<OrderRow[]>([]);
  const [preorders, setPreorders] = useState<RetailerOrderLifecycleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [actingId, setActingId] = useState<string | null>(null);
  const [dialog, setDialog] = useState<{
    orderId: string;
    kind: 'propose' | 'reject' | 'preorder-reject';
  } | null>(null);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState('');

  const loadActive = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await warehouseApi.getWarehouseOrders(filter ? { state: filter } : {});
      setOrders(
        (data.orders || []).map((row) => {
          const portalRow = row as unknown as {
            order_id: string;
            retailer_name?: string;
            state?: string;
            status?: string;
            total_uzs?: number;
            total_minor?: number;
            created_at?: string;
            updated_at?: string;
          };
          return {
            order_id: portalRow.order_id,
            retailer_name: portalRow.retailer_name ?? '',
            state: portalRow.state ?? portalRow.status ?? '',
            total_uzs: portalRow.total_uzs ?? portalRow.total_minor ?? 0,
            created_at: portalRow.created_at ?? portalRow.updated_at ?? '',
          };
        }),
      );
    } catch {
      if (!silent) toast('Failed to load orders', 'error');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [filter, toast]);

  const loadPreorders = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePreorders();
      setPreorders(data.preorders ?? data.items ?? []);
    } catch (err) {
      if (!silent) toast(err instanceof ApiError ? err.message : 'Failed to load pre-orders', 'error');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [toast]);

  const load = useCallback(async (silent = false) => {
    if (tab === 'preorders') {
      await loadPreorders(silent);
    } else {
      await loadActive(silent);
    }
  }, [loadActive, loadPreorders, tab]);

  useEffect(() => {
    void load();
  }, [load]);

  useWarehouseWsRefresh(() => {
    void load(true);
  });

  useWarehouseSessionReconcile(() => {
    void load(true);
  });

  const activePagination = usePagination(orders, 24);
  const preorderPagination = usePagination(preorders, 24);
  const page = tab === 'preorders' ? preorderPagination.page : activePagination.page;
  const pageCount = tab === 'preorders' ? preorderPagination.pageCount : activePagination.pageCount;
  const pageItems = tab === 'preorders' ? preorderPagination.pageItems : activePagination.pageItems;
  const next = tab === 'preorders' ? preorderPagination.next : activePagination.next;
  const prev = tab === 'preorders' ? preorderPagination.prev : activePagination.prev;
  const reset = tab === 'preorders' ? preorderPagination.reset : activePagination.reset;

  useEffect(() => {
    reset();
  }, [filter, tab, reset]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  const setTab = (nextTab: OrdersTab) => {
    router.replace(nextTab === 'preorders' ? '/orders?tab=preorders' : '/orders');
  };

  const openDetail = (orderId: string) => {
    router.push(`/orders/${orderId}${tab === 'preorders' ? '?from=preorders' : ''}`);
  };

  const closeDialog = () => {
    setDialog(null);
    setReason('');
    setProposedDate('');
  };

  async function submitDialog() {
    if (!dialog) return;
    const trimmedReason = reason.trim();
    setActingId(dialog.orderId);
    try {
      if (dialog.kind === 'reject') {
        if (!trimmedReason) {
          toast('Reason is required', 'error');
          return;
        }
        const resp = await warehouseOps.rejectOrder(dialog.orderId, trimmedReason);
        toast(`Order cancelled · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      } else if (dialog.kind === 'propose') {
        if (!proposedDate || !trimmedReason) {
          toast('New delivery date and reason are required', 'error');
          return;
        }
        const resp = await warehouseOps.proposeOrderDelivery(
          dialog.orderId,
          isoDeliveryDate(proposedDate),
          trimmedReason,
        );
        toast(`New delivery date proposed · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      } else if (dialog.kind === 'preorder-reject') {
        if (!trimmedReason) {
          toast('Reason is required', 'error');
          return;
        }
        const resp = await warehouseOps.rejectPreorder(dialog.orderId, trimmedReason);
        toast(`Pre-order rejected · ${resp.status ?? 'ok'}`, 'success');
      }
      closeDialog();
      await load(true);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Action failed', 'error');
    } finally {
      setActingId(null);
    }
  }

  const exportCsv = () => {
    if (tab === 'preorders') {
      downloadCsv(
        'warehouse-preorders.csv',
        ['order_id', 'status', 'requested_delivery_date', 'total_minor'],
        preorders.map((row) => [
          row.order_id,
          row.status,
          row.requested_delivery_date ?? '',
          String(row.total_minor ?? 0),
        ]),
      );
      return;
    }
    downloadCsv(
      `warehouse-orders${filter ? `-${filter.toLowerCase()}` : ''}.csv`,
      ['order_id', 'retailer_name', 'state', 'total_uzs', 'created_at'],
      orders.map((order) => [
        order.order_id,
        order.retailer_name ?? '',
        order.state,
        String(order.total_uzs),
        order.created_at,
      ]),
    );
  };

  const dialogCopy = useMemo(() => {
    if (!dialog) return null;
    if (dialog.kind === 'reject') {
      return {
        title: 'Cancel order',
        description: 'Cancels the order and notifies the retailer immediately.',
        confirmLabel: 'Cancel order',
        destructive: true,
        reasonRequired: true,
      };
    }
    if (dialog.kind === 'propose') {
      return {
        title: 'Delay delivery',
        description: 'Choose a new delivery date. The retailer is notified and can accept or reject the change.',
        confirmLabel: 'Propose new date',
        destructive: false,
        reasonRequired: true,
      };
    }
    return {
      title: 'Reject pre-order',
      description: 'This cancels the scheduled pre-order.',
      confirmLabel: 'Reject pre-order',
      destructive: true,
      reasonRequired: true,
    };
  }, [dialog]);

  const activePageItems = tab === 'active' ? (pageItems as OrderRow[]) : [];
  const preorderPageItems = tab === 'preorders' ? (pageItems as RetailerOrderLifecycleResponse[]) : [];

  return (
    <PageTransition>
      <PageChrome
        icon="orders"
        title="Orders"
        description="Active fulfillment queue and scheduled pre-orders. Double-click a card for full detail."
        actions={
          <div className="flex gap-2 items-center flex-wrap">
            {tab === 'active' ? (
              <select
                value={filter}
                onChange={(e) => {
                  setFilter(e.target.value);
                  setLoading(true);
                }}
                className="px-3 py-1.5 rounded-lg border text-sm"
                style={{
                  background: 'var(--field-background)',
                  borderColor: 'var(--field-border)',
                  color: 'var(--field-foreground)',
                }}
              >
                <option value="">All States</option>
                {['PENDING', 'LOADED', 'IN_TRANSIT', 'DELAYED', 'ARRIVED', 'COMPLETED', 'CANCELLED'].map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            ) : null}
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => {
                setLoading(true);
                void load();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary active-press"
            >
              <Icon name="refresh" size={16} /> Refresh
            </motion.button>
          </div>
        }
      >
        <div className="flex gap-2 mb-4">
          <button
            type="button"
            className={`px-4 py-2 rounded-lg text-sm font-medium ${tab === 'active' ? 'portal-btn portal-btn--primary' : 'button--secondary'}`}
            onClick={() => setTab('active')}
          >
            Active orders
          </button>
          <button
            type="button"
            className={`px-4 py-2 rounded-lg text-sm font-medium ${tab === 'preorders' ? 'portal-btn portal-btn--primary' : 'button--secondary'}`}
            onClick={() => setTab('preorders')}
          >
            Pre-orders
          </button>
        </div>

        {loading ? (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="md-skeleton h-28 rounded-xl" />
            ))}
          </div>
        ) : (tab === 'preorders' ? preorders : orders).length === 0 ? (
          <EmptyState
            variant={filter ? 'no-results' : 'no-data'}
            headline={tab === 'preorders' ? 'No pre-orders' : 'No orders found'}
            body={
              tab === 'preorders'
                ? 'Scheduled manual pre-orders will appear here.'
                : filter
                  ? `No orders found with state "${filter}".`
                  : 'There are no orders recorded in this warehouse yet.'
            }
          />
        ) : (
          <>
            <ListToolbar
              page={page}
              pageCount={pageCount}
              totalLabel={`${tab === 'preorders' ? preorders.length : orders.length} ${tab === 'preorders' ? 'pre-orders' : 'orders'}`}
              onPrev={prev}
              onNext={next}
              onExport={exportCsv}
            />
            <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {tab === 'active'
                ? activePageItems.map((order, index) => (
                    <OrderOpsCard
                      key={order.order_id}
                      orderId={order.order_id}
                      retailerName={order.retailer_name}
                      state={order.state}
                      amountLabel={`${fmt(order.total_uzs)} UZS`}
                      meta={order.created_at ? new Date(order.created_at).toLocaleString() : undefined}
                      index={index}
                      disabled={actingId === order.order_id}
                      onOpenDetail={() => openDetail(order.order_id)}
                      onDelay={() => {
                        setDialog({ orderId: order.order_id, kind: 'propose' });
                        setReason('');
                        setProposedDate(new Date().toISOString().slice(0, 10));
                      }}
                      onReject={() => {
                        setDialog({ orderId: order.order_id, kind: 'reject' });
                        setReason('');
                      }}
                    />
                  ))
                : preorderPageItems.map((row, index) => (
                    <OrderOpsCard
                      key={row.order_id}
                      orderId={row.order_id}
                      retailerName={row.order_source || 'Manual pre-order'}
                      state={row.status}
                      amountLabel={`${fmt(Math.round((row.total_minor ?? 0) / 100))} ${row.currency || 'UZS'}`}
                      meta={row.requested_delivery_date
                        ? `Requested ${new Date(row.requested_delivery_date).toLocaleDateString()}`
                        : undefined}
                      badge={showsReviewBadge(row) ? 'Review delivery' : row.preorder_badge}
                      index={index}
                      disabled={actingId === row.order_id}
                      onOpenDetail={() => openDetail(row.order_id)}
                      onDelay={() => {
                        setDialog({ orderId: row.order_id, kind: 'propose' });
                        setReason('');
                        setProposedDate((row.requested_delivery_date ?? '').slice(0, 10) || new Date().toISOString().slice(0, 10));
                      }}
                      onReject={() => {
                        setDialog({ orderId: row.order_id, kind: 'preorder-reject' });
                        setReason('');
                      }}
                      delayLabel="Propose delivery"
                      rejectLabel="Reject pre-order"
                      canDelayOverride
                      canRejectOverride
                    />
                  ))}
            </div>
          </>
        )}
      </PageChrome>

      {dialog && dialogCopy ? (
        <>
          <OrderActionDialog
            open={dialog.kind !== 'propose'}
            title={dialogCopy.title}
            description={dialogCopy.description}
            confirmLabel={dialogCopy.confirmLabel}
            destructive={dialogCopy.destructive}
            reason={reason}
            onReasonChange={setReason}
            reasonRequired={dialogCopy.reasonRequired}
            submitting={actingId === dialog.orderId}
            onConfirm={() => void submitDialog()}
            onClose={closeDialog}
          />
          {dialog.kind === 'propose' ? (
            <dialog
              open
              className="rounded-xl border border-[var(--border)] bg-[var(--surface)] p-5 backdrop:bg-black/40 max-w-md w-[calc(100%-2rem)] fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 z-50"
            >
              <div className="space-y-4">
                <div>
                  <h2 className="text-lg font-semibold">Delay delivery</h2>
                  <p className="text-sm text-[var(--muted)] mt-1">
                    Choose a new delivery date. The retailer is notified and can accept or reject.
                  </p>
                </div>
                <label className="block text-sm">
                  <span className="text-[var(--muted)]">Proposed date</span>
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
                <textarea
                  value={reason}
                  onChange={(e) => setReason(e.target.value)}
                  placeholder="Reason (required)"
                  rows={3}
                  className="w-full px-3 py-2 rounded-lg border text-sm"
                  style={{
                    background: 'var(--field-background)',
                    borderColor: 'var(--field-border)',
                    color: 'var(--field-foreground)',
                  }}
                />
                <div className="flex justify-end gap-2">
                  <button type="button" className="button--secondary px-4 py-2 rounded-lg text-sm" onClick={closeDialog}>
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="portal-btn portal-btn--primary px-4 py-2 rounded-lg text-sm disabled:opacity-50"
                    disabled={actingId === dialog.orderId || !proposedDate || !reason.trim()}
                    onClick={() => void submitDialog()}
                  >
                    {actingId === dialog.orderId ? 'Working…' : 'Propose new date'}
                  </button>
                </div>
              </div>
            </dialog>
          ) : null}
        </>
      ) : null}
    </PageTransition>
  );
}
