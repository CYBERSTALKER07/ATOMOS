"use client";

import { useState, useMemo, useCallback, useEffect, Suspense } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
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
import { Chip } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import { OrderFilters } from "../../../components/orders/OrderFilters";
import { OrderList } from "../../../components/orders/OrderList";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import EmptyState from "../../../components/EmptyState";
import { PageSection } from "../../../components/PageSection";
import { VirtualScrollList } from "@pegasusx/ui-kit/desktop";
import { ListRowSkeleton } from "../../../components/Skeleton";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { OrderTimelinePanel } from "../../../components/OrderTimelinePanel";
import { FileClaimPanel } from "../../../components/FileClaimPanel";
import {
  confirmAiOrder,
  rejectAiOrder,
  confirmPreorder,
  editPreorder,
  acceptDeliveryProposal,
  rejectDeliveryProposal,
  getClaimEligibility,
  type ClaimEligibility,
} from "../../../lib/api";
import {
  retailerCancelKey,
  retailerRequestCancelKey,
} from "@pegasusx/api-client";
import { useOptionalWebSocket } from "../../../lib/ws";
import { getRetailerProfile } from "@/lib/retailer-profile";
import type { Order, TrackingResponse } from "../../../lib/types";
import { usePortalT } from "@/lib/i18n";

const chipCfg: Record<
  string,
  { color: "warning" | "success" | "default" | "danger"; label: string }
