"use client";

import { useState, useMemo, useCallback, useEffect, useRef } from "react";
import { Chip, Skeleton } from "@heroui/react";
import { Truck, Package, Users, MapPin, X } from "lucide-react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useWsEvent, type WsMessage } from "../../../lib/ws";
import type { TrackingResponse, TrackingOrder } from "../../../lib/types";
import MapGL, { Marker, NavigationControl } from "react-map-gl/maplibre";
import maplibregl from "maplibre-gl";
import "maplibre-gl/dist/maplibre-gl.css";

const TASHKENT: [number, number] = [69.2401, 41.2995];

const LIGHT_STYLE = "https://basemaps.cartocdn.com/gl/positron-gl-style/style.json";
const DARK_STYLE = "https://basemaps.cartocdn.com/gl/dark-matter-gl-style/style.json";

const chipCfg: Record<string, { color: "warning" | "success" | "default" | "danger"; label: string }> = {
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  DISPATCHED: { color: "warning", label: "Dispatched" },
  PENDING: { color: "default", label: "Pending" },
  LOADED: { color: "default", label: "Loaded" },
  ARRIVED: { color: "success", label: "Arrived" },
  ARRIVING: { color: "success", label: "Arriving" },
};

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ") + "";
}

function useColorScheme(): "light" | "dark" {
  const [scheme, setScheme] = useState<"light" | "dark">("light");
  useEffect(() => {
    const mq = window.matchMedia("(prefers-color-scheme: dark)");
    setScheme(mq.matches ? "dark" : "light");
    const handler = (e: MediaQueryListEvent) => setScheme(e.matches ? "dark" : "light");
    mq.addEventListener("change", handler);
    return () => mq.removeEventListener("change", handler);
  }, []);
  return scheme;
}

