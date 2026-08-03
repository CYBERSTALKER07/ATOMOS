import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import { OrderOpsCard } from '@/components/orders';

interface PreordersListProps {
  items: RetailerOrderLifecycleResponse[];
  actingId: string | null;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (row: RetailerOrderLifecycleResponse) => void;
  onReject: (row: RetailerOrderLifecycleResponse) => void;
}

function showsReviewBadge(row: RetailerOrderLifecycleResponse): boolean {
  return String(row.confirmation_status) === 'PENDING_WAREHOUSE' || row.preorder_badge === 'REVIEW_DELIVERY';
}

const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

export function PreordersList({
  items,
  actingId,
  onOpenDetail,
  onProposeDate,
  onReject,
}: PreordersListProps) {
  return (
    <>
      {items.map((row, index) => (
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
          detailOpenMode="single"
          onOpenDetail={() => onOpenDetail(row.order_id)}
          onProposeDate={() => onProposeDate(row)}
          onReject={() => onReject(row)}
          proposeDateLabel="Propose date"
          rejectLabel="Reject pre-order"
          canProposeDateOverride
          canRejectOverride
        />
      ))}
    </>
  );
}
