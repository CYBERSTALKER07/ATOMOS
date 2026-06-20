'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ApiError, supplierVetOrderKey } from '@pegasusx/api-client';
import type { SupplierOrder } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { downloadCsv } from '@/lib/csv';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';
import { ListToolbar } from '@/components/ListToolbar';
import { OrderActionDialog, OrderOpsCard } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';

type OrderFilter = 'ACTIVE' | 'REVIEW' | 'SCHEDULED' | 'COMPLETED' | 'CANCELLED';

const supplierApi = createSupplierApi();
const PAGE_SIZE = 25;
const filterLabels: Record<OrderFilter, string> = {
  ACTIVE: 'Active Orders',
  REVIEW: 'Awaiting Review',
  SCHEDULED: 'Scheduled pre-orders',
  COMPLETED: 'Completed',
  CANCELLED: 'Cancelled',
};

function formatMoney(order: SupplierOrder) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency: order.currency,
      maximumFractionDigits: 2,
    }).format(order.total_minor / 100);
  } catch {
    return `${order.total_minor} ${order.currency}`;
  }
}

function formatTimestamp(value: string) {
  const timestamp = Date.parse(value);
  if (Number.isNaN(timestamp)) return value;
  return new Date(timestamp).toLocaleString();
}

function liveStatusLabel(order: SupplierOrder) {
  if (order.live_location_available && order.driver_location) {
    return `Live ${formatTimestamp(order.driver_location.received_at)}`;
  }
  if (order.driver_id) return 'Stale';
  return 'Unassigned';
}