export default function TrackingPage() {
  const { data, loading, mutate } = useLiveData<TrackingResponse>("/v1/retailer/tracking", 15_000);
  const [orders, setOrders] = useState<TrackingOrder[]>([]);
  const [selectedSupplierIds, setSelectedSupplierIds] = useState<Set<string>>(new Set());
  const [selectedOrder, setSelectedOrder] = useState<TrackingOrder | null>(null);
  const mapRef = useRef<maplibregl.Map | null>(null);
  const colorScheme = useColorScheme();

  // Sync polling data into local state
  useEffect(() => {
    if (data?.orders) setOrders(data.orders);
  }, [data]);

  // WS: DRIVER_APPROACHING — instant green transition + position update
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
                driver_latitude: (msg.driver_latitude as number) ?? o.driver_latitude,
                driver_longitude: (msg.driver_longitude as number) ?? o.driver_longitude,
              }
            : o,
        ),
      );
    }, []),
  );

  // WS: ORDER_COMPLETED — remove from map
  useWsEvent(
    "ORDER_COMPLETED",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      if (!orderId) return;
      setOrders((prev) => prev.filter((o) => o.order_id !== orderId));
      setSelectedOrder((prev) => (prev?.order_id === orderId ? null : prev));
    }, []),
  );

  // WS: ORDER_STATUS_CHANGED — refresh tracking data
  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback(() => {
      mutate();
    }, [mutate]),
  );

  // Derived: unique suppliers
  const suppliers = useMemo(() => {
    const map = new Map<string, string>();
    for (const o of orders) {
      if (!map.has(o.supplier_id)) map.set(o.supplier_id, o.supplier_name || "Unknown");
    }
    return Array.from(map, ([id, name]) => ({ id, name })).sort((a, b) => a.name.localeCompare(b.name));
  }, [orders]);

  // Derived: visible orders (with driver location + matching supplier filter)
  const visibleOrders = useMemo(() => {
    return orders.filter((o) => {
      if (o.driver_latitude == null || o.driver_longitude == null) return false;
      if (selectedSupplierIds.size > 0 && !selectedSupplierIds.has(o.supplier_id)) return false;
      return true;
    });
  }, [orders, selectedSupplierIds]);

  // KPI metrics
  const approachingCount = useMemo(() => orders.filter((o) => o.is_approaching || o.state === "ARRIVED").length, [orders]);
  const avgItems = useMemo(() => {
    if (orders.length === 0) return 0;
    return Math.round(orders.reduce((sum, o) => sum + o.items.length, 0) / orders.length);
  }, [orders]);

  // Fit camera to visible markers
  useEffect(() => {
    const map = mapRef.current;
    if (!map || visibleOrders.length === 0) return;
    const bounds = new maplibregl.LngLatBounds();
    for (const o of visibleOrders) {
      bounds.extend([o.driver_longitude!, o.driver_latitude!]);
    }
    map.fitBounds(bounds, { padding: 80, maxZoom: 15, duration: 600 });
  }, [visibleOrders]);

  function toggleSupplier(id: string) {
    setSelectedSupplierIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  return (
    <div className="flex flex-col h-full gap-6 p-8">
      {/* Page header */}
      <div>
        <h1 className="md-typescale-headline-large" style={{ color: "var(--foreground)" }}>
          Delivery Tracking
        </h1>
        <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
          Live driver positions for your active orders
        </p>
      </div>

      {/* KPI strip */}
      <BentoGrid>
        <BentoCard delay={0}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl" style={{ background: "var(--surface)" }}>
              <Package size={18} style={{ color: "var(--muted)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Active Deliveries</p>
              <p className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                <CountUp end={orders.length} />
              </p>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={60}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl" style={{ background: "var(--success)", opacity: 0.12 }}>
              <MapPin size={18} style={{ color: "var(--success)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Approaching</p>
              <p className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                <CountUp end={approachingCount} />
              </p>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={120}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl" style={{ background: "var(--surface)" }}>
              <Users size={18} style={{ color: "var(--muted)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Suppliers</p>
              <p className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                <CountUp end={suppliers.length} />
              </p>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={180}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex items-center justify-center w-10 h-10 rounded-xl" style={{ background: "var(--surface)" }}>
              <Truck size={18} style={{ color: "var(--muted)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Avg Items / Order</p>
              <p className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                <CountUp end={avgItems} />
              </p>
            </div>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* Supplier filter chips */}
      {suppliers.length > 1 && (
        <div className="flex gap-2 flex-wrap">
          {suppliers.map((s) => {
            const active = selectedSupplierIds.size === 0 || selectedSupplierIds.has(s.id);
            return (
              <Chip
                key={s.id}
                variant={active ? "primary" : "soft"}
                className="cursor-pointer select-none transition-all duration-200"
                style={active ? { background: "var(--accent)", color: "var(--accent-foreground)" } : { background: "var(--surface)", color: "var(--muted)" }}
                onClick={() => toggleSupplier(s.id)}
              >
                {s.name}
              </Chip>
            );
          })}
        </div>
      )}

      {/* Map + info card */}
      <div className="relative flex-1 min-h-100 rounded-2xl overflow-hidden" style={{ border: "1px solid var(--border)" }}>
        {loading && orders.length === 0 ? (
          <Skeleton className="w-full h-full rounded-2xl" />
        ) : (
          <MapGL
            mapLib={maplibregl}
            mapStyle={colorScheme === "dark" ? DARK_STYLE : LIGHT_STYLE}
            initialViewState={{ longitude: TASHKENT[0], latitude: TASHKENT[1], zoom: 12 }}
            style={{ width: "100%", height: "100%" }}
            onLoad={(e) => { mapRef.current = e.target; }}
            onClick={() => setSelectedOrder(null)}
          >
            <NavigationControl position="top-right" />
            {visibleOrders.map((order) => {
              const isGreen = order.is_approaching || order.state === "ARRIVED";
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
                  <div
                    className="flex items-center justify-center rounded-full cursor-pointer transition-all duration-200"
                    style={{
                      width: 36,
                      height: 36,
                      background: isGreen ? "var(--success)" : "var(--accent)",
                      boxShadow: "0 2px 8px rgba(0,0,0,0.18)",
                    }}
                  >
                    <Truck size={16} color={isGreen ? "#fff" : "var(--accent-foreground)"} />
                  </div>
                </Marker>
              );
            })}
          </MapGL>
        )}

        {/* Active badge */}
        {visibleOrders.length > 0 && (
          <div
            className="absolute top-4 left-4 flex items-center gap-2 px-3 py-1.5 rounded-full md-typescale-label-medium"
            style={{ background: "var(--background)", border: "1px solid var(--border)", color: "var(--foreground)" }}
          >
            <Truck size={14} />
            <span className="tabular-nums">{visibleOrders.length}</span> active
          </div>
        )}

        {/* Empty state overlay */}
        {!loading && visibleOrders.length === 0 && (
          <div className="absolute inset-0 flex flex-col items-center justify-center gap-3" style={{ background: "var(--background)", opacity: 0.9 }}>
            <div className="flex items-center justify-center w-16 h-16 rounded-2xl" style={{ background: "var(--surface)" }}>
              <Truck size={28} style={{ color: "var(--muted)" }} />
            </div>
            <p className="md-typescale-body-medium" style={{ color: "var(--muted)" }}>
              No active deliveries with driver location
            </p>
          </div>
        )}

        {/* Selected order info card */}
        {selectedOrder && (
          <div
            className="absolute bottom-4 left-4 right-4 max-w-md bento-card p-4"
            style={{
              background: "var(--background)",
              border: "1px solid var(--border)",
              animation: "slideUp 200ms cubic-bezier(0.2, 0, 0, 1)",
            }}
          >
            <div className="flex items-start justify-between gap-3">
              <div className="flex items-center gap-2 min-w-0">
                <div
                  className="w-2 h-2 rounded-full shrink-0"
                  style={{
                    background: selectedOrder.is_approaching || selectedOrder.state === "ARRIVED"
                      ? "var(--success)"
                      : "var(--accent)",
                  }}
                />
                <span
                  className="md-typescale-title-small truncate"
                  style={{ color: "var(--foreground)" }}
                >
                  {selectedOrder.supplier_name || "Unknown Supplier"}
                </span>
              </div>
              <div className="flex items-center gap-2 shrink-0">
                <Chip size="sm" variant="soft" color={chipCfg[selectedOrder.state]?.color ?? "default"}>
                  {chipCfg[selectedOrder.state]?.label ?? selectedOrder.state.replace(/_/g, " ")}
                </Chip>
                <button
                  className="flex items-center justify-center w-6 h-6 rounded-full transition-colors"
                  style={{ color: "var(--muted)" }}
                  onClick={() => setSelectedOrder(null)}
                >
                  <X size={14} />
                </button>
              </div>
            </div>

            <p className="md-typescale-body-small mt-2 line-clamp-2" style={{ color: "var(--muted)" }}>
              {selectedOrder.items.map((i) => `${i.product_name} ×${i.quantity}`).join(", ") || "No items"}
            </p>

            <p className="md-typescale-label-medium tabular-nums mt-1" style={{ color: "var(--foreground)" }}>
              {formatAmount(selectedOrder.total_amount)}
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
