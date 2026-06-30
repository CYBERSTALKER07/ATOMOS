"use client";

import { useState, useMemo, useCallback } from "react";
import {
  TrendingUp,
  BarChart3,
  Brain,
  Zap,
  RefreshCw,
  ArrowUpRight,
  Package,
  AlertTriangle,
  WifiOff,
  CheckSquare,
  Square,
  Minus,
  Plus,
  Loader2,
  ChevronRight,
} from "lucide-react";
import { Chip } from "@heroui/react";
import { PageChrome } from "@/components/PageChrome";
import { motion, AnimatePresence } from "framer-motion";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import EmptyState from "../../../components/EmptyState";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { correctPrediction } from "../../../lib/api";
import { retailerOrderCreateKey } from "@pegasusx/api-client";
import { useOptionalWebSocket } from "../../../lib/ws";
import type { Prediction, RetailerAnalytics } from "../../../lib/types";

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

const urgencyCfg: Record<
  string,
  { color: "danger" | "warning" | "default"; label: string }
> = {
  WAITING: { color: "danger", label: "REORDER NOW" },
  DORMANT: { color: "warning", label: "MONITOR" },
  EXECUTED: { color: "default", label: "ORDERED" },
};

export default function InsightsPage() {
  const {
    data: predictions,
    loading: loadingPred,
    error: predictionsError,
    isRefreshing: isPredictionsRefreshing,
    mutate: refreshPred,
  } = useLiveData<Prediction[]>("/v1/ai/predictions");
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

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [submitting, setSubmitting] = useState(false);
  const [correctingId, setCorrectingId] = useState<string | null>(null);
  const [orderResult, setOrderResult] = useState<"success" | "error" | null>(
    null,
  );

  const predList = predictions ?? [];
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
        message: "Insights access is partially restricted for this account.",
      };
    }
    if (loadIssue === "offline") {
      return {
        kind: "warning" as const,
        icon: WifiOff,
        message: "Offline mode active. Showing latest cached intelligence signals.",
      };
    }
    if (loadIssue === "error") {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Insights sync degraded. Auto-retry is active.",
      };
    }
    if (ws && !ws.isConnected) {
      return {
        kind: "warning" as const,
        icon: AlertTriangle,
        message: "Live socket reconnecting. New signals may be delayed.",
      };
    }
    if (isRefreshing && !loadingPred && !loadingAnalytics) {
      return {
        kind: "refreshing" as const,
        icon: RefreshCw,
        message: "Syncing intelligence feeds...",
      };
    }
    return null;
  }, [isRefreshing, loadIssue, loadingAnalytics, loadingPred, ws]);

  const aiEmptyState = useMemo(() => {
    if (loadIssue === "restricted") {
      return {
        headline: "Insights access restricted",
        body: "Your account cannot load replenishment signals right now.",
        variant: "restricted" as const,
      };
    }
    if (loadIssue === "offline") {
      return {
        headline: "Insights are offline",
        body: "Reconnect to refresh prediction signals and analytics.",
        variant: "offline" as const,
      };
    }
    if (loadIssue === "error") {
      return {
        headline: "Signals unavailable",
        body: "Prediction feeds could not be loaded right now.",
        variant: "error" as const,
      };
    }
    return {
      headline: "No actionable signals detected",
      body: "AI has no urgent replenishment recommendations for this cycle.",
      variant: "no-predictions" as const,
    };
  }, [loadIssue]);

  const toggleSelect = useCallback((id: string, defaultQty: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) {
        next.delete(id);
      } else {
        next.add(id);
        setQuantities((q) => ({ ...q, [id]: q[id] ?? defaultQty }));
      }
      return next;
    });
  }, []);

  const selectAll = useCallback(() => {
    if (selected.size === predList.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(predList.map((p) => p.id)));
      const qMap: Record<string, number> = {};
      predList.forEach((p) => {
        qMap[p.id] =
          quantities[p.id] ?? p.predicted_quantity ?? p.predictedQuantity ?? 1;
      });
      setQuantities(qMap);
    }
  }, [predList, selected.size, quantities]);

  const updateQty = useCallback((id: string, delta: number) => {
    setQuantities((prev) => ({
      ...prev,
      [id]: Math.max(1, (prev[id] ?? 1) + delta),
    }));
  }, []);

  const totalSelectedUnits = useMemo(
    () => Array.from(selected).reduce((s, id) => s + (quantities[id] ?? 0), 0),
    [selected, quantities],
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

  const getRetailerId = useCallback(() => {
    if (typeof localStorage === "undefined") return "";
    try {
      const profile = JSON.parse(
        localStorage.getItem("retailer_profile") || "null",
      ) as { id?: string } | null;
      return profile?.id ?? "";
    } catch {
      return "";
    }
  }, []);

  const handleCorrectPrediction = useCallback(
    async (predictionId: string, payload: Record<string, unknown>) => {
      setCorrectingId(predictionId);
      try {
        const res = await correctPrediction(
          predictionId,
          payload,
          `retailer-prediction-correct:${predictionId}`,
        );
        if (!res.ok) {
          throw new Error(`Correction failed with ${res.status}`);
        }
        await refreshPred();
      } finally {
        setCorrectingId(null);
      }
    },
    [refreshPred],
  );

  const createOrder = useCallback(async () => {
    if (selected.size === 0) return;

    setSubmitting(true);
    setOrderResult(null);

    try {
      const retailerId = getRetailerId();
      if (!retailerId) {
        throw new Error("Retailer profile not found. Please log in again.");
      }

      const orderItems = predList
        .filter((item) => selected.has(item.id))
        .map((item) => ({
          product_id: item.product_id ?? item.id,
          quantity:
            quantities[item.id] ??
            item.predicted_quantity ??
            item.predictedQuantity ??
            1,
        }));

      const idempotencyKey = retailerOrderCreateKey(
        orderItems
          .map((item) => `${item.product_id}:${item.quantity}`)
          .sort()
          .join("|"),
      );

      const res = await apiFetch("/v1/order/create", {
        method: "POST",
        headers: { "Idempotency-Key": idempotencyKey },
        body: JSON.stringify({
          retailer_id: retailerId,
          items: orderItems,
        }),
      });

      if (!res.ok) {
        throw new Error((await res.text()) || "Failed to submit procurement order.");
      }

      setSelected(new Set());
      setQuantities({});
      setOrderResult("success");
      await refreshPred();
    } catch {
      setOrderResult("error");
    } finally {
      setSubmitting(false);
    }
  }, [getRetailerId, predList, quantities, refreshPred, selected]);

  const loading = loadingPred || loadingAnalytics;

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="insights"
        title="Intelligence Hub"
        description="Predictive demand signals and network analytics."
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

      {orderResult && (
        <div className="mb-6">
          <Chip
            color={orderResult === "success" ? "success" : "danger"}
            variant="secondary"
            className="font-light"
          >
            {orderResult === "success"
              ? "Procurement order submitted. Signals are refreshing."
              : "Procurement order failed. Retry when the connection is stable."}
          </Chip>
        </div>
      )}

      <BentoGrid className="mb-8">
        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Monthly Spend
              </span>
              <BarChart3 size={18} style={{ color: "var(--desk-accent)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={totalThisMonth}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkRevenue} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                AI Signals
              </span>
              <Brain size={18} style={{ color: "var(--desk-info)" }} />
            </div>
            <div className="flex items-end justify-between">
              <CountUp
                end={predList.length}
                className="md-typescale-metric text-[var(--desk-text-primary)]"
              />
              <MiniSparkline data={sparkOrders} width={80} height={32} />
            </div>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Top Product
              </span>
              <TrendingUp size={18} style={{ color: "var(--desk-success)" }} />
            </div>
            <span className="md-typescale-title-medium font-light truncate text-[var(--desk-text-primary)]">
              {topProducts[0]?.product_name || "Calculating..."}
            </span>
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              {topProducts[0]?.quantity || 0} units staged
            </p>
          </div>
        </BentoCard>

        <BentoCard interactive={false}>
          <div className="flex flex-col gap-1">
            <div className="flex items-center justify-between mb-2">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)]">
                Efficiency
              </span>
              <Zap size={18} style={{ color: "var(--desk-warning)" }} />
            </div>
            <CountUp
              end={94}
              className="md-typescale-metric text-[var(--desk-text-primary)]"
              suffix="%"
            />
            <p className="md-typescale-body-small text-[var(--desk-text-secondary)]">
              Prediction precision
            </p>
          </div>
        </BentoCard>
      </BentoGrid>

      <div className="flex gap-8 min-h-[520px]">
        {/* Main: AI Picks */}
        <div className="flex-1 flex flex-col gap-4">
          <div className="flex items-center justify-between">
            <h2 className="md-typescale-title-large font-light text-[var(--desk-text-primary)]">
              AI Replenishment Picks
            </h2>
            {predList.length > 0 && (
              <button
                onClick={selectAll}
                className="text-[var(--desk-accent)] md-typescale-label-small font-light uppercase tracking-widest hover:underline"
              >
                {selected.size === predList.length
                  ? "Deselect All"
                  : "Select All"}
              </button>
            )}
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
                {predList.map((item) => {
                  const cfg = urgencyCfg[item.status] || urgencyCfg.DORMANT;
                  const isSelected = selected.has(item.id);
                  const qty =
                    quantities[item.id] ??
                    item.predicted_quantity ??
                    item.predictedQuantity ??
                    1;
                  return (
                    <motion.div
                      key={item.id}
                      layout
                      initial={{ opacity: 0, y: 10 }}
                      animate={{ opacity: 1, y: 0 }}
                      className={`flex items-center gap-4 p-4 rounded-2xl border transition-all ${
                        isSelected
                          ? "bg-[var(--desk-surface)] border-[var(--desk-accent)] shadow-md ring-2 ring-[var(--desk-accent-soft)]"
                          : "bg-[var(--desk-surface)] border-[var(--desk-border)] hover:border-[var(--desk-border-strong)]"
                      }`}
                    >
                      <button
                        onClick={() =>
                          toggleSelect(
                            item.id,
                            item.predicted_quantity ??
                              item.predictedQuantity ??
                              1,
                          )
                        }
                        className="shrink-0 text-[var(--desk-text-tertiary)] hover:text-[var(--desk-accent)] transition-colors"
                      >
                        {isSelected ? (
                          <CheckSquare
                            size={22}
                            className="text-[var(--desk-accent)]"
                          />
                        ) : (
                          <Square size={22} />
                        )}
                      </button>

                      <div
                        className={`w-12 h-12 rounded-xl flex items-center justify-center shrink-0 ${isSelected ? "bg-[var(--desk-accent-soft)] text-[var(--desk-accent)]" : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-tertiary)]"}`}
                      >
                        <Package size={20} />
                      </div>

                      <div className="flex-1 min-w-0">
                        <div className="flex items-center gap-3 mb-1">
                          <span className="md-typescale-title-small font-light text-[var(--desk-text-primary)] truncate">
                            {item.product_name ??
                              item.productName ??
                              "Predicted Item"}
                          </span>
                          <span
                            className={`text-[9px] font-black tracking-tighter px-2 py-0.5 rounded bg-[var(--desk-surface-subtle)] border border-[var(--desk-border)] text-[var(--desk-text-tertiary)]`}
                          >
                            {cfg.label}
                          </span>
                        </div>
                        <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                          {item.reasoning}
                        </p>
                      </div>

                      {isSelected ? (
                        <div className="flex items-center gap-3 bg-[var(--desk-canvas)] p-1 rounded-xl">
                          <button
                            onClick={() => updateQty(item.id, -1)}
                            className="w-8 h-8 rounded-lg bg-[var(--desk-surface)] flex items-center justify-center shadow-sm active:scale-90 transition-all"
                          >
                            <Minus size={14} />
                          </button>
                          <span className="md-typescale-title-small font-light w-6 text-center tabular-nums">
                            {qty}
                          </span>
                          <button
                            onClick={() => updateQty(item.id, 1)}
                            className="w-8 h-8 rounded-lg bg-[var(--desk-surface)] flex items-center justify-center shadow-sm active:scale-90 transition-all"
                          >
                            <Plus size={14} />
                          </button>
                        </div>
                      ) : (
                        <div className="flex flex-col items-end gap-2">
                          <div className="text-right">
                            <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                              {(item.predicted_amount ??
                                item.predictedAmount ??
                                0
                              ).toLocaleString()}
                            </p>
                            <p className="text-[10px] font-light text-[var(--desk-text-tertiary)] uppercase tracking-widest">
                              {item.predicted_quantity ??
                                item.predictedQuantity ??
                                1}{" "}
                              UNITS
                            </p>
                          </div>
                          <button
                            type="button"
                            disabled={correctingId === item.id}
                            onClick={() =>
                              void handleCorrectPrediction(item.id, {
                                status: "REJECTED",
                              })
                            }
                            className="text-[10px] font-light uppercase tracking-wide text-[var(--desk-text-tertiary)] hover:text-red-600"
                          >
                            {correctingId === item.id ? "Updating…" : "Dismiss signal"}
                          </button>
                        </div>
                      )}
                    </motion.div>
                  );
                })}
              </div>
            )}
          </AnimatePresence>

          {selected.size > 0 && (
            <motion.div
              initial={{ y: 20, opacity: 0 }}
              animate={{ y: 0, opacity: 1 }}
              className="sticky bottom-4 p-4 bg-[var(--desk-text-primary)] rounded-2xl shadow-xl flex items-center justify-between text-white border border-white/10 backdrop-blur-md"
            >
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-white/10 flex items-center justify-center font-light">
                  {selected.size}
                </div>
                <div>
                  <p className="text-xs font-light opacity-60 uppercase tracking-widest">
                    Staged Assets
                  </p>
                  <p className="text-sm font-light">
                    {totalSelectedUnits.toLocaleString()} units total
                  </p>
                </div>
              </div>
              <button
                type="button"
                onClick={() => void createOrder()}
                disabled={submitting}
                className="portal-btn portal-btn--primary font-light h-11 px-8 rounded-xl shadow-lg hover:scale-105 active:scale-95 transition-all"
              >
                {submitting ? (
                  <>
                    <Loader2 size={14} className="mr-2 animate-spin" />
                    Executing...
                  </>
                ) : (
                  "Execute Procurement"
                )}
              </button>
            </motion.div>
          )}
        </div>

        {/* Sidebar: Analytics */}
        <aside className="w-[360px] shrink-0 hidden lg:flex flex-col gap-6">
          <div className="p-8 bg-[var(--desk-text-primary)] rounded-3xl text-white shadow-2xl relative overflow-hidden">
            <Zap className="absolute top-[-10px] right-[-10px] w-32 h-32 opacity-10 rotate-12" />
            <span className="md-typescale-label-small uppercase tracking-[0.2em] opacity-60 mb-4 block">
              Fleet Efficiency
            </span>
            <h3 className="md-typescale-display-small font-light mb-2 tabular-nums">
              {totalThisMonth.toLocaleString()}
            </h3>
            <p className="text-sm opacity-60 font-medium">
              Monthly Operational Volume
            </p>
            <div className="mt-8 pt-6 border-t border-white/10 flex items-center justify-between">
              <span className="text-xs font-light opacity-60 uppercase tracking-widest">
                Trend
              </span>
              <div className="flex items-center gap-2 text-[var(--desk-success)] font-light">
                <ArrowUpRight size={16} />
                <span>+12.4%</span>
              </div>
            </div>
          </div>

          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
            <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-6">
              Top Resource Demand
            </h3>
            {topProducts.length === 0 ? (
              <div className="rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-4 text-center">
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] uppercase font-light tracking-widest">
                  No product demand rankings available yet
                </p>
              </div>
            ) : (
              <div className="space-y-4">
                {topProducts.slice(0, 5).map((item, i) => (
                  <div
                    key={item.product_id}
                    className="flex items-center gap-4 group"
                  >
                    <div className="w-8 h-8 rounded-lg bg-[var(--desk-surface-subtle)] flex items-center justify-center text-[10px] font-black text-[var(--desk-text-tertiary)] group-hover:bg-[var(--desk-accent-soft)] group-hover:text-[var(--desk-accent)] transition-colors">
                      {i + 1}
                    </div>
                    <div className="flex-1 min-w-0">
                      <p className="md-typescale-body-medium font-light text-[var(--desk-text-primary)] truncate">
                        {item.product_name}
                      </p>
                      <p className="text-[10px] text-[var(--desk-text-tertiary)] font-light uppercase">
                        {item.quantity} Units
                      </p>
                    </div>
                    <ChevronRight
                      size={14}
                      className="text-[var(--desk-text-tertiary)] opacity-0 group-hover:opacity-100 transition-opacity"
                    />
                  </div>
                ))}
              </div>
            )}
          </div>

          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
            <h3 className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-4">
              Detailed spend series
            </h3>
            {detailedError ? (
              <p className="text-sm text-[var(--desk-text-tertiary)]">
                Advanced analytics unavailable right now.
              </p>
            ) : detailedSeries.length === 0 ? (
              <p className="text-sm text-[var(--desk-text-tertiary)]">
                No completed-order series yet.
              </p>
            ) : (
              <div className="space-y-2 max-h-48 overflow-y-auto">
                {detailedSeries.slice(0, 8).map((row) => (
                  <div
                    key={row.order_id}
                    className="flex items-center justify-between py-2 border-b border-[var(--desk-border)] last:border-0"
                  >
                    <span className="text-xs font-mono text-[var(--desk-text-secondary)]">
                      #{row.order_id.slice(-8)}
                    </span>
                    <span className="text-sm font-light tabular-nums">
                      {(row.total_minor ?? 0).toLocaleString()}{" "}
                      {row.currency ?? "UZS"}
                    </span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </aside>
      </div>
      </PageChrome>
    </div>
  );
}
