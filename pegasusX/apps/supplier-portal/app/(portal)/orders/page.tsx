'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ApiError } from '@pegasusx/api-client';
import { cacheGet, cacheSet } from '@pegasusx/desktop-cache';
import { isTauri } from '@pegasusx/desktop-bridge';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import type { SupplierOrder } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import { supplierOrdersCacheKey } from '@/lib/supplier-cache-keys';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { downloadCsv } from '@/lib/csv';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';
import { ListToolbar } from '@/components/ListToolbar';
import { OrderActionDialog, OrderOpsCard, ReDispatchDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';

type OrderFilter = 'ACTIVE' | 'SCHEDULED' | 'COMPLETED' | 'CANCELLED';

const supplierApi = createSupplierApi();
const WEB_PAGE_SIZE = 25;
const DESKTOP_PAGE_SIZE = 200;
const filterLabels: Record<OrderFilter, string> = {
  ACTIVE: 'Active Orders',
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

function isoDeliveryDate(dateInput: string): string {
  return `${dateInput.slice(0, 10)}T12:00:00+05:00`;
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
  const pageSize = isTauri() ? DESKTOP_PAGE_SIZE : WEB_PAGE_SIZE;
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
    kind: 'propose' | 'reject' | 'reassign';
  } | null>(null);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState(() => new Date().toISOString().slice(0, 10));

  const loadOrders = useCallback(async (silent = false) => {
    const query =
      filter === 'SCHEDULED'
        ? { limit: pageSize, offset: page * pageSize, status: 'SCHEDULED' }
        : { limit: pageSize, offset: page * pageSize, filter };
    const cacheKey = supplierOrdersCacheKey(query);
    let hydratedFromCache = false;

    if (isTauri()) {
      const cached = await cacheGet<{ orders: SupplierOrder[]; total: number }>(cacheKey);
      if (cached) {
        setOrders(cached.orders);
        setTotal(cached.total);
        setLoading(false);
        hydratedFromCache = true;
      }
    }

    if (!silent && !hydratedFromCache) {
      setLoading(true);
      setError(null);
    }
    try {
      const response = await supplierApi.getSupplierOrders(query);
      setOrders(response.orders);
      setTotal(response.total ?? response.orders.length);
      if (isTauri()) {
        void cacheSet(cacheKey, {
          orders: response.orders,
          total: response.total ?? response.orders.length,
        });
      }
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'load_supplier_orders_failed';
      if (!hydratedFromCache) {
        if (!silent) {
          setError(message);
          toast(message, 'error');
        }
      }
    } finally {
      if (!silent || !hydratedFromCache) {
        setLoading(false);
      }
    }
  }, [filter, page, pageSize, toast]);

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

  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const currentPage = Math.min(page, pageCount - 1);

  const runWarehouseMutation = async () => {
    if (!dialog?.warehouseId) {
      toast('Warehouse scope missing for this order', 'error');
      return;
    }
    if (dialog.kind === 'reject' && !reason.trim()) {
      toast('Reason is required', 'error');
      return;
    }
    if (dialog.kind === 'propose' && (!proposedDate || !reason.trim())) {
      toast('New delivery date and reason are required', 'error');
      return;
    }
    setActingId(dialog.orderId);
    try {
      const resp =
        dialog.kind === 'propose'
          ? await supplierWarehouseOps.proposeOrderDelivery(
              dialog.orderId,
              dialog.warehouseId,
              isoDeliveryDate(proposedDate),
              reason.trim(),
            )
          : await supplierWarehouseOps.rejectOrder(dialog.orderId, dialog.warehouseId, reason.trim());
      toast(
        dialog.kind === 'propose'
          ? `New delivery date proposed · retailer notified · ${resp.status ?? 'ok'}`
          : `Order cancelled · retailer notified · ${resp.status ?? 'ok'}`,
        'success',
      );
      setDialog(null);
      setReason('');
      setProposedDate(new Date().toISOString().slice(0, 10));
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
        filter === 'SCHEDULED'
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
          {(['ACTIVE', 'SCHEDULED', 'COMPLETED', 'CANCELLED'] as OrderFilter[]).map((nextFilter) => (
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
          <VirtualScrollList
            className="-mx-1 px-1"
            height="calc(100vh - 280px)"
            items={orders}
            itemKey={(order) => order.order_id}
            renderItem={(order, index) => {
              const flags = orderActionFlags(order.status);
              const warehouseOpsEnabled = canWarehouseOps(order);
              return (
                <div className="pb-3">
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
                    showOpsMenu
                    canDelayOverride={warehouseOpsEnabled && flags.canDelay}
                    canRejectOverride={warehouseOpsEnabled && flags.canReject}
                    onDelay={
                      warehouseOpsEnabled
                        ? () => {
                            setDialog({ orderId: order.order_id, warehouseId: order.warehouse_id, kind: 'propose' });
                            setReason('');
                            setProposedDate(new Date().toISOString().slice(0, 10));
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
                    canReassignOverride={flags.canReassign}
                    onReassign={() => {
                      setDialog({ orderId: order.order_id, warehouseId: order.warehouse_id, kind: 'reassign' });
                    }}
                  />
                </div>
              );
            }}
          />
        )}
      </div>

      <OrderActionDialog
        open={dialog?.kind === 'reject'}
        title="Cancel order"
        description="Cancels the order and notifies the retailer immediately. Reason is required."
        confirmLabel="Cancel order"
        destructive
        reason={reason}
        onReasonChange={setReason}
        reasonRequired
        submitting={actingId !== null}
        onClose={() => {
          setDialog(null);
          setReason('');
        }}
        onConfirm={() => void runWarehouseMutation()}
      />
      <ReDispatchDialog
        open={dialog?.kind === 'reassign'}
        orderId={dialog?.kind === 'reassign' ? dialog.orderId : ''}
        onClose={() => setDialog(null)}
        onSuccess={() => {
          setDialog(null);
          toast('Order reassigned successfully.', 'success');
          void loadOrders();
        }}
      />
      {dialog?.kind === 'propose' ? (
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
              <span className="text-[var(--muted)]">New delivery date</span>
              <input
                type="date"
                value={proposedDate}
                onChange={(e) => setProposedDate(e.target.value)}
                className="mt-1 w-full px-3 py-2 rounded-lg border text-sm md-input-outlined"
              />
            </label>
            <textarea
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              placeholder="Reason (required)"
              rows={3}
              className="w-full px-3 py-2 rounded-lg border text-sm md-input-outlined"
            />
            <div className="flex justify-end gap-2">
              <button
                type="button"
                className="md-btn md-btn-outlined px-4 py-2"
                onClick={() => {
                  setDialog(null);
                  setReason('');
                }}
              >
                Cancel
              </button>
              <button
                type="button"
                className="md-btn md-btn-tonal px-4 py-2 disabled:opacity-50"
                disabled={actingId !== null || !proposedDate || !reason.trim()}
                onClick={() => void runWarehouseMutation()}
              >
                {actingId ? 'Working…' : 'Propose new date'}
              </button>
            </div>
          </div>
        </dialog>
      ) : null}
    </PageChrome>
  );
}
