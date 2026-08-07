import { useCallback, useEffect, useState } from "react";
import { usePolling } from "@pegasusx/api-client";
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import type { OrderStatus } from "@pegasusx/types";
import type { SupplierDashboardResponse } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { supplierDashboardCacheKey } from "@/lib/supplier-cache-keys";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";

export interface DashboardMetrics {
  ordersByStatus: Partial<Record<OrderStatus, number>>;
  revenueToday: number;
  revenueChangePct: number;
  activeDrivers: number;
  totalDrivers: number;
  deliveryCompletionRate: number;
  retailersOrderedToday: number;
  totalRetailers: number;
  fleetVuUsed: number;
  fleetVuTotal: number;
}

export interface DispatchManifest {
  id: string;
  status: "DRAFT" | "LOADING" | "DISPATCHED";
  ordersCount: number;
  driverName: string;
}

export interface WsEventLog {
  id: string;
  type: string;
  timestamp: string;
  description: string;
}

export interface DashboardData {
  metrics: DashboardMetrics;
  recentManifests: DispatchManifest[];
  recentEvents: WsEventLog[];
}

type DashboardCacheBundle = {
  data: DashboardData;
  isPaymentConfigured: boolean | null;
};

function emptyOrderStatusCounts(): Partial<Record<OrderStatus, number>> {
  return {
    PENDING: 0,
    SCHEDULED: 0,
    AUTO_ACCEPTED: 0,
    LOADED: 0,
    IN_TRANSIT: 0,
    ARRIVED: 0,
    AWAITING_PAYMENT: 0,
    PENDING_CASH_COLLECTION: 0,
    COMPLETED: 0,
    CANCELLED: 0,
    DELAYED: 0,
  };
}

const api = createSupplierApi();

function mapDashboard(resp: SupplierDashboardResponse): DashboardData {
  const ordersByStatus = emptyOrderStatusCounts();
  for (const [key, value] of Object.entries(resp.orders_by_status ?? {})) {
    const normalized = key.toUpperCase() as OrderStatus;
    if (normalized in ordersByStatus) {
      ordersByStatus[normalized] = value;
    }
  }

  const recentManifests: DispatchManifest[] = (resp.recent_manifests ?? []).map((row) => ({
    id: row.manifest_id,
    status: (row.status === "LOADING" || row.status === "DISPATCHED" ? row.status : "DRAFT") as
      | "DRAFT"
      | "LOADING"
      | "DISPATCHED",
    ordersCount: row.orders_count,
    driverName: row.driver_name,
  }));

  const recentEvents: WsEventLog[] = (resp.activity_events ?? []).map((event) => ({
    id: event.id,
    type: event.type,
    timestamp: event.timestamp,
    description: event.description,
  }));

  return {
    metrics: {
      ordersByStatus,
      revenueToday: resp.today_revenue_minor ?? 0,
      revenueChangePct: 0,
      activeDrivers: resp.active_drivers ?? 0,
      totalDrivers: Math.max(resp.total_drivers ?? 0, resp.active_drivers ?? 0, 1),
      deliveryCompletionRate: resp.delivery_completion_rate_pct ?? 0,
      retailersOrderedToday: resp.retailers_ordered_today ?? 0,
      totalRetailers: Math.max(resp.total_retailers ?? 0, resp.retailers_ordered_today ?? 0, 1),
      fleetVuUsed: resp.fleet_vu_used ?? 0,
      fleetVuTotal: Math.max(resp.fleet_vu_total ?? 1, 1),
    },
    recentManifests,
    recentEvents,
  };
}

const empty: DashboardData = {
  metrics: {
    ordersByStatus: emptyOrderStatusCounts(),
    revenueToday: 0,
    revenueChangePct: 0,
    activeDrivers: 0,
    totalDrivers: 1,
    deliveryCompletionRate: 0,
    retailersOrderedToday: 0,
    totalRetailers: 1,
    fleetVuUsed: 0,
    fleetVuTotal: 1,
  },
  recentManifests: [],
  recentEvents: [],
};

export function useDashboardData() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [isPaymentConfigured, setIsPaymentConfigured] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    const cacheKey = supplierDashboardCacheKey();
    let hydratedFromCache = false;

    if (isTauri()) {
      const cached = await cacheGet<DashboardCacheBundle>(cacheKey, {
        maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
      });
      if (cached?.data) {
        setData(cached.data);
        setIsPaymentConfigured(cached.isPaymentConfigured);
        setLoading(false);
        hydratedFromCache = true;
      }
    }

    if (!silent && !hydratedFromCache) {
      setLoading(true);
    }

    try {
      const [dashResp, profResp] = await Promise.all([
        api.getSupplierDashboard(),
        api.getSupplierProfile(),
      ]);
      const nextData = mapDashboard(dashResp);
      const paymentConfigured =
        Boolean(profResp?.selected_gateways && profResp.selected_gateways.length > 0);
      setData(nextData);
      setIsPaymentConfigured(paymentConfigured);
      setError(null);
      if (isTauri()) {
        void cacheSet(cacheKey, {
          data: nextData,
          isPaymentConfigured: paymentConfigured,
        });
      }
    } catch (err) {
      if (!hydratedFromCache) {
        setError(err instanceof Error ? err.message : "load_dashboard_failed");
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useSupplierSessionReconcile(() => {
    void refresh(true);
  });

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh(true);
    },
    60_000,
    [refresh],
    { pauseWhenHidden: true },
  );

  return {
    ...(data ?? empty),
    isPaymentConfigured,
    loading: loading && !data,
    error,
  };
}
