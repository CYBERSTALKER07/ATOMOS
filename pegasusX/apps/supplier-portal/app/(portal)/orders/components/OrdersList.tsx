"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useState } from 'react';
import { useRouter } from 'next/navigation';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import type { SupplierOrder } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-core';
import { orderActionFlags } from '@/lib/order-actions';
import { supplierWarehouseOps } from '@/lib/supplier-warehouse-ops';
import { canAdminOrderOps } from '@/lib/admin-scope';
import { OrderActionDialog, OrderOpsCard, ReDispatchDialog, ProposeDelayDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';

export type OrderFilter = 'ACTIVE' | 'SCHEDULED' | 'COMPLETED' | 'CANCELLED';

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

export interface OrdersListProps {
  orders: SupplierOrder[];
  filter: OrderFilter;
  loading: boolean;
  error: string | null;
  onRefresh: () => Promise<void>;
}

export function OrdersList({ orders, filter, loading, error, onRefresh }: OrdersListProps) {
  const t = usePortalT();
  const router = useRouter();
  const { push: toast } = useToast();
  
  // Recompute admin permission just like the page did
  const showAdminOps = React.useMemo(() => canAdminOrderOps(), []);

  const [actingId, setActingId] = useState<string | null>(null);
  const [dialog, setDialog] = useState<{
    orderId: string;
    warehouseId?: string;
    kind: 'propose' | 'reject' | 'reassign';
  } | null>(null);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState(() => new Date().toISOString().slice(0, 10));

  const canWarehouseOps = (order: SupplierOrder) =>
    showAdminOps && Boolean(order.warehouse_id) && (filter === 'ACTIVE' || filter === 'SCHEDULED');

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
      await onRefresh();
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'order_mutation_failed', 'error');
    } finally {
      setActingId(null);
    }
  };

  if (loading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="h-28 rounded-xl bg-[var(--color-md-surface-container-low)] animate-pulse" />
        ))}
      </div>
    );
  }

  if (error) {
    return <EmptyState icon="error" headline={t("supplier_portal.residual.text.could_not_load_orders")} body={error} />;
  }

  if (orders.length === 0) {
    return (
      <EmptyState
        icon="orders"
        headline={`No ${filterLabels[filter].toLowerCase()}`}
        body={t("supplier_portal.residual.text.orders_matching_this_filter_will_appear_here")}
      />
    );
  }

  return (
    <>
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

      <OrderActionDialog
        open={dialog?.kind === 'reject'}
        title={t("supplier_portal.orders.components.orders_list.text.cancel_order")}
        description={t("supplier_portal.residual.text.cancels_the_order_and_notifies_the_retailer_immediately_reason_i")}
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
          void onRefresh();
        }}
      />
      
      <ProposeDelayDialog
        open={dialog?.kind === 'propose'}
        proposedDate={proposedDate}
        reason={reason}
        submitting={actingId !== null}
        onProposedDateChange={setProposedDate}
        onReasonChange={setReason}
        onClose={() => {
          setDialog(null);
          setReason('');
        }}
        onConfirm={() => void runWarehouseMutation()}
      />
    </>
  );
}
