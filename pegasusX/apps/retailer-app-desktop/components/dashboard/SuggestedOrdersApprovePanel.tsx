"use client";

import { useCallback, useEffect, useMemo, useState } from "react";
import { Check, Loader2, RefreshCw, Sparkles } from "lucide-react";
import { PageSection } from "@/components/PageSection";
import { DemandSourceChips } from "@/components/ui/DemandSourceChips";
import { usePortalT } from "@/lib/i18n";
import {
  approveReplenishmentSuggestion,
  getRetailerPlanningForecast,
  getRetailerReorderSuggestions,
  type ApproveReplenishmentSuggestionResponse,
  type RetailerForecastRun,
  type RetailerForecastValue,
  type RetailerReorderSuggestionItem,
} from "@/lib/api";
import { formatBpsPercent } from "@/lib/money";
import { getRetailerId } from "@/lib/retailer-profile";

function formatMilli(milli: number): string {
  const safe = BigInt(Math.trunc(milli || 0));
  const neg = safe < BigInt(0);
  const abs = neg ? -safe : safe;
  const whole = abs / BigInt(1000);
  const rem = abs % BigInt(1000);
  const frac = rem === BigInt(0) ? "" : `.${rem.toString().padStart(3, "0").replace(/0+$/, "")}`;
  return `${neg ? "-" : ""}${whole.toString()}${frac}`;
}

