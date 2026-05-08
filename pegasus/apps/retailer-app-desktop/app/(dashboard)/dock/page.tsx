"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
import { Chip, Skeleton } from "@heroui/react";
import {
  Truck,
  Package,
  QrCode,
  Clock,
  MapPin,
  ChevronDown,
  ChevronRight,
  Inbox,
} from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";
import { QRCodeSVG } from "qrcode.react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import { useLiveData } from "../../../lib/hooks";
import { useWsEvent, type WsMessage } from "../../../lib/ws";
import type { TrackingResponse, TrackingOrder } from "../../../lib/types";

/* ── Config ── */

const chipCfg: Record<
  string,
  {
    color: "warning" | "success" | "default" | "danger" | "accent";
    label: string;
  }
> = {
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
  return amount.toLocaleString("en-US").replace(/,/g, " ");
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
  const { data, loading } = useLiveData<TrackingResponse>(
    "/v1/retailer/tracking",
    15_000,
  );
  const [orders, setOrders] = useState<TrackingOrder[]>([]);
  const [expandedSuppliers, setExpandedSuppliers] = useState<Set<string>>(
    new Set(),
  );
  const [revealedTokens, setRevealedTokens] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (data?.orders) setOrders(data.orders);
  }, [data]);

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
              }
            : o,
        ),
      );
    }, []),
  );

  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback((msg: WsMessage) => {
      const orderId = msg.order_id as string | undefined;
      const newState = msg.state as string | undefined;
      if (!orderId || !newState) return;
      setOrders((prev) =>
        prev.map((o) =>
          o.order_id === orderId ? { ...o, state: newState } : o,
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
    }, []),
  );

  const activeOrders = useMemo(
    () =>
      orders.filter((o) =>
        [
          "DISPATCHED",
          "IN_TRANSIT",
          "ARRIVING",
          "ARRIVED",
          "AWAITING_PAYMENT",
        ].includes(o.state),
      ),
    [orders],
  );

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
      if (order.is_approaching || order.state === "ARRIVING")
        group.hasApproaching = true;
      if (order.state === "ARRIVED" || order.state === "AWAITING_PAYMENT")
        group.hasArrived = true;
    }
    return Array.from(map.values()).sort(
      (a, b) => b.totalAmount - a.totalAmount,
    );
  }, [activeOrders]);

  useEffect(() => {
    setExpandedSuppliers(new Set(supplierGroups.map((g) => g.supplierId)));
  }, [supplierGroups.length]);

  const toggleSupplier = (id: string) => {
    setExpandedSuppliers((prev) => {
      const next = new Set(prev);
      next.has(id) ? next.delete(id) : next.add(id);
      return next;
    });
  };

  const toggleToken = (orderId: string) => {
    setRevealedTokens((prev) => {
      const next = new Set(prev);
      next.has(orderId) ? next.delete(orderId) : next.add(orderId);
      return next;
    });
  };

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="mb-8">
        <h1 className="md-typescale-display-small font-bold tracking-tight text-[var(--desk-text-primary)]">
          Dock Control
        </h1>
        <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
          Real-time arrival queue and proximity-locked secure verification.
        </p>
      </header>

      <BentoGrid className="mb-8">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Queue
              </span>
              <Inbox size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <CountUp
              end={activeOrders.length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Active inbound nodes
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Arrived
              </span>
              <MapPin size={18} style={{ color: "var(--desk-success)" }} />
            </div>
            <CountUp
              end={activeOrders.filter((o) => o.state === "ARRIVED").length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Ready for receipt
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                In Transit
              </span>
              <Truck size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <CountUp
              end={activeOrders.filter((o) => o.state === "IN_TRANSIT").length}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              En route to dock
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Throughput
              </span>
              <Clock size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={92}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
              suffix="%"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              On-time performance
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      <div className="max-w-5xl space-y-4">
        <AnimatePresence mode="popLayout">
          {loading ? (
            [0, 1, 2].map((i) => (
              <div
                key={i}
                className="h-32 rounded-3xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
              />
            ))
          ) : supplierGroups.length === 0 ? (
            <div className="py-20 text-center opacity-40">
              <Package size={64} className="mx-auto mb-4" />
              <p className="md-typescale-body-large text-[var(--desk-text-secondary)]">
                The dock is currently clear
              </p>
            </div>
          ) : (
            supplierGroups.map((group) => (
              <motion.div
                key={group.supplierId}
                layout
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-3xl overflow-hidden shadow-[var(--shadow-sm)]"
              >
                <button
                  onClick={() => toggleSupplier(group.supplierId)}
                  className="flex w-full items-center justify-between p-5 hover:bg-[var(--desk-surface-subtle)] transition-colors text-left"
                >
                  <div className="flex items-center gap-4">
                    <div className="w-12 h-12 rounded-2xl bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] flex items-center justify-center font-bold text-xl">
                      {group.supplierName.charAt(0)}
                    </div>
                    <div>
                      <h3 className="md-typescale-title-medium font-bold text-[var(--desk-text-primary)]">
                        {group.supplierName}
                      </h3>
                      <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] font-bold uppercase tracking-widest">
                        {group.orders.length} ACTIVE NODES ·{" "}
                        {formatAmount(group.totalAmount)} UZS
                      </p>
                    </div>
                  </div>
                  <div className="flex items-center gap-4">
                    {group.hasArrived ? (
                      <span className="px-3 py-1 rounded-lg bg-[var(--desk-success)] text-white text-[10px] font-black uppercase tracking-widest">
                        At Dock
                      </span>
                    ) : group.hasApproaching ? (
                      <span className="px-3 py-1 rounded-lg bg-[var(--desk-accent)] text-white text-[10px] font-black uppercase tracking-widest">
                        Approaching
                      </span>
                    ) : null}
                    {expandedSuppliers.has(group.supplierId) ? (
                      <ChevronDown size={20} />
                    ) : (
                      <ChevronRight size={20} />
                    )}
                  </div>
                </button>

                <AnimatePresence>
                  {expandedSuppliers.has(group.supplierId) && (
                    <motion.div
                      initial={{ height: 0, opacity: 0 }}
                      animate={{ height: "auto", opacity: 1 }}
                      exit={{ height: 0, opacity: 0 }}
                      className="border-t border-[var(--desk-border)] divide-y divide-[var(--desk-border)]"
                    >
                      {group.orders.map((order) => (
                        <div
                          key={order.order_id}
                          className="p-5 flex items-center justify-between gap-6 hover:bg-[var(--desk-surface-subtle)] transition-colors"
                        >
                          <div className="flex-1 min-w-0">
                            <div className="flex items-center gap-3 mb-1">
                              <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                                #{order.order_id.slice(-8)}
                              </span>
                              <Chip
                                size="sm"
                                variant="flat"
                                className="font-bold text-[10px]"
                              >
                                {order.state}
                              </Chip>
                            </div>
                            <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase tracking-widest font-bold">
                              {formatAmount(order.total_amount)} UZS ·{" "}
                              {order.items.length} SKUS
                            </p>
                          </div>

                          <div className="flex items-center gap-4">
                            {order.state === "ARRIVED" ||
                            order.state === "AWAITING_PAYMENT" ? (
                              <>
                                <button
                                  onClick={() => toggleToken(order.order_id)}
                                  className={`flex items-center gap-2 px-4 h-10 rounded-xl font-bold transition-all active:scale-95 ${revealedTokens.has(order.order_id) ? "bg-[var(--desk-text-primary)] text-white" : "bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] text-[var(--desk-text-primary)]"}`}
                                >
                                  <QrCode size={18} />
                                  {revealedTokens.has(order.order_id)
                                    ? "Hide"
                                    : "Reveal Token"}
                                </button>
                                {revealedTokens.has(order.order_id) &&
                                  order.delivery_token && (
                                    <motion.div
                                      initial={{ scale: 0 }}
                                      animate={{ scale: 1 }}
                                      className="p-2 bg-white rounded-xl shadow-lg border border-[var(--desk-border)]"
                                    >
                                      <QRCodeSVG
                                        value={order.delivery_token}
                                        size={80}
                                        bgColor="transparent"
                                        fgColor="var(--desk-text-primary)"
                                      />
                                    </motion.div>
                                  )}
                              </>
                            ) : (
                              <div className="flex items-center gap-2 text-[var(--desk-text-tertiary)] opacity-40">
                                <Clock size={16} />
                                <span className="text-[10px] font-bold uppercase tracking-widest">
                                  Locked Until Proximity
                                </span>
                              </div>
                            )}
                          </div>
                        </div>
                      ))}
                    </motion.div>
                  )}
                </AnimatePresence>
              </motion.div>
            ))
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}
