"use client";

import { usePortalT } from "@/lib/i18n";
import { useState, useMemo, useCallback } from "react";
import { useRetailerSessionReconcile } from "../../../lib/use-retailer-session-reconcile";
import {
  RefreshCw,
  Package,
  AlertTriangle,
  WifiOff,
} from "lucide-react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import EmptyState from "../../../components/EmptyState";
import SpendAnalytics from "./SpendAnalytics";
import { InsightsSummary } from "../../../components/insights/InsightsSummary";
import { InsightsSidebar } from "../../../components/insights/InsightsSidebar";
import { SellThroughPanel } from "../../../components/insights/SellThroughPanel";
import { useLiveData } from "../../../lib/hooks";
import { confirmAiOrder, rejectAiOrder } from "../../../lib/api";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { RetailerAIPrediction, RetailerAIPredictionsResponse, RetailerAnalytics } from "../../../lib/types";
import {
  aiPredictionQty,
  aiPredictionTitle,
  formatMinorAmount,
} from "../../../lib/types";

type LoadIssue = "restricted" | "offline" | "error";

type DetailedAnalytics = {
  series?: Array<{
    order_id: string;
    total_minor: number;
    currency?: string;
    created_at: string;
  }>;
  by_category?: unknown[];
  from?: string;
  to?: string;
};

const EMPTY_AI_ITEMS: RetailerAIPrediction[] = [];

