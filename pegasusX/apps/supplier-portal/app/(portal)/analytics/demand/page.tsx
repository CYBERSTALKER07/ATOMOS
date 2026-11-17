"use client";

import Link from "next/link";
import type { Route } from "next";
import { useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  CartesianGrid,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";
import { createSupplierApi } from "@/lib/api";
import {
  createIntelRoiApi,
  extractControlError,
  formatMilliUnits,
  formatRateBps,
  formatSignedBps,
  milliToChartUnits,
  type ForecastExplanation,
  type ForecastRegressor,
  type ForecastRun,
  type ForecastValue,
} from "@/lib/intel-roi-api";
import {
  forecastConfidenceFromDemand,
  formatForecastUpdatedAt,
  isForecastStale,
} from "@/lib/forecast-confidence";
import { usePortalT } from "@/lib/i18n";
import { useSupplierSessionReconcile } from "@/lib/use-supplier-session-reconcile";
import type {
  SupplierDemandAccuracyResponse,
  SupplierDemandHistoryResponse,
  SupplierDemandSummaryResponse,
} from "@pegasusx/types";
import { ForecastConfidenceCard } from "@/components/ForecastConfidenceCard";
import { KpiStatCard, KpiStatGrid } from "@/components/KpiStatCard";
import { PageChrome } from "@/components/PageChrome";
import { PortalInput, PortalSection, PortalSelect } from "@/components/portal";
import StatusBadge from "@/components/StatusBadge";

function pctLabel(ratio: number | null | undefined): string {
  if (ratio == null || Number.isNaN(ratio)) return "—";
  return `${Math.round(ratio * 100)}%`;
}

export default function DemandAnalyticsPage() {
  return (
    <Suspense fallback={null}>
      <DemandAnalyticsInner />
    </Suspense>
  );
}

function DemandAnalyticsInner() {
  const t = usePortalT();
  const tRef = useRef(t);
  tRef.current = t;
  const searchParams = useSearchParams();
  const baseApi = useMemo(() => createSupplierApi(), []);
  const intelApi = useMemo(() => createIntelRoiApi(baseApi), [baseApi]);

  const [skuFilter, setSkuFilter] = useState<string>(() => (searchParams.get("sku") || "").trim());
  const [locationId, setLocationId] = useState<string>(() => (searchParams.get("location_id") || "").trim());
  const [horizonDays, setHorizonDays] = useState<string>(() => {
    const raw = (searchParams.get("horizon_days") || "14").trim();
    return ["7", "14", "28", "30"].includes(raw) ? raw : "14";
  });
  const [runIdFilter] = useState<string>(() => (searchParams.get("run_id") || "").trim());
  const [promoDiscountBps, setPromoDiscountBps] = useState<string>("0");
  const [activePromo, setActivePromo] = useState<boolean>(false);
  const [includeExplain, setIncludeExplain] = useState<boolean>(true);

  const [loading, setLoading] = useState<boolean>(true);
  const [refreshTick, setRefreshTick] = useState<number>(0);
  useSupplierSessionReconcile(() => setRefreshTick((tick) => tick + 1));

  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [busyAction, setBusyAction] = useState<string | null>(null);

  // V2 state
  const [forecastRun, setForecastRun] = useState<ForecastRun | null>(null);
  const [forecastValues, setForecastValues] = useState<ForecastValue[]>([]);
  const [explanations, setExplanations] = useState<ForecastExplanation[]>([]);
  const [adjustmentDrafts, setAdjustmentDrafts] = useState<Record<string, string>>({});
  const [commentDrafts, setCommentDrafts] = useState<Record<string, string>>({});

  // Legacy V1 demand compatibility state
  const [summary, setSummary] = useState<SupplierDemandSummaryResponse | null>(null);
  const [history, setHistory] = useState<SupplierDemandHistoryResponse | null>(null);
  const [accuracy, setAccuracy] = useState<SupplierDemandAccuracyResponse | null>(null);

  const loadForecasts = useCallback(async () => {
    setLoading(true);
    setError(null);
    const parsedHorizon = Number.parseInt(horizonDays, 10);
    const v2Promise = intelApi.getPlanningForecast({
      sku: skuFilter.trim() || undefined,
      location_id: locationId.trim() || undefined,
      horizon_days: Number.isFinite(parsedHorizon) ? parsedHorizon : 14,
      run_id: runIdFilter || undefined,
      limit: 120,
    });
    const legacySupported =
      typeof baseApi.getSupplierDemandToday === "function" &&
      typeof baseApi.getSupplierDemandHistory === "function" &&
      typeof baseApi.getSupplierDemandAccuracy === "function";

    const [v2Res, sumRes, histRes, accRes] = await Promise.allSettled([
      v2Promise,
      legacySupported ? baseApi.getSupplierDemandToday() : Promise.resolve(null),
      legacySupported ? baseApi.getSupplierDemandHistory() : Promise.resolve(null),
      legacySupported ? baseApi.getSupplierDemandAccuracy({ days: 28 }) : Promise.resolve(null),
    ]);

    if (v2Res.status === "fulfilled") {
      setForecastRun(v2Res.value.run ?? null);
      setForecastValues(v2Res.value.forecasts ?? []);
      setExplanations(v2Res.value.explanations ?? []);
    } else if (!legacySupported || sumRes.status === "rejected") {
      const parsed = extractControlError(v2Res.reason, tRef.current("supplier_portal.forecast_v2.load_failed"));
      setError(parsed.message);
    }

    if (sumRes.status === "fulfilled" && sumRes.value) setSummary(sumRes.value);
    if (histRes.status === "fulfilled" && histRes.value) setHistory(histRes.value);
    if (accRes.status === "fulfilled" && accRes.value) setAccuracy(accRes.value);
    setLoading(false);
  }, [baseApi, horizonDays, intelApi, locationId, runIdFilter, skuFilter]);

  useEffect(() => {
    void loadForecasts();
  }, [loadForecasts, refreshTick]);

  const handleRunForecast = async (e: React.FormEvent) => {
    e.preventDefault();
    const horizon = Number.parseInt(horizonDays, 10);
    const discountBps = Number.parseInt(promoDiscountBps || "0", 10);
    setBusyAction("run");
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.runPlanningForecast({
        location_id: locationId.trim() || undefined,
        sku: skuFilter.trim() || undefined,
        horizon_days: Number.isFinite(horizon) && horizon > 0 ? horizon : 14,
        promo_discount_bps: Number.isFinite(discountBps) ? discountBps : 0,
        active_promo: activePromo,
        explain: includeExplain,
      });
      setForecastRun(res.run);
      setForecastValues(res.forecasts ?? []);
      setExplanations(res.explanations ?? []);
      setNotice(
        t("supplier_portal.forecast_v2.run_success", {
          runId: res.run.run_id,
          winner: res.run.winner_model,
        }),
      );
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.forecast_v2.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const handleDecideExplanation = async (
    explanation: ForecastExplanation,
    decision: "APPROVE" | "REJECT",
  ) => {
    const rawAdj = adjustmentDrafts[explanation.explanation_id];
    const parsedAdj =
      rawAdj !== undefined && rawAdj.trim() !== ""
        ? Number.parseInt(rawAdj.trim(), 10)
        : explanation.adjustment_bps;
    setBusyAction(`exp-${explanation.explanation_id}`);
    setError(null);
    setNotice(null);
    try {
      const res = await intelApi.approveForecastExplanation(explanation.explanation_id, {
        decision,
        adjustment_bps: Number.isFinite(parsedAdj) ? parsedAdj : explanation.adjustment_bps,
        comment: commentDrafts[explanation.explanation_id]?.trim() || undefined,
      });
      setExplanations((prev) =>
        prev.map((row) =>
          row.explanation_id === res.explanation.explanation_id ? res.explanation : row,
        ),
      );
      setNotice(
        t("supplier_portal.forecast_v2.adjustment_decided", {
          explanationId: res.explanation.explanation_id,
          status: res.explanation.status,
        }),
      );
    } catch (err) {
      const parsed = extractControlError(err, t("supplier_portal.forecast_v2.load_failed"));
      setError(parsed.message);
    } finally {
      setBusyAction(null);
    }
  };

  const v2ChartData = useMemo(() => {
    const winnerRows = forecastValues.filter((row) => row.is_winner);
    const sourceRows = winnerRows.length > 0 ? winnerRows : forecastValues;
    const byDate = new Map<
      string,
      { date: string; p10: number; p50: number; p90: number; baseline: number }
    >();
    for (const row of sourceRows) {
      const dateKey = row.forecast_date.slice(0, 10);
      const curr = byDate.get(dateKey) ?? {
        date: dateKey.slice(5) || dateKey,
        p10: 0,
        p50: 0,
        p90: 0,
        baseline: 0,
      };
      curr.p10 += milliToChartUnits(row.p10_milli);
      curr.p50 += milliToChartUnits(row.p50_milli);
      curr.p90 += milliToChartUnits(row.p90_milli);
      curr.baseline += milliToChartUnits(row.baseline_p50_milli);
      byDate.set(dateKey, curr);
    }
    return Array.from(byDate.values());
  }, [forecastValues]);

  const activeRegressors = useMemo<ForecastRegressor[]>(() => {
    const seen = new Map<string, ForecastRegressor>();
    for (const reg of forecastRun?.regressors ?? []) {
      seen.set(`${reg.name}:${reg.category}`, reg);
    }
    for (const row of forecastValues) {
      for (const reg of row.regressors ?? []) {
        if (!seen.has(`${reg.name}:${reg.category}`)) {
          seen.set(`${reg.name}:${reg.category}`, reg);
        }
      }
    }
    return Array.from(seen.values());
  }, [forecastRun?.regressors, forecastValues]);

  const legacyChartData = useMemo(
    () =>
      (history?.time_series ?? []).map((point) => ({
        date: point.date.slice(5),
        baseline: point.predicted_qty,
        actual: point.actual_qty,
      })),
    [history],
  );

  return (
    <PageChrome
      icon="analytics"
      title={t("supplier_portal.forecast_v2.title")}
      description={t("supplier_portal.forecast_v2.subtitle")}
      loading={loading && !forecastRun && !summary}
      skeletonVariant="dashboard"
      error={error}
      actions={
        <div className="flex flex-wrap items-center gap-2">
          <Link href={"/proposals" as Route} className="portal-btn portal-btn--outline text-sm">
            {t("supplier_portal.forecast_v2.open_proposals")}
          </Link>
          <Link href={"/analytics/demand/flywheel" as Route} className="portal-btn portal-btn--outline text-sm">
            POS flywheel
          </Link>
          <Link href={"/analytics/demand/signals" as Route} className="portal-btn portal-btn--outline text-sm">
            Manage Signals
          </Link>
          <Link href={"/analytics" as Route} className="portal-btn portal-btn--ghost text-sm">
            Back to analytics
          </Link>
        </div>
      }
    >
      <div className="flex flex-col gap-6">
        {notice ? (
          <div
            role="status"
            className="rounded-xl border px-4 py-3 text-sm"
            style={{
              background: "var(--desk-success-soft)",
              borderColor: "color-mix(in srgb, var(--desk-success) 35%, transparent)",
              color: "var(--desk-success)",
            }}
          >
            {notice}
          </div>
        ) : null}

        <PortalSection
          icon="analytics"
          title={t("supplier_portal.forecast_v2.run_forecast")}
          description={t("supplier_portal.forecast_v2.subtitle")}
        >
          <form onSubmit={handleRunForecast} className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-6 gap-3 items-end">
            <label className="flex flex-col gap-1 text-xs">
              <span style={{ color: "var(--desk-text-secondary)" }}>
                {t("supplier_portal.forecast_v2.location_id")}
              </span>
              <PortalInput
                aria-label={t("supplier_portal.forecast_v2.location_id")}
                value={locationId}
                onChange={(e) => setLocationId(e.target.value)}
                placeholder="wh-tashkent-1"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs">
              <span style={{ color: "var(--desk-text-secondary)" }}>
                {t("supplier_portal.forecast_v2.sku_filter")}
              </span>
              <PortalInput
                aria-label={t("supplier_portal.forecast_v2.sku_filter")}
                value={skuFilter}
                onChange={(e) => setSkuFilter(e.target.value)}
                placeholder="SKU-1001"
              />
            </label>
            <label className="flex flex-col gap-1 text-xs">
              <span style={{ color: "var(--desk-text-secondary)" }}>
                {t("supplier_portal.forecast_v2.horizon_days")}
              </span>
              <PortalSelect
                aria-label={t("supplier_portal.forecast_v2.horizon_days")}
                value={horizonDays}
                onChange={(e) => setHorizonDays(e.target.value)}
              >
                <option value="7">7d</option>
                <option value="14">14d</option>
                <option value="28">28d</option>
                <option value="30">30d</option>
              </PortalSelect>
            </label>
            <label className="flex flex-col gap-1 text-xs">
              <span style={{ color: "var(--desk-text-secondary)" }}>
                {t("supplier_portal.forecast_v2.promo_discount_bps")}
              </span>
              <PortalInput
                aria-label={t("supplier_portal.forecast_v2.promo_discount_bps")}
                type="number"
                min={0}
                max={9000}
                value={promoDiscountBps}
                onChange={(e) => setPromoDiscountBps(e.target.value)}
              />
            </label>
            <div className="flex flex-col gap-2 text-xs pb-1">
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={activePromo}
                  onChange={(e) => setActivePromo(e.target.checked)}
                />
                <span>{t("supplier_portal.forecast_v2.active_promo")}</span>
              </label>
              <label className="inline-flex items-center gap-2 cursor-pointer">
                <input
                  type="checkbox"
                  checked={includeExplain}
                  onChange={(e) => setIncludeExplain(e.target.checked)}
                />
                <span>{t("supplier_portal.forecast_v2.include_explanation")}</span>
              </label>
            </div>
            <div className="flex items-center gap-2">
              <button
                type="submit"
                disabled={busyAction === "run"}
                className="portal-btn portal-btn--primary w-full"
              >
                {busyAction === "run"
                  ? t("supplier_portal.forecast_v2.running")
                  : t("supplier_portal.forecast_v2.run_forecast")}
              </button>
            </div>
          </form>
        </PortalSection>

        {forecastRun ? (
          <>
            <KpiStatGrid columns={4}>
              <KpiStatCard
                label={t("supplier_portal.forecast_v2.champion_wape")}
                value={formatRateBps(forecastRun.champion_wape_bps)}
                sub={`${forecastRun.champion_model} · Bias ${formatSignedBps(forecastRun.champion_bias_bps ?? 0)}`}
              />
              <KpiStatCard
                label={t("supplier_portal.forecast_v2.challenger_wape")}
                value={formatRateBps(forecastRun.challenger_wape_bps)}
                sub={`${forecastRun.challenger_model} · Bias ${formatSignedBps(forecastRun.challenger_bias_bps ?? 0)}`}
              />
              <KpiStatCard
                label={t("supplier_portal.forecast_v2.winner_model")}
                value={forecastRun.winner_model}
                sub={`${forecastRun.horizon_days}d · ${forecastRun.feature_source || "classic+calendar"}`}
              />
              <KpiStatCard
                label={t("supplier_portal.forecast_v2.wape_improvement")}
                value={formatSignedBps(forecastRun.wape_improvement_bps)}
                sub={`Run ${forecastRun.run_id.slice(0, 12)}`}
              />
            </KpiStatGrid>

            <PortalSection
              icon="calendar"
              title={t("supplier_portal.forecast_v2.regressors_title")}
              description={t("supplier_portal.forecast_v2.intervals_desc")}
            >
              {activeRegressors.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                  No active calendar/promo regressors in this window.
                </p>
              ) : (
                <div className="flex flex-wrap gap-2">
                  {activeRegressors.map((reg) => (
                    <div
                      key={`${reg.name}:${reg.category}`}
                      className="rounded-lg border px-3 py-2 text-xs flex flex-col gap-0.5"
                      style={{
                        borderColor: "var(--desk-border)",
                        background: "var(--desk-surface-raised)",
                      }}
                    >
                      <div className="flex items-center gap-2">
                        <span className="font-semibold" style={{ color: "var(--desk-text-primary)" }}>
                          {reg.name}
                        </span>
                        <StatusBadge state={reg.category} />
                        <span className="font-mono" style={{ color: "var(--desk-accent-strong)" }}>
                          {formatSignedBps(reg.multiplier_bps)}
                        </span>
                      </div>
                      {reg.description ? (
                        <span style={{ color: "var(--desk-text-secondary)" }}>{reg.description}</span>
                      ) : null}
                    </div>
                  ))}
                </div>
              )}

              {v2ChartData.length > 0 ? (
                <div className="mt-4">
                  <h3 className="text-sm font-medium mb-2" style={{ color: "var(--desk-text-primary)" }}>
                    {t("supplier_portal.forecast_v2.intervals_title")}
                  </h3>
                  <div className="h-72">
                    <ResponsiveContainer width="100%" height="100%">
                      <LineChart data={v2ChartData}>
                        <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
                        <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                        <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                        <Tooltip />
                        <Legend />
                        <Line type="monotone" dataKey="p90" name="P90" stroke="var(--desk-warning)" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                        <Line type="monotone" dataKey="p50" name="P50 (Median)" stroke="var(--desk-accent)" strokeWidth={2.5} dot={false} />
                        <Line type="monotone" dataKey="p10" name="P10" stroke="var(--desk-text-secondary)" strokeWidth={1.5} strokeDasharray="4 4" dot={false} />
                        <Line type="monotone" dataKey="baseline" name="Baseline P50" stroke="var(--desk-success)" strokeWidth={1.5} dot={false} />
                      </LineChart>
                    </ResponsiveContainer>
                  </div>
                </div>
              ) : null}
            </PortalSection>

            <PortalSection
              icon="ai"
              title={t("supplier_portal.forecast_v2.explanations_title")}
              description={t("supplier_portal.forecast_v2.explanations_desc")}
            >
              {explanations.length === 0 ? (
                <p className="text-sm" style={{ color: "var(--desk-text-secondary)" }}>
                  No demand explanations generated for this run yet.
                </p>
              ) : (
                <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
                  {explanations.map((exp) => {
                    const adjValue =
                      adjustmentDrafts[exp.explanation_id] ?? String(exp.adjustment_bps);
                    const commentValue = commentDrafts[exp.explanation_id] ?? "";
                    const isPending = exp.status === "PENDING" || exp.status === "PROPOSED";
                    const expBusy = busyAction === `exp-${exp.explanation_id}`;
                    return (
                      <div
                        key={exp.explanation_id}
                        className="rounded-xl border p-4 flex flex-col gap-3"
                        style={{
                          borderColor: "var(--desk-border)",
                          background: "var(--desk-surface-raised)",
                        }}
                      >
                        <div className="flex flex-wrap items-center justify-between gap-2">
                          <div className="flex items-center gap-2">
                            <span className="font-mono text-xs font-semibold">{exp.sku}</span>
                            {exp.location_id ? (
                              <span className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                                · {exp.location_id}
                              </span>
                            ) : null}
                            <StatusBadge state={exp.source} />
                            <StatusBadge state={exp.status} />
                          </div>
                          <span className="text-xs font-mono" style={{ color: "var(--desk-text-secondary)" }}>
                            {exp.model_name}
                          </span>
                        </div>

                        <p className="text-sm" style={{ color: "var(--desk-text-primary)" }}>
                          {exp.summary_text}
                        </p>

                        <div className="grid grid-cols-3 gap-2 text-xs rounded-lg border p-2.5" style={{ borderColor: "var(--desk-border)", background: "var(--desk-surface)" }}>
                          <div>
                            <div style={{ color: "var(--desk-text-secondary)" }}>Baseline P50</div>
                            <div className="font-mono font-semibold">{formatMilliUnits(exp.baseline_p50_milli)}</div>
                          </div>
                          <div>
                            <div style={{ color: "var(--desk-text-secondary)" }}>Adjusted P50</div>
                            <div className="font-mono font-semibold">{formatMilliUnits(exp.adjusted_p50_milli)}</div>
                          </div>
                          <div>
                            <div style={{ color: "var(--desk-text-secondary)" }}>FVA / Adj</div>
                            <div className="font-mono font-semibold">
                              {formatSignedBps(exp.forecast_value_add_bps)} · {formatSignedBps(exp.adjustment_bps)}
                            </div>
                          </div>
                        </div>

                        {(exp.evidence_refs ?? []).length > 0 ? (
                          <div className="flex flex-wrap gap-1.5">
                            {exp.evidence_refs!.map((ev, idx) => (
                              <span
                                key={`${ev.ref_type}-${ev.ref_id}-${idx}`}
                                className="rounded-md border px-2 py-1 text-[11px] font-mono"
                                style={{
                                  borderColor: "var(--desk-border)",
                                  background: "var(--desk-surface)",
                                  color: "var(--desk-text-secondary)",
                                }}
                              >
                                {ev.ref_type}:{ev.ref_id} · {ev.label} ({ev.value_text})
                              </span>
                            ))}
                          </div>
                        ) : null}

                        {isPending ? (
                          <div className="pt-2 border-t flex flex-col gap-2" style={{ borderColor: "var(--desk-border)" }}>
                            <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                              <PortalInput
                                aria-label={`Adjustment bps ${exp.explanation_id}`}
                                type="number"
                                value={adjValue}
                                onChange={(e) =>
                                  setAdjustmentDrafts((prev) => ({
                                    ...prev,
                                    [exp.explanation_id]: e.target.value,
                                  }))
                                }
                                placeholder="Adjustment bps"
                              />
                              <PortalInput
                                aria-label={`Approval comment ${exp.explanation_id}`}
                                value={commentValue}
                                onChange={(e) =>
                                  setCommentDrafts((prev) => ({
                                    ...prev,
                                    [exp.explanation_id]: e.target.value,
                                  }))
                                }
                                placeholder="Review note"
                              />
                            </div>
                            <div className="flex flex-wrap gap-2">
                              <button
                                type="button"
                                disabled={expBusy}
                                className="portal-btn portal-btn--primary text-xs"
                                onClick={() => void handleDecideExplanation(exp, "APPROVE")}
                              >
                                {t("supplier_portal.forecast_v2.approve_adjustment")}
                              </button>
                              <button
                                type="button"
                                disabled={expBusy}
                                className="portal-btn portal-btn--outline text-xs"
                                onClick={() => void handleDecideExplanation(exp, "REJECT")}
                              >
                                {t("supplier_portal.forecast_v2.reject_adjustment")}
                              </button>
                            </div>
                          </div>
                        ) : exp.approved_by ? (
                          <p className="text-xs" style={{ color: "var(--desk-text-secondary)" }}>
                            Reviewed by {exp.approved_by}
                            {exp.approval_note ? ` · ${exp.approval_note}` : ""}
                          </p>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              )}
            </PortalSection>

            {forecastValues.length > 0 ? (
              <PortalSection
                icon="inventory"
                title={t("supplier_portal.forecast_v2.forecast_table_title")}
              >
                <div className="overflow-x-auto">
                  <table className="desk-table w-full text-sm">
                    <thead>
                      <tr style={{ color: "var(--desk-text-secondary)" }}>
                        <th className="p-2.5 text-left">Date</th>
                        <th className="p-2.5 text-left">Location</th>
                        <th className="p-2.5 text-left">SKU</th>
                        <th className="p-2.5 text-left">Model</th>
                        <th className="p-2.5 text-right">P10</th>
                        <th className="p-2.5 text-right">P50</th>
                        <th className="p-2.5 text-right">P90</th>
                        <th className="p-2.5 text-right">Baseline P50</th>
                        <th className="p-2.5 text-right">WAPE</th>
                        <th className="p-2.5 text-right">Confidence</th>
                      </tr>
                    </thead>
                    <tbody>
                      {forecastValues.slice(0, 40).map((row, idx) => (
                        <tr
                          key={`${row.forecast_date}-${row.location_id}-${row.sku}-${row.model}-${idx}`}
                          style={{ borderTop: "1px solid var(--desk-border)" }}
                        >
                          <td className="p-2.5 font-mono text-xs">{row.forecast_date.slice(0, 10)}</td>
                          <td className="p-2.5 font-mono text-xs">{row.location_id}</td>
                          <td className="p-2.5 font-mono text-xs">{row.sku}</td>
                          <td className="p-2.5">
                            <span className="text-xs font-medium">{row.model}</span>
                            {row.is_winner ? (
                              <span className="ml-1.5">
                                <StatusBadge state="APPROVED" label="Winner" />
                              </span>
                            ) : null}
                          </td>
                          <td className="p-2.5 text-right font-mono">{formatMilliUnits(row.p10_milli)}</td>
                          <td className="p-2.5 text-right font-mono font-semibold">{formatMilliUnits(row.p50_milli)}</td>
                          <td className="p-2.5 text-right font-mono">{formatMilliUnits(row.p90_milli)}</td>
                          <td className="p-2.5 text-right font-mono">{formatMilliUnits(row.baseline_p50_milli)}</td>
                          <td className="p-2.5 text-right font-mono">{formatRateBps(row.wape_bps)}</td>
                          <td className="p-2.5 text-right font-mono">{formatRateBps(row.confidence_bps)}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </PortalSection>
            ) : null}
          </>
        ) : (
          <div className="desk-card p-6 text-sm" style={{ color: "var(--desk-text-secondary)" }}>
            {t("supplier_portal.forecast_v2.no_v2_runs")}
          </div>
        )}

        {summary ? (
          <>
            <KpiStatGrid columns={3}>
              <KpiStatCard label={t("supplier_portal.residual.text.predictions")} value={summary.prediction_count} />
              <KpiStatCard label={t("portal.nav.retailers")} value={summary.total_retailers} />
              <KpiStatCard label={t("supplier_portal.residual.text.forecast_units")} value={summary.total_pallets} />
            </KpiStatGrid>
            {accuracy?.enabled ? (
              <KpiStatGrid columns={4}>
                <KpiStatCard label={t("supplier_portal.residual.text.28d_forecast_units")} value={accuracy.forecast_units} />
                <KpiStatCard label={t("supplier_portal.residual.text.28d_actual_units")} value={accuracy.actual_units} />
                <KpiStatCard
                  label={t("supplier_portal.residual.text.wape_28d")}
                  value={accuracy.actual_units > 0 ? pctLabel(accuracy.wape_28) : "—"}
                  sub={
                    accuracy.actual_units > 0
                      ? `Bias ${pctLabel(accuracy.bias_28)} · TS ${accuracy.tracking_signal.toFixed(2)}`
                      : "No scored actuals yet"
                  }
                />
                <KpiStatCard
                  label={t("supplier_portal.residual.text.tracking_alerts")}
                  value={accuracy.alert_count}
                  sub={accuracy.alert_count > 0 ? "|TS| > 4 on one or more series" : "All series in control"}
                />
              </KpiStatGrid>
            ) : null}
            <ForecastConfidenceCard
              confidence={forecastConfidenceFromDemand(summary)}
              updatedAt={formatForecastUpdatedAt(summary.generated_at)}
              stale={isForecastStale(summary.generated_at)}
            />
          </>
        ) : null}

        {legacyChartData.length > 0 ? (
          <section className="desk-card p-6">
            <h2 className="bento-card-title">{t("supplier_portal.analytics.demand.text.baseline_vs_actual_14d")}</h2>
            <div className="h-64 mt-4">
              <ResponsiveContainer width="100%" height="100%">
                <LineChart data={legacyChartData}>
                  <CartesianGrid stroke="var(--desk-border)" strokeDasharray="3 3" />
                  <XAxis dataKey="date" tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                  <YAxis allowDecimals={false} tick={{ fill: "var(--desk-text-secondary)", fontSize: 12 }} />
                  <Tooltip />
                  <Legend />
                  <Line type="monotone" dataKey="baseline" name="Baseline" stroke="var(--desk-accent)" strokeWidth={2} dot={false} />
                  <Line type="monotone" dataKey="actual" name="Actual" stroke="var(--desk-success)" strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          </section>
        ) : null}
      </div>
    </PageChrome>
  );
}
