"use client";

import { useState, useMemo, useCallback, useEffect } from "react";
import { useRouter, useSearchParams, usePathname } from "next/navigation";
import { useWsEvent } from "../../../lib/ws";
import {
  Copy,
  Truck,
  CheckCircle2,
  PackageOpen,
  MoreVertical,
  Clock,
  Layers3,
  ArrowUpRight,
  RefreshCw,
  AlertTriangle,
  WifiOff,
  XCircle,
  Loader2,
} from "lucide-react";
import { Button, Chip } from "@heroui/react";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import EmptyState from "../../../components/EmptyState";
import { PageSection } from "../../../components/PageSection";
import { ListRowSkeleton } from "../../../components/Skeleton";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { confirmAiOrder, rejectAiOrder, confirmPreorder, editPreorder, acceptDeliveryProposal, rejectDeliveryProposal } from "../../../lib/api";
import {
  retailerCancelKey,
  retailerRequestCancelKey,
} from "@pegasusx/api-client";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Order, RetailerProfile, TrackingResponse } from "../../../lib/types";

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

type LoadIssue = "restricted" | "offline" | "error";

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
    error: ordersError,
    isRefreshing: isOrdersRefreshing,
    mutate: mutateOrders,
  } = useLiveData<Order[]>(ordersUrl, 30000);
  const [activeTab, setActiveTab] = useState<"ALL" | "ACTIVE" | "COMPLETED">(
    "ALL",
  );
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState(false);
  const [verifying, setVerifying] = useState(false);
  const [aiActionPending, setAiActionPending] = useState(false);
  const [preorderActionPending, setPreorderActionPending] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const ws = useOptionalWebSocket();
  const router = useRouter();
  const searchParams = useSearchParams();
  const pathname = usePathname();

  const refreshAll = useCallback(() => {
    void mutateOrders();
  }, [mutateOrders]);

  useWsEvent(
    "ORDER_COMPLETED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "ORDER_STATUS_CHANGED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "ORDER_REASSIGNED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "DRIVER_APPROACHING",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_DATE_PROPOSED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_DATE_ACCEPTED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_DATE_REJECTED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_CANCELLED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_NUDGE",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_CONFIRMATION",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_AUTO_ACCEPTED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_CONFIRMED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );
  useWsEvent(
    "PRE_ORDER_EDITED",
    useCallback(() => mutateOrders(), [mutateOrders]),
  );

  const { data: trackingData, mutate: mutateTracking } = useLiveData<TrackingResponse>(
    selectedId ? "/v1/retailer/tracking" : "",
    15000,
  );

  useEffect(() => {
    const epoch = ws?.reconnectEpoch ?? 0;
    if (epoch === 0) return;
    if (!aiActionPending && !cancelling && !verifying) return;
    setAiActionPending(false);
    setCancelling(false);
    setVerifying(false);
    setActionError("Connection restored — verify order status before retrying.");
    void mutateOrders();
    if (selectedId) void mutateTracking();
  }, [ws?.reconnectEpoch, aiActionPending, cancelling, verifying, mutateOrders, mutateTracking, selectedId]);

  const trackingDetail = useMemo(() => {
    if (!selectedId || !trackingData?.orders) return null;
    return trackingData.orders.find((order) => order.order_id === selectedId) ?? null;
  }, [selectedId, trackingData]);

  const handleConfirmAiOrder = useCallback(async (orderId: string) => {
    setAiActionPending(true);
    setActionError(null);
    try {
      const res = await confirmAiOrder(orderId);
      if (!res.ok) {
        throw new Error(`Confirm AI order failed with ${res.status}`);
      }
      await Promise.all([mutateOrders(), mutateTracking()]);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Confirm AI order failed");
    } finally {
      setAiActionPending(false);
    }
  }, [mutateOrders, mutateTracking]);

  const handleRejectAiOrder = useCallback(async (orderId: string) => {
    setAiActionPending(true);
    setActionError(null);
    try {
      const res = await rejectAiOrder(orderId, "Retailer rejected");
      if (!res.ok) {
        throw new Error(`Reject AI order failed with ${res.status}`);
      }
      await Promise.all([mutateOrders(), mutateTracking()]);
    } catch (err) {
      setActionError(err instanceof Error ? err.message : "Reject AI order failed");
    } finally {
      setAiActionPending(false);
    }
  }, [mutateOrders, mutateTracking]);

  const handleConfirmPreorder = useCallback(
    async (orderId: string) => {
      setPreorderActionPending(true);
      setActionError(null);
      try {
        const res = await confirmPreorder(orderId);
        if (!res.ok) {
          throw new Error(`Confirm preorder failed with ${res.status}`);
        }
        await Promise.all([mutateOrders(), mutateTracking()]);
      } catch (err) {
        setActionError(
          err instanceof Error ? err.message : "Confirm preorder failed",
        );
      } finally {
        setPreorderActionPending(false);
      }
    },
    [mutateOrders, mutateTracking],
  );

  const handleEditPreorder = useCallback(
    async (order: Order) => {
      setPreorderActionPending(true);
      setActionError(null);
      try {
        const deliveryDate =
          order.deliver_before ?? order.auto_confirm_at ?? "";
        const lineItems = (order.items ?? []).map((item) => ({
          sku: item.sku_id || item.line_item_id,
          name: item.sku_name || "Item",
          quantity: item.quantity,
          unit_price_minor: item.unit_price,
        }));
        const res = await editPreorder(order.order_id, deliveryDate, lineItems);
        if (!res.ok) {
          throw new Error(`Edit preorder failed with ${res.status}`);
        }
        await Promise.all([mutateOrders(), mutateTracking()]);
      } catch (err) {
        setActionError(
          err instanceof Error ? err.message : "Edit preorder failed",
        );
      } finally {
        setPreorderActionPending(false);
      }
    },
    [mutateOrders, mutateTracking],
  );

  const handleAcceptDeliveryProposal = useCallback(
    async (orderId: string) => {
      setPreorderActionPending(true);
      setActionError(null);
      try {
        const res = await acceptDeliveryProposal(orderId);
        if (!res.ok) {
          throw new Error(`Accept delivery proposal failed with ${res.status}`);
        }
        await Promise.all([mutateOrders(), mutateTracking()]);
      } catch (err) {
        setActionError(
          err instanceof Error ? err.message : "Accept delivery proposal failed",
        );
      } finally {
        setPreorderActionPending(false);
      }
    },
    [mutateOrders, mutateTracking],
  );

  const handleRejectDeliveryProposal = useCallback(
    async (orderId: string) => {
      setPreorderActionPending(true);
      setActionError(null);
      try {
        const res = await rejectDeliveryProposal(orderId, "Retailer declined proposed date");
        if (!res.ok) {
          throw new Error(`Reject delivery proposal failed with ${res.status}`);
        }
        await Promise.all([mutateOrders(), mutateTracking()]);
      } catch (err) {
        setActionError(
          err instanceof Error ? err.message : "Reject delivery proposal failed",
        );
      } finally {
        setPreorderActionPending(false);
      }
    },
    [mutateOrders, mutateTracking],
  );

  useEffect(() => {
    const pathMatch = pathname?.match(/\/orders\/([^/?]+)/);
    if (pathMatch?.[1]) {
      setSelectedId(pathMatch[1]);
      return;
    }
    const orderId = searchParams.get("order_id");
    if (orderId) {
      setSelectedId(orderId);
    }
  }, [pathname, searchParams]);

  const cancellableStates = useMemo(
    () =>
      new Set([
        "PENDING",
        "PENDING_REVIEW",
        "SCHEDULED",
        "AUTO_ACCEPTED",
        "LOADED",
      ]),
    [],
  );
  const requestCancelStates = useMemo(
    () => new Set(["DISPATCHED", "IN_TRANSIT", "ARRIVING", "ARRIVED"]),
    [],
  );

  const handleCancelOrder = useCallback(
    async (orderId: string, state: string) => {
      if (!profile?.id) {
        setActionError("Retailer profile not found. Please log in again.");
        return;
      }
      setCancelling(true);
      setActionError(null);
      try {
        const useRequestCancel = requestCancelStates.has(state);
        const endpoint = useRequestCancel
          ? "/v1/orders/request-cancel"
          : "/v1/order/cancel";
        const res = await apiFetch(endpoint, {
          method: "POST",
          headers: {
            "Idempotency-Key": useRequestCancel
              ? retailerRequestCancelKey(orderId)
              : retailerCancelKey(orderId),
            "Content-Type": "application/json",
          },
          body: JSON.stringify({
            order_id: orderId,
            retailer_id: profile.id,
            reason: "Retailer requested cancellation",
          }),
        });
        if (!res.ok) {
          const errBody = await res.json().catch(() => null);
          throw new Error(
            errBody?.error ||
              errBody?.message ||
              `Cancel failed with ${res.status}`,
          );
        }
        await Promise.all([mutateOrders(), mutateTracking()]);
      } catch (err) {
        setActionError(err instanceof Error ? err.message : "Cancel order failed");
      } finally {
        setCancelling(false);
      }
    },
    [mutateOrders, mutateTracking, profile?.id, requestCancelStates],
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
  const detail = useMemo(() => {
    if (!selected) return null;
    if (!trackingDetail) return selected;
    return {
      ...selected,
      state: trackingDetail.state,
      amount: trackingDetail.total_amount,
      items: trackingDetail.items.map((item) => ({
        line_item_id: item.product_id,
        order_id: trackingDetail.order_id,
        sku_id: item.product_id,
        sku_name: item.product_name,
        quantity: item.quantity,
        unit_price: item.unit_price,
        status: trackingDetail.state,
      })),
    };
  }, [selected, trackingDetail]);
  const showAiActions =
    detail?.state === "PENDING_REVIEW" ||
    (detail?.order_source === "AI_PREDICTED" && detail?.state === "PENDING");
  const showPreorderActions =
    detail?.order_source === "MANUAL_PREORDER" &&
    detail?.state === "SCHEDULED" &&
    (detail?.confirmation_status === "DRAFT" || !detail?.confirmation_status);
  const showDeliveryProposalReview =
    detail?.confirmation_status === "PENDING_WAREHOUSE" ||
    detail?.preorder_badge === "REVIEW_DELIVERY";
  const showCancelAction =
    !!detail &&
    (cancellableStates.has(detail.state) ||
      requestCancelStates.has(detail.state));

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const message = actionError ?? ordersError?.message;
    const status = (ordersError as (Error & { status?: number }) | null)?.status;
    if (!message && status == null) return null;
    if (status === 401 || status === 403 || /forbidden|restricted|access/i.test(message ?? "")) {
      return "restricted";
    }
    if (
      (typeof navigator !== "undefined" && !navigator.onLine) ||
      /failed to fetch|network|load failed|offline/i.test(message ?? "")
    ) {
      return "offline";
    }
    return "error";
  }, [actionError, ordersError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Orders access is partially restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing latest cached order data.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Order sync degraded. Auto-retry is active.",
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Live socket reconnecting. Event updates may be delayed.",
      };
    }
    if (isOrdersRefreshing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing order feeds...",
      };
    }
    return null;
  }, [isOrdersRefreshing, loadIssue, loading, ws]);

  const listEmptyState = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: "Orders access restricted",
        body: "Your account currently cannot load logistics orders.",
        variant: "restricted" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: "Orders are offline",
        body: "Reconnect your network and retry to refresh order status.",
        variant: "offline" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: "Orders unavailable",
        body: "Order feeds could not be loaded right now.",
        variant: "error" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (list.length === 0) {
      return {
        headline: "No orders yet",
        body: "New and active logistics orders will appear here.",
        variant: "no-orders" as const,
        action: "Refresh",
        onAction: refreshAll,
      };
    }
    return {
      headline: `No ${activeTab.toLowerCase()} orders found`,
      body: "Try switching tabs to inspect other workflow states.",
      variant: "no-results" as const,
      action: "Show All",
      onAction: () => setActiveTab("ALL"),
    };
  }, [activeTab, list.length, loadIssue, refreshAll]);

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
          variant="primary"
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
          variant="tertiary"
          size="sm"
          isIconOnly
          isDisabled={isOrdersRefreshing}
          onPress={refreshAll}
          className="text-[var(--desk-text-tertiary)]"
        >
          <RefreshCw size={16} className={isOrdersRefreshing ? "animate-spin" : ""} />
        </Button>
      </div>

      {syncBanner && (
        <motion.div
          initial={{ opacity: 0, y: -6 }}
          animate={{ opacity: 1, y: 0 }}
          className={`mb-6 flex items-center justify-between gap-3 rounded-2xl border px-4 py-3 ${
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
            <span className="md-typescale-body-small font-bold uppercase tracking-wide">
              {syncBanner.message}
            </span>
          </div>
          {syncBanner.kind !== "refreshing" && (
            <button
              onClick={refreshAll}
              className="rounded-lg border border-current/30 px-3 py-1 text-[11px] font-bold uppercase tracking-wide hover:bg-current/10"
            >
              Retry
            </button>
          )}
        </motion.div>
      )}

      <div className="flex gap-8 min-h-[520px]">
        <PageSection
          title="Order queue"
          description={`${filtered.length} orders in ${activeTab.toLowerCase()} view.`}
          className="w-[440px] shrink-0 !overflow-visible"
        >
          <div
            className="flex flex-col gap-2 overflow-y-auto pr-2 !mt-0 !p-0"
            style={{ maxHeight: "calc(100vh - 440px)" }}
          >
          <AnimatePresence mode="popLayout">
            {loading ? (
              <div className="flex flex-col gap-2">
                <ListRowSkeleton count={4} />
              </div>
            ) : filtered.length === 0 ? (
              <EmptyState
                headline={listEmptyState.headline}
                body={listEmptyState.body}
                variant={listEmptyState.variant}
                action={listEmptyState.action}
                onAction={listEmptyState.onAction}
              />
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
                      {(order.preorder_badge === "REVIEW_DELIVERY" ||
                        order.confirmation_status === "PENDING_WAREHOUSE") && (
                        <Chip size="sm" color="warning" variant="flat" className="mt-1">
                          Review Delivery
                        </Chip>
                      )}
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
        </PageSection>

        <PageSection
          title="Order details"
          description={detail ? `Manifest and actions for #${detail.order_id.slice(-8)}.` : "Select an order from the queue."}
          className="flex-1 min-w-0"
        >
        <div className="flex flex-col overflow-hidden !mt-0 !p-0 min-h-[480px]">
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
                  variant="tertiary"
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

              {showAiActions && (
                <div className="mb-10 flex flex-wrap gap-3">
                  <Button
                    variant="secondary"
                    isDisabled={aiActionPending}
                    onPress={() => void handleRejectAiOrder(detail.order_id)}
                    className="h-11 px-5 rounded-xl font-bold"
                  >
                    Reject Suggestion
                  </Button>
                  <Button
                    variant="primary"
                    isDisabled={aiActionPending}
                    onPress={() => void handleConfirmAiOrder(detail.order_id)}
                    className="h-11 px-5 rounded-xl font-bold"
                    style={{ background: "var(--desk-accent)", color: "white" }}
                  >
                    {aiActionPending ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      "Confirm Suggestion"
                    )}
                  </Button>
                </div>
              )}

              {showDeliveryProposalReview && (
                <div className="mb-10 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-5 space-y-4">
                  <div className="flex items-center gap-2">
                    <Chip color="warning" variant="soft">Review Delivery</Chip>
                    <span className="md-typescale-label-medium text-[var(--desk-text-secondary)]">
                      Warehouse proposed a new delivery date
                    </span>
                  </div>
                  {detail?.proposed_delivery_date && (
                    <p className="md-typescale-body-medium">
                      Proposed: {detail.proposed_delivery_date}
                    </p>
                  )}
                  {detail?.delivery_proposal_reason && (
                    <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
                      {detail.delivery_proposal_reason}
                    </p>
                  )}
                  <div className="flex flex-wrap gap-3">
                    <Button
                      variant="secondary"
                      isDisabled={preorderActionPending}
                      onPress={() => detail && void handleRejectDeliveryProposal(detail.order_id)}
                      className="h-11 px-5 rounded-xl font-bold"
                    >
                      Reject Proposal
                    </Button>
                    <Button
                      variant="primary"
                      isDisabled={preorderActionPending}
                      onPress={() => detail && void handleAcceptDeliveryProposal(detail.order_id)}
                      className="h-11 px-5 rounded-xl font-bold"
                      style={{ background: "var(--desk-accent)", color: "white" }}
                    >
                      {preorderActionPending ? (
                        <Loader2 size={16} className="animate-spin" />
                      ) : (
                        "Accept Date"
                      )}
                    </Button>
                  </div>
                </div>
              )}

              {showPreorderActions && (
                <div className="mb-10 flex flex-wrap gap-3">
                  <Button
                    variant="secondary"
                    isDisabled={preorderActionPending}
                    onPress={() => detail && void handleEditPreorder(detail)}
                    className="h-11 px-5 rounded-xl font-bold"
                  >
                    Edit Preorder
                  </Button>
                  <Button
                    variant="primary"
                    isDisabled={preorderActionPending}
                    onPress={() =>
                      detail && void handleConfirmPreorder(detail.order_id)
                    }
                    className="h-11 px-5 rounded-xl font-bold"
                    style={{ background: "var(--desk-accent)", color: "white" }}
                  >
                    {preorderActionPending ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      "Confirm Preorder"
                    )}
                  </Button>
                </div>
              )}

              {showCancelAction && (
                <div className="mb-10">
                  <Button
                    variant="secondary"
                    isDisabled={cancelling}
                    onPress={() => void handleCancelOrder(detail.order_id, detail.state)}
                    className="h-11 px-5 rounded-xl font-bold text-red-700 border border-red-200"
                  >
                    {cancelling ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : requestCancelStates.has(detail.state) ? (
                      <>
                        <XCircle size={16} className="mr-2" />
                        Request cancellation
                      </>
                    ) : (
                      <>
                        <XCircle size={16} className="mr-2" />
                        Cancel order
                      </>
                    )}
                  </Button>
                </div>
              )}

              {actionError && (
                <div className="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-semibold text-red-700">
                  {actionError}
                </div>
              )}

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
            <div className="flex-1 flex flex-col items-center justify-center py-16">
              <EmptyState
                headline="Select an order"
                body="Choose a node from the queue to inspect manifest lines and actions."
                variant="no-orders"
              />
            </div>
          )}
        </div>
        </PageSection>
      </div>
    </div>
  );
}
