"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
import { Chip, Skeleton } from "@heroui/react";
import { Truck, Package, QrCode, Clock, MapPin, ChevronDown, ChevronRight } from "lucide-react";
import { QRCodeSVG } from "qrcode.react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useWsEvent, type WsMessage } from "../../../lib/ws";
import type { TrackingResponse, TrackingOrder } from "../../../lib/types";

/* ── Config ── */

const chipCfg: Record<string, { color: "warning" | "success" | "default" | "danger" | "accent"; label: string }> = {
  DISPATCHED: { color: "warning", label: "Dispatched" },
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  ARRIVING: { color: "accent", label: "Arriving" },
  ARRIVED: { color: "success", label: "Arrived" },
  ARRIVED_SHOP_CLOSED: { color: "warning", label: "Shop Closed" },
  AWAITING_PAYMENT: { color: "danger", label: "Awaiting Payment" },
  PENDING_CASH_COLLECTION: { color: "warning", label: "Cash Collection" },
  PENDING: { color: "default", label: "Pending" },
  PENDING_REVIEW: { color: "default", label: "Pending Review" },
  LOADED: { color: "default", label: "Loaded" },
  COMPLETED: { color: "success", label: "Completed" },
  CANCELLED: { color: "danger", label: "Cancelled" },
  CANCEL_REQUESTED: { color: "danger", label: "Cancel Requested" },
  NO_CAPACITY: { color: "danger", label: "No Capacity" },
  SCHEDULED: { color: "default", label: "Scheduled" },
  AUTO_ACCEPTED: { color: "default", label: "Auto-Accepted" },
  QUARANTINE: { color: "danger", label: "Quarantined" },
  DELIVERED_ON_CREDIT: { color: "success", label: "Delivered (Credit)" },
};

function formatAmount(amount: number): string {
  return amount.toLocaleString("en-US").replace(/,/g, " ") + "";
}