export default function OrdersPage() {
  const router = useRouter();
  const { push: toast } = useToast();
  const showAdminOps = useMemo(() => canAdminOrderOps(), []);
  const [orders, setOrders] = useState<SupplierOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [filter, setFilter] = useState<OrderFilter>('ACTIVE');
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);
  const [actingId, setActingId] = useState<string | null>(null);
  const [dialog, setDialog] = useState<{
    orderId: string;
    warehouseId?: string;
    kind: 'delay' | 'reject';
  } | null>(null);
  const [reason, setReason] = useState('');

  const loadOrders = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
      setError(null);
    }
    try {
      const query =
        filter === 'REVIEW'
          ? { limit: PAGE_SIZE, offset: page * PAGE_SIZE, status: 'AWAITING_REVIEW' }
          : filter === 'SCHEDULED'
            ? { limit: PAGE_SIZE, offset: page * PAGE_SIZE, status: 'SCHEDULED' }
            : { limit: PAGE_SIZE, offset: page * PAGE_SIZE, filter };
      const response = await supplierApi.getSupplierOrders(query);
      setOrders(response.orders);
      setTotal(response.total ?? response.orders.length);
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'load_supplier_orders_failed';
      if (!silent) {
        setError(message);
        toast(message, 'error');
      }
    } finally {
      if (!silent) setLoading(false);
    }
  }, [filter, page, toast]);

  useEffect(() => {
    void loadOrders();
  }, [loadOrders]);

  useEffect(() => {
    setPage(0);
  }, [filter]);

  useSupplierWsRefresh(() => {
    void loadOrders(true);
  }, { eventTypes: SUPPLIER_ORDERS_REFRESH_EVENTS });

  useSupplierSessionReconcile(() => {
    if (actingId) {
      setActingId(null);
      toast('Connection restored — order list refreshed from server.', 'info');
    }
    void loadOrders(true);
  });

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const currentPage = Math.min(page, pageCount - 1);

  const vetOrder = async (orderId: string, decision: 'APPROVED' | 'REJECTED') => {
    setActingId(orderId);
    try {
      await supplierApi.vetSupplierOrder(
        { order_id: orderId, decision },
        supplierVetOrderKey(orderId, decision),
      );
      toast(`Order ${decision.toLowerCase()}`, 'success');
      await loadOrders();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'vet_failed', 'error');
    } finally {
      setActingId(null);
    }
  };

  const runWarehouseMutation = async () => {
    if (!dialog?.warehouseId) {
      toast('Warehouse scope missing for this order', 'error');
      return;
    }
    if (dialog.kind === 'reject' && !reason.trim()) {
      toast('Reason is required', 'error');
      return;
    }
    setActingId(dialog.orderId);
    try {
      const resp =
        dialog.kind === 'delay'
          ? await supplierWarehouseOps.delayOrder(dialog.orderId, dialog.warehouseId, reason.trim() || undefined)
          : await supplierWarehouseOps.rejectOrder(dialog.orderId, dialog.warehouseId, reason.trim());
      toast(`${dialog.kind === 'delay' ? 'Delayed' : 'Rejected'} · ${resp.status ?? 'ok'}`, 'success');
      setDialog(null);
      setReason('');
      await loadOrders();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'order_mutation_failed', 'error');
    } finally {
      setActingId(null);
    }
  };

  const exportCsv = async () => {
    setExporting(true);
    try {
      const query =
        filter === 'REVIEW'
          ? { limit: 300, offset: 0, status: 'AWAITING_REVIEW' }
          : filter === 'SCHEDULED'
            ? { limit: 300, offset: 0, status: 'SCHEDULED' }
            : { limit: 300, offset: 0, filter };
      const response = await supplierApi.getSupplierOrders(query);
      downloadCsv(
        `supplier-orders-${filter.toLowerCase()}.csv`,
        ['order_id', 'status', 'retailer_id', 'driver_id', 'total_minor', 'currency', 'updated_at'],
        response.orders.map((order) => [
          order.order_id,
          order.status,
          order.retailer_id,
          order.driver_id ?? '',
          String(order.total_minor),
          order.currency,
          order.updated_at,
        ]),
      );
    } catch (err) {
      toast(err instanceof Error ? err.message : 'export_failed', 'error');
    } finally {
      setExporting(false);
    }
  };

  const canWarehouseOps = (order: SupplierOrder) =>
    showAdminOps && Boolean(order.warehouse_id) && (filter === 'ACTIVE' || filter === 'SCHEDULED');

  return (
    <PageChrome
      icon="orders"
      title="Orders"
      description="Durable supplier-scoped orders with assignment and live driver snapshots."
      actions={
        <div className="flex flex-wrap gap-2">
          {(['ACTIVE', 'REVIEW', 'SCHEDULED', 'COMPLETED', 'CANCELLED'] as OrderFilter[]).map((nextFilter) => (
            <button
              key={nextFilter}
              type="button"
              className="md-chip"
              aria-pressed={filter === nextFilter}
              onClick={() => setFilter(nextFilter)}
            >
              {filterLabels[nextFilter]}
            </button>
          ))}
        </div>
      }
    >
      <div className="space-y-4">
        <ListToolbar
          page={currentPage}
          pageCount={pageCount}
          totalLabel={`${total} ${filterLabels[filter].toLowerCase()}`}
          onPrev={() => setPage((value) => Math.max(value - 1, 0))}
          onNext={() => setPage((value) => Math.min(value + 1, pageCount - 1))}
          onExport={() => void exportCsv()}
          exportDisabled={exporting}
        />

        {loading ? (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <div key={i} className="h-28 rounded-xl bg-[var(--color-md-surface-container-low)] animate-pulse" />
            ))}
          </div>
        ) : error ? (
          <EmptyState icon="error" headline="Could not load orders" body={error} />
        ) : orders.length === 0 ? (
          <EmptyState
            icon="orders"
            headline={`No ${filterLabels[filter].toLowerCase()}`}
            body="Orders matching this filter will appear here."
          />
        ) : (
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
            {orders.map((order, index) => {
              const flags = orderActionFlags(order.status);
              const warehouseOpsEnabled = canWarehouseOps(order);
              return (
                <div key={order.order_id} className="space-y-2">
                  <OrderOpsCard
                    index={index}
                    orderId={order.order_id}
                    retailerName={order.retailer_id}
                    state={order.status}
                    amountLabel={formatMoney(order)}
                    meta={`${formatTimestamp(order.updated_at)} · ${liveStatusLabel(order)}`}
                    badge={filter === 'SCHEDULED' ? 'Pre-order' : undefined}
                    disabled={actingId === order.order_id}
                    onOpenDetail={() => router.push(`/orders/${order.order_id}` as '/orders')}
                    showOpsMenu={filter !== 'REVIEW'}
                    canDelayOverride={warehouseOpsEnabled && flags.canDelay}
                    canRejectOverride={warehouseOpsEnabled && flags.canReject}
                    onDelay={
                      warehouseOpsEnabled
                        ? () => {
                            setDialog({ orderId: order.order_id, warehouseId: order.warehouse_id, kind: 'delay' });
                            setReason('');
                          }
                        : undefined
                    }
                    onReject={
                      warehouseOpsEnabled
                        ? () => {
                            setDialog({ orderId: order.order_id, warehouseId: order.warehouse_id, kind: 'reject' });
                            setReason('');
                          }
                        : undefined
                    }
                  />
                  {filter === 'REVIEW' ? (
                    <div className="flex gap-2 px-1">
                      <button
                        type="button"
                        className="md-btn md-btn-tonal md-typescale-label-medium flex-1 px-3 py-1.5"
                        disabled={actingId === order.order_id}
                        onClick={() => void vetOrder(order.order_id, 'APPROVED')}
                      >
                        Approve
                      </button>
                      <button
                        type="button"
                        className="md-btn md-btn-outlined md-typescale-label-medium flex-1 px-3 py-1.5"
                        disabled={actingId === order.order_id}
                        onClick={() => void vetOrder(order.order_id, 'REJECTED')}
                      >
                        Reject
                      </button>
                    </div>
                  ) : null}
                </div>
              );
            })}
          </div>
        )}
      </div>

      <OrderActionDialog
        open={dialog !== null}
        title={dialog?.kind === 'reject' ? 'Reject order' : 'Delay delivery'}
        description={
          dialog?.kind === 'reject'
            ? 'The retailer will be notified. Reason is required.'
            : 'Delay this delivery and notify the retailer.'
        }
        confirmLabel={dialog?.kind === 'reject' ? 'Reject' : 'Delay'}
        destructive={dialog?.kind === 'reject'}
        reason={reason}
        onReasonChange={setReason}
        reasonRequired={dialog?.kind === 'reject'}
        submitting={actingId !== null}
        onClose={() => {
          setDialog(null);
          setReason('');
        }}
        onConfirm={() => void runWarehouseMutation()}
      />
    </PageChrome>
  );
}
