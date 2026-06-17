'use client';

import React, { useCallback, useEffect, useMemo, useState } from 'react';

import { ApiError, supplierVetOrderKey } from '@pegasusx/api-client';
import { createSupplierApi } from '@/lib/api';
import { canAdminOrderOps, canAssignOrder, canPatchOrderStatus } from '@/lib/admin-scope';
import { downloadCsv } from '@/lib/csv';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { ListToolbar } from '@/components/ListToolbar';
import { AdminOrderOpsPanel } from '@/components/AdminOrderOpsPanel';
import { useToast } from '@/components/Toast';
import StatusBadge from '@/components/StatusBadge';
import { PortalSurface } from '../_components/PortalSurface';
import type { SupplierOrder } from '@pegasusx/types';

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
  if (Number.isNaN(timestamp)) {
    return value;
  }
  return new Date(timestamp).toLocaleString();
}

function liveStatusLabel(order: SupplierOrder) {
  if (order.live_location_available && order.driver_location) {
    return `Live ${formatTimestamp(order.driver_location.received_at)}`;
  }
  if (order.driver_id) {
    return 'Stale';
  }
  return 'Unassigned';
}

export default function OrdersPage() {
  const { push: toast } = useToast();
  const showAdminOps = useMemo(() => canAdminOrderOps(), []);
  const canAssign = useMemo(() => canAssignOrder(), []);
  const canPatchStatus = useMemo(() => canPatchOrderStatus(), []);
  const [orders, setOrders] = useState<SupplierOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [filter, setFilter] = useState<OrderFilter>('ACTIVE');
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const [vettingId, setVettingId] = useState<string | null>(null);
  const [adminBusyId, setAdminBusyId] = useState<string | null>(null);

  const loadOrders = useCallback(async () => {
    setLoading(true);
    setError(null);
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
      const message =
        err instanceof ApiError ? err.message : 'load_supplier_orders_failed';
      setError(message);
      toast(message, 'error');
    } finally {
      setLoading(false);
    }
  }, [filter, page, toast]);

  useEffect(() => {
    void loadOrders();
  }, [loadOrders]);

  useEffect(() => {
    setPage(0);
  }, [filter]);

  useSupplierSessionReconcile(() => {
    if (adminBusyId) {
      setAdminBusyId(null);
      toast('Connection restored — order list refreshed from server.', 'info');
    }
    void loadOrders();
  });

  const pageCount = Math.max(1, Math.ceil(total / PAGE_SIZE));
  const currentPage = Math.min(page, pageCount - 1);

  const vetOrder = async (orderId: string, decision: 'APPROVED' | 'REJECTED') => {
    setVettingId(orderId);
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
      setVettingId(null);
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

  return (
    <PortalSurface
      title="Orders"
      description="Durable supplier-scoped orders with assignment and live driver snapshots."
      actions={
        <div className="flex flex-wrap gap-2">
          {(['ACTIVE', 'REVIEW', 'COMPLETED', 'CANCELLED'] as OrderFilter[]).map((nextFilter) => (
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

      <div className="md-card p-0 overflow-x-auto">
        <table className="desk-table w-full">
          <thead>
            <tr className="border-b border-[var(--color-md-outline-variant)] bg-[var(--color-md-surface-container-low)]">
              <th className="md-typescale-label-medium p-4 font-medium">Order ID</th>
              <th className="md-typescale-label-medium p-4 font-medium">Updated</th>
              <th className="md-typescale-label-medium p-4 font-medium">Retailer</th>
              <th className="md-typescale-label-medium p-4 font-medium">Status</th>
              <th className="md-typescale-label-medium p-4 font-medium">Assignment</th>
              <th className="md-typescale-label-medium p-4 font-medium">Live</th>
              <th className="md-typescale-label-medium p-4 font-medium text-right">Total</th>
              {filter === 'REVIEW' || showAdminOps ? (
                <th className="md-typescale-label-medium p-4 font-medium text-right">Actions</th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {loading ? (
              <tr>
                <td colSpan={filter === 'REVIEW' || showAdminOps ? 8 : 7} className="p-8 text-center text-[var(--color-md-outline)]">
                  Loading supplier orders…
                </td>
              </tr>
            ) : error ? (
              <tr>
                <td colSpan={filter === 'REVIEW' || showAdminOps ? 8 : 7} className="p-8 text-center text-[var(--color-md-error)]">
                  {error}
                </td>
              </tr>
            ) : orders.length === 0 ? (
              <tr>
                <td colSpan={filter === 'REVIEW' || showAdminOps ? 8 : 7} className="p-8 text-center text-[var(--color-md-outline)]">
                  No {filterLabels[filter].toLowerCase()} at this time.
                </td>
              </tr>
            ) : (
              orders.map((order) => (
                <tr key={order.order_id} className="border-b border-[var(--color-md-outline-variant)] align-top">
                  <td className="p-4 font-medium">{order.order_id}</td>
                  <td className="p-4 text-[var(--color-md-outline)]">{formatTimestamp(order.updated_at)}</td>
                  <td className="p-4 text-[var(--color-md-outline)]">{order.retailer_id}</td>
                  <td className="p-4">
                    <StatusBadge state={order.status} />
                    {order.decision ? (
                      <div className="text-xs text-[var(--color-md-outline)] mt-1">Decision: {order.decision}</div>
                    ) : null}
                  </td>
                  <td className="p-4 text-[var(--color-md-outline)]">
                    {order.driver_id ? (
                      <>
                        <div>Driver: {order.driver_id}</div>
                        <div className="text-xs">Route: {order.route_id || 'pending'}</div>
                      </>
                    ) : (
                      'Awaiting assignment'
                    )}
                  </td>
                  <td className="p-4 text-[var(--color-md-outline)]">
                    <div>{liveStatusLabel(order)}</div>
                    {order.driver_location ? (
                      <div className="text-xs">
                        {order.driver_location.latitude.toFixed(4)}, {order.driver_location.longitude.toFixed(4)}
                      </div>
                    ) : null}
                  </td>
                  <td className="p-4 text-right">{formatMoney(order)}</td>
                  {filter === 'REVIEW' || showAdminOps ? (
                    <td className="p-4 text-right align-top">
                      {filter === 'REVIEW' ? (
                        <div className="flex justify-end gap-2">
                          <button
                            type="button"
                            className="md-btn md-btn-tonal md-typescale-label-medium px-3 py-1"
                            disabled={vettingId === order.order_id}
                            onClick={() => void vetOrder(order.order_id, 'APPROVED')}
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            className="md-btn md-btn-outlined md-typescale-label-medium px-3 py-1"
                            disabled={vettingId === order.order_id}
                            onClick={() => void vetOrder(order.order_id, 'REJECTED')}
                          >
                            Reject
                          </button>
                        </div>
                      ) : null}
                      {showAdminOps ? (
                        <AdminOrderOpsPanel
                          order={order}
                          busy={adminBusyId === order.order_id}
                          canAssign={canAssign}
                          canPatchStatus={canPatchStatus}
                          onBusyChange={setAdminBusyId}
                          onSuccess={() => {
                            toast('Order updated', 'success');
                            void loadOrders();
                          }}
                          onError={(message) => toast(message, 'error')}
                        />
                      ) : null}
                    </td>
                  ) : null}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>
      </div>
    </PortalSurface>
  );
}
