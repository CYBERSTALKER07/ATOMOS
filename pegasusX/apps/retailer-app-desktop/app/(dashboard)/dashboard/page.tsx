"use client";

import { useCallback, useMemo } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import Link from "next/link";
import {
  RefreshCw,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { CreditProfileCard } from "@/components/CreditProfileCard";
import { motion } from "framer-motion";
import { KpiGrid } from "../../../components/dashboard/KpiGrid";
import { QuickReorderSection } from "../../../components/dashboard/QuickReorderSection";
import { AiPredictionSection } from "../../../components/dashboard/AiPredictionSection";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import { useOptionalWebSocket } from "../../../lib/ws";
import { getRetailerId } from "@/lib/retailer-profile";
import type { Order, Prediction, Product } from "../../../lib/types";
import { isPredictionBlocked } from "../../../lib/types";

const EMPTY_ORDERS: Order[] = [];
const EMPTY_PREDICTIONS: Prediction[] = [];
const EMPTY_PRODUCTS: Product[] = [];

type LoadIssue = "restricted" | "offline" | "error";

export default function DashboardPage() {
  const retailerID = getRetailerId();
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

            <div className="mb-6 max-w-xl">
              <CreditProfileCard />
            </div>

            <KpiGrid
              activeOrdersLength={activeOrders.length}
              predictionListLength={predictionList.length}
              productListLength={productList.length}
              cartQuantity={cartQuantity}
              completedOrdersLength={completedOrders.length}
              blockedPredictionCount={blockedPredictionCount}
              uniqueSuppliersCount={new Set(productList.map((p) => p.supplier_id)).size}
            />

            <div className="grid gap-8 xl:grid-cols-[1.2fr_0.8fr]">
              <QuickReorderSection
                reorderProducts={reorderProducts}
                onRefresh={refreshProducts}
                onAddToCart={addToCart}
              />

              <AiPredictionSection
                predictionList={predictionList}
                onRefresh={refreshPredictions}
              />
            </div>
          </motion.div>
      </PageChrome>
    </div>
  );
}
