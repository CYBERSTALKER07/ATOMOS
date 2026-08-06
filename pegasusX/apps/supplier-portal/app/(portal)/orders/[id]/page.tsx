'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import { useParams, useRouter } from 'next/navigation';
import type { SupplierOrder, WarehouseOrderDetail } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import Icon from '@/components/Icon';
import { OrderTimelinePanel } from '@/components/OrderTimelinePanel';
import { PageChrome } from '@/components/PageChrome';
import { OrderStateChip } from '@/components/orders';
import { OrderLineItems } from '@/components/orders/OrderLineItems';
import { OrderOpsActions } from '@/components/orders/OrderOpsActions';
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
  const t = usePortalT();
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
      <PageChrome icon="orders" title={t("order.error.not_found")} description={t("supplier_portal.residual.text.this_order_is_not_visible_in_your_supplier_scope")}>
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
      description={t("supplier_portal.residual.text.supplier_scoped_order_detail_with_warehouse_admin_actions_when_a")}
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
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">{t("supplier_portal.analytics.demand.flywheel.text.retailer")}</p>
              <p className="font-medium">{retailerLabel}</p>
            </div>
            <div>
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">{t("supplier_portal.compliance.text.status")}</p>
              <OrderStateChip state={state} />
            </div>
            <div>
              <p className="md-typescale-label-medium text-[var(--color-md-outline)]">{t("supplier_portal.orders._id_.text.total")}</p>
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
          {(state === 'COMPLETED' || state === 'FISCAL_FAILED' || state === 'FISCALIZING') && (
            <div className="flex flex-wrap gap-2 pt-2">
              <button
                type="button"
                className="md-btn md-btn-tonal text-sm px-3 py-1.5"
                disabled={acting}
                onClick={() => {
                  void import('@/lib/order-receipt').then((m) =>
                    m.openSupplierOrderReceipt(orderId, 'html').catch((err) =>
                      toast(err instanceof Error ? err.message : 'Receipt unavailable', 'error'),
                    ),
                  );
                }}
              >
                View receipt
              </button>
              <button
                type="button"
                className="md-btn md-btn-outlined text-sm px-3 py-1.5"
                disabled={acting}
                onClick={() => {
                  void import('@/lib/order-receipt').then((m) =>
                    m.openSupplierOrderReceipt(orderId, 'pdf').catch((err) =>
                      toast(err instanceof Error ? err.message : 'PDF unavailable', 'error'),
                    ),
                  );
                }}
              >
                Download PDF
              </button>
            </div>
          )}
        </div>

        <div className="md-card p-5">
          <h2 className="text-sm font-semibold uppercase tracking-wider text-[var(--color-md-outline)] mb-3">{t("supplier_portal.orders._id_.text.status_history")}</h2>
          <OrderTimelinePanel orderId={orderId} />
        </div>

        <OrderLineItems detail={detail} />
        <OrderOpsActions
          canWarehouseMutate={canWarehouseMutate}
          flags={flags}
          acting={acting}
          proposedDate={proposedDate}
          setProposedDate={setProposedDate}
          reason={reason}
          setReason={setReason}
          onDelay={() =>
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
          onReject={() =>
            void runMutation(
              'Order cancelled · retailer notified',
              () => supplierWarehouseOps.rejectOrder(orderId, warehouseId!, reason.trim()),
              true,
            )
          }
        />
      </div>
    </PageChrome>
  );
}