function timeAgo(iso: string): string {
  const diff = Date.now() - new Date(iso).getTime();
  const mins = Math.floor(diff / 60_000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `${hrs}h ago`;
  return `${Math.floor(hrs / 24)}d ago`;
}

/* ── Types ── */

interface SupplierGroup {
  supplierId: string;
  supplierName: string;
  orders: TrackingOrder[];
  totalAmount: number;
  hasApproaching: boolean;
  hasArrived: boolean;
}

/* ── Page ── */

export default function DockPage() {
  const { data, loading } = useLiveData<TrackingResponse>("/v1/retailer/tracking", 15_000);
  const [orders, setOrders] = useState<TrackingOrder[]>([]);
  const [expandedSuppliers, setExpandedSuppliers] = useState<Set<string>>(new Set());
  const [revealedTokens, setRevealedTokens] = useState<Set<string>>(new Set());

  // Sync polling data
  useEffect(() => {
    if (data?.orders) setOrders(data.orders);
  }, [data]);

  // WS: DRIVER_APPROACHING — live position + approaching flag
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
                state: o.state === "IN_TRANSIT" ? "ARRIVING" : o.state,
                driver_latitude: (msg.driver_latitude as number) ?? o.driver_latitude,
                driver_longitude: (msg.driver_longitude as number) ?? o.driver_longitude,
              }
            : o,
        ),
      );
    }, []),
  );

  // WS: ORDER_STATUS_CHANGED — generic state update
  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      const newState = msg.state as string | undefined;
      if (!orderId || !newState) return;
      setOrders((prev) =>
        prev.map((o) => (o.order_id === orderId ? { ...o, state: newState } : o)),
      );
    }, []),
  );

  // WS: ORDER_COMPLETED — remove from dock
  useWsEvent(
    "ORDER_COMPLETED",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      if (!orderId) return;
      setOrders((prev) => prev.filter((o) => o.order_id !== orderId));
      setRevealedTokens((prev) => { const next = new Set(prev); next.delete(orderId); return next; });
    }, []),
  );

  // Active orders only (exclude PENDING/LOADED — not in delivery)
  const activeOrders = useMemo(
    () => orders.filter((o) => ["DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED", "AWAITING_PAYMENT"].includes(o.state)),
    [orders],
  );

  // Group by supplier
  const supplierGroups: SupplierGroup[] = useMemo(() => {
    const map = new Map<string, SupplierGroup>();
    for (const order of activeOrders) {
      const sid = order.supplier_id;
      let group = map.get(sid);
      if (!group) {
        group = {
          supplierId: sid,
          supplierName: order.supplier_name || sid.slice(0, 8),
          orders: [],
          totalAmount: 0,
          hasApproaching: false,
          hasArrived: false,
        };
        map.set(sid, group);
      }
      group.orders.push(order);
      group.totalAmount += order.total_amount;
      if (order.is_approaching || order.state === "ARRIVING") group.hasApproaching = true;
      if (order.state === "ARRIVED" || order.state === "AWAITING_PAYMENT") group.hasArrived = true;
    }
    // Sort: has arrived/approaching first
    return Array.from(map.values()).sort((a, b) => {
      if (a.hasArrived !== b.hasArrived) return a.hasArrived ? -1 : 1;
      if (a.hasApproaching !== b.hasApproaching) return a.hasApproaching ? -1 : 1;
      return b.totalAmount - a.totalAmount;
    });
  }, [activeOrders]);

  // Auto-expand all by default
  useEffect(() => {
    setExpandedSuppliers(new Set(supplierGroups.map((g) => g.supplierId)));
  }, [supplierGroups.length]);

  const toggleSupplier = useCallback((id: string) => {
    setExpandedSuppliers((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  }, []);

  const toggleToken = useCallback((orderId: string) => {
    setRevealedTokens((prev) => {
      const next = new Set(prev);
      next.has(orderId) ? next.delete(orderId) : next.add(orderId);
      return next;
    });
  }, []);

  // QR is only visible when driver is approaching or arrived
  const canShowQR = useCallback(
    (order: TrackingOrder) => order.is_approaching || order.state === "ARRIVING" || order.state === "ARRIVED" || order.state === "AWAITING_PAYMENT",
    [],
  );

  /* ── KPIs ── */
  const totalActive = activeOrders.length;
  const suppliersInbound = supplierGroups.length;
  const arrivedCount = activeOrders.filter((o) => o.state === "ARRIVED" || o.state === "AWAITING_PAYMENT").length;
  const inTransitCount = activeOrders.filter((o) => ["DISPATCHED", "IN_TRANSIT", "ARRIVING"].includes(o.state)).length;

  /* ── Render ── */
  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div>
          <h1 className="md-typescale-headline-large" style={{ color: "var(--foreground)" }}>
            Dock Manager
          </h1>
          <p className="md-typescale-body-medium" style={{ color: "var(--muted)" }}>
            Active deliveries grouped by supplier · QR unlocks on proximity
          </p>
        </div>
      </div>

      {/* KPI Strip */}
      <BentoGrid className="grid-cols-4">
        <BentoCard delay={0}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl" style={{ background: "var(--surface)" }}>
              <Package size={18} style={{ color: "var(--muted)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Active Orders</p>
              <div className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                {loading ? <Skeleton className="h-6 w-12 rounded" /> : <CountUp end={totalActive} />}
              </div>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={60}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl" style={{ background: "var(--surface)" }}>
              <Truck size={18} style={{ color: "var(--muted)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Suppliers Inbound</p>
              <div className="md-typescale-headline-small tabular-nums" style={{ color: "var(--foreground)" }}>
                {loading ? <Skeleton className="h-6 w-12 rounded" /> : <CountUp end={suppliersInbound} />}
              </div>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={120}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl" style={{ background: "var(--surface)" }}>
              <MapPin size={18} style={{ color: "var(--success)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>Arrived</p>
              <div className="md-typescale-headline-small tabular-nums" style={{ color: "var(--success)" }}>
                {loading ? <Skeleton className="h-6 w-12 rounded" /> : <CountUp end={arrivedCount} />}
              </div>
            </div>
          </div>
        </BentoCard>
        <BentoCard delay={180}>
          <div className="flex items-center gap-3 p-4">
            <div className="flex h-10 w-10 items-center justify-center rounded-xl" style={{ background: "var(--surface)" }}>
              <Clock size={18} style={{ color: "var(--warning)" }} />
            </div>
            <div>
              <p className="md-typescale-label-small uppercase tracking-widest" style={{ color: "var(--muted)" }}>In Transit</p>
              <div className="md-typescale-headline-small tabular-nums" style={{ color: "var(--warning)" }}>
                {loading ? <Skeleton className="h-6 w-12 rounded" /> : <CountUp end={inTransitCount} />}
              </div>
            </div>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* Supplier Groups */}
      {loading ? (
        <div className="space-y-4">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-40 w-full rounded-2xl" />
          ))}
        </div>
      ) : supplierGroups.length === 0 ? (
        <div className="bento-card flex flex-col items-center justify-center gap-3 py-16">
          <Package size={48} style={{ color: "var(--muted)" }} />
          <p className="md-typescale-title-medium" style={{ color: "var(--muted)" }}>No active deliveries</p>
          <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
            Orders will appear here once dispatched
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {supplierGroups.map((group) => (
            <SupplierSection
              key={group.supplierId}
              group={group}
              expanded={expandedSuppliers.has(group.supplierId)}
              onToggle={() => toggleSupplier(group.supplierId)}
              revealedTokens={revealedTokens}
              onToggleToken={toggleToken}
              canShowQR={canShowQR}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Supplier Section ── */

function SupplierSection({
  group,
  expanded,
  onToggle,
  revealedTokens,
  onToggleToken,
  canShowQR,
}: {
  group: SupplierGroup;
  expanded: boolean;
  onToggle: () => void;
  revealedTokens: Set<string>;
  onToggleToken: (id: string) => void;
  canShowQR: (order: TrackingOrder) => boolean;
}) {
  const statusChip = group.hasArrived
    ? { color: "success" as const, label: "At Dock" }
    : group.hasApproaching
      ? { color: "accent" as const, label: "Approaching" }
      : { color: "warning" as const, label: "En Route" };

  return (
    <div className="bento-card overflow-hidden">
      {/* Supplier Header */}
      <button
        onClick={onToggle}
        className="flex w-full items-center justify-between p-4 text-left transition-colors hover:bg-[var(--surface)]"
      >
        <div className="flex items-center gap-3">
          <div
            className="flex h-10 w-10 items-center justify-center rounded-xl font-semibold md-typescale-label-large"
            style={{ background: "var(--accent)", color: "var(--accent-foreground)" }}
          >
            {group.supplierName.charAt(0).toUpperCase()}
          </div>
          <div>
            <p className="md-typescale-title-medium" style={{ color: "var(--foreground)" }}>
              {group.supplierName}
            </p>
            <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
              {group.orders.length} order{group.orders.length !== 1 ? "s" : ""} · {formatAmount(group.totalAmount)}
            </p>
          </div>
        </div>
        <div className="flex items-center gap-3">
          <Chip size="sm" color={statusChip.color} variant="soft">
            {statusChip.label}
          </Chip>
          {expanded ? (
            <ChevronDown size={18} style={{ color: "var(--muted)" }} />
          ) : (
            <ChevronRight size={18} style={{ color: "var(--muted)" }} />
          )}
        </div>
      </button>

      {/* Order Cards */}
      {expanded && (
        <div className="border-t" style={{ borderColor: "var(--border)" }}>
          {group.orders.map((order) => (
            <OrderRow
              key={order.order_id}
              order={order}
              qrRevealed={revealedTokens.has(order.order_id)}
              onToggleQR={() => onToggleToken(order.order_id)}
              qrAllowed={canShowQR(order)}
            />
          ))}
        </div>
      )}
    </div>
  );
}

/* ── Order Row ── */

function OrderRow({
  order,
  qrRevealed,
  onToggleQR,
  qrAllowed,
}: {
  order: TrackingOrder;
  qrRevealed: boolean;
  onToggleQR: () => void;
  qrAllowed: boolean;
}) {
  const chip = chipCfg[order.state] ?? { color: "default" as const, label: order.state };
  const hasPosition = order.driver_latitude != null && order.driver_longitude != null;

  return (
    <div
      className="flex items-start gap-4 border-b px-4 py-3 last:border-b-0 transition-colors hover:bg-[var(--surface)]"
      style={{ borderColor: "var(--border)" }}
    >
      {/* Order Info */}
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="md-typescale-title-small truncate" style={{ color: "var(--foreground)" }}>
            {order.order_id.slice(0, 8)}
          </p>
          <Chip size="sm" color={chip.color} variant="soft">
            {chip.label}
          </Chip>
        </div>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--muted)" }}>
          {formatAmount(order.total_amount)} · {order.items.length} item{order.items.length !== 1 ? "s" : ""}
          {order.order_source ? ` · ${order.order_source}` : ""}
        </p>
        <p className="md-typescale-body-small" style={{ color: "var(--muted)" }}>
          {timeAgo(order.created_at)}
          {hasPosition && (
            <span>
              {" "}· <MapPin size={12} className="inline" /> Driver tracked
            </span>
          )}
        </p>
        {/* Line items preview */}
        {order.items.length > 0 && (
          <div className="mt-2 flex flex-wrap gap-1">
            {order.items.slice(0, 3).map((item) => (
              <span
                key={item.product_id}
                className="md-typescale-label-small rounded-md px-2 py-0.5"
                style={{ background: "var(--surface)", color: "var(--muted)" }}
              >
                {item.product_name} ×{item.quantity}
              </span>
            ))}
            {order.items.length > 3 && (
              <span
                className="md-typescale-label-small rounded-md px-2 py-0.5"
                style={{ background: "var(--surface)", color: "var(--muted)" }}
              >
                +{order.items.length - 3} more
              </span>
            )}
          </div>
        )}
      </div>

      {/* QR Section */}
      <div className="flex flex-col items-center gap-2">
        {qrAllowed ? (
          <>
            <button
              onClick={onToggleQR}
              className="flex h-10 items-center gap-2 rounded-xl px-3 transition-colors md-typescale-label-large"
              style={{
                background: qrRevealed ? "var(--accent)" : "var(--surface)",
                color: qrRevealed ? "var(--accent-foreground)" : "var(--foreground)",
              }}
            >
              <QrCode size={16} />
              {qrRevealed ? "Hide" : "Show QR"}
            </button>
            {qrRevealed && order.delivery_token && (
              <div
                className="rounded-xl p-3"
                style={{ background: "var(--background)", border: "1px solid var(--border)" }}
              >
                <QRCodeSVG
                  value={order.delivery_token}
                  size={120}
                  level="M"
                  bgColor="transparent"
                  fgColor="var(--foreground)"
                />
              </div>
            )}
          </>
        ) : (
          <div className="flex h-10 items-center gap-2 rounded-xl px-3 opacity-40" style={{ background: "var(--surface)" }}>
            <QrCode size={16} style={{ color: "var(--muted)" }} />
            <span className="md-typescale-label-small" style={{ color: "var(--muted)" }}>
              Proximity locked
            </span>
          </div>
        )}
      </div>
    </div>
  );
}
