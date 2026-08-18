'use client';

import { usePortalT } from "@/lib/i18n";
import { useRouter } from 'next/navigation';
import type { WarehouseDispatchOrder } from '@pegasusx/types';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';
import { moneyCurrency } from '@pegasusx/api-client';

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
  const t = usePortalT();
  return (
    <PageSection
      title={`Undispatched orders (${orders.length})`}
      description={t("warehouse_portal.residual.text.select_for_dispatch_double_click_a_card_for_order_detail")}
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
        <EmptyState variant="no-data" headline={t("warehouse_portal.residual.text.all_orders_dispatched")} body={t("warehouse_portal.residual.text.no_pending_orders_need_assignment_right_now")} />
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
                  amountLabel={`${fmt(order.total_uzs)} ${moneyCurrency()} · ${formatVU(order.volume_vu ?? 0)} VU`.trim()}
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
