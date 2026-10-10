import React from 'react';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';
import { moneyCurrency } from '@pegasusx/api-core';

export type OrdersTab = 'active' | 'preorders';

export interface OrderRow {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at: string;
}

export interface OrdersListProps {
  tab: OrdersTab;
  loading: boolean;
  filter: string;
  activeItems: OrderRow[];
  preorderItems: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string, isPreorder: boolean, currentDate?: string) => void;
  onReject: (orderId: string, isPreorder: boolean) => void;
}

function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean {
  return String(row.confirmation_status) === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

export function OrdersList({
  tab,
  loading,
  filter,
  activeItems,
  preorderItems,
  actingId,
  onOpenDetail,
  onProposeDate,
  onReject,
}: OrdersListProps) {
  if (loading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="md-skeleton h-28 rounded-xl" />
        ))}
      </div>
    );
  }

  const isEmpty = tab === 'preorders' ? preorderItems.length === 0 : activeItems.length === 0;

  if (isEmpty) {
    return (
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
    );
  }

  return (
    <div className="wh-ops-grid mt-4">
      {tab === 'active'
        ? activeItems.map((order, index) => (
            <OrderOpsCard
              key={order.order_id}
              orderId={order.order_id}
              retailerName={order.retailer_name}
              state={order.state}
              amountLabel={`${fmt(order.total_uzs)} ${moneyCurrency()}`.trim()}
              meta={order.created_at ? new Date(order.created_at).toLocaleString() : undefined}
              index={index}
              disabled={actingId === order.order_id}
              detailOpenMode="single"
              onOpenDetail={() => onOpenDetail(order.order_id)}
              onProposeDate={() => onProposeDate(order.order_id, false)}
              onReject={() => onReject(order.order_id, false)}
            />
          ))
        : preorderItems.map((row, index) => (
            <OrderOpsCard
              key={row.order_id}
              orderId={row.order_id}
              retailerName={row.order_source || 'Manual pre-order'}
              state={row.status}
              amountLabel={`${fmt(Math.round((row.total_minor ?? 0) / 100))} ${moneyCurrency(row.currency)}`.trim()}
              meta={
                row.requested_delivery_date
                  ? `Requested ${new Date(row.requested_delivery_date).toLocaleDateString()}`
                  : undefined
              }
              badge={showsReviewBadge(row) ? 'Review delivery' : row.preorder_badge}
              index={index}
              disabled={actingId === row.order_id}
              detailOpenMode="single"
              onOpenDetail={() => onOpenDetail(row.order_id)}
              onProposeDate={() => onProposeDate(row.order_id, true, row.requested_delivery_date)}
              onReject={() => onReject(row.order_id, true)}
              proposeDateLabel="Propose date"
              rejectLabel="Reject pre-order"
              canProposeDateOverride
              canRejectOverride
            />
          ))}
    </div>
  );
}
