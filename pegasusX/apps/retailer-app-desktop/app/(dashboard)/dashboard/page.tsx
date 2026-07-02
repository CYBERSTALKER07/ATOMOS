"use client";

import { useCallback, useMemo } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import Link from "next/link";
import {
  ShoppingCart,
  PackageSearch,
  Inbox,
  Truck,
  Brain,
  Package,
  ArrowUpRight,
  Layers3,
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { motion } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import EmptyState from "../../../components/EmptyState";
import { PageSection } from "../../../components/PageSection";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Order, Prediction, Product } from "../../../lib/types";
import { isPredictionBlocked } from "../../../lib/types";

const EMPTY_ORDERS: Order[] = [];
const EMPTY_PREDICTIONS: Prediction[] = [];
const EMPTY_PRODUCTS: Product[] = [];

type LoadIssue = "restricted" | "offline" | "error";

export default function DashboardPage() {
  const getProfileId = () => {
    if (typeof localStorage === "undefined") return "";
    try {
      const parsed = JSON.parse(
        localStorage.getItem("retailer_profile") || "null",
      ) as { id?: string } | null;
      return parsed?.id ?? "";
    } catch {
      return "";
    }
  };

  const retailerID = getProfileId();
  const ordersPath = retailerID
    ? `/v1/retailers/${retailerID}/orders`
    : "/v1/orders";

  const {
    data: orders,
    loading: loadingOrders,
    error: ordersError,
    isRefreshing: isOrdersRefreshing,
    mutate: refreshOrders,
  } = useLiveData<Order[]>(ordersPath, 30000);
  const {
    data: predictions,
    loading: loadingPred,
    error: predictionsError,
    isRefreshing: isPredictionsRefreshing,
    mutate: refreshPredictions,
  } = useLiveData<Prediction[]>("/v1/ai/predictions");
  const {
    data: products,
    loading: loadingProducts,
    error: productsError,
    isRefreshing: isProductsRefreshing,
    mutate: refreshProducts,
  } = useLiveData<Product[]>("/v1/catalog/products");
  const ws = useOptionalWebSocket();
  const { addToCart, items } = useCart();

  const orderList = orders ?? EMPTY_ORDERS;
  const predictionList = predictions ?? EMPTY_PREDICTIONS;
  const productList = products ?? EMPTY_PRODUCTS;
  const cartQuantity = items.reduce((total, item) => total + item.quantity, 0);
  const isRefreshing =
    isOrdersRefreshing || isPredictionsRefreshing || isProductsRefreshing;

  const refreshAll = useCallback(() => {
    void refreshOrders();
    void refreshPredictions();
    void refreshProducts();
  }, [refreshOrders, refreshPredictions, refreshProducts]);

  const activeOrders = useMemo(
    () =>
      orderList.filter(
        (order) => order.state !== "COMPLETED" && order.state !== "CANCELLED",
      ),
    [orderList],
  );
  const completedOrders = useMemo(
    () => orderList.filter((order) => order.state === "COMPLETED"),
    [orderList],
  );
  const reorderProducts = useMemo(() => productList.slice(0, 8), [productList]);
  const blockedPredictionCount = useMemo(
    () => predictionList.filter((item) => isPredictionBlocked(item)).length,
    [predictionList],
  );

  const loading = loadingOrders || loadingPred || loadingProducts;

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const errors = [ordersError, predictionsError, productsError].filter(
      Boolean,
    ) as Array<Error & { status?: number }>;
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
  }, [ordersError, predictionsError, productsError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Dashboard access is partially restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing the latest cached operations data.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Operations sync degraded. Auto-retry is active.",
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
        message: "Syncing dashboard feeds...",
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loading, ws]);

  useRetailerSessionReconcile(() => {
    void refreshOrders();
    void refreshPredictions();
    void refreshProducts();
  });

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="dashboard"
        title="Operations Hub"
        description="Active deliveries, restock signals, and fleet telemetry."
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={isRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-10 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
              />
              {isRefreshing ? "Syncing" : "Sync"}
            </button>
            <Link href="/orders">
              <button type="button" className="portal-btn portal-btn--ghost h-10 px-5 rounded-xl font-light">
                Review Orders
              </button>
            </Link>
            <Link href="/catalog">
              <button type="button" className="portal-btn portal-btn--primary h-10 px-5 rounded-xl font-light shadow-[var(--shadow-sm)]">
                Open Catalog
              </button>
            </Link>
          </div>
        }
      >
          <motion.div layout initial={{ opacity: 0 }} animate={{ opacity: 1 }}>

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

            <BentoGrid className="mb-10">
              <BentoCard
                span={2}
                interactive={false}
                className="flex flex-col justify-between"
              >
                <div className="flex flex-wrap items-start justify-between gap-4">
                  <div>
                    <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                      Focus today
                    </span>
                    <h2 className="mt-2 text-2xl font-light tracking-tight text-[var(--desk-text-primary)] leading-tight">
                      Predictive Replenishment
                    </h2>
                    <p className="mt-2 md-typescale-body-medium text-[var(--desk-text-secondary)]">
                      {activeOrders.length} inbound nodes and{" "}
                      {predictionList.length} AI signals are shaping your next
                      run.
                    </p>
                  </div>
                  <div className="px-3 py-1 rounded-lg bg-[var(--desk-accent-soft)] text-[var(--desk-accent)] font-light text-xs">
                    {productList.length} SKUS
                  </div>
                </div>

                <div className="grid grid-cols-3 gap-3 mt-6">
                  <QuickAction
                    href="/catalog"
                    icon={PackageSearch}
                    label="Catalog"
                  />
                  <QuickAction
                    href="/orders"
                    icon={ShoppingCart}
                    label="Orders"
                  />
                  <QuickAction href="/insights" icon={Brain} label="Reorder suggestions" />
                </div>
              </BentoCard>

              <KpiCard
                label="Active Nodes"
                value={activeOrders.length}
                sub={`${completedOrders.length} archived`}
                icon={
                  <Truck size={18} style={{ color: "var(--desk-accent)" }} />
                }
              />
              <KpiCard
                label="AI Signals"
                value={predictionList.length}
                sub={
                  blockedPredictionCount > 0
                    ? `${blockedPredictionCount} blocked (sparse history)`
                    : "Restock Ready"
                }
                icon={<Brain size={18} style={{ color: "var(--desk-info)" }} />}
              />
              <KpiCard
                label="Staged Cart"
                value={cartQuantity}
                sub="Items in queue"
                icon={
                  <ShoppingCart
                    size={18}
                    style={{ color: "var(--desk-success)" }}
                  />
                }
              />
              <KpiCard
                label="Suppliers"
                value={new Set(productList.map((p) => p.supplier_id)).size}
                sub="Active partners"
                icon={
                  <Layers3 size={18} style={{ color: "var(--desk-warning)" }} />
                }
              />
            </BentoGrid>

            <div className="grid gap-8 xl:grid-cols-[1.2fr_0.8fr]">
              <PageSection
                title="Quick Reorder"
                description="Stage repeat purchases from your approved catalog."
              >
                {reorderProducts.length === 0 ? (
                  <EmptyState
                    headline="No products ready for reorder"
                    body="Catalog feeds are still populating reorder candidates."
                    variant="no-products"
                    action="Sync"
                    onAction={refreshProducts}
                  />
                ) : (
                  <div className="grid grid-cols-1 md:grid-cols-2 gap-4 !mt-0">
                    {reorderProducts.map((p) => (
                      <motion.button
                        key={p.id}
                        layout
                        whileHover={{ y: -2 }}
                        whileTap={{ scale: 0.98 }}
                        onClick={() => addToCart(p)}
                        className="flex items-center gap-4 p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl text-left hover:shadow-md transition-shadow group"
                      >
                        <div className="w-12 h-12 rounded-xl bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[var(--desk-text-tertiary)] group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                          <Package size={20} />
                        </div>
                        <div className="flex-1 min-w-0">
                          <p className="md-typescale-title-small font-light truncate text-[var(--desk-text-primary)]">
                            {p.name}
                          </p>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] truncate uppercase tracking-widest">
                            {p.supplier_name}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                            {p.price.toLocaleString()}
                          </p>
                          <ArrowUpRight
                            size={14}
                            className="ml-auto opacity-20 group-hover:opacity-100 transition-opacity"
                          />
                        </div>
                      </motion.button>
                    ))}
                  </div>
                )}
              </PageSection>

              <PageSection
                title="AI Restock"
                description="High-confidence replenishment signals for this cycle."
                actions={
                  <Link
                    href="/insights"
                    className="text-[var(--desk-accent)] md-typescale-label-small font-light uppercase tracking-widest hover:underline"
                  >
                    View All
                  </Link>
                }
              >
                <div className="flex flex-col gap-3 !mt-0">
                  {predictionList.length === 0 ? (
                    <EmptyState
                      headline="No AI restock signals"
                      body="Prediction feed is currently empty for this cycle."
                      variant="no-predictions"
                      action="Sync"
                      onAction={refreshPredictions}
                    />
                  ) : (
                    predictionList.slice(0, 5).map((forecast) => (
                      <div
                        key={forecast.id}
                        className="p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl flex items-center gap-4"
                      >
                        <div
                          className={`w-10 h-10 rounded-full border-2 flex items-center justify-center text-xs font-light ${forecast.confidence > 0.8 ? "border-[var(--desk-success)] text-[var(--desk-success)]" : "border-[var(--desk-warning)] text-[var(--desk-warning)]"}`}
                        >
                          {Math.round(forecast.confidence * 100)}%
                        </div>
                        <div className="flex-1 min-w-0">
                          <div className="flex items-center gap-2 min-w-0">
                            <p className="md-typescale-title-small font-light truncate text-[var(--desk-text-primary)]">
                              {forecast.productName}
                            </p>
                            {isPredictionBlocked(forecast) ? (
                              <span className="shrink-0 text-[9px] font-black tracking-tighter px-2 py-0.5 rounded bg-amber-100 border border-amber-200 text-amber-800">
                                INSUFFICIENT HISTORY
                              </span>
                            ) : null}
                          </div>
                          <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                            {forecast.reasoning}
                          </p>
                        </div>
                        <div className="text-right">
                          <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                            {forecast.predictedQuantity}
                          </p>
                          <p className="text-[10px] text-[var(--desk-text-tertiary)] uppercase font-light tracking-tighter">
                            Units
                          </p>
                        </div>
                      </div>
                    ))
                  )}
                </div>
              </PageSection>
            </div>
          </motion.div>
      </PageChrome>
    </div>
  );
}

