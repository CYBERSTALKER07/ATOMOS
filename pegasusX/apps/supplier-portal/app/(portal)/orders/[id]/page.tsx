'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import type { SupplierOrder, WarehouseOrderDetail } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import Icon from '@/components/Icon';
import { OrderTimelinePanel } from '@/components/OrderTimelinePanel';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { createSupplierApi } from '@/lib/api';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';

const supplierApi = createSupplierApi();

function formatMoney(minor: number, currency: string) {
  try {
    return new Intl.NumberFormat(undefined, {
      style: 'currency',
      currency,
      maximumFractionDigits: 2,
    }).format(minor / 100);
  } catch {
    return `${minor} ${currency}`;
  }
}

export default function SupplierOrderDetailPage() {
  const params = useParams();
  const router = useRouter();
  const { push: toast } = useToast();
  const orderId = String(params.id ?? '');
  const showAdminOps = useMemo(() => canAdminOrderOps(), []);

  const [listRow, setListRow] = useState<SupplierOrder | null>(null);
  const [detail, setDetail] = useState<WarehouseOrderDetail | null>(null);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState(false);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState(() => new Date().toISOString().slice(0, 10));

  function isoDeliveryDate(dateInput: string): string {
    return `${dateInput.slice(0, 10)}T12:00:00+05:00`;
  }

  const warehouseId = listRow?.warehouse_id;

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const list = await supplierApi.getSupplierOrders({ limit: 50, offset: 0, filter: 'ACTIVE' });
      let row = list.orders.find((o) => o.order_id === orderId) ?? null;
      if (!row) {
        const scheduled = await supplierApi.getSupplierOrders({ limit: 50, offset: 0, status: 'SCHEDULED' });
        row = scheduled.orders.find((o) => o.order_id === orderId) ?? row;
      }
      setListRow(row);

      if (row?.warehouse_id && showAdminOps) {
        try {
          const wh = await supplierWarehouseOps.getOrderDetail(orderId, row.warehouse_id);
          setDetail(wh);
        } catch {
          setDetail(null);
        }
      } else {
        setDetail(null);
      }
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Failed to load order', 'error');
    } finally {
      setLoading(false);
    }
  }, [orderId, showAdminOps, toast]);

  useEffect(() => {
    void load();
  }, [load]);

  useSupplierWsRefresh(() => {
    void load();
  }, { eventTypes: SUPPLIER_ORDERS_REFRESH_EVENTS });

  useSupplierSessionReconcile(() => {
    void load();
    if (acting) {
      setActing(false);
      toast('Connection restored — verify order status before retrying.', 'info');
    }
  });

  const state = (detail?.state ?? detail?.status ?? listRow?.status ?? '').toUpperCase();
  const flags = orderActionFlags(state);
  const canWarehouseMutate = showAdminOps && Boolean(warehouseId);

  async function runMutation(label: string, fn: () => Promise<{ status?: string }>, requiresReason = false) {
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
      toast(err instanceof ApiError ? err.message : `${label} failed`, 'error');
    } finally {
      setActing(false);
    }
  }

  if (!loading && !listRow && !detail) {
    return (
      <PageChrome icon="orders" title="Order not found" description="This order is not visible in your supplier scope.">
        <Link href="/orders" className="md-btn md-btn-tonal inline-flex items-center gap-1.5 px-4 py-2">
          <Icon name="arrow_back" size={16} /> Back to orders
        </Link>
      </PageChrome>
    );
  }

  const retailerLabel = detail?.retailer_name || listRow?.retailer_id || '—';
  const totalMinor = detail?.total_minor ?? listRow?.total_minor ?? 0;
  const currency = listRow?.currency ?? 'UZS';
  const totalUzs = detail?.total_uzs;

  return (
    <PageChrome
      icon="orders"
      title={`Order ${orderId.slice(0, 8)}…`}
      description="Supplier-scoped order detail with warehouse admin actions when available."
      loading={loading}
      actions={
        <Link href="/orders" className="md-btn md-btn-outlined inline-flex items-center gap-1.5 px-3 py-1.5">
          <Icon name="arrow_back" size={16} /> Back
        </Link>
      }
    >
      <div className="max-w-4xl space-y-6">
        <div className="md-card p-5 space-y-4">
          <div className="flex justify-between gap-4 flex-wrap">
            <div>
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Retailer</p>
              <p className="font-medium">{retailerLabel}</p>
            </div>
            <div>
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Status</p>
              <OrderStateChip state={state} />
            </div>
            <div>
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">Total</p>
              <p className="font-mono tabular-nums">
                {totalUzs != null
                  ? new Intl.NumberFormat('uz-UZ').format(totalUzs)
                  : formatMoney(totalMinor, currency)}
              </p>
            </div>
          </div>
          <p className="text-xs font-mono text-[var(--color-md-outline)]">{orderId}</p>
          {listRow?.driver_id ? (
            <p className="text-sm text-[var(--color-md-outline)]">
              Driver {listRow.driver_id} · Route {listRow.route_id || 'pending'}
            </p>
          ) : null}
        </div>

        <div className="md-card p-5">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-md-outline)] mb-3">Status history</h2>
          <OrderTimelinePanel orderId={orderId} />
        </div>

        {(detail?.line_items?.length ?? 0) > 0 ? (
          <div className="md-card p-0 overflow-hidden">
            <div className="px-5 py-3 border-b border-[var(--color-md-outline-variant)] text-sm font-semibold uppercase tracking-wider text-[var(--color-md-outline)]">
              Line items
            </div>
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="border-b border-[var(--color-md-outline-variant)]">
                  <th className="text-left py-2 px-4">Product</th>
                  <th className="text-right py-2 px-4">Qty</th>
                  <th className="text-right py-2 px-4">Unit</th>
                </tr>
              </thead>
              <tbody>
                {detail?.line_items?.map((item, idx) => (
                  <tr key={`${item.product_id ?? idx}`} className="border-b border-[var(--color-md-outline-variant)] last:border-0">
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

        {canWarehouseMutate && (flags.canDelay || flags.canReject) ? (
          <div className="md-card p-5 space-y-4">
            <p className="md-typescale-title-small">Warehouse admin actions</p>
            {flags.canDelay ? (
              <label className="block text-sm">
                <span className="text-[var(--color-md-outline)]">New delivery date</span>
                <input
                  type="date"
                  value={proposedDate}
                  onChange={(e) => setProposedDate(e.target.value)}
                  className="md-input-outlined w-full px-3 py-2 mt-1"
                  disabled={acting}
                />
              </label>
            ) : null}
            <textarea
              className="md-input-outlined w-full px-3 py-2 min-h-[80px]"
              placeholder="Reason (required)"
              value={reason}
              onChange={(e) => setReason(e.target.value)}
              disabled={acting}
            />
            <div className="flex flex-wrap gap-2">
              {flags.canDelay ? (
                <button
                  type="button"
                  className="md-btn md-btn-tonal px-4 py-2"
                  disabled={acting || !proposedDate || !reason.trim()}
                  onClick={() =>
                    void runMutation(
                      'Delivery date proposed · retailer notified',
                      () =>
                        supplierWarehouseOps.proposeOrderDelivery(
                          orderId,
                          warehouseId!,
                          isoDeliveryDate(proposedDate),
                          reason.trim(),
                        ),
                      true,
                    )
                  }
                >
                  Delay delivery
                </button>
              ) : null}
              {flags.canReject ? (
                <button
                  type="button"
                  className="md-btn md-btn-outlined px-4 py-2 text-[var(--color-md-error)]"
                  disabled={acting}
                  onClick={() =>
                    void runMutation(
                      'Order cancelled · retailer notified',
                      () => supplierWarehouseOps.rejectOrder(orderId, warehouseId!, reason.trim()),
                      true,
                    )
                  }
                >
                  Cancel order
                </button>
              ) : null}
            </div>
          </div>
        ) : null}
      </div>
    </PageChrome>
  );
}
