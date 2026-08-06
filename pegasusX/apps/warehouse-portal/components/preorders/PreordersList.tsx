"use client";

import { usePortalT } from "@/lib/i18n";
import React from 'react';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';

export interface PreordersListProps {
  loading: boolean;
  items: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string, isPreorder: boolean, currentDate?: string) => void;
  onReject: (orderId: string, isPreorder: boolean) => void;
}

function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean {
  return String(row.confirmation_status) === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

export function PreordersList({
  loading,
  items,
  actingId,
  onOpenDetail,
  onProposeDate,
  onReject,
}: PreordersListProps) {
  const t = usePortalT();
  if (loading) {
    return (
      <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3">
        {Array.from({ length: 6 }).map((_, i) => (
          <div key={i} className="md-skeleton h-28 rounded-xl" />
        ))}
      </div>
    );
  }

  if (items.length === 0) {
    return (
      <EmptyState
        variant="no-data"
        headline={t("warehouse_portal.residual.text.no_pre_orders")}
        body={t("warehouse_portal.residual.text.scheduled_manual_pre_orders_will_appear_here")}
      />
    );
  }

  return (
    <div className="wh-ops-grid mt-4">
      {items.map((row, index) => (
        <OrderOpsCard
          key={row.order_id}
          orderId={row.order_id}
          retailerName={row.order_source || 'Manual pre-order'}
          state={row.status}
          amountLabel={`${fmt(Math.round((row.total_minor ?? 0) / 100))} ${row.currency || 'UZS'}`}
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
