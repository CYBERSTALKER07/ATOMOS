'use client';

import { useRouter } from 'next/navigation';
import type { WarehouseDispatchOrder } from '@pegasusx/types';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';

function fmt(n: number) {
  return new Intl.NumberFormat('uz-UZ').format(n);
}

function formatVU(vu: number) {
  return vu.toFixed(1);
}

interface DispatchOrderListProps {
  orders: WarehouseDispatchOrder[];
  selectedOrderIds: Set<string>;
  allSelected: boolean;
  opsActingId: string | null;
  toggleOrder: (orderId: string) => void;
  toggleSelectAll: () => void;
  onOpenDetail: (orderId: string) => void;
  onProposeDate: (orderId: string) => void;
  onReject: (orderId: string) => void;
}

/**
 * Undispatched orders section of the Dispatch screen.
 *
 * Renders a VirtualScrollList of order cards with selection checkboxes,
 * ops menu actions (propose date / reject), and a select-all toggle.
 */
export default function DispatchOrderList({
  orders,
  selectedOrderIds,
  allSelected,
  opsActingId,
  toggleOrder,
  toggleSelectAll,
  onOpenDetail,
  onProposeDate,
  onReject,
}: DispatchOrderListProps) {
  return (
    <PageSection
      title={`Undispatched orders (${orders.length})`}
      description="Select for dispatch. Double-click a card for order detail."
      actions={
        orders.length > 0 ? (
          <label className="flex items-center gap-2 text-xs text-(--muted) cursor-pointer">
            <input type="checkbox" checked={allSelected} onChange={toggleSelectAll} />
            Select all
          </label>
        ) : undefined
      }
    >
      {orders.length === 0 ? (
        <EmptyState variant="no-data" headline="All orders dispatched" body="No pending orders need assignment right now." />
      ) : (
        <VirtualScrollList
          className="-mx-5 px-5"
          height="28rem"
          items={orders}
          itemKey={(order) => order.order_id}
          renderItem={(order, index) => (
            <div className="flex items-start gap-2 pb-3">
              <label className="flex items-center pt-4 shrink-0 cursor-pointer">
                <input
                  type="checkbox"
                  checked={selectedOrderIds.has(order.order_id)}
                  onChange={() => toggleOrder(order.order_id)}
                />
              </label>
              <div className="flex-1 min-w-0">
                <OrderOpsCard
                  orderId={order.order_id}
                  retailerName={order.retailer_name || 'Unknown'}
                  state="PENDING"
                  amountLabel={`${fmt(order.total_uzs)} UZS · ${formatVU(order.volume_vu ?? 0)} VU`}
                  index={index}
                  disabled={opsActingId === order.order_id}
                  detailOpenMode="double"
                  onOpenDetail={() => onOpenDetail(order.order_id)}
                  onProposeDate={() => onProposeDate(order.order_id)}
                  onReject={() => onReject(order.order_id)}
                />
              </div>
            </div>
          )}
        />
      )}
    </PageSection>
  );
}
