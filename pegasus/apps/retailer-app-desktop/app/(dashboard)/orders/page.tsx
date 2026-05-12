"use client";

import { useState, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useWsEvent } from "../../../lib/ws";
import {
  Copy,
  Truck,
  CheckCircle2,
  PackageOpen,
  MoreVertical,
  Clock,
  ArrowUpRight,
  RefreshCw,
  AlertTriangle,
  XCircle,
  Loader2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import EmptyState from "../../../components/EmptyState";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import type { Order, RetailerProfile } from "../../../lib/types";

const chipCfg: Record<
  string,
  { color: "warning" | "success" | "default" | "danger"; label: string }
> = {
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  COMPLETED: { color: "success", label: "Completed" },
  PENDING: { color: "default", label: "Order Placed" },
  PENDING_REVIEW: { color: "default", label: "Pending Review" },
  LOADED: { color: "default", label: "Approved" },
  DISPATCHED: { color: "warning", label: "Dispatched" },
  ARRIVING: { color: "success", label: "Arriving" },
  ARRIVED: { color: "success", label: "Driver Arrived" },
  ARRIVED_SHOP_CLOSED: { color: "warning", label: "Shop Closed" },
  AWAITING_PAYMENT: { color: "warning", label: "Awaiting Payment" },
  PENDING_CASH_COLLECTION: { color: "warning", label: "Cash Collection" },
  CANCELLED: { color: "danger", label: "Cancelled" },
  CANCEL_REQUESTED: { color: "danger", label: "Cancel Requested" },
  NO_CAPACITY: { color: "danger", label: "No Capacity" },
  SCHEDULED: { color: "default", label: "Scheduled" },
  AUTO_ACCEPTED: { color: "default", label: "Auto-Accepted" },
  QUARANTINE: { color: "danger", label: "Quarantined" },
  DELIVERED_ON_CREDIT: { color: "success", label: "Delivered (Credit)" },
};

export default function OrdersPage() {
  const getProfile = (): RetailerProfile | null => {
    if (typeof localStorage === "undefined") return null;
    try {
      return JSON.parse(localStorage.getItem("retailer_profile") || "null");
    } catch {
      return null;
    }
  };

  const profile = getProfile();
  const ordersUrl = profile?.id
    ? `/v1/retailers/${profile.id}/orders`
    : "/v1/orders";
  const {
    data: orders,
    loading,
    mutate,
  } = useLiveData<Order[]>(ordersUrl, 30000);
  const [activeTab, setActiveTab] = useState<"ALL" | "ACTIVE" | "COMPLETED">(
    "ALL",
  );
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const router = useRouter();

  useWsEvent(
    "ORDER_COMPLETED",
    useCallback(() => mutate(), [mutate]),
  );
  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback(() => mutate(), [mutate]),
  );
  useWsEvent(
    "ORDER_REASSIGNED",
    useCallback(() => mutate(), [mutate]),
  );
  useWsEvent(
    "DRIVER_APPROACHING",
    useCallback(() => mutate(), [mutate]),
  );

  const { data: orderDetail } = useLiveData<Order>(
    selectedId ? `/v1/orders/${selectedId}` : "",
  );

  const list = orders ?? [];
  const filtered = useMemo(() => {
    if (activeTab === "ACTIVE")
      return list.filter(
        (o) => o.state !== "COMPLETED" && o.state !== "CANCELLED",
      );
    if (activeTab === "COMPLETED")
      return list.filter((o) => o.state === "COMPLETED");
    return list;
  }, [activeTab, list]);

  const kpi = useMemo(() => {
    const active = list.filter(
      (o) => o.state === "IN_TRANSIT" || o.state === "DISPATCHED",
    ).length;
    const pending = list.filter(
      (o) => o.state === "PENDING" || o.state === "SCHEDULED",
    ).length;
    const completed = list.filter((o) => o.state === "COMPLETED").length;
    const totalRev = list
      .filter((o) => o.state === "COMPLETED")
      .reduce((s, o) => s + o.amount, 0);
    return { active, pending, completed, totalRev, total: list.length };
  }, [list]);

  const sparkActive = useMemo(
    () =>
      Array.from({ length: 12 }, (_, i) =>
        Math.max(0, kpi.active + Math.sin(i * 0.8) * 2),
      ),
    [kpi.active],
  );

  const selected = selectedId
    ? (list.find((o) => o.order_id === selectedId) ?? null)
    : (list[0] ?? null);
  const detail = orderDetail ?? selected;

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <header className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-display-small font-bold tracking-tight text-[var(--desk-text-primary)]">
            Logistics Tracking
          </h1>
          <p className="mt-1 md-typescale-body-large text-[var(--desk-text-secondary)]">
            Monitor inbound nodes and verify delivery manifests.
          </p>
        </div>
        <Button
          variant="solid"
          onPress={() => router.push("/catalog")}
          className="h-11 px-6 rounded-xl font-bold transition-all shadow-[var(--shadow-sm)]"
          style={{ background: "var(--desk-accent)", color: "white" }}
        >
          <PackageOpen size={18} className="mr-2" /> New Order
        </Button>
      </header>

      <BentoGrid className="mb-8">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                En Route
              </span>
              <Truck size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={kpi.active}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkActive} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Completed
              </span>
              <CheckCircle2
                size={18}
                style={{ color: "var(--desk-success)" }}
              />
            </div>
            <CountUp
              end={kpi.completed}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Nodes delivered
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Pending
              </span>
              <Clock size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={kpi.pending}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Awaiting dispatch
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Settled
              </span>
              <Layers3 size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <CountUp
              end={kpi.totalRev}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              UZS Volume
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      <div className="flex items-center gap-3 mb-6 border-b border-[var(--desk-border)] pb-3">
        {(["ALL", "ACTIVE", "COMPLETED"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`px-5 py-2 rounded-full md-typescale-label-large font-bold transition-all ${
              activeTab === tab
                ? "bg-[var(--desk-text-primary)] text-white shadow-[var(--shadow-sm)]"
                : "text-[var(--desk-text-secondary)] hover:bg-[var(--desk-surface-subtle)]"
            }`}
          >
            {tab}
          </button>
        ))}
        <div className="flex-1" />
        <Button
          variant="light"
          size="sm"
          isIconOnly
          onPress={() => mutate()}
          className="text-[var(--desk-text-tertiary)]"
        >
          <RefreshCw size={16} />
        </Button>
      </div>

      <div className="flex gap-8 min-h-[520px]">
        {/* Order List */}
        <div
          className="w-[440px] shrink-0 flex flex-col gap-2 overflow-y-auto pr-2"
          style={{ maxHeight: "calc(100vh - 440px)" }}
        >
          <AnimatePresence mode="popLayout">
            {loading ? (
              [0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-24 rounded-2xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
                />
              ))
            ) : filtered.length === 0 ? (
              <EmptyState headline="No orders found" variant="no-results" />
            ) : (
              filtered.map((order) => {
                const isSelected =
                  (selectedId ?? list[0]?.order_id) === order.order_id;
                const c = chipCfg[order.state] || chipCfg.PENDING;
                return (
                  <motion.button
                    key={order.order_id}
                    layout
                    initial={{ opacity: 0, y: 10 }}
                    animate={{ opacity: 1, y: 0 }}
                    onClick={() => setSelectedId(order.order_id)}
                    className={`flex items-center gap-4 p-4 rounded-2xl border transition-all text-left group ${
                      isSelected
                        ? "bg-[var(--desk-surface)] border-[var(--desk-accent)] shadow-md ring-2 ring-[var(--desk-accent-soft)]"
                        : "bg-[var(--desk-surface)] border-[var(--desk-border)] hover:border-[var(--desk-border-strong)]"
                    }`}
                  >
                    <div
                      className={`w-11 h-11 rounded-xl flex items-center justify-center shrink-0 transition-colors ${isSelected ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)] group-hover:text-[var(--desk-text-secondary)]"}`}
                    >
                      <PackageOpen size={20} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center justify-between mb-1">
                        <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                          #{order.order_id.slice(-8)}
                        </span>
                        <span
                          className={`text-[10px] font-bold uppercase tracking-widest px-2 py-0.5 rounded-md ${c.color === "success" ? "bg-green-100 text-green-700" : c.color === "warning" ? "bg-orange-100 text-orange-700" : "bg-gray-100 text-gray-700"}`}
                        >
                          {c.label}
                        </span>
                      </div>
                      <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] truncate">
                        {order.payment_gateway || "UNSPECIFIED"}
                      </p>
                    </div>
                    <div className="text-right">
                      <p className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                        {order.amount.toLocaleString()}
                      </p>
                      <ArrowUpRight
                        size={14}
                        className={`ml-auto transition-opacity ${isSelected ? "opacity-100 text-[var(--desk-accent)]" : "opacity-20 group-hover:opacity-100"}`}
                      />
                    </div>
                  </motion.button>
                );
              })
            )}
          </AnimatePresence>
        </div>

        {/* Detail Panel */}
        <div className="flex-1 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)] flex flex-col overflow-hidden">
          {detail ? (
            <div className="p-8 flex-1 overflow-y-auto">
              <div className="flex items-start justify-between mb-8">
                <div>
                  <div className="flex items-center gap-3 mb-2 text-[var(--desk-text-tertiary)]">
                    <span className="md-typescale-label-small font-bold uppercase tracking-[0.2em]">
                      {detail.state.replace("_", " ")}
                    </span>
                    <span className="w-1.5 h-1.5 rounded-full bg-[var(--desk-border-strong)]" />
                    <span className="md-typescale-label-small font-mono">
                      {detail.order_id}
                    </span>
                  </div>
                  <h2 className="md-typescale-display-small font-bold text-[var(--desk-text-primary)]">
                    Order Details
                  </h2>
                </div>
                <Button
                  isIconOnly
                  variant="light"
                  className="text-[var(--desk-text-tertiary)]"
                >
                  <MoreVertical size={20} />
                </Button>
              </div>

              <div className="grid grid-cols-2 gap-4 mb-10">
                <div className="p-5 rounded-2xl bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]">
                  <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2 block">
                    Assigned Route
                  </span>
                  <div className="flex items-center gap-3">
                    <Truck size={20} className="text-[var(--desk-accent)]" />
                    <span className="md-typescale-title-medium font-bold text-[var(--desk-text-primary)]">
                      {detail.route_id
                        ? detail.route_id.slice(-8)
                        : "Pending Assignment"}
                    </span>
                  </div>
                </div>
                <div className="p-5 rounded-2xl bg-[var(--desk-text-primary)] text-white shadow-lg">
                  <span className="md-typescale-label-small uppercase tracking-widest opacity-60 mb-2 block">
                    Settlement Amount
                  </span>
                  <div className="flex items-center justify-between">
                    <span className="md-typescale-display-small font-bold tabular-nums">
                      {detail.amount.toLocaleString()}
                    </span>
                    <CheckCircle2 size={24} className="opacity-40" />
                  </div>
                </div>
              </div>

              <div className="mb-10">
                <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-6">
                  Manifest Content
                </h3>
                <div className="space-y-2">
                  {(detail.items ?? []).map((item) => (
                    <div
                      key={item.line_item_id}
                      className="flex items-center justify-between p-4 bg-[var(--desk-canvas)] rounded-xl border border-[var(--desk-border)]"
                    >
                      <div className="flex items-center gap-4">
                        <div className="w-10 h-10 bg-[var(--desk-surface)] rounded-lg flex items-center justify-center border border-[var(--desk-border)]">
                          <PackageOpen
                            size={18}
                            className="text-[var(--desk-text-tertiary)]"
                          />
                        </div>
                        <div>
                          <p className="md-typescale-body-medium font-bold text-[var(--desk-text-primary)]">
                            {item.sku_name || "Generic SKU"}
                          </p>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
                            QTY: {item.quantity}
                          </p>
                        </div>
                      </div>
                      <span className="md-typescale-title-small font-bold text-[var(--desk-text-primary)]">
                        {(item.unit_price * item.quantity).toLocaleString()}
                      </span>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          ) : (
            <div className="flex-1 flex flex-col items-center justify-center opacity-40 grayscale">
              <PackageOpen size={64} strokeWidth={1} />
              <p className="mt-4 md-typescale-body-large">
                Select an active node
              </p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
