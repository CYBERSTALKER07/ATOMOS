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
import { CommandBoard } from "../../../components/dashboard/CommandBoard";
import { KpiGrid } from "../../../components/dashboard/KpiGrid";
import { QuickReorderSection } from "../../../components/dashboard/QuickReorderSection";
import { AiPredictionSection } from "../../../components/dashboard/AiPredictionSection";
import { useLiveData } from "../../../lib/hooks";
import { useCart } from "../../../lib/cart";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Product, RetailerAIPredictionsResponse } from "../../../lib/types";
import type { RetailerControlTowerPulse } from "@pegasusx/types";
import { usePortalT } from "@/lib/i18n";

const EMPTY_PRODUCTS: Product[] = [];

type LoadIssue = "restricted" | "offline" | "error";

export default function DashboardPage() {
  const t = usePortalT();
  const {
    data: pulse,
    loading: loadingPulse,
    error: pulseError,
    isRefreshing: isPulseRefreshing,
    mutate: refreshPulse,
  } = useLiveData<RetailerControlTowerPulse>("/v1/retailer/control-tower/pulse", 60000);
  const {
    data: predictions,
    loading: loadingPred,
    error: predictionsError,
    isRefreshing: isPredictionsRefreshing,
    mutate: refreshPredictions,
  } = useLiveData<RetailerAIPredictionsResponse>("/v1/retailer/ai/predictions");
  const {
    data: products,
    loading: loadingProducts,
    error: productsError,
    isRefreshing: isProductsRefreshing,
    mutate: refreshProducts,
  } = useLiveData<Product[]>("/v1/catalog/products");
  const ws = useOptionalWebSocket();
  const { addToCart, items } = useCart();

  const predictionList = predictions?.items ?? [];
  const productList = products ?? EMPTY_PRODUCTS;
  const cartQuantity = items.reduce((total, item) => total + item.quantity, 0);
  const isRefreshing =
    isPulseRefreshing || isPredictionsRefreshing || isProductsRefreshing;

  const refreshAll = useCallback(() => {
    void refreshPulse();
    void refreshPredictions();
    void refreshProducts();
  }, [refreshPulse, refreshPredictions, refreshProducts]);

  const reorderProducts = useMemo(() => productList.slice(0, 8), [productList]);
  const loading = loadingPulse || loadingPred || loadingProducts;

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const errors = [pulseError, predictionsError, productsError].filter(
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
  }, [pulseError, predictionsError, productsError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.dashboard_access_is_partially_restricted_for_this_account"),
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: t("retailer_desktop.residual.text.offline_mode_active_showing_the_latest_cached_operations_data"),
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.operations_sync_degraded_auto_retry_is_active"),
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.live_socket_reconnecting_event_updates_may_be_delayed"),
      };
    }
    if (isRefreshing && !loading) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: t("retailer_desktop.residual.text.syncing_dashboard_feeds"),
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loading, ws]);

  useRetailerSessionReconcile(() => {
    void refreshPulse();
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
        title={t("portal.page.dashboard.retailer.title")}
        description={t("portal.page.dashboard.retailer.description")}
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
              {isRefreshing
                ? t("portal.page.dashboard.action.syncing")
                : t("portal.page.dashboard.action.sync")}
            </button>
            <Link href="/orders">
              <button type="button" className="portal-btn portal-btn--ghost h-10 px-5 rounded-xl font-light">
                {t("portal.page.dashboard.action.review_orders")}
              </button>
            </Link>
            <Link href="/catalog">
              <button type="button" className="portal-btn portal-btn--primary h-10 px-5 rounded-xl font-light shadow-[var(--shadow-sm)]">
                {t("portal.page.dashboard.action.open_catalog")}
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

            <CommandBoard
              pulse={pulse}
              loading={loadingPulse}
              error={pulseError ? "control_tower_pulse_failed" : null}
            />

            {pulseError || (loadingPulse && !pulse) ? null : (
              <KpiGrid
                activeOrdersLength={pulse?.open_orders ?? 0}
                predictionListLength={predictionList.length}
                productListLength={productList.length}
                cartQuantity={cartQuantity}
                completedOrdersLength={pulse?.orders_by_status?.COMPLETED ?? 0}
                blockedPredictionCount={0}
                uniqueSuppliersCount={pulse?.orders_by_supplier?.length ?? 0}
              />
            )}

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
