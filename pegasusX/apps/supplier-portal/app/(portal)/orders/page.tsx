'use client';

import React, { useCallback, useEffect, useState } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from '@pegasusx/desktop-cache';
import { isTauri } from '@pegasusx/desktop-bridge';
import type { SupplierOrder } from '@pegasusx/types';
import { createSupplierApi } from '@/lib/api';
import { supplierOrdersCacheKey } from '@/lib/supplier-cache-keys';
import { downloadCsv } from '@/lib/csv';
import { SUPPLIER_ORDERS_REFRESH_EVENTS } from '@/lib/supplier-ws-events';
import { useSupplierSessionReconcile } from '@/lib/use-supplier-session-reconcile';
import { useSupplierWsRefresh } from '@/lib/use-supplier-ws-refresh';
import { ListToolbar } from '@/components/ListToolbar';
import { useToast } from '@/components/Toast';
import { PageChrome } from '@/components/PageChrome';
import { usePortalT } from '@/lib/i18n';
import { OrdersList, type OrderFilter } from './components/OrdersList';

const supplierApi = createSupplierApi();
const WEB_PAGE_SIZE = 25;
const DESKTOP_PAGE_SIZE = 200;

export default function OrdersPage() {
  const t = usePortalT();
  const filterLabels: Record<OrderFilter, string> = {
    ACTIVE: t('portal.page.orders.filter.active'),
    SCHEDULED: t('portal.page.orders.filter.scheduled'),
    COMPLETED: t('portal.page.orders.filter.completed'),
    CANCELLED: t('portal.page.orders.filter.cancelled'),
  };
  const { push: toast } = useToast();
  const pageSize = isTauri() ? DESKTOP_PAGE_SIZE : WEB_PAGE_SIZE;
  const [orders, setOrders] = useState<SupplierOrder[]>([]);
  const [total, setTotal] = useState(0);
  const [filter, setFilter] = useState<OrderFilter>('ACTIVE');
  const [page, setPage] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [exporting, setExporting] = useState(false);

  const loadOrders = useCallback(async (silent = false) => {
    const query =
      filter === 'SCHEDULED'
        ? { limit: pageSize, offset: page * pageSize, status: 'SCHEDULED' }
        : { limit: pageSize, offset: page * pageSize, filter };
    const cacheKey = supplierOrdersCacheKey(query);
    let hydratedFromCache = false;

    if (isTauri()) {
      const cached = await cacheGet<{ orders: SupplierOrder[]; total: number }>(cacheKey, {
        maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
      });
      if (cached) {
        setOrders(cached.orders);
        setTotal(cached.total);
        setLoading(false);
        hydratedFromCache = true;
      }
    }

    if (!silent && !hydratedFromCache) {
      setLoading(true);
      setError(null);
    }
    try {
      const response = await supplierApi.getSupplierOrders(query);
      setOrders(response.orders);
      setTotal(response.total ?? response.orders.length);
      if (isTauri()) {
        void cacheSet(cacheKey, {
          orders: response.orders,
          total: response.total ?? response.orders.length,
        });
      }
    } catch (err) {
      const message = err instanceof ApiError ? err.message : 'load_supplier_orders_failed';
      if (!hydratedFromCache) {
        if (!silent) {
          setError(message);
          toast(message, 'error');
        }
      }
    } finally {
      if (!silent || !hydratedFromCache) {
        setLoading(false);
      }
    }
  }, [filter, page, pageSize, toast]);

  useEffect(() => {
    void loadOrders();
  }, [loadOrders]);

  useEffect(() => {
    setPage(0);
  }, [filter]);

  useSupplierWsRefresh(() => {
    void loadOrders(true);
  }, { eventTypes: SUPPLIER_ORDERS_REFRESH_EVENTS });

  useSupplierSessionReconcile(() => {
    void loadOrders(true);
  });

  const pageCount = Math.max(1, Math.ceil(total / pageSize));
  const currentPage = Math.min(page, pageCount - 1);

  const exportCsv = async () => {
    setExporting(true);
    try {
      const query =
        filter === 'SCHEDULED'
          ? { limit: 300, offset: 0, status: 'SCHEDULED' }
          : { limit: 300, offset: 0, filter };
      const response = await supplierApi.getSupplierOrders(query);
      downloadCsv(
        `supplier-orders-${filter.toLowerCase()}.csv`,
        ['order_id', 'status', 'retailer_id', 'driver_id', 'total_minor', 'currency', 'updated_at'],
        response.orders.map((order) => [
          order.order_id,
          order.status,
          order.retailer_id,
          order.driver_id ?? '',
          String(order.total_minor),
          order.currency,
          order.updated_at,
        ]),
      );
    } catch (err) {
      toast(err instanceof Error ? err.message : 'export_failed', 'error');
    } finally {
      setExporting(false);
    }
  };

  return (
    <PageChrome
      icon="orders"
      title={t('portal.page.orders.supplier.title')}
      description={t('portal.page.orders.supplier.description')}
      actions={
        <div className="flex flex-wrap gap-2">
          {(['ACTIVE', 'SCHEDULED', 'COMPLETED', 'CANCELLED'] as OrderFilter[]).map((nextFilter) => (
            <button
              key={nextFilter}
              type="button"
              className="md-chip"
              aria-pressed={filter === nextFilter}
              onClick={() => setFilter(nextFilter)}
            >
              {filterLabels[nextFilter]}
            </button>
          ))}
        </div>
      }
    >
      <div className="space-y-4">
        <ListToolbar
          page={currentPage}
          pageCount={pageCount}
          totalLabel={`${total} ${filterLabels[filter].toLowerCase()}`}
          onPrev={() => setPage((value) => Math.max(value - 1, 0))}
          onNext={() => setPage((value) => Math.min(value + 1, pageCount - 1))}
          onExport={() => void exportCsv()}
          exportDisabled={exporting}
        />

        <OrdersList 
          orders={orders}
          filter={filter}
          loading={loading}
          error={error}
          onRefresh={async () => { await loadOrders() }}
        />
      </div>
    </PageChrome>
  );
}
