"use client";

import { usePortalT } from "@/lib/i18n";
import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { AlertTriangle, RefreshCw, WifiOff } from "lucide-react";
import { motion } from "framer-motion";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import { useLiveData } from "@/lib/hooks";
import type {
  ActiveFulfillmentsResponse,
  TrackingOrder,
  TrackingResponse,
} from "@/lib/types";
import { useOptionalWebSocket, useWsEvent, type WsMessage } from "../../../lib/ws";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "../../../components/PageSection";
import NetworkPulsePanel from "../../../components/NetworkPulsePanel";
import { TrackingMap } from "../../../components/tracking/TrackingMap";
import { TrackingStatus } from "../../../components/tracking/TrackingStatus";


const TASHKENT: [number, number] = [69.2401, 41.2995];

const LIGHT_STYLE =
  "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json";
const DARK_STYLE =
  "https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json";

const chipCfg: Record<
  string,
  {
    color: "warning" | "success" | "default" | "danger" | "accent";
    label: string;
  }
> = {
  IN_TRANSIT: { color: "warning", label: t("retailer_desktop.residual.text.in_transit") },
  DISPATCHED: { color: "warning", label: t("supplier_portal.dispatch.text.dispatched") },
  PENDING: { color: "default", label: t("retailer_desktop.residual.text.pending") },
  PENDING_REVIEW: { color: "default", label: t("retailer_desktop.residual.text.pending_review") },
  LOADED: { color: "default", label: t("retailer_desktop.residual.text.loaded") },
  ARRIVED: { color: "success", label: t("retailer_desktop.residual.text.arrived") },
  ARRIVING: { color: "accent", label: t("retailer_desktop.residual.text.arriving") },
  ARRIVED_SHOP_CLOSED: { color: "warning", label: t("retailer_desktop.residual.text.shop_closed") },
  AWAITING_PAYMENT: { color: "warning", label: t("retailer_desktop.residual.text.awaiting_payment") },
  PENDING_CASH_COLLECTION: { color: "warning", label: t("retailer_desktop.residual.text.cash_collection") },
  COMPLETED: { color: "success", label: t("portal.page.orders.filter.completed") },
  FISCALIZING: { color: "warning", label: t("retailer_desktop.residual.text.pending_fiscal") },
  FISCAL_FAILED: { color: "danger", label: t("retailer_desktop.residual.text.fiscal_failed") },
  CANCELLED: { color: "danger", label: t("portal.page.orders.filter.cancelled") },
  CANCEL_REQUESTED: { color: "danger", label: t("retailer_desktop.residual.text.cancel_requested") },
  NO_CAPACITY: { color: "danger", label: t("retailer_desktop.residual.text.no_capacity") },
  SCHEDULED: { color: "default", label: t("supplier_portal.demand.signals.text.scheduled") },
  AUTO_ACCEPTED: { color: "default", label: t("retailer_desktop.residual.text.auto_accepted") },
  QUARANTINE: { color: "danger", label: t("retailer_desktop.residual.text.quarantined") },
  DELIVERED_ON_CREDIT: { color: "success", label: t("retailer_desktop.residual.text.delivered_credit") },
};

type LoadIssue = "restricted" | "offline" | "error";

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ");
}

function useColorScheme(): "light" | "dark" {
  const [scheme, setScheme] = useState<"light" | "dark">("light");
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    setScheme(mq.matches ? "dark" : "light");
    const handler = (e: MediaQueryListEvent) =>
      setScheme(e.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);
  return scheme;
}

