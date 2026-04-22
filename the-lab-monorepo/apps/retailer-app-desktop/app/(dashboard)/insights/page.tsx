"use client";

import { useState, useMemo, useCallback } from "react";
import {
  TrendingUp, TrendingDown, BarChart3, Brain, Zap, RefreshCcw,
  ShoppingCart, Star, ArrowUpRight, ArrowDownRight, Package, AlertTriangle,
  CheckSquare, Square, Minus, Plus, Loader2, XCircle,
} from "lucide-react";
import { Button, Chip, Skeleton } from "@heroui/react";
import { BentoGrid, BentoCard } from "../../../components/BentoGrid";
import CountUp from "../../../components/CountUp";
import MiniSparkline from "../../../components/MiniSparkline";
import { useLiveData } from "../../../lib/hooks";
import { apiFetch } from "../../../lib/auth";
import { useCart } from "../../../lib/cart";
import type { Prediction, RetailerAnalytics, TopProduct } from "../../../lib/types";

const urgencyCfg: Record<string, { color: "danger" | "warning" | "default"; label: string }> = {
  WAITING: { color: "danger", label: "Reorder Now" },
  DORMANT: { color: "warning", label: "Monitor" },
  EXECUTED: { color: "default", label: "Ordered" },
};

export default function InsightsPage() {
  const { data: predictions, loading: loadingPred, mutate: refreshPred } = useLiveData<Prediction[]>("/v1/ai/predictions");
  const { data: analytics, loading: loadingAnalytics } = useLiveData<RetailerAnalytics>("/v1/retailer/analytics/expenses");
  const { addToCart } = useCart();

  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [quantities, setQuantities] = useState<Record<string, number>>({});
  const [submitting, setSubmitting] = useState(false);
  const [orderResult, setOrderResult] = useState<string | null>(null);
  const [rejectingId, setRejectingId] = useState<string | null>(null);

  const predList = predictions ?? [];
  const topProducts = analytics?.top_products ?? [];
  const totalThisMonth = analytics?.total_this_month ?? 0;
  const totalLastMonth = analytics?.total_last_month ?? 0;
  const monthlyExpenses = analytics?.monthly_expenses ?? [];

  const toggleSelect = useCallback((id: string, defaultQty: number) => {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) { next.delete(id); } else { next.add(id); setQuantities((q) => ({ ...q, [id]: q[id] ?? defaultQty })); }
      return next;
    });
  }, []);

  const selectAll = useCallback(() => {
    if (selected.size === predList.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(predList.map((p) => p.id)));
      const qMap: Record<string, number> = {};
      predList.forEach((p) => { qMap[p.id] = quantities[p.id] ?? p.predictedQuantity; });
      setQuantities(qMap);
    }
  }, [predList, selected.size, quantities]);

  const updateQty = useCallback((id: string, delta: number) => {
    setQuantities((prev) => ({ ...prev, [id]: Math.max(1, (prev[id] ?? 1) + delta) }));
  }, []);

  const totalSelectedUnits = useMemo(
    () => Array.from(selected).reduce((s, id) => s + (quantities[id] ?? 0), 0),
    [selected, quantities],
  );

  const createOrder = useCallback(async () => {
    if (selected.size === 0) return;
    setSubmitting(true);
    setOrderResult(null);
    try {
      const items = predList
        .filter((p) => selected.has(p.id))
        .map((p) => ({ product_id: p.id, quantity: quantities[p.id] ?? p.predictedQuantity }));
      const res = await apiFetch("/v1/order/create", {
        method: "POST",
        body: JSON.stringify({ items }),
      });
      if (!res.ok) throw new Error("Order creation failed");
      setOrderResult("success");
      setSelected(new Set());
      setQuantities({});
    } catch {
      setOrderResult("error");
    } finally {
      setSubmitting(false);
    }
  }, [predList, selected, quantities]);

  const addSelectedToCart = useCallback(() => {
    predList
      .filter((p) => selected.has(p.id))
      .forEach((p) => {
        addToCart({ id: p.id, name: p.productName, price: p.predictedAmount / (p.predictedQuantity || 1), supplier_id: "", image_url: "" }, quantities[p.id] ?? p.predictedQuantity);
      });
    setSelected(new Set());
  }, [predList, selected, quantities, addToCart]);

  const rejectPrediction = useCallback(async (predictionId: string) => {
    setRejectingId(predictionId);
    try {
      const res = await apiFetch(`/v1/ai/predictions/correct?prediction_id=${encodeURIComponent(predictionId)}`, {
        method: "PATCH",
        body: JSON.stringify({ status: "DISMISSED" }),
      });
      if (res.ok) refreshPred();
    } catch { /* swallow */ }
    finally { setRejectingId(null); }
  }, [refreshPred]);

  const correctPrediction = useCallback(async (predictionId: string, newQty: number) => {
    try {
      await apiFetch(`/v1/ai/predictions/correct?prediction_id=${encodeURIComponent(predictionId)}`, {
        method: "PATCH",
        body: JSON.stringify({ amount: null, status: "WAITING" }),
      });
      refreshPred();
    } catch { /* swallow */ }
  }, [refreshPred]);

  const sparkRevenue = useMemo(() => {
    if (monthlyExpenses.length > 0) return monthlyExpenses.map((m) => m.total);
    return Array.from({ length: 14 }, (_, i) => 800 + i * 45 + Math.sin(i * 0.9) * 120);
  }, [monthlyExpenses]);

  const sparkOrders = useMemo(() => Array.from({ length: 14 }, (_, i) => 12 + i * 2 + Math.cos(i * 0.7) * 4), []);

  const revenueDelta = totalLastMonth > 0
    ? (((totalThisMonth - totalLastMonth) / totalLastMonth) * 100).toFixed(0)
    : null;

  const loading = loadingPred || loadingAnalytics;

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
          <div className="flex-1 flex flex-col gap-2">
            {[0, 1, 2, 3].map((i) => <Skeleton key={i} className="h-20 rounded-2xl" />)}
          </div>
          <Skeleton className="w-[400px] h-80 rounded-2xl" />
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8">
      {/* ── Header ── */}
      <header className="mb-6 flex items-end justify-between gap-4 flex-wrap">
        <div>
          <h1 className="md-typescale-headline-large">Intelligence & Insights</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: "var(--muted)" }}>
            AI-powered recommendations, performance metrics, and demand forecasting.
          </p>
        </div>
        <Button variant="outline" onPress={() => refreshPred()} className="md-btn md-btn-outlined md-typescale-label-large px-4 h-10 flex items-center gap-2">
          <RefreshCcw size={16} /> Refresh Forecast
        </Button>
      </header>

      {/* ── KPI Bento ── */}
      <BentoGrid className="mb-8">
        <BentoCard delay={0}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Spend MTD</span>
              <BarChart3 size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={totalThisMonth} className="md-kpi-value" suffix="" />
              <MiniSparkline data={sparkRevenue} width={72} height={28} />
            </div>
            {revenueDelta && (
              <div className="flex items-center gap-1.5">
                <ArrowUpRight size={14} strokeWidth={2} style={{ color: Number(revenueDelta) >= 0 ? "var(--success)" : "var(--danger)" }} />
                <span className="md-kpi-sub" style={{ color: Number(revenueDelta) >= 0 ? "var(--success)" : "var(--danger)" }}>
                  {revenueDelta}% vs last month
                </span>
              </div>
            )}
          </div>
        </BentoCard>

        <BentoCard delay={60}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">AI Predictions</span>
              <Brain size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <div className="flex items-end justify-between gap-4">
              <CountUp end={predList.length} className="md-kpi-value" />
              <MiniSparkline data={sparkOrders} width={72} height={28} />
            </div>
            <span className="md-kpi-sub">
              {predList.filter((p) => p.status === "WAITING").length} actionable
            </span>
          </div>
        </BentoCard>

        <BentoCard delay={120}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Top Products</span>
              <TrendingUp size={18} strokeWidth={1.5} style={{ color: "var(--success)" }} />
            </div>
            <CountUp end={topProducts.length} className="md-kpi-value" />
            <span className="md-kpi-sub">By purchase volume</span>
          </div>
        </BentoCard>

        <BentoCard delay={180}>
          <div className="md-kpi-card">
            <div className="flex items-center justify-between">
              <span className="md-kpi-label">Last Month</span>
              <TrendingDown size={18} strokeWidth={1.5} style={{ color: "var(--muted)" }} />
            </div>
            <CountUp end={totalLastMonth} className="md-kpi-value" suffix="" />
            <span className="md-kpi-sub">Previous period baseline</span>
          </div>
        </BentoCard>
      </BentoGrid>

      {/* ── Split: AI Picks + Top Sellers Panel ── */}
      <div className="flex gap-6 min-h-[420px]">

        {/* Left: AI Replenishment */}
        <div className="flex-1 flex flex-col gap-2 overflow-y-auto max-h-[calc(100dvh-440px)] pr-1">
          <div className="flex items-center gap-3 mb-3">
            <Brain size={20} style={{ color: "var(--accent)" }} />
            <h2 className="md-typescale-title-large font-semibold text-foreground">AI Replenishment Picks</h2>
            {predList.length > 0 && (
              <button
                onClick={selectAll}
                className="md-typescale-label-small font-bold text-accent ml-auto cursor-pointer hover:underline"
              >
                {selected.size === predList.length ? "Deselect All" : "Select All"}
              </button>
            )}
          </div>

          {/* Order result banner */}
          {orderResult === "success" && (
            <div className="bento-card flex items-center gap-2" style={{ borderLeft: "3px solid var(--success)" }}>
              <span className="md-typescale-body-medium text-foreground font-semibold">Order created successfully!</span>
              <button onClick={() => setOrderResult(null)} className="ml-auto text-muted cursor-pointer">Dismiss</button>
            </div>
          )}
          {orderResult === "error" && (
            <div className="bento-card flex items-center gap-2" style={{ borderLeft: "3px solid var(--danger)" }}>
              <AlertTriangle size={16} style={{ color: "var(--danger)" }} />
              <span className="md-typescale-body-medium text-foreground">Order failed. Try again.</span>
              <button onClick={() => setOrderResult(null)} className="ml-auto text-muted cursor-pointer">Dismiss</button>
            </div>
          )}

          {predList.length === 0 ? (
            <div className="bento-card flex flex-col items-center justify-center py-8 gap-2">
              <Brain size={32} style={{ color: "var(--muted)" }} />
              <p className="md-typescale-body-medium text-muted">No predictions available yet</p>
            </div>
          ) : (
            predList.map((item) => {
              const cfg = urgencyCfg[item.status] ?? urgencyCfg.DORMANT;
              const isSelected = selected.has(item.id);
              const qty = quantities[item.id] ?? item.predictedQuantity;
              return (
                <div
                  key={item.id}
                  className="bento-card transition-all"
                  style={isSelected ? { borderColor: "var(--accent)", borderWidth: 2 } : undefined}
                >
                  <div className="flex items-center gap-4">
                    {/* Checkbox */}
                    <button
                      onClick={() => toggleSelect(item.id, item.predictedQuantity)}
                      className="shrink-0 cursor-pointer"
                    >
                      {isSelected ? (
                        <CheckSquare size={22} style={{ color: "var(--accent)" }} />
                      ) : (
                        <Square size={22} style={{ color: "var(--muted)" }} />
                      )}
                    </button>

                    <div className="w-12 h-12 rounded-xl flex items-center justify-center shrink-0" style={{ background: "var(--surface)" }}>
                      <Package size={20} style={{ color: "var(--muted)" }} />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2.5">
                        <span className="md-typescale-title-small font-semibold text-foreground truncate">
                          {item.productName || `Prediction ${item.id.slice(-6)}`}
                        </span>
                        <Chip size="sm" color={cfg.color} variant="soft" className="shrink-0">{cfg.label}</Chip>
                      </div>
                      <div className="flex items-center gap-3 mt-1.5">
                        <span className="md-typescale-body-small text-muted">
                          {item.reasoning || "AI-generated recommendation"}
                        </span>
                      </div>
                    </div>

                    {/* Quantity controls if selected */}
                    {isSelected ? (
                      <div className="flex items-center gap-2 shrink-0">
                        <button
                          onClick={() => updateQty(item.id, -1)}
                          className="w-7 h-7 rounded-lg flex items-center justify-center cursor-pointer"
                          style={{ background: "var(--surface)" }}
                        >
                          <Minus size={14} style={{ color: "var(--foreground)" }} />
                        </button>
                        <span className="md-typescale-title-small font-bold tabular-nums w-8 text-center">{qty}</span>
                        <button
                          onClick={() => updateQty(item.id, 1)}
                          className="w-7 h-7 rounded-lg flex items-center justify-center cursor-pointer"
                          style={{ background: "var(--surface)" }}
                        >
                          <Plus size={14} style={{ color: "var(--foreground)" }} />
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-3 shrink-0">
                        <div className="text-right">
                          <div className="flex items-center gap-2 justify-end">
                            <span className="md-typescale-label-small uppercase tracking-widest font-semibold" style={{ color: "var(--muted)" }}>Qty</span>
                            <span className="md-typescale-title-small font-bold tabular-nums">{item.predictedQuantity}</span>
                          </div>
                          <span className="md-typescale-label-small font-semibold tabular-nums" style={{ color: "var(--accent)" }}>
                            {item.predictedAmount.toLocaleString()}
                          </span>
                        </div>
                        {item.status === "WAITING" && (
                          <button
                            onClick={() => rejectPrediction(item.id)}
                            disabled={rejectingId === item.id}
                            className="p-1.5 rounded-lg hover:bg-surface cursor-pointer transition-colors disabled:opacity-50"
                            title="Dismiss prediction"
                          >
                            {rejectingId === item.id ? (
                              <Loader2 size={14} className="animate-spin text-muted" />
                            ) : (
                              <XCircle size={16} style={{ color: "var(--muted)" }} />
                            )}
                          </button>
                        )}
                      </div>
                    )}
                  </div>
                </div>
              );
            })
          )}

          {/* Action Bar */}
          {selected.size > 0 && (
            <div className="sticky bottom-0 mt-3 p-4 rounded-2xl border border-[var(--border)] flex items-center gap-3" style={{ background: "var(--background)" }}>
              <Chip size="sm" color="default" variant="soft">{selected.size} selected</Chip>
              <span className="md-typescale-label-large font-semibold text-foreground tabular-nums">{totalSelectedUnits} units</span>
              <div className="flex-1" />
              <Button
                variant="primary"
                onPress={createOrder}
                isDisabled={submitting}
                className="md-btn md-btn-filled md-typescale-label-large h-10 px-5 flex items-center gap-2"
              >
                {submitting ? <Loader2 size={16} className="animate-spin" /> : <ShoppingCart size={16} />} Create Order
              </Button>
              <Button
                variant="outline"
                onPress={addSelectedToCart}
                className="md-btn md-btn-outlined md-typescale-label-large h-10 px-5 flex items-center gap-2"
              >
                <ShoppingCart size={16} /> Add to Cart
              </Button>
            </div>
          )}
        </div>

        {/* Right: Performance Panel */}
        <div className="w-full lg:w-[360px] xl:w-[400px] shrink-0 hidden lg:flex flex-col gap-4">
          <h2 className="md-typescale-title-large font-semibold text-foreground">Top Products This Month</h2>

          <div className="bento-card" style={{ background: "var(--accent)", color: "var(--accent-foreground)" }}>
            <div className="flex items-center gap-3 mb-2">
              <Zap size={18} />
              <span className="md-typescale-title-small font-bold">Spend Summary</span>
            </div>
            <div className="flex items-end gap-2">
              <span className="md-typescale-headline-medium font-bold tabular-nums">{totalThisMonth.toLocaleString()}</span>
              <span className="md-typescale-body-small font-medium opacity-80 pb-0.5">this month</span>
            </div>
            {totalLastMonth > 0 && (
              <span className="md-typescale-body-small font-medium opacity-80 mt-1 block">
                vs {totalLastMonth.toLocaleString()} last month
              </span>
            )}
          </div>

          <div className="bento-card">
            <div className="flex flex-col gap-3">
              {topProducts.length === 0 ? (
                <p className="md-typescale-body-medium text-muted text-center py-4">No product data yet</p>
              ) : (
                topProducts.slice(0, 5).map((item, i) => (
                  <div key={item.product_id} className="flex items-center gap-3">
                    <span className="md-typescale-label-large font-bold tabular-nums w-5 text-right text-muted">{i + 1}</span>
                    <div className="flex-1 min-w-0">
                      <span className="md-typescale-body-medium font-medium text-foreground block truncate">{item.product_name}</span>
                      <span className="md-typescale-label-small text-muted tabular-nums">{item.quantity} units</span>
                    </div>
                    <span className="md-typescale-label-small font-semibold tabular-nums text-foreground">
                      {item.total.toLocaleString()}
                    </span>
                  </div>
                ))
              )}
            </div>
          </div>

          {(analytics?.top_suppliers ?? []).length > 0 && (
            <div className="bento-card">
              <div className="flex items-center gap-3 mb-2">
                <Star size={16} style={{ color: "var(--warning)" }} />
                <span className="md-typescale-title-small font-semibold text-foreground">Top Suppliers</span>
              </div>
              <div className="flex flex-col gap-2 mt-2">
                {(analytics?.top_suppliers ?? []).slice(0, 3).map((ts) => (
                  <div key={ts.supplier_id} className="flex justify-between items-center">
                    <span className="md-typescale-body-medium font-medium text-foreground">{ts.supplier_name}</span>
                    <span className="md-typescale-label-small tabular-nums text-muted">{ts.order_count} orders</span>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