> = {
  IN_TRANSIT: { color: "warning", label: "In Transit" },
  COMPLETED: { color: "success", label: "Completed" },
  FISCALIZING: { color: "warning", label: "Pending fiscal" },
  FISCAL_FAILED: { color: "danger", label: "Fiscal failed" },
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

function OrdersPageContent() {
  const t = usePortalT();
  const profile = getRetailerProfile();
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
  const [claimElig, setClaimElig] = useState<ClaimEligibility | null>(null);
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
      setActionError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.confirm_ai_order_failed"));
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
      setActionError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.reject_ai_order_failed"));
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
          err instanceof Error ? err.message : t("retailer_desktop.residual.text.confirm_preorder_failed"),
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
          err instanceof Error ? err.message : t("retailer_desktop.residual.text.edit_preorder_failed"),
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
          err instanceof Error ? err.message : t("retailer_desktop.residual.text.accept_delivery_proposal_failed"),
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
          err instanceof Error ? err.message : t("retailer_desktop.residual.text.reject_delivery_proposal_failed"),
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
        setActionError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.cancel_order_failed"));
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
  const claimCandidate =
    !!detail &&
    (detail.state === "COMPLETED" || detail.state === "DELIVERED_ON_CREDIT");
  const showFileClaim = claimCandidate && claimElig?.eligible === true;
  const showClaimWindowClosed =
    claimCandidate && claimElig != null && !claimElig.eligible;

  useEffect(() => {
    if (!detail?.order_id || !claimCandidate) {
      setClaimElig(null);
      return;
    }
    let cancelled = false;
    void getClaimEligibility(detail.order_id)
      .then((e) => {
        if (!cancelled) setClaimElig(e);
      })
      .catch(() => {
        // Fallback: show panel; FileClaimPanel re-fetches / server enforces.
        if (!cancelled) {
          setClaimElig({
            eligible: true,
            ends_at: null,
            window_hours: 48,
            hours_remaining: 48,
            policy_source: "DEFAULT",
          });
        }
      });
    return () => {
      cancelled = true;
    };
  }, [detail?.order_id, claimCandidate]);

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
        message: t("retailer_desktop.residual.text.orders_access_is_partially_restricted_for_this_account"),
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: t("retailer_desktop.residual.text.offline_mode_active_showing_latest_cached_order_data"),
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.order_sync_degraded_auto_retry_is_active"),
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.live_socket_reconnecting_event_updates_may_be_delayed"),
      };
    }
    if (isOrdersRefreshing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: t("retailer_desktop.residual.text.syncing_order_feeds"),
      };
    }
    return null;
  }, [isOrdersRefreshing, loadIssue, loading, ws]);

  const listEmptyState = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: t("retailer_desktop.residual.text.orders_access_restricted"),
        body: "Your account currently cannot load logistics orders.",
        variant: "restricted" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: t("retailer_desktop.residual.text.orders_are_offline"),
        body: "Reconnect your network and retry to refresh order status.",
        variant: "offline" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: t("retailer_desktop.residual.text.orders_unavailable"),
        body: "Order feeds could not be loaded right now.",
        variant: "error" as const,
        action: "Retry",
        onAction: refreshAll,
      };
    }
    if (list.length === 0) {
      return {
        headline: t("retailer_desktop.residual.text.no_orders_yet"),
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

  useRetailerSessionReconcile(() => {
    void mutateOrders();
    void mutateTracking();
  });

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="orders"
        title={t("portal.page.orders.retailer.title")}
        description={t("portal.page.orders.retailer.description")}
        loading={loading}
        skeletonVariant="table"
        actions={
          <button
            type="button"
            onClick={() => router.push("/catalog")}
            className="portal-btn portal-btn--primary h-11 px-6 rounded-xl font-light shadow-[var(--shadow-sm)]"
          >
            <PackageOpen size={18} className="mr-2" /> {t("portal.page.orders.action.new_order")}
          </button>
        }
      >

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
                {t("portal.page.orders.filter.completed")}
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

      <OrderFilters
        activeTab={activeTab}
        setActiveTab={setActiveTab}
        isOrdersRefreshing={isOrdersRefreshing}
        refreshAll={refreshAll}
      />

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

      <div className="flex gap-8 min-h-[520px]">
        <PageSection
          title={t("portal.page.orders.section.order_queue")}
          description={`${filtered.length} orders in ${activeTab.toLowerCase()} view.`}
          className="w-[440px] shrink-0 !overflow-visible"
        >
          <div
            className="flex flex-col gap-2 overflow-y-auto pr-2 !mt-0 !p-0"
            style={{ maxHeight: "calc(100vh - 440px)" }}
          >
          <OrderList
            loading={loading}
            filtered={filtered}
            listEmptyState={listEmptyState}
            selectedId={selectedId}
            setSelectedId={setSelectedId}
            chipCfg={chipCfg}
            list={list}
          />
          </div>
        </PageSection>

        <PageSection
          title={t("retailer_desktop.orders.text.order_details")}
          description={detail ? `Manifest and actions for #${detail.order_id.slice(-8)}.` : "Select an order from the queue."}
          className="flex-1 min-w-0"
        >
        <div className="flex flex-col overflow-hidden !mt-0 !p-0 min-h-[480px]">
          {detail ? (
            <div className="p-8 flex-1 overflow-y-auto">
              <div className="flex items-start justify-between mb-8">
                <div>
                  <div className="flex items-center gap-3 mb-2 text-[var(--desk-text-tertiary)]">
                    <span className="md-typescale-label-small font-light uppercase tracking-[0.2em]">
                      {detail.state.replace("_", " ")}
                    </span>
                    <span className="w-1.5 h-1.5 rounded-full bg-[var(--desk-border-strong)]" />
                    <span className="md-typescale-label-small font-mono">
                      {detail.order_id}
                    </span>
                  </div>
                  <h2 className="md-typescale-display-small font-light text-[var(--desk-text-primary)]">
                    Order Details
                  </h2>
                </div>
                <button
                  type="button"
                  className="portal-btn portal-btn--ghost desk-icon-btn text-[var(--desk-text-tertiary)]"
                  aria-label={t("retailer_desktop.orders.text.more_options")}
                >
                  <MoreVertical size={20} />
                </button>
              </div>

              <div className="grid grid-cols-2 gap-4 mb-10">
                <div className="p-5 rounded-2xl bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]">
                  <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2 block">
                    Assigned Route
                  </span>
                  <div className="flex items-center gap-3">
                    <Truck size={20} className="text-[var(--desk-accent)]" />
                    <span className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
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
                    <span className="md-typescale-display-small font-light tabular-nums">
                      {detail.amount.toLocaleString()}
                    </span>
                    <CheckCircle2 size={24} className="opacity-40" />
                  </div>
                </div>
              </div>

              {showAiActions && (
                <div className="mb-10 flex flex-wrap gap-3">
                  <button
                    type="button"
                    disabled={aiActionPending}
                    onClick={() => void handleRejectAiOrder(detail.order_id)}
                    className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
                  >
                    Reject Suggestion
                  </button>
                  <button
                    type="button"
                    disabled={aiActionPending}
                    onClick={() => void handleConfirmAiOrder(detail.order_id)}
                    className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
                  >
                    {aiActionPending ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      "Confirm Suggestion"
                    )}
                  </button>
                </div>
              )}

              {showDeliveryProposalReview && (
                <div className="mb-10 rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-5 space-y-4">
                  <div className="flex items-center gap-2">
                    <Chip color="warning" variant="soft">{t("retailer_desktop.orders.text.review_delivery")}</Chip>
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
                    <button
                      type="button"
                      disabled={preorderActionPending}
                      onClick={() => detail && void handleRejectDeliveryProposal(detail.order_id)}
                      className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
                    >
                      Reject Proposal
                    </button>
                    <button
                      type="button"
                      disabled={preorderActionPending}
                      onClick={() => detail && void handleAcceptDeliveryProposal(detail.order_id)}
                      className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
                    >
                      {preorderActionPending ? (
                        <Loader2 size={16} className="animate-spin" />
                      ) : (
                        "Accept Date"
                      )}
                    </button>
                  </div>
                </div>
              )}

              {showPreorderActions && (
                <div className="mb-10 flex flex-wrap gap-3">
                  <button
                    type="button"
                    disabled={preorderActionPending}
                    onClick={() => detail && void handleEditPreorder(detail)}
                    className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
                  >
                    Edit Preorder
                  </button>
                  <button
                    type="button"
                    disabled={preorderActionPending}
                    onClick={() =>
                      detail && void handleConfirmPreorder(detail.order_id)
                    }
                    className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light"
                  >
                    {preorderActionPending ? (
                      <Loader2 size={16} className="animate-spin" />
                    ) : (
                      "Confirm Preorder"
                    )}
                  </button>
                </div>
              )}

              {showCancelAction && (
                <div className="mb-10">
                  <button
                    type="button"
                    disabled={cancelling}
                    onClick={() => void handleCancelOrder(detail.order_id, detail.state)}
                    className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light text-red-700 border border-red-200"
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
                  </button>
                </div>
              )}

              {showClaimWindowClosed && (
                <div className="mb-10 rounded-xl border border-border px-4 py-3 text-sm text-muted-foreground">
                  Window closed — claim filing is no longer available for this
                  order
                  {claimElig?.ends_at
                    ? ` (ended ${new Date(claimElig.ends_at).toLocaleString()})`
                    : ""}
                  .
                </div>
              )}
              {showFileClaim && (
                <div className="mb-10">
                  <FileClaimPanel order={detail} />
                </div>
              )}

              {actionError && (
                <div className="mb-6 rounded-xl border border-red-200 bg-red-50 px-4 py-3 text-sm font-semibold text-red-700">
                  {actionError}
                </div>
              )}

              <div className="mb-10">
                <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-4">
                  Status history
                </h3>
                <OrderTimelinePanel orderId={detail.order_id} />
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
                          <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)]">
                            {item.sku_name || "Generic SKU"}
                          </p>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)]">
                            QTY: {item.quantity}
                          </p>
                        </div>
                      </div>
                      <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
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
                headline={t("retailer_desktop.residual.text.select_an_order")}
                body={t("retailer_desktop.residual.text.choose_a_node_from_the_queue_to_inspect_manifest_lines_and_actio")}
                variant="no-orders"
              />
            </div>
          )}
        </div>
        </PageSection>
      </div>
      </PageChrome>
    </div>
  );
}

export default function OrdersPage() {
  const t = usePortalT();
  return (
    <Suspense fallback={<div className="p-8">{t("retailer_desktop.orders.text.loading_orders")}</div>}>
      <OrdersPageContent />
    </Suspense>
  );
}
