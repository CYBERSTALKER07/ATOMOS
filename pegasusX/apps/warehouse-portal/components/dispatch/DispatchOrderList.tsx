import { useRouter } from 'next/navigation';
import type { WarehouseDispatchOrder } from '@pegasusx/types';
import { VirtualScrollList } from '@pegasusx/ui-kit/desktop';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { OrderOpsCard } from '@/components/orders';

function formatVU(value: number) {
  return value.toFixed(1);
}

const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

interface DispatchOrderListProps {
  orders: WarehouseDispatchOrder[];
  selectedOrderIds: Set<string>;
  allSelected: boolean;
  toggleSelectAll: () => void;
  toggleOrder: (orderId: string) => void;
  opsActingId: string | null;
  setOpsDialog: (dialog: { orderId: string; kind: 'propose' | 'reject' } | null) => void;
  setOpsReason: (reason: string) => void;
  setOpsProposedDate: (date: string) => void;
}

export function DispatchOrderList({
  orders,
  selectedOrderIds,
  allSelected,
  toggleSelectAll,
  toggleOrder,
  opsActingId,
  setOpsDialog,
  setOpsReason,
  setOpsProposedDate,
}: DispatchOrderListProps) {
  const router = useRouter();

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
                  onOpenDetail={() => router.push(`/orders/${order.order_id}?from=dispatch`)}
                  onProposeDate={() => {
                    setOpsDialog({ orderId: order.order_id, kind: 'propose' });
                    setOpsReason('');
                    setOpsProposedDate(new Date().toISOString().slice(0, 10));
                  }}
                  onReject={() => {
                    setOpsDialog({ orderId: order.order_id, kind: 'reject' });
                    setOpsReason('');
                  }}
                />
              </div>
            </div>
          )}
        />
      )}
    </PageSection>
  );
}