export default function InsightsPage() {
  const t = usePortalT();
  const {
    data: predictions,
    loading: loadingPred,
    error: predictionsError,
    isRefreshing: isPredictionsRefreshing,
    mutate: refreshPred,
  } = useLiveData<RetailerAIPredictionsResponse>("/v1/retailer/ai/predictions");
  const {
    data: analytics,
    loading: loadingAnalytics,
    error: analyticsError,
    isRefreshing: isAnalyticsRefreshing,
    mutate: refreshAnalytics,
  } = useLiveData<RetailerAnalytics>("/v1/retailer/analytics/expenses");
  const {
    data: detailedAnalytics,
    loading: loadingDetailed,
    error: detailedError,
    mutate: refreshDetailed,
  } = useLiveData<DetailedAnalytics>("/v1/retailer/analytics/detailed");
  const ws = useOptionalWebSocket();

  const [actingId, setActingId] = useState<string | null>(null);

  const predList = predictions?.items ?? EMPTY_AI_ITEMS;
  const topProducts = analytics?.top_products ?? [];
  const totalThisMonth = analytics?.total_this_month ?? 0;
  const monthlyExpenses = analytics?.monthly_expenses ?? [];
  const detailedSeries = detailedAnalytics?.series ?? [];
  const isRefreshing = isPredictionsRefreshing || isAnalyticsRefreshing || loadingDetailed;

  const refreshAll = useCallback(() => {
    void refreshPred();
    void refreshAnalytics();
    void refreshDetailed();
  }, [refreshAnalytics, refreshDetailed, refreshPred]);

  const loadIssue = useMemo<LoadIssue | null>(() => {
    const errors = [predictionsError, analyticsError, detailedError].filter(Boolean) as Array<
      Error & { status?: number }
    >;
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
  }, [analyticsError, detailedError, predictionsError]);

  const syncBanner = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.insights_access_is_partially_restricted_for_this_account"),
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: t("retailer_desktop.residual.text.offline_mode_active_showing_latest_cached_intelligence_signals"),
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.insights_sync_degraded_auto_retry_is_active"),
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: t("retailer_desktop.residual.text.live_socket_reconnecting_new_signals_may_be_delayed"),
      };
    }
    if (isRefreshing && !loadingPred && !loadingAnalytics) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: t("retailer_desktop.residual.text.syncing_intelligence_feeds"),
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loadingAnalytics, loadingPred, ws]);

  const aiEmptyState = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: t("retailer_desktop.residual.text.insights_access_restricted"),
        body: "Your account cannot load replenishment signals right now.",
        variant: "restricted" as const,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: t("retailer_desktop.residual.text.insights_are_offline"),
        body: "Reconnect to refresh prediction signals and analytics.",
        variant: "offline" as const,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: t("retailer_desktop.residual.text.signals_unavailable"),
        body: "Prediction feeds could not be loaded right now.",
        variant: "error" as const,
      };
    }
    return {
      headline: t("retailer_desktop.residual.text.no_actionable_signals_detected"),
      body: "AI has no urgent replenishment recommendations for this cycle.",
      variant: "no-predictions" as const,
    };
  }, [loadIssue]);

  const runAiAction = useCallback(
    async (orderId: string, action: "confirm" | "reject") => {
      setActingId(orderId);
      try {
        const res =
          action === "confirm"
            ? await confirmAiOrder(orderId)
            : await rejectAiOrder(orderId, "Retailer rejected");
        if (!res.ok) {
          throw new Error(`ai_${action}_failed_${res.status}`);
        }
        await refreshPred();
      } finally {
        setActingId(null);
      }
    },
    [refreshPred],
  );

  const sparkRevenue = useMemo(() => {
    if (monthlyExpenses.length > 0) return monthlyExpenses.map((m) => m.total);
    return Array.from(
      { length: 14 },
      (_, i) => 800 + i * 45 + Math.sin(i * 0.9) * 120,
    );
  }, [monthlyExpenses]);

  const sparkOrders = useMemo(
    () =>
      Array.from({ length: 14 }, (_, i) => 12 + i * 2 + Math.cos(i * 0.7) * 4),
    [],
  );

  const loading = loadingPred || loadingAnalytics;

  useRetailerSessionReconcile(() => {
    void refreshPred();
    void refreshAnalytics();
    void refreshDetailed();
  });

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="insights"
        title={t("retailer_desktop.insights.text.intelligence_hub")}
        description={t("retailer_desktop.residual.text.predictive_demand_signals_and_network_analytics")}
        loading={loading}
        skeletonVariant="table"
        actions={
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
            {isRefreshing ? "Syncing" : "Sync Signals"}
          </button>
        }
      >

      <SellThroughPanel />

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

      <InsightsSummary
        totalThisMonth={totalThisMonth}
        sparkRevenue={sparkRevenue}
        predList={predList}
        sparkOrders={sparkOrders}
        topProducts={topProducts}
      />

      <div className="mb-8">
        <SpendAnalytics
          spendTrend={detailedSeries.map((row) => ({
            day: row.created_at?.slice(0, 10) || row.order_id,
            spend: (row.total_minor ?? 0) / 100,
          }))}
          categorySpend={
            Array.isArray(detailedAnalytics?.by_category)
              ? detailedAnalytics.by_category
                  .map((row) => {
                    if (!row || typeof row !== "object") return null;
                    const r = row as Record<string, unknown>;
                    const name =
                      (typeof r.name === "string" && r.name) ||
                      (typeof r.category === "string" && r.category) ||
                      "";
                    const value =
                      typeof r.value === "number"
                        ? r.value
                        : typeof r.total_minor === "number"
                          ? r.total_minor / 100
                          : NaN;
                    if (!name || Number.isNaN(value)) return null;
                    return { name, value };
                  })
                  .filter((x): x is { name: string; value: number } => x != null)
              : []
          }
        />
      </div>

      <div className="flex gap-8 min-h-[520px]">
        {/* Main: AI Picks */}
        <div className="flex-1 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
              Pending AI preorders
            </h2>
          </div>

          <AnimatePresence mode="popLayout">
            {loading ? (
              [0, 1, 2, 3].map((i) => (
                <div
                  key={i}
                  className="h-24 rounded-2xl animate-pulse bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)]"
                />
              ))
            ) : predList.length === 0 ? (
              <div className="rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-6">
                <EmptyState
                  headline={aiEmptyState.headline}
                  body={aiEmptyState.body}
                  variant={aiEmptyState.variant}
                  action="Sync Signals"
                  onAction={refreshAll}
                />
              </div>
            ) : (
              <div className="flex flex-col gap-2">
                {predList.map((item) => (
                    <motion.div
                      key={item.order_id}
                      layout
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      className="flex items-center gap-4 p-4 rounded-2xl border transition-all bg-[var(--desk-surface)] border-[var(--desk-border)] hover:border-[var(--desk-border-strong)]"
                    >
                      <div className="w-12 h-12 rounded-xl flex items-center justify-center shrink-0 bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)]">
                        <Package size={20} />
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-3 mb-1">
                          <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)] truncate">
                            {aiPredictionTitle(item)}
                          </span>
                          <span className="text-[9px] font-black tracking-tighter px-2 py-0.5 rounded bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] text-[var(--desk-text-tertiary)]">
                            {(item.confirmation_status || "PENDING").replace(/_/g, " ")}
                          </span>
                        </div>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                          {item.requested_delivery_date
                            ? item.requested_delivery_date.slice(0, 10)
                            : item.order_id}
                        </p>
                      </div>

                      <div className="flex flex-col items-end gap-2">
                        <div className="text-right">
                          <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                            {formatMinorAmount(item.total_minor, item.currency)}
                          </p>
                          <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                            {aiPredictionQty(item)} UNITS
                          </p>
                        </div>
                        <div className="flex items-center gap-3">
                          <button
                            type="button"
                            disabled={actingId === item.order_id}
                            onClick={() => void runAiAction(item.order_id, "confirm")}
                            className="text-[10px] font-light uppercase tracking-wide text-[var(--desk-accent)] hover:underline disabled:opacity-50"
                          >
                            Confirm
                          </button>
                          <button
                            type="button"
                            disabled={actingId === item.order_id}
                            onClick={() => void runAiAction(item.order_id, "reject")}
                            className="text-[10px] font-light uppercase tracking-wide text-[var(--desk-text-tertiary)] hover:text-red-600 disabled:opacity-50"
                          >
                            Reject
                          </button>
                        </div>
                      </div>
                    </motion.div>
                ))}
              </div>
            )}
          </AnimatePresence>
        </div>

        {/* Sidebar: Analytics */}
        <InsightsSidebar
          totalThisMonth={totalThisMonth}
          topProducts={topProducts}
          detailedError={detailedError}
          detailedSeries={detailedSeries}
        />
      </div>
      </PageChrome>
    </div>
  );
}
