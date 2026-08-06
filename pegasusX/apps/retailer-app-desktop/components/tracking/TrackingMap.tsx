"use client";

import { usePortalT } from "@/lib/i18n";
import { useMemo, useEffect, useRef } from "react";
import { Truck, X } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { Skeleton } from "../Skeleton";
import EmptyState from "../EmptyState";
import MapGL, { Layer, Marker, NavigationControl, Source } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";
import { PageSection } from "../PageSection";
import type { TrackingOrder } from "../../lib/types";

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
  FISCALIZING: { color: "warning", label: "Pending fiscal" },
  FISCAL_FAILED: { color: "danger", label: "Fiscal failed" },
  CANCELLED: { color: "danger", label: "Cancelled" },
  CANCEL_REQUESTED: { color: "danger", label: "Cancel Requested" },
  NO_CAPACITY: { color: "danger", label: "No Capacity" },
  SCHEDULED: { color: "default", label: "Scheduled" },
  AUTO_ACCEPTED: { color: "default", label: "Auto-Accepted" },
  QUARANTINE: { color: "danger", label: "Quarantined" },
  DELIVERED_ON_CREDIT: { color: "success", label: "Delivered (Credit)" },
};

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ");
}

interface TrackingMapProps {
  loading: boolean;
  visibleOrders: TrackingOrder[];
  ordersLength: number;
  emptyStateTitle: string;
  emptyStateMessage: string;
  loadIssue: string | null;
  refreshAll: () => void;
  colorScheme: "light" | "dark";
  activeFulfillmentCount: number;
  selectedOrder: TrackingOrder | null;
  setSelectedOrder: (order: TrackingOrder | null) => void;
}

export function TrackingMap({
  loading,
  visibleOrders,
  ordersLength,
  emptyStateTitle,
  emptyStateMessage,
  loadIssue,
  refreshAll,
  colorScheme,
  activeFulfillmentCount,
  selectedOrder,
  setSelectedOrder,
}: TrackingMapProps) {
  const t = usePortalT();
  const mapRef = useRef<maplibregl.Map | null>(null);

  const routeLines = useMemo(() => {
    const features = visibleOrders
      .map((order) => {
        const coords = order.route_geometry?.coordinates;
        if (!coords || coords.length < 2) return null;
        return {
          type: "Feature" as const,
          properties: { order_id: order.order_id },
          geometry: {
            type: "LineString" as const,
            coordinates: coords.map((p) => [p.lng, p.lat] as [number, number]),
          },
        };
      })
      .filter((f): f is NonNullable<typeof f> => f !== null);
    return { type: "FeatureCollection" as const, features };
  }, [visibleOrders]);

  useEffect(() => {
    const map = mapRef.current;
    if (!map || visibleOrders.length === 0) return;
    const bounds = new maplibregl.LngLatBounds();
    let hasBounds = false;
    for (const o of visibleOrders) {
      if (o.driver_longitude != null && o.driver_latitude != null) {
        bounds.extend([o.driver_longitude, o.driver_latitude]);
        hasBounds = true;
      }
      for (const p of o.route_geometry?.coordinates ?? []) {
        bounds.extend([p.lng, p.lat]);
        hasBounds = true;
      }
    }
    if (hasBounds && !bounds.isEmpty()) {
      map.fitBounds(bounds, { padding: 80, maxZoom: 15, duration: 600 });
    }
  }, [visibleOrders]);

  return (
      <PageSection
        title={t("retailer_desktop.tracking.tracking_map.text.live_map")}
        description={t("retailer_desktop.residual.text.driver_positions_for_active_inbound_deliveries")}
        className="flex-1 min-h-[500px] flex flex-col"
      >
      <div className="relative flex-1 min-h-[500px] rounded-2xl overflow-hidden border border-[var(--desk-border)] !mt-0 !p-0">
        <AnimatePresence mode="popLayout">
          {loading && ordersLength === 0 ? (
            <motion.div
              key="loading"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="absolute inset-0 z-20 bg-[var(--desk-surface)]"
            >
              <Skeleton className="w-full h-full" style={{ borderRadius: 0 }} />
            </motion.div>
          ) : visibleOrders.length === 0 ? (
            <motion.div
              key="empty"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1 }}
              className="absolute inset-0 z-10 flex items-center justify-center bg-[var(--desk-surface-subtle)]/80 backdrop-blur-sm p-6"
            >
              <EmptyState
                headline={emptyStateTitle}
                body={emptyStateMessage}
                variant={
                  loadIssue === "restricted"
                    ? "restricted"
                    : loadIssue === "offline"
                      ? "offline"
                      : loadIssue === "error"
                        ? "error"
                        : "no-orders"
                }
                action={loadIssue ? "Retry" : undefined}
                onAction={loadIssue ? refreshAll : undefined}
              />
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
          {routeLines.features.length > 0 ? (
            <Source id="tracking-routes" type="geojson" data={routeLines}>
              <Layer
                id="tracking-routes-line"
                type="line"
                paint={{
                  "line-color": "#2563eb",
                  "line-width": 4,
                  "line-opacity": 0.85,
                }}
              />
            </Source>
          ) : null}
          {visibleOrders.map((order) => {
            if (order.driver_longitude == null || order.driver_latitude == null) {
              return null;
            }
            const isApproaching =
              order.is_approaching || order.state === "ARRIVED";
            return (
              <Marker
                key={order.order_id}
                longitude={order.driver_longitude}
                latitude={order.driver_latitude}
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
      </PageSection>
  );
}
