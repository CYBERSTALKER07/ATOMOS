import { useCallback, useEffect, useRef, useState } from "react";
import { usePolling } from "@pegasusx/api-client";
import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import { isTauri } from "@pegasusx/desktop-bridge";
import type { OrderStatus } from "@pegasusx/types";
import type { ManifestState, SupplierDashboardResponse } from "@pegasusx/types";
import {
  canonicalizeOrderStatus,
  emptyManifestStateCounts,
  emptyOrderStatusCounts,
  incrementOrderStatusCount,
} from "@pegasusx/types";
import {
  DASHBOARD_ROLLUP_REFRESH_EVENTS,
  shouldRefetchDashboardRollup,
} from "@pegasusx/ws-refresh-contract";
import { createSupplierApi } from "@/lib/api";
import { orderStatusFromWsRaw } from "@/lib/dashboard-command";
import { supplierDashboardCacheKey } from "@/lib/supplier-cache-keys";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";

export interface DashboardMetrics {
  ordersByStatus: Partial<Record<OrderStatus, number>>;
  manifestsByState: Record<ManifestState, number>;
  revenueToday: number;
  completedToday: number;
  attemptedToday: number;
  activeDrivers: number;
  totalDrivers: number;
  deliveryCompletionRate: number;
  retailersOrderedToday: number;
  totalRetailers: number;
  fleetVuUsed: number;
  fleetVuTotal: number;
  fleetVuAvailable: boolean;
  asOf: string | null;
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

const api = createSupplierApi();

function mapDashboard(resp: SupplierDashboardResponse): DashboardData {
  const ordersByStatus = emptyOrderStatusCounts();
  for (const [key, value] of Object.entries(resp.orders_by_status ?? {})) {
    const normalized = canonicalizeOrderStatus(key) as OrderStatus;
    if (normalized in ordersByStatus) {
      ordersByStatus[normalized] = value;
    }
  }

  const manifestsByState = emptyManifestStateCounts();
  for (const [key, value] of Object.entries(resp.manifests_by_state ?? {})) {
    const state = key.toUpperCase() as ManifestState;
    if (state in manifestsByState) {
      manifestsByState[state] = value;
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
      manifestsByState,
      revenueToday: resp.today_revenue_minor ?? 0,
      completedToday: resp.deliveries_completed_today ?? 0,
      attemptedToday: resp.deliveries_attempted_today ?? 0,
      activeDrivers: resp.active_drivers ?? 0,
      totalDrivers: resp.total_drivers ?? 0,
      deliveryCompletionRate: resp.delivery_completion_rate_pct ?? 0,
      retailersOrderedToday: resp.retailers_ordered_today ?? 0,
      totalRetailers: resp.total_retailers ?? 0,
      fleetVuUsed: resp.fleet_vu_used ?? 0,
      fleetVuTotal: Math.max(resp.fleet_vu_total ?? 0, 0),
      fleetVuAvailable: resp.fleet_vu_available === true,
      asOf: resp.updated_at ?? null,
    },
    recentManifests,
    recentEvents,
  };
}

const empty: DashboardData = {
  metrics: {
    ordersByStatus: emptyOrderStatusCounts(),
    manifestsByState: emptyManifestStateCounts(),
    revenueToday: 0,
    completedToday: 0,
    attemptedToday: 0,
    activeDrivers: 0,
    totalDrivers: 0,
    deliveryCompletionRate: 0,
    retailersOrderedToday: 0,
    totalRetailers: 0,
    fleetVuUsed: 0,
    fleetVuTotal: 0,
    fleetVuAvailable: false,
    asOf: null,
  },
  recentManifests: [],
  recentEvents: [],
};

export function useDashboardData() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [isPaymentConfigured, setIsPaymentConfigured] = useState<boolean | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const etagRef = useRef<string | null>(null);

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
      const dashResp = await api.getSupplierDashboardConditional(etagRef.current ?? undefined);
      if (!dashResp.notModified) {
        etagRef.current = dashResp.etag;
        const nextData = mapDashboard(dashResp.data);
        setData(nextData);
        if (!silent) {
          const profResp = await api.getSupplierProfile();
          const paymentConfigured =
            Boolean(profResp?.selected_gateways && profResp.selected_gateways.length > 0);
          setIsPaymentConfigured(paymentConfigured);
          if (isTauri()) {
            void cacheSet(cacheKey, {
              data: nextData,
              isPaymentConfigured: paymentConfigured,
            });
          }
        } else if (isTauri()) {
          void cacheSet(cacheKey, {
            data: nextData,
            isPaymentConfigured,
          });
        }
      } else if (dashResp.etag) {
        etagRef.current = dashResp.etag;
      }
      setError(null);
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

  useSupplierWsRefresh(
    (eventType, raw) => {
      if (eventType === "ORDER_STATUS_CHANGED") {
        const status = orderStatusFromWsRaw(raw);
        if (status) {
          setData((prev) => {
            if (!prev) return prev;
            return {
              ...prev,
              metrics: {
                ...prev.metrics,
                ordersByStatus: incrementOrderStatusCount(prev.metrics.ordersByStatus, status),
              },
            };
          });
        }
      }
      if (shouldRefetchDashboardRollup(eventType)) {
        void refresh(true);
      }
    },
    { eventTypes: DASHBOARD_ROLLUP_REFRESH_EVENTS, debounceMs: 500 },
  );

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh(true);
    },
    60_000,
    [refresh],
    { pauseWhenHidden: true, immediate: false },
  );

  return {
    ...(data ?? empty),
    isPaymentConfigured,
    loading: loading && !data,
    error,
  };
}
