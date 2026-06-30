"use client";

import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { Button, Skeleton } from "@heroui/react";
import {
  Truck,
  Package,
  Users,
  MapPin,
  X,
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useWsEvent, useOptionalWebSocket, type WsMessage } from "../../../lib/ws";
import type {
  ActiveFulfillmentsResponse,
  TrackingResponse,
  TrackingOrder,
} from "../../../lib/types";
import MapGL, { Marker, NavigationControl } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

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
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  DISPATCHED: { color: "warning", label: "Dispatched" },
  PENDING: { color: "default", label: "Pending" },
  PENDING_REVIEW: { color: "default", label: "Pending Review" },
  LOADED: { color: "default", label: "Loaded" },
  ARRIVED: { color: "success", label: "Arrived" },
  ARRIVING: { color: "accent", label: "Arriving" },
  ARRIVED_SHOP_CLOSED: { color: "warning", label: "Shop Closed" },
  AWAITING_PAYMENT: { color: "warning", label: "Awaiting Payment" },
  PENDING_CASH_COLLECTION: { color: "warning", label: "Cash Collection" },
  COMPLETED: { color: "success", label: "Completed" },
  CANCELLED: { color: "danger", label: "Cancelled" },
  CANCEL_REQUESTED: { color: "danger", label: "Cancel Requested" },
  NO_CAPACITY: { color: "danger", label: "No Capacity" },
  SCHEDULED: { color: "default", label: "Scheduled" },
  AUTO_ACCEPTED: { color: "default", label: "Auto-Accepted" },
  QUARANTINE: { color: "danger", label: "Quarantined" },
  DELIVERED_ON_CREDIT: { color: "success", label: "Delivered (Credit)" },
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
        message: "Tracking access restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing the latest known telemetry.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Telemetry sync degraded. Auto-retry is active.",
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Live socket reconnecting. Event updates may be delayed.",
      };
    }
    if (isRefreshing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing live telemetry...",
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

  useEffect(() => {
    const map = mapRef.current;
    if (!map || visibleOrders.length === 0) return;
    const bounds = new maplibregl.LngLatBounds();
    for (const o of visibleOrders)
      bounds.extend([o.driver_longitude!, o.driver_latitude!]);
    map.fitBounds(bounds, { padding: 80, maxZoom: 15, duration: 600 });
  }, [visibleOrders]);

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

  return (
    <div
      className="min-h-full p-6 md:p-8 flex flex-col gap-6"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-display-small font-light tracking-tight text-[var(--desk-text-primary)]">
            Telemetry Control
          </h1>
          <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
            Live fleet orchestration and inbound logistics monitoring.
          </p>
        </div>
        <Button
          variant="secondary"
          onPress={refreshAll}
          isDisabled={isRefreshing}
          className="h-10 px-5 rounded-xl font-light text-[var(--desk-text-secondary)]"
        >
          <RefreshCw
            size={16}
            className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
          />
          {isRefreshing ? "Syncing" : "Sync GPS"}
        </Button>
      </header>

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

      <BentoGrid className="mb-2">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Active Deliveries
              </span>
              <Package size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <CountUp
              end={activeFulfillmentCount}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Inbound orders in motion
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Approaching
              </span>
              <MapPin size={18} style={{ color: "var(--desk-success)" }} />
            </div>
            <CountUp
              end={approachingCount}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Immediate vicinity
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Suppliers
              </span>
              <Users size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <CountUp
              end={suppliers.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Contracted partners
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Avg Items / Order
              </span>
              <Truck size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={avgItems}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Basket density
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      {suppliers.length > 1 && (
        <div className="flex flex-wrap items-center gap-2">
          {suppliers.map((s) => {
            const active =
              selectedSupplierIds.size === 0 || selectedSupplierIds.has(s.id);
            return (
              <button
                key={s.id}
                onClick={() => toggleSupplier(s.id)}
                className={`px-5 py-2 rounded-full md-typescale-label-large font-light transition-all ${
                  active
                    ? "bg-[var(--desk-accent)] text-white shadow-[var(--shadow-sm)]"
                    : "bg-[var(--desk-surface)] text-[var(--desk-text-secondary)] border border-[var(--desk-border)] hover:bg-[var(--desk-surface-subtle)]"
                }`}
              >
                {s.name}
              </button>
            );
          })}
        </div>
      )}

      <div className="relative flex-1 min-h-[500px] rounded-3xl overflow-hidden border border-[var(--desk-border)] shadow-[var(--shadow-md)] bg-[var(--desk-surface)]">
        <AnimatePresence mode="popLayout">
          {loading && orders.length === 0 ? (
            <motion.div
              key="loading"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="absolute inset-0 z-20 bg-[var(--desk-surface)]"
            >
              <Skeleton className="w-full h-full" />
            </motion.div>
          ) : visibleOrders.length === 0 ? (
            <motion.div
              key="empty"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="absolute inset-0 z-10 flex flex-col items-center justify-center gap-4 bg-[var(--desk-surface-subtle)]/80 backdrop-blur-sm"
            >
              <div className="w-20 h-20 rounded-3xl bg-[var(--desk-surface)] border border-[var(--desk-border)] flex items-center justify-center shadow-lg">
                <Truck
                  size={40}
                  className="text-[var(--desk-text-tertiary)] opacity-40"
                />
              </div>
              <p className="md-typescale-title-medium font-light text-[var(--desk-text-secondary)]">
                {emptyStateTitle}
              </p>
              <p className="max-w-md text-center md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase font-light tracking-widest">
                {emptyStateMessage}
              </p>
            </motion.div>
          ) : null}
        </AnimatePresence>

        <MapGL
          mapLib={maplibregl}
          mapStyle={colorScheme === "dark" ? DARK_STYLE : LIGHT_STYLE}
          initialViewState={{
            longitude: TASHKENT[0],
            latitude: TASHKENT[1],
            zoom: 12,
          }}
          style={{ width: "100%", height: "100%" }}
          onLoad={(e) => {
            mapRef.current = e.target;
          }}
          onClick={() => setSelectedOrder(null)}
        >
          <NavigationControl position="top-right" />
          {visibleOrders.map((order) => {
            const isApproaching =
              order.is_approaching || order.state === "ARRIVED";
            return (
              <Marker
                key={order.order_id}
                longitude={order.driver_longitude!}
                latitude={order.driver_latitude!}
                anchor="center"
                onClick={(e) => {
                  e.originalEvent.stopPropagation();
                  setSelectedOrder(order);
                }}
              >
                <motion.div
                  initial={{ scale: 0 }}
                  animate={{ scale: 1 }}
                  whileHover={{ scale: 1.2 }}
                  className={`w-10 h-10 rounded-full flex items-center justify-center cursor-pointer shadow-xl border-2 border-white transition-colors duration-500 ${isApproaching ? "bg-[var(--desk-success)]" : "bg-[var(--desk-accent)]"}`}
                >
                  <Truck size={18} color="white" />
                </motion.div>
              </Marker>
            );
          })}
        </MapGL>

        {activeFulfillmentCount > 0 && (
          <div
            className="absolute top-4 left-4 flex items-center gap-2 px-3 py-1.5 rounded-full md-typescale-label-medium"
            style={{
              background: "var(--background)",
              border: "1px solid var(--border)",
              color: "var(--foreground)",
            }}
          >
            <Truck size={14} />
            <span className="tabular-nums">{activeFulfillmentCount}</span> active
          </div>
        )}

        {/* Selected order info card */}
        <AnimatePresence>
          {selectedOrder && (
            <motion.div
              layout
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              exit={{ y: 20, opacity: 0 }}
              className="absolute bottom-6 left-6 right-6 lg:left-auto lg:w-[400px] bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl p-6 shadow-2xl z-30"
            >
              <div className="flex items-start justify-between mb-4">
                <div className="flex items-center gap-3">
                  <div
                    className={`w-3 h-3 rounded-full ${selectedOrder.is_approaching || selectedOrder.state === "ARRIVED" ? "bg-[var(--desk-success)]" : "bg-[var(--desk-accent)]"}`}
                  />
                  <div>
                    <h3 className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                      {selectedOrder.supplier_name || "Unknown Supplier"}
                    </h3>
                    <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                      ID: #{selectedOrder.order_id.slice(-8)}
                    </p>
                  </div>
                </div>
                <button
                  onClick={() => setSelectedOrder(null)}
                  className="w-8 h-8 rounded-full hover:bg-[var(--desk-surface-subtle)] flex items-center justify-center transition-colors"
                >
                  <X size={16} />
                </button>
              </div>

              <div className="flex flex-wrap gap-1 mb-6">
                {selectedOrder.items.slice(0, 4).map((item) => (
                  <span
                    key={item.product_id}
                    className="px-2 py-0.5 rounded bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] text-[9px] font-black text-[var(--desk-text-tertiary)] uppercase tracking-tighter"
                  >
                    {item.product_name} ×{item.quantity}
                  </span>
                ))}
                {selectedOrder.items.length > 4 && (
                  <span className="text-[9px] font-black text-[var(--desk-text-tertiary)] ml-1">
                    +{selectedOrder.items.length - 4} MORE
                  </span>
                )}
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-[var(--desk-border)]">
                <div>
                  <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest mb-0.5">
                    Asset Value
                  </p>
                  <p className="md-typescale-title-medium font-light text-[var(--desk-text-primary)] tabular-nums">
                    {formatAmount(selectedOrder.total_amount)}{" "}
                    <small className="text-[10px] ml-0.5 opacity-60">UZS</small>
                  </p>
                </div>
                <div className="px-3 py-1 rounded-lg bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-black text-[10px] tracking-widest">
                  {chipCfg[selectedOrder.state]?.label ??
                    selectedOrder.state.replace(/_/g, " ")}
                </div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