export function SuggestedOrdersApprovePanel({
  onApproved,
}: {
  onApproved?: (res: ApproveReplenishmentSuggestionResponse) => void;
}) {
  const t = usePortalT();
  const retailerId = getRetailerId();

  const [suggestions, setSuggestions] = useState<RetailerReorderSuggestionItem[]>([]);
  const [forecastRun, setForecastRun] = useState<RetailerForecastRun | null>(null);
  const [forecastValues, setForecastValues] = useState<RetailerForecastValue[]>([]);
  const [qtyDrafts, setQtyDrafts] = useState<Record<string, string>>({});
  const [approvedMap, setApprovedMap] = useState<
    Record<string, ApproveReplenishmentSuggestionResponse>
  >({});
  const [loading, setLoading] = useState<boolean>(true);
  const [busyKey, setBusyKey] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);

  const loadData = useCallback(async () => {
    setLoading(true);
    setError(null);
    const [sugRes, fcRes] = await Promise.allSettled([
      getRetailerReorderSuggestions("OPEN"),
      getRetailerPlanningForecast({ horizon_days: 14, limit: 24 }),
    ]);

    if (sugRes.status === "fulfilled") {
      setSuggestions(sugRes.value);
    } else {
      setError(
        sugRes.reason instanceof Error
          ? sugRes.reason.message
          : "Failed to load reorder suggestions",
      );
    }

    if (fcRes.status === "fulfilled") {
      setForecastRun(fcRes.value.run ?? null);
      setForecastValues(fcRes.value.forecasts ?? []);
    }

    setLoading(false);
  }, []);

  useEffect(() => {
    void loadData();
  }, [loadData]);

  const forecastBySku = useMemo(() => {
    const map = new Map<string, RetailerForecastValue>();
    for (const row of forecastValues) {
      if (!map.has(row.sku) || row.is_winner) {
        map.set(row.sku, row);
      }
    }
    return map;
  }, [forecastValues]);

  const handleOneTapApprove = async (row: RetailerReorderSuggestionItem) => {
    const targetId =
      row.suggestion_id ||
      (row.retailer_id
        ? `${row.retailer_id}:${row.sku}`
        : retailerId
          ? `${retailerId}:${row.sku}`
          : row.sku);
    const draftStr = qtyDrafts[row.sku];
    const parsedQty =
      draftStr !== undefined && draftStr.trim() !== ""
        ? Number.parseInt(draftStr.trim(), 10)
        : row.suggested_qty;

    setBusyKey(row.sku);
    setError(null);
    setNotice(null);
    try {
      const res = await approveReplenishmentSuggestion(targetId, {
        retailer_id: row.retailer_id || retailerId || undefined,
        sku: row.sku,
        qty: Number.isFinite(parsedQty) && parsedQty > 0 ? parsedQty : row.suggested_qty,
        mode: "order",
      });
      setApprovedMap((prev) => ({ ...prev, [row.sku]: res }));
      setNotice(
        t("retailer_desktop.suggested_orders.approved_notice", {
          sku: res.sku,
          qty: res.approved_qty,
          status: res.status,
        }),
      );
      onApproved?.(res);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Approval failed");
    } finally {
      setBusyKey(null);
    }
  };

  const winnerWape = forecastRun
    ? `${formatBpsPercent(
        forecastRun.winner_model === forecastRun.challenger_model
          ? forecastRun.challenger_wape_bps
          : forecastRun.champion_wape_bps,
        1,
      )}%`
    : null;

  return (
    <PageSection
      title={t("retailer_desktop.suggested_orders.title")}
      description={t("retailer_desktop.suggested_orders.description")}
      actions={
        <button
          type="button"
          onClick={() => void loadData()}
          disabled={loading}
          className="portal-btn portal-btn--ghost h-9 px-3 rounded-xl text-xs inline-flex items-center gap-1.5"
        >
          <RefreshCw size={13} className={loading ? "animate-spin" : ""} />
          Sync
        </button>
      }
    >
      {forecastRun ? (
        <div
          data-testid="retailer-forecast-summary"
          className="rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface-subtle)] p-3 flex flex-wrap items-center justify-between gap-2 text-xs"
        >
          <div className="flex items-center gap-2">
            <Sparkles size={14} className="text-[var(--desk-accent)]" />
            <span className="font-medium text-[var(--desk-text-primary)]">
              {t("retailer_desktop.suggested_orders.forecast_winner", {
                model: forecastRun.winner_model,
                wape: winnerWape || "—",
              })}
            </span>
          </div>
          {(forecastRun.regressors ?? []).length > 0 ? (
            <div className="flex flex-wrap gap-1.5">
              {forecastRun.regressors!.slice(0, 4).map((reg) => (
                <span
                  key={reg.name}
                  className="rounded-md border border-[var(--desk-border)] bg-[var(--desk-surface)] px-2 py-0.5 font-mono text-[10px]"
                >
                  {reg.name}
                </span>
              ))}
            </div>
          ) : null}
        </div>
      ) : null}

      {notice ? (
        <div
          role="status"
          className="rounded-xl border border-emerald-500/30 bg-emerald-500/10 px-3 py-2 text-xs text-emerald-700"
        >
          {notice}
        </div>
      ) : null}

      {error ? (
        <div
          role="alert"
          className="rounded-xl border border-red-500/30 bg-red-500/10 px-3 py-2 text-xs text-red-600"
        >
          {error}
        </div>
      ) : null}

      {loading && suggestions.length === 0 ? (
        <p className="text-xs text-[var(--desk-text-secondary)]">Loading suggested orders...</p>
      ) : suggestions.length === 0 ? (
        <p className="text-xs text-[var(--desk-text-secondary)]">
          {t("retailer_desktop.suggested_orders.no_suggestions")}
        </p>
      ) : (
        <div className="space-y-2.5">
          {suggestions.slice(0, 10).map((row) => {
            const approved = approvedMap[row.sku];
            const fc = forecastBySku.get(row.sku);
            const qtyVal = qtyDrafts[row.sku] ?? String(row.suggested_qty);
            const isBusy = busyKey === row.sku;
            return (
              <div
                key={row.suggestion_id || row.sku}
                className="flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)] px-4 py-3"
              >
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="text-sm font-medium text-[var(--desk-text-primary)]">
                      {row.sku_name ? `${row.sku_name} (${row.sku})` : row.sku}
                    </span>
                    {approved ? (
                      <span className="rounded-full bg-emerald-500/15 text-emerald-700 px-2 py-0.5 text-[10px] font-semibold">
                        {approved.status}
                        {approved.order_id ? ` · ${approved.order_id}` : ""}
                      </span>
                    ) : null}
                  </div>
                  <p className="text-xs text-[var(--desk-text-secondary)] mt-0.5">
                    Suggested {row.suggested_qty}
                    {row.current_stock != null ? ` · Stock ${row.current_stock}` : ""}
                    {row.safety_stock != null ? ` · SS ${Math.round(row.safety_stock)}` : ""}
                    {fc
                      ? ` · P10–P90 ${formatMilli(fc.p10_milli)} / ${formatMilli(fc.p50_milli)} / ${formatMilli(fc.p90_milli)}`
                      : ""}
                  </p>
                  <div className="mt-1">
                    <DemandSourceChips sources={row.sources} />
                  </div>
                </div>

                <div className="flex items-center gap-2 shrink-0">
                  <input
                    type="number"
                    min={1}
                    aria-label={`Quantity for ${row.sku}`}
                    value={qtyVal}
                    onChange={(e) =>
                      setQtyDrafts((prev) => ({ ...prev, [row.sku]: e.target.value }))
                    }
                    disabled={Boolean(approved) || isBusy}
                    className="w-20 h-9 rounded-lg border border-[var(--desk-border)] bg-[var(--desk-surface)] px-2 text-xs font-mono text-right"
                  />
                  <button
                    type="button"
                    disabled={Boolean(approved) || isBusy}
                    onClick={() => void handleOneTapApprove(row)}
                    className="portal-btn portal-btn--primary h-9 px-3 rounded-lg text-xs inline-flex items-center gap-1.5 disabled:opacity-50"
                  >
                    {isBusy ? (
                      <>
                        <Loader2 size={13} className="animate-spin" />
                        {t("retailer_desktop.suggested_orders.approving")}
                      </>
                    ) : approved ? (
                      <>
                        <Check size={13} />
                        {approved.status}
                      </>
                    ) : (
                      <>
                        <Check size={13} />
                        {t("retailer_desktop.suggested_orders.approve_one_tap")}
                      </>
                    )}
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </PageSection>
  );
}
