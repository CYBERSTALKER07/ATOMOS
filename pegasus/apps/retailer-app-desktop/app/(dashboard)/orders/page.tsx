"use client";

import { useState, useMemo, useCallback } from "react";
import { useRouter } from "next/navigation";
import { useWsEvent } from "../../../lib/ws";
import {
  Copy, Truck, CheckCircle2, PackageOpen, MoreVertical,
  Clock, ArrowUpRight, Filter, AlertTriangle, XCircle, Loader2,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import type { Order, RetailerProfile } from "../../../lib/types";

const chipCfg: Record<string, { color: "warning" | "success" | "default" | "danger"; label: string }> = {
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
    if (typeof localStorage === 'undefined') return null;
    try { return JSON.parse(localStorage.getItem('retailer_profile') || 'null'); } catch { return null; }
  };

  const profile = getProfile();
  const ordersUrl = profile?.id ? `/v1/orders?retailer_id=${profile.id}` : "/v1/orders";
  const { data: orders, loading, error, mutate } = useLiveData<Order[]>(ordersUrl, 30000);
  const [activeTab, setActiveTab] = useState<"ALL" | "ACTIVE" | "COMPLETED">("ALL");
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const router = useRouter();

  // Auto-refresh orders list when backend pushes order events
  useWsEvent("ORDER_COMPLETED", useCallback(() => mutate(), [mutate]));
  useWsEvent("ORDER_STATUS_CHANGED", useCallback(() => mutate(), [mutate]));
  useWsEvent("DRIVER_APPROACHING", useCallback(() => mutate(), [mutate]));
  useWsEvent("PRE_ORDER_AUTO_ACCEPTED", useCallback(() => mutate(), [mutate]));
  useWsEvent("PRE_ORDER_CONFIRMED", useCallback(() => mutate(), [mutate]));
  useWsEvent("PRE_ORDER_EDITED", useCallback(() => mutate(), [mutate]));

  // Fetch detail for selected order (includes line items)
  const { data: orderDetail } = useLiveData<Order>(
    selectedId ? `/v1/orders/${selectedId}` : "",
  );

  const cancelOrder = useCallback(async (order: Order) => {
    const profile = getProfile();
    if (!profile?.id) { setActionError('Profile not found. Please re-login.'); return; }
    setCancelling(true);
    setActionError(null);
    try {
      const res = await apiFetch('/v1/order/cancel', {
        method: 'POST',
        headers: {
          "Idempotency-Key": `retailer-cancel:${order.order_id}:${order.version ?? 0}`,
        },
        body: JSON.stringify({
          order_id: order.order_id,
          retailer_id: profile.id,
          version: order.version ?? 0,
        }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => null);
        throw new Error(errBody?.error || `Cancel failed (${res.status})`);
      }
      mutate();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Cancel failed');
    } finally { setCancelling(false); }
  }, [mutate]);

  const verifyOrder = useCallback(async (order: Order) => {
    setVerifying(true);
    setActionError(null);
    try {
      const res = await apiFetch(`/v1/orders/${order.order_id}/status`, {
        method: 'PATCH',
        body: JSON.stringify({ status: 'COMPLETED' }),
      });
      if (!res.ok) {
        const errBody = await res.json().catch(() => null);
        throw new Error(errBody?.error || `Verify failed (${res.status})`);
      }
      mutate();
    } catch (err: unknown) {
      setActionError(err instanceof Error ? err.message : 'Verify failed');
    } finally { setVerifying(false); }
  }, [mutate]);

  const list = orders ?? [];

  const filtered = useMemo(() => {
    if (activeTab === "ACTIVE") return list.filter((o) => o.state !== "COMPLETED" && o.state !== "CANCELLED");
    if (activeTab === "COMPLETED") return list.filter((o) => o.state === "COMPLETED");
    return list;
  }, [activeTab, list]);

  const kpi = useMemo(() => {
    const active = list.filter((o) => o.state === "IN_TRANSIT" || o.state === "DISPATCHED").length;
    const pending = list.filter((o) => o.state === "PENDING" || o.state === "SCHEDULED").length;
    const completed = list.filter((o) => o.state === "COMPLETED").length;
    const totalRev = list.filter((o) => o.state === "COMPLETED").reduce((s, o) => s + o.amount, 0);
    return { active, pending, completed, totalRev, total: list.length };
  }, [list]);

  const sparkActive = useMemo(() => Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.active + Math.sin(i * 0.8) * 2)), [kpi.active]);
  const sparkCompleted = useMemo(() => Array.from({ length: 12 }, (_, i) => Math.max(0, kpi.completed * 0.3 + i * (kpi.completed * 0.06))), [kpi.completed]);

  const selected = selectedId ? list.find((o) => o.order_id === selectedId) ?? null : list[0] ?? null;
  const detail = orderDetail ?? selected;
  const cfg = detail ? (chipCfg[detail.state] ?? chipCfg.PENDING) : chipCfg.PENDING;

  /* ── Loading skeleton ── */
  if (loading) {
    return (
      <div className="min-h-full p-6 md:p-8">
        <Skeleton className="h-8 w-64 rounded-lg mb-2" />
        <Skeleton className="h-4 w-96 rounded-lg mb-8" />
        <div className="grid grid-cols-4 gap-4 mb-8">
          {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-28 rounded-2xl" />)}
        </div>
        <div className="flex gap-6">
          <div className="w-[480px] flex flex-col gap-2">
            {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-20 rounded-2xl" />)}
          </div>
          <Skeleton className="flex-1 h-96 rounded-2xl" />
        </div>
      </div>
    );
  }

  /* ── Error state ── */
  if (error) {
    return (
      <div className="min-h-full p-6 md:p-8 flex flex-col items-center justify-center gap-4">
        <AlertTriangle size={32} style={{ color: "var(--danger)" }} />
        <p className="md-typescale-title-medium text-foreground">Failed to load orders</p>
        <p className="md-typescale-body-medium text-muted">{error.message}</p>
        <Button onPress={() => mutate()} className="md-btn md-btn-outlined">Retry</Button>
      </div>
    );
  }

  /* ── Empty state ── */
  if (list.length === 0) {
    return (
      <div className="min-h-full p-6 md:p-8 flex flex-col items-center justify-center gap-4">
        <PackageOpen size={48} style={{ color: "var(--muted)" }} />
        <p className="md-typescale-title-large font-semibold text-foreground">No orders yet</p>
        <p className="md-typescale-body-medium text-muted">Your incoming deliveries will appear here.</p>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8">
      {/* ── Header ── */}
      <header className="mb-6 flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="md-typescale-headline-large">Orders & Tracking</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Monitor incoming deliveries. Verify manifests and confirm receipt.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <Button variant="primary" className="md-btn md-btn-filled md-typescale-label-large px-5 h-10 flex items-center gap-2" onPress={() => router.push("/catalog")}>
            <PackageOpen size={18} /> Fast Order
          </Button>
        </div>
      </header>

      {/* ── KPI Bento ── */}
      <BentoGrid className="mb-8">
        <BentoCard delay={0}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">In Transit</span>
              <Truck size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={kpi.active} className="md-kpi-value" />
              <MiniSparkline data={sparkActive} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">{kpi.pending} pending</span>
          </div>
        </BentoCard>

        <BentoCard delay={60}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Completed</span>
              <CheckCircle2 size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={kpi.completed} className="md-kpi-value" />
              <MiniSparkline data={sparkCompleted} width={72} height={28} />
            </div>
            {kpi.total > 0 && (
              <div className="flex items-center gap-1.5">
                <ArrowUpRight size={14} strokeWidth={2} style={{ color: 'var(--success)' }} />
                <span className="md-kpi-sub" style={{ color: 'var(--success)' }}>
                  {((kpi.completed / kpi.total) * 100).toFixed(0)}%
                </span>
              </div>
            )}
          </div>
        </BentoCard>

        <BentoCard delay={120}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Pending</span>
              <Clock size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <CountUp end={kpi.pending} className="md-kpi-value" />
            <span className="md-kpi-sub">Awaiting dispatch</span>
          </div>
        </BentoCard>

        <BentoCard delay={180}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Settled</span>
              <CheckCircle2 size={18} strokeWidth={1.5} style={{ color: 'var(--muted)' }} />
            </div>
            <CountUp end={kpi.totalRev} className="md-kpi-value" suffix="" />
            <span className="md-kpi-sub">Revenue confirmed</span>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* ── Filter Tabs ── */}
      <div className="flex items-center gap-3 mb-6 border-b border-[var(--border)] pb-3 flex-wrap">
        {(["ALL", "ACTIVE", "COMPLETED"] as const).map((tab) => (
          <button
            key={tab}
            onClick={() => setActiveTab(tab)}
            className={`md-typescale-label-large px-4 py-2 rounded-full font-semibold transition-colors cursor-pointer ${
              activeTab === tab
                ? 'bg-accent text-accent-foreground'
                : 'text-muted hover:text-foreground hover:bg-surface'
            }`}
          >
            {tab === "ALL" ? `All (${list.length})` : tab === "ACTIVE" ? "Active" : "Completed"}
          </button>
        ))}
        <div className="flex-1" />
        <Button variant="ghost" className="text-muted md-typescale-label-large flex items-center gap-2" onPress={() => mutate()}>
          <Filter size={16} /> Refresh
        </Button>
      </div>

      {/* ── Split: Order List + Detail Panel ── */}
      <div className="flex gap-6 min-h-[480px]">
        
        {/* Left: Order List */}
        <div className="w-full lg:w-[420px] xl:w-[480px] shrink-0 flex flex-col gap-2 overflow-y-auto max-h-[calc(100dvh-420px)] pr-1">
          {filtered.map((order) => {
            const c = chipCfg[order.state] ?? chipCfg.PENDING;
            const isSelected = (selectedId ?? list[0]?.order_id) === order.order_id;
            return (
              <button
                key={order.order_id}
                onClick={() => setSelectedId(order.order_id)}
                className={`bento-card w-full text-left cursor-pointer transition-all duration-150 ${
                  isSelected ? 'ring-2 ring-accent border-accent' : ''
                }`}
              >
                <div className="flex items-center gap-4">
                  <div className="w-10 h-10 rounded-xl flex items-center justify-center shrink-0" style={{ background: 'var(--surface)' }}>
                    <PackageOpen size={18} style={{ color: 'var(--muted)' }} />
                  </div>
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center justify-between gap-2">
                      <span className="md-typescale-title-small font-semibold text-foreground truncate">
                        #{order.order_id.slice(-8)}
                      </span>
                      <Chip size="sm" color={c.color} variant="soft" className="shrink-0">{c.label}</Chip>
                    </div>
                    <div className="flex items-center gap-2 mt-1">
                      <span className="md-typescale-body-small text-muted truncate">
                        {order.payment_gateway || "UNSPECIFIED"}
                      </span>
                      <span className="text-muted opacity-40">·</span>
                      <span className="md-typescale-label-small font-mono tabular-nums text-muted">
                        {order.items?.length ?? 0} items
                      </span>
                    </div>
                  </div>
                  <div className="text-right shrink-0">
                    <span className="md-typescale-label-large font-semibold tabular-nums">
                      {order.amount.toLocaleString()}
                    </span>
                  </div>
                </div>
              </button>
            );
          })}
        </div>

        {/* Right: Detail Panel */}
        <div className="hidden lg:flex flex-1 flex-col bento-card overflow-y-auto">
          {detail ? (
            <div className="p-6 flex-1">
              {/* Order Header */}
              <div className="flex items-start justify-between mb-6">
                <div>
                  <Chip size="sm" color={cfg.color} variant="soft" className="font-bold tracking-widest uppercase mb-3">
                    {cfg.label}
                  </Chip>
                  <h2 className="md-typescale-headline-small font-semibold text-foreground">
                    Order #{detail.order_id.slice(-8)}
                  </h2>
                  <div className="flex items-center gap-2 mt-1.5 md-typescale-body-small text-muted">
                    <span className="font-mono tabular-nums">{detail.order_id}</span>
                    <Copy size={13} className="cursor-pointer hover:text-foreground transition-colors" />
                    {detail.deliver_before && (
                      <>
                        <span className="opacity-40">·</span>
                        <span>Deliver by {new Date(detail.deliver_before).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}</span>
                      </>
                    )}
                  </div>
                </div>
                <Button isIconOnly variant="ghost" className="text-muted">
                  <MoreVertical size={18} />
                </Button>
              </div>

              {/* Detail Cards */}
              <div className="grid grid-cols-1 xl:grid-cols-2 gap-4 mb-8">
                <div className="bento-card">
                  <span className="md-kpi-label">Route</span>
                  <p className="md-typescale-title-small font-semibold flex items-center gap-2 mt-2 text-foreground">
                    <Truck size={16} style={{ color: 'var(--accent)' }} />
                    {detail.route_id ? detail.route_id.slice(-8) : "Unassigned"}
                  </p>
                  {detail.state === "IN_TRANSIT" && (
                    <p className="md-typescale-body-small text-muted mt-1">Currently en route</p>
                  )}
                </div>
                <div className="bento-card" style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}>
                  <span className="md-typescale-label-small uppercase tracking-widest opacity-80 font-semibold">Payment</span>
                  <p className="md-typescale-headline-small font-bold mt-1 tabular-nums">
                    {detail.amount.toLocaleString()}
                  </p>
                  <p className="md-typescale-body-small mt-1 opacity-80 flex items-center gap-1.5 font-medium">
                    <CheckCircle2 size={14} />
                    {detail.payment_gateway || "UNSPECIFIED"}
                  </p>
                </div>
              </div>

              {/* ── Delivery Progress Tracker ── */}
              {(() => {
                const stages = ["PENDING", "LOADED", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "COMPLETED"] as const;
                const stageLabels: Record<string, string> = {
                  PENDING: "Order Placed",
                  LOADED: "Approved",
                  DISPATCHED: "Dispatched",
                  IN_TRANSIT: "In Transit",
                  ARRIVED: "Driver Arrived",
                  COMPLETED: "Delivered",
                };
                const idx = stages.indexOf(detail.state as typeof stages[number]);
                const isCancelled = detail.state === "CANCELLED";
                return (
                  <div className="mb-8">
                    <h3 className="md-typescale-title-small font-semibold text-foreground mb-4">Delivery Progress</h3>
                    {isCancelled ? (
                      <div className="flex items-center gap-2 p-3 rounded-xl" style={{ background: "rgba(220,38,38,0.08)" }}>
                        <AlertTriangle size={16} style={{ color: "var(--danger)" }} />
                        <span className="md-typescale-body-medium font-semibold" style={{ color: "var(--danger)" }}>Order Cancelled</span>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1">
                        {stages.map((stage, i) => {
                          const isComplete = i <= idx;
                          const isCurrent = i === idx;
                          return (
                            <div key={stage} className="flex items-center gap-1 flex-1">
                              <div className="flex flex-col items-center gap-1.5 flex-1">
                                <div
                                  className="w-7 h-7 rounded-full flex items-center justify-center text-xs font-bold shrink-0 transition-colors"
                                  style={{
                                    background: isComplete ? "var(--accent)" : "var(--surface)",
                                    color: isComplete ? "var(--accent-foreground)" : "var(--muted)",
                                    boxShadow: isCurrent ? "0 0 0 3px var(--accent), 0 0 0 5px var(--background)" : "none",
                                  }}
                                >
                                  {isComplete && i < idx ? "✓" : i + 1}
                                </div>
                                <span
                                  className="md-typescale-label-small text-center font-medium"
                                  style={{ color: isCurrent ? "var(--accent)" : isComplete ? "var(--foreground)" : "var(--muted)" }}
                                >
                                  {stageLabels[stage]}
                                </span>
                              </div>
                              {i < stages.length - 1 && (
                                <div
                                  className="h-0.5 flex-1 rounded-full mt-[-18px]"
                                  style={{ background: i < idx ? "var(--accent)" : "var(--surface)" }}
                                />
                              )}
                            </div>
                          );
                        })}
                      </div>
                    )}
                    {(detail.state === "IN_TRANSIT" || detail.state === "DISPATCHED") && detail.route_id && (
                      <div className="mt-3 p-3 rounded-xl flex items-center gap-3" style={{ background: "var(--surface)" }}>
                        <Truck size={16} style={{ color: "var(--accent)" }} />
                        <span className="md-typescale-body-small text-foreground">
                          {detail.state === "DISPATCHED" ? "Truck loaded & sealed — awaiting departure" : `Driver is en route — Route #${detail.route_id.slice(-6)}`}
                        </span>
                        {detail.deliver_before && (
                          <span className="ml-auto md-typescale-label-small text-muted">
                            ETA: {new Date(detail.deliver_before).toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" })}
                          </span>
                        )}
                      </div>
                    )}
                  </div>
                );
              })()}

              {/* Manifest */}
              <div className="border-t border-[var(--border)] pt-6">
                <h3 className="md-typescale-title-small font-semibold text-foreground mb-4">
                  Manifest ({detail.items?.length ?? 0} items)
                </h3>
                <div className="space-y-3">
                  {(detail.items ?? []).map((item) => (
                    <div key={item.line_item_id} className="flex justify-between items-center p-3 rounded-xl border border-[var(--border)] hover:bg-surface transition-colors cursor-pointer">
                      <div className="flex gap-3 items-center">
                        <div className="w-10 h-10 rounded-lg flex items-center justify-center" style={{ background: 'var(--surface)' }}>
                          <PackageOpen size={16} style={{ color: 'var(--muted)' }} />
                        </div>
                        <div>
                          <p className="md-typescale-body-medium font-semibold text-foreground">
                            {item.sku_name || item.sku_id} x{item.quantity}
                          </p>
                          <p className="md-typescale-body-small text-muted">{item.status}</p>
                        </div>
                      </div>
                      <span className="md-typescale-label-large font-semibold tabular-nums">
                        {(item.unit_price * item.quantity).toLocaleString()}
                      </span>
                    </div>
                  ))}
                  {(!detail.items || detail.items.length === 0) && (
                    <p className="md-typescale-body-medium text-muted text-center py-4">No line items</p>
                  )}
                </div>
              </div>

              {/* Actions */}
              {detail.state !== "COMPLETED" && detail.state !== "CANCELLED" && (
                <div className="mt-8 pt-6 border-t border-[var(--border)] flex flex-col gap-3">
                  {actionError && (
                    <div className="p-3 rounded-xl flex items-center gap-2" style={{ background: 'rgba(220,38,38,0.08)' }}>
                      <AlertTriangle size={14} style={{ color: 'var(--danger)' }} />
                      <span className="md-typescale-body-small" style={{ color: 'var(--danger)' }}>{actionError}</span>
                    </div>
                  )}
                  <div className="flex gap-3">
                    {detail.state === "ARRIVED" && (
                      <Button
                        variant="primary"
                        onPress={() => verifyOrder(detail)}
                        isDisabled={verifying}
                        className="md-btn md-btn-filled flex-1 md-typescale-label-large h-11 flex items-center justify-center gap-2"
                      >
                        {verifying ? <Loader2 size={16} className="animate-spin" /> : <CheckCircle2 size={16} />}
                        Verify & Receipt
                      </Button>
                    )}
                    {detail.state === "PENDING" && !detail.route_id && (
                      <Button
                        variant="danger"
                        onPress={() => cancelOrder(detail)}
                        isDisabled={cancelling}
                        className="md-typescale-label-large h-11 flex items-center justify-center gap-2"
                      >
                        {cancelling ? <Loader2 size={16} className="animate-spin" /> : <XCircle size={16} />}
                        Cancel Order
                      </Button>
                    )}
                    {detail.state !== "PENDING" && detail.state !== "ARRIVED" && (
                      <Button variant="outline" className="md-typescale-label-large h-11 flex items-center justify-center gap-2">
                        <AlertTriangle size={16} /> Flag Issue
                      </Button>
                    )}
                  </div>
                </div>
              )}
            </div>
          ) : (
            <div className="flex-1 flex items-center justify-center">
              <p className="md-typescale-body-medium text-muted">Select an order to view details</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