function QuickAction({
  href,
  icon: Icon,
  label,
}: {
  href: string;
  icon: React.ElementType;
  label: string;
}) {
  useRetailerSessionReconcile(() => {
    void refreshOrders();
    void refreshPredictions();
    void refreshProducts();
  });

  return (
    <Link
      href={href}
      className="flex flex-col items-center gap-2 p-3 bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] rounded-xl hover:bg-[var(--desk-accent-soft)] hover:border-[var(--desk-accent)] hover:text-[var(--desk-accent)] transition-all active:scale-95 group"
    >
      <Icon
        size={20}
        strokeWidth={1.5}
        className="group-hover:scale-110 transition-transform"
      />
      <span className="md-typescale-label-small font-light uppercase tracking-widest">
        {label}
      </span>
    </Link>
  );
}

function KpiCard({
  label,
  value,
  sub,
  icon,
}: {
  label: string;
  value: number;
  sub: string;
  icon: React.ReactNode;
}) {
  useRetailerSessionReconcile(() => {
    void refreshOrders();
    void refreshPredictions();
    void refreshProducts();
  });

  return (
    <BentoCard interactive={false}>
      <div className="flex items-center justify-between mb-2">
        <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
          {label}
        </span>
        {icon}
      </div>
      <CountUp
        end={value}
        className="md-typescale-metric text-[var(--desk-text-primary)]"
      />
      <p className="md-typescale-body-small text-[var(--desk-text-secondary)] mt-1">
        {sub}
      </p>
    </BentoCard>
  );
}
