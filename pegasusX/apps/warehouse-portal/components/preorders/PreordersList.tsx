<<<<<<< HEAD
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import { OrderOpsCard } from '@/components/orders';

interface PreordersListProps {
  items: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (row: RetailerOrderLifecycleResponse) => void;
  onReject: (row: RetailerOrderLifecycleResponse) => void;
=======
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
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
}

function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean {
  return String(row.confirmation_status) === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

export function PreordersList({
<<<<<<< HEAD
=======
  loading,
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  items,
  actingId,
  onOpenDetail,
  onProposeDate,
  onReject,
}: PreordersListProps) {
<<<<<<< HEAD
  return (
    <>
=======
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
        headline="No pre-orders"
        body="Scheduled manual pre-orders will appear here."
      />
    );
  }

  return (
    <div className="wh-ops-grid mt-4">
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
      {items.map((row, index) => (
        <OrderOpsCard
          key={row.order_id}
          orderId={row.order_id}
          retailerName={row.order_source || 'Manual pre-order'}
          state={row.status}
          amountLabel={`${fmt(Math.round((row.total_minor ?? 0) / 100))} ${row.currency || 'UZS'}`}
<<<<<<< HEAD
          meta={row.requested_delivery_date
            ? `Requested ${new Date(row.requested_delivery_date).toLocaleDateString()}`
            : undefined}
=======
          meta={
            row.requested_delivery_date
              ? `Requested ${new Date(row.requested_delivery_date).toLocaleDateString()}`
              : undefined
          }
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
          badge={showsReviewBadge(row) ? 'Review delivery' : row.preorder_badge}
          index={index}
          disabled={actingId === row.order_id}
          detailOpenMode="single"
          onOpenDetail={() => onOpenDetail(row.order_id)}
<<<<<<< HEAD
          onProposeDate={() => onProposeDate(row)}
          onReject={() => onReject(row)}
=======
          onProposeDate={() => onProposeDate(row.order_id, true, row.requested_delivery_date)}
          onReject={() => onReject(row.order_id, true)}
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
          proposeDateLabel="Propose date"
          rejectLabel="Reject pre-order"
          canProposeDateOverride
          canRejectOverride
        />
      ))}
<<<<<<< HEAD
    </>
=======
    </div>
>>>>>>> 5fbd72145092e2ede05adb999b291e8ffbaa19a8
  );
}
