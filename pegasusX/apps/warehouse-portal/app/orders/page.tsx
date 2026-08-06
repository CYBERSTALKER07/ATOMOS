'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { useRouter, useSearchParams } from 'next/navigation';
import type { RetailerOrderLifecycleResponse } from '@pegasusx/types';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import { warehouseOps } from '@/lib/warehouse-ops';
import { downloadCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import { useWarehouseWsRefresh } from '@/lib/use-warehouse-ws-refresh';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { ListToolbar } from '@/components/ListToolbar';
import { PageChrome } from '@/components/PageChrome';
import { OrderActionDialog, OrderProposeDateDialog } from '@/components/orders';
import { useToast } from '@/components/Toast';
import { motion } from 'framer-motion';
import { OrdersList, type OrderRow, type OrdersTab } from './components/OrdersList';
import { PreordersList } from '@/components/preorders/PreordersList';
import { usePortalT } from '@/lib/i18n';

function isoDeliveryDate(dateInput: string): string {
  const dateOnly = dateInput.slice(0, 10);
  return `${dateOnly}T12:00:00+05:00`;
}

export default function OrdersPage() {
  const t = usePortalT();
  const router = useRouter();
  const searchParams = useSearchParams();
  const { toast } = useToast();
  const tab: OrdersTab = searchParams.get('tab') === 'preorders' ? 'preorders' : 'active';

  const [orders, setOrders] = useState<OrderRow[]>([]);
  const [preorders, setPreorders] = useState<RetailerOrderLifecycleResponse[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');
  const [actingId, setActingId] = useState<string | null>(null);
  const [dialog, setDialog] = useState<{
    orderId: string;
    kind: 'propose' | 'reject' | 'preorder-reject';
  } | null>(null);
  const [reason, setReason] = useState('');
  const [proposedDate, setProposedDate] = useState('');

  const loadActive = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await warehouseApi.getWarehouseOrders(filter ? { state: filter } : {});
      setOrders(
        (data.orders || []).map((row) => {
          const portalRow = row as unknown as {
            order_id: string;
            retailer_name?: string;
            state?: string;
            status?: string;
            total_uzs?: number;
            total_minor?: number;
            created_at?: string;
            updated_at?: string;
          };
          return {
            order_id: portalRow.order_id,
            retailer_name: portalRow.retailer_name ?? '',
            state: portalRow.state ?? portalRow.status ?? '',
            total_uzs: portalRow.total_uzs ?? portalRow.total_minor ?? 0,
            created_at: portalRow.created_at ?? portalRow.updated_at ?? '',
          };
        }),
      );
    } catch {
      if (!silent) toast('Failed to load orders', 'error');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [filter, toast]);

  const loadPreorders = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const data = await warehouseApi.getWarehousePreorders();
      setPreorders(data.preorders ?? data.items ?? []);
    } catch (err) {
      if (!silent) toast(err instanceof ApiError ? err.message : 'Failed to load pre-orders', 'error');
    } finally {
      if (!silent) setLoading(false);
    }
  }, [toast]);

  const load = useCallback(async (silent = false) => {
    if (tab === 'preorders') {
      await loadPreorders(silent);
    } else {
      await loadActive(silent);
    }
  }, [loadActive, loadPreorders, tab]);

  useEffect(() => {
    void load();
  }, [load]);

  useWarehouseWsRefresh(() => {
    void load(true);
  });

  useWarehouseSessionReconcile(() => {
    void load(true);
  });

  const activePagination = usePagination(orders, 24);
  const preorderPagination = usePagination(preorders, 24);
  const page = tab === 'preorders' ? preorderPagination.page : activePagination.page;
  const pageCount = tab === 'preorders' ? preorderPagination.pageCount : activePagination.pageCount;
  const pageItems = tab === 'preorders' ? preorderPagination.pageItems : activePagination.pageItems;
  const next = tab === 'preorders' ? preorderPagination.next : activePagination.next;
  const prev = tab === 'preorders' ? preorderPagination.prev : activePagination.prev;
  const reset = tab === 'preorders' ? preorderPagination.reset : activePagination.reset;

  useEffect(() => {
    reset();
  }, [filter, tab, reset]);

  const setTab = (nextTab: OrdersTab) => {
    router.replace(nextTab === 'preorders' ? '/orders?tab=preorders' : '/orders');
  };

  const openDetail = (orderId: string) => {
    router.push(`/orders/${orderId}${tab === 'preorders' ? '?from=preorders' : ''}`);
  };

  const handleProposeDate = (orderId: string, isPreorder: boolean, currentDate?: string) => {
    setDialog({ orderId, kind: 'propose' });
    setReason('');
    setProposedDate(currentDate ? currentDate.slice(0, 10) : new Date().toISOString().slice(0, 10));
  };

  const handleReject = (orderId: string, isPreorder: boolean) => {
    setDialog({ orderId, kind: isPreorder ? 'preorder-reject' : 'reject' });
    setReason('');
  };

  const closeDialog = () => {
    setDialog(null);
    setReason('');
    setProposedDate('');
  };

  async function submitDialog() {
    if (!dialog) return;
    const trimmedReason = reason.trim();
    setActingId(dialog.orderId);
    try {
      if (dialog.kind === 'reject') {
        if (!trimmedReason) {
          toast('Reason is required', 'error');
          return;
        }
        const resp = await warehouseOps.rejectOrder(dialog.orderId, trimmedReason);
        toast(`Order cancelled · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      } else if (dialog.kind === 'propose') {
        if (!proposedDate || !trimmedReason) {
          toast('New delivery date and reason are required', 'error');
          return;
        }
        const resp = await warehouseOps.proposeOrderDelivery(
          dialog.orderId,
          isoDeliveryDate(proposedDate),
          trimmedReason,
        );
        toast(`New delivery date proposed · retailer notified · ${resp.status ?? 'ok'}`, 'success');
      } else if (dialog.kind === 'preorder-reject') {
        if (!trimmedReason) {
          toast('Reason is required', 'error');
          return;
        }
        const resp = await warehouseOps.rejectPreorder(dialog.orderId, trimmedReason);
        toast(`Pre-order rejected · ${resp.status ?? 'ok'}`, 'success');
      }
      closeDialog();
      await load(true);
    } catch (err) {
      toast(err instanceof ApiError ? err.message : 'Action failed', 'error');
    } finally {
      setActingId(null);
    }
  }

  const exportCsv = () => {
    if (tab === 'preorders') {
      downloadCsv(
        'warehouse-preorders.csv',
        ['order_id', 'status', 'requested_delivery_date', 'total_minor'],
        preorders.map((row) => [
          row.order_id,
          row.status,
          row.requested_delivery_date ?? '',
          String(row.total_minor ?? 0),
        ]),
      );
      return;
    }
    downloadCsv(
      `warehouse-orders${filter ? `-${filter.toLowerCase()}` : ''}.csv`,
      ['order_id', 'retailer_name', 'state', 'total_uzs', 'created_at'],
      orders.map((order) => [
        order.order_id,
        order.retailer_name ?? '',
        order.state,
        String(order.total_uzs),
        order.created_at,
      ]),
    );
  };

  const dialogCopy = useMemo(() => {
    if (!dialog) return null;
    if (dialog.kind === 'reject') {
      return {
        title: 'Cancel order',
        description: 'Cancels the order and notifies the retailer immediately.',
        confirmLabel: 'Cancel order',
        destructive: true,
        reasonRequired: true,
      };
    }
    if (dialog.kind === 'preorder-reject') {
      return {
        title: 'Reject pre-order',
        description: 'This cancels the scheduled pre-order.',
        confirmLabel: 'Reject pre-order',
        destructive: true,
        reasonRequired: true,
      };
    }
    return null;
  }, [dialog]);

  const activePageItems = tab === 'active' ? (pageItems as OrderRow[]) : [];
  const preorderPageItems = tab === 'preorders' ? (pageItems as RetailerOrderLifecycleResponse[]) : [];

  return (
    <PageTransition>
      <PageChrome
        icon="orders"
        title={t('portal.page.orders.warehouse.title')}
        description={t('portal.page.orders.warehouse.description')}
        actions={
          <div className="flex gap-2 items-center flex-wrap">
            {tab === 'active' ? (
              <select
                value={filter}
                onChange={(e) => {
                  setFilter(e.target.value);
                  setLoading(true);
                }}
                className="px-3 py-1.5 rounded-lg border text-sm"
                style={{
                  background: 'var(--field-background)',
                  borderColor: 'var(--field-border)',
                  color: 'var(--field-foreground)',
                }}
              >
                <option value="">{t('portal.page.orders.filter.all_states')}</option>
                {['PENDING', 'LOADED', 'IN_TRANSIT', 'DELAYED', 'ARRIVED', 'COMPLETED', 'CANCELLED'].map((s) => (
                  <option key={s} value={s}>{s}</option>
                ))}
              </select>
            ) : null}
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => {
                setLoading(true);
                void load();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary active-press"
            >
              <Icon name="refresh" size={16} /> {t('portal.page.orders.action.refresh')}
            </motion.button>
          </div>
        }
      >
        <div className="wh-tab-bar mb-5" role="tablist" aria-label={t("warehouse_portal.orders.text.order_views")}>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'active'}
            className={`wh-tab${tab === 'active' ? ' wh-tab--active' : ''}`}
            onClick={() => setTab('active')}
          >
            {t('portal.page.orders.filter.active_tab')}
          </button>
          <button
            type="button"
            role="tab"
            aria-selected={tab === 'preorders'}
            className={`wh-tab${tab === 'preorders' ? ' wh-tab--active' : ''}`}
            onClick={() => setTab('preorders')}
          >
            {t('portal.page.orders.filter.preorders_tab')}
          </button>
        </div>

        <ListToolbar
          page={page}
          pageCount={pageCount}
          totalLabel={`${tab === 'preorders' ? preorders.length : orders.length} ${tab === 'preorders' ? 'pre-orders' : 'orders'}`}
          onPrev={prev}
          onNext={next}
          onExport={exportCsv}
        />
        
        {tab === 'active' ? (
          <OrdersList
            tab={tab}
            loading={loading}
            filter={filter}
            activeItems={activePageItems}
            preorderItems={preorderPageItems}
            actingId={actingId}
            onOpenDetail={openDetail}
            onProposeDate={handleProposeDate}
            onReject={handleReject}
          />
        ) : (
          <PreordersList
            loading={loading}
            items={preorderPageItems}
            actingId={actingId}
            onOpenDetail={openDetail}
            onProposeDate={handleProposeDate}
            onReject={handleReject}
          />
        )}
      </PageChrome>

      {dialog && dialogCopy ? (
        <>
          <OrderActionDialog
            open={dialog.kind !== 'propose'}
            title={dialogCopy.title}
            description={dialogCopy.description}
            confirmLabel={dialogCopy.confirmLabel}
            destructive={dialogCopy.destructive}
            reason={reason}
            onReasonChange={setReason}
            reasonRequired={dialogCopy.reasonRequired}
            submitting={actingId === dialog.orderId}
            onConfirm={() => void submitDialog()}
            onClose={closeDialog}
          />
          <OrderProposeDateDialog
            open={dialog.kind === 'propose'}
            proposedDate={proposedDate}
            onProposedDateChange={setProposedDate}
            reason={reason}
            onReasonChange={setReason}
            submitting={actingId === dialog.orderId}
            onConfirm={() => void submitDialog()}
            onClose={closeDialog}
          />
        </>
      ) : null}
    </PageTransition>
  );
}