export default function TrackingPage() {
  const t = usePortalT();
  const {
    data: trackingData,
    loading,
    error: trackingError,
    isRefreshing: isTrackingRefreshing,
    mutate: mutateTracking,
  } = useLiveData<TrackingResponse>("/v1/retailer/tracking", 15_000);
  const {
    data: fulfillmentData,
    error: fulfillmentError,
    isRefreshing: isFulfillmentRefreshing,
    mutate: mutateFulfillments,
  } = useLiveData<ActiveFulfillmentsResponse>(
    "/v1/retailer/active-fulfillment",
    30_000,
  );
  const ws = useOptionalWebSocket();
  const [orders, setOrders] = useState<TrackingOrder[]>([]);
  const [selectedSupplierIds, setSelectedSupplierIds] = useState<Set<string>>(
    new Set(),
  );
  const [selectedOrder, setSelectedOrder] = useState<TrackingOrder | null>(
    null,
  );
  const mapRef = useRef<maplibregl.Map | null>(null);
  const colorScheme = useColorScheme();
  const isRefreshing = isTrackingRefreshing || isFulfillmentRefreshing;
  const refreshAll = useCallback(() => {
    void mutateTracking();
    void mutateFulfillments();
  }, [mutateFulfillments, mutateTracking]);

  useEffect(() => {
    if (trackingData?.orders) setOrders(trackingData.orders);
  }, [trackingData]);

  useWsEvent(
    "DRIVER_APPROACHING",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      if (!orderId) return;
      setOrders((prev) =>
        prev.map((o) =>
          o.order_id === orderId
            ? {
                ...o,
                is_approaching: true,
                driver_latitude:
                  (msg.driver_latitude as number) ?? o.driver_latitude,
                driver_longitude:
                  (msg.driver_longitude as number) ?? o.driver_longitude,
              }
            : o,
        ),
      );
    }, []),
  );

  useWsEvent(
    "ORDER_COMPLETED",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      if (!orderId) return;
      setOrders((prev) => prev.filter((o) => o.order_id !== orderId));
      setSelectedOrder((prev) => (prev?.order_id === orderId ? null : prev));
      void mutateFulfillments();
    }, [mutateFulfillments]),
  );

  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback(() => {
      refreshAll();
    }, [refreshAll]),
  );

  useWsEvent(
    "ORDER_REASSIGNED",
    useCallback(() => {
      refreshAll();
    }, [refreshAll]),
  );

  const suppliers = useMemo(() => {
    const map = new Map<string, string>();
    for (const o of orders) {
      if (!map.has(o.supplier_id))
        map.set(o.supplier_id, o.supplier_name || "Unknown");
    }
    return Array.from(map, ([id, name]) => ({ id, name })).sort((a, b) =>
      a.name.localeCompare(b.name),
    );
  }, [orders]);

  const visibleOrders = useMemo(() => {
    return orders.filter((o) => {
      if (o.driver_latitude == null || o.driver_longitude == null) return false;
      if (
        selectedSupplierIds.size > 0 &&
        !selectedSupplierIds.has(o.supplier_id)
      )
        return false;
      return true;
    });
  }, [orders, selectedSupplierIds]);

  const activeFulfillmentCount = useMemo(
    () => fulfillmentData?.count ?? orders.length,
    [fulfillmentData?.count, orders.length],
  );
  const recentReceipts = trackingData?.recent_receipts ?? [];

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const errors = [trackingError, fulfillmentError].filter(Boolean) as Array<
      Error & { status?: number }
    >;
    if (errors.length === 0) return null;
    if (errors.some((err) => err.status === 401 || err.status === 403)) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      errors.some((err) =>
        /Failed to fetch|NetworkError|Load failed/i.test(err.message),
      )
    ) {
      return "offline";
    }
    return "error";
  }, [fulfillmentError, trackingError]);

  const emptyStateTitle = useMemo(() => {
    if (loadIssue === "restricted") return "Tracking access restricted";
    if (loadIssue === "offline") return "Tracking is offline";
    if (loadIssue === "error") return "Tracking unavailable";
    if (activeFulfillmentCount > 0) return "Driver location pending";
    return "No active geospatial nodes";
  }, [activeFulfillmentCount, loadIssue]);

  const emptyStateMessage = useMemo(() => {
    if (loadIssue === "restricted") {
      return "Your account cannot view retailer tracking right now.";
    }
    if (loadIssue === "offline") {
      return "Live tracking is offline. Reconnect to refresh driver positions.";
    }
    if (loadIssue === "error") {
      return "Tracking data could not be loaded right now.";
    }
    if (activeFulfillmentCount > 0) {
      return "Active deliveries exist, but live driver location is not available yet.";
    }
    return "No active deliveries with driver location";
  }, [activeFulfillmentCount, loadIssue]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.tracking_access_restricted_for_this_account"),
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: t("retailer_desktop.residual.text.offline_mode_active_showing_the_latest_known_telemetry"),
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.telemetry_sync_degraded_auto_retry_is_active"),
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.live_socket_reconnecting_event_updates_may_be_delayed"),
      };
    }
    if (isRefreshing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: t("retailer_desktop.residual.text.syncing_live_telemetry"),
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loading, ws]);

  const approachingCount = useMemo(
    () => orders.filter((o) => o.is_approaching || o.state === "ARRIVED").length,
    [orders],
  );
  const avgItems = useMemo(() => {
    if (orders.length === 0) return 0;
    return Math.round(
      orders.reduce((sum, order) => sum + order.items.length, 0) / orders.length,
    );
  }, [orders]);



  const toggleSupplier = (id: string) => {
    setSelectedSupplierIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
      }
      return next;
    });
  };

  useRetailerSessionReconcile(() => {
    void mutateTracking();
    void mutateFulfillments();
  });

  return (
    <div
      className="min-h-full p-6 md:p-8 flex flex-col gap-6"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="tracking"
        title={t("retailer_desktop.tracking.text.telemetry_control")}
        description={t("retailer_desktop.residual.text.live_fleet_orchestration_and_inbound_logistics_monitoring")}
        loading={loading && orders.length === 0}
        skeletonVariant="table"
        actions={
          <button
            type="button"
            onClick={refreshAll}
            disabled={isRefreshing}
            className="portal-btn portal-btn--ghost h-10 px-5 rounded-xl font-light"
          >
            <RefreshCw
              size={16}
              className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
            />
            {isRefreshing ? "Syncing" : "Sync GPS"}
          </button>
        }
      >

      <PageSection title={t("retailer_desktop.tracking.text.network_pulse")} description={t("retailer_desktop.residual.text.recent_order_and_notification_activity_for_your_account")}>
        <NetworkPulsePanel />
      </PageSection>

      {syncBanner && (
        <motion.div
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          className={`flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
            syncBanner.kind === "refreshing"
              ? "border-[var(--desk-info)]/30 bg-[var(--desk-info)]/5 text-[var(--desk-info)]"
              : "border-[var(--desk-warning)]/30 bg-[var(--desk-warning)]/10 text-[var(--desk-warning)]"
          }`}
        >
          <div className="flex items-center gap-2">
            <syncBanner.icon
              size={16}
              className={syncBanner.kind === "refreshing" ? "animate-spin" : ""}
            />
            <span className="md-typescale-body-small font-light uppercase tracking-wide">
              {syncBanner.message}
            </span>
          </div>
          {syncBanner.kind !== "refreshing" && (
            <button
              onClick={refreshAll}
              className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-light uppercase tracking-wide hover:bg-current/10"
            >
              Retry
            </button>
          )}
        </motion.div>
      )}

      <TrackingStatus
        activeFulfillmentCount={activeFulfillmentCount}
        approachingCount={approachingCount}
        suppliers={suppliers}
        avgItems={avgItems}
        recentReceipts={recentReceipts}
        selectedSupplierIds={selectedSupplierIds}
        toggleSupplier={toggleSupplier}
      />

      <TrackingMap
        loading={loading}
        visibleOrders={visibleOrders}
        ordersLength={orders.length}
        emptyStateTitle={emptyStateTitle}
        emptyStateMessage={emptyStateMessage}
        loadIssue={loadIssue}
        refreshAll={refreshAll}
        colorScheme={colorScheme}
        activeFulfillmentCount={activeFulfillmentCount}
        selectedOrder={selectedOrder}
        setSelectedOrder={setSelectedOrder}
      />
      </PageChrome>
    </div>
  );
}
