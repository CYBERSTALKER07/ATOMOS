"use client";

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import {
  AlertTriangle,
  Play,
  RefreshCw,
  Info,
  Loader2,
  History,
  ShoppingCart,
} from "lucide-react";
import { DemandSourceChips } from "@pegasusx/ui-kit/portal";
import { AutoOrderRules } from "@/components/auto-order/AutoOrderRules";
import { AutoOrderList } from "@/components/auto-order/AutoOrderList";
import { PageChrome } from "@/components/PageChrome";
import { useLiveData } from "@/lib/hooks";
import { apiFetch } from "@/lib/auth";
import { getRetailerId } from "@/lib/retailer-profile";
import type {
  AutoOrderExecutionMode,
  AutoOrderRun,
  AutoOrderSettings,
  AutoOrderShadowProposal,
  Prediction,
} from "@/lib/types";

type RetailerReorderSuggestion = {
  sku: string;
  suggested_qty: number;
  adjusted_demand_per_day?: number;
  current_stock?: number;
  sources?: string[];
  sell_through_velocity?: number;
  status?: string;
};

export default function AutoOrderPage() {
  const t = usePortalT();
  const {
    data: settings,
    loading: settingsLoading,
    error: settingsError,
    isRefreshing: isSettingsRefreshing,
    mutate: mutateSettings,
  } = useLiveData<AutoOrderSettings>("/v1/retailer/settings/auto-order");

  const [predictions, setPredictions] = useState<Prediction[]>([]);
  const [predictionsLoading, setPredictionsLoading] = useState(true);
  const [predictionsError, setPredictionsError] = useState<Error | null>(null);

  const [syncMessage, setSyncMessage] = useState<string | null>(null);
  const [runs, setRuns] = useState<AutoOrderRun[]>([]);
  const [runsLoading, setRunsLoading] = useState(true);
  const [runsError, setRunsError] = useState<string | null>(null);
  const [running, setRunning] = useState(false);
  const [runningMode, setRunningMode] = useState<AutoOrderExecutionMode | null>(
    null,
  );
  const [lastRun, setLastRun] = useState<AutoOrderRun | null>(null);
  const [reorderSuggestions, setReorderSuggestions] = useState<
    RetailerReorderSuggestion[]
  >([]);
  const [shadowProposals, setShadowProposals] = useState<
    AutoOrderShadowProposal[]
  >([]);
  const [placeConfirmOpen, setPlaceConfirmOpen] = useState(false);

  const retailerId = getRetailerId();
  const executionMode: AutoOrderExecutionMode = (() => {
    const m = settings?.execution_mode;
    if (m === "off" || m === "shadow" || m === "draft" || m === "place") return m;
    return settings?.global_enabled ? "draft" : "off";
  })();

  const fetchPredictions = useCallback(async () => {
    if (!retailerId) return;
    setPredictionsLoading(true);
    setPredictionsError(null);
    try {
      const res = await apiFetch(`/v1/ai/predictions?retailer_id=${retailerId}`);
      if (!res.ok) throw new Error("Predictions fetch failed");
      const data = await res.json();
      setPredictions(Array.isArray(data) ? data : []);
    } catch (err: unknown) {
      setPredictionsError(err instanceof Error ? err : new Error("Predictions fetch failed"));
    } finally {
      setPredictionsLoading(false);
    }
  }, [retailerId]);

  const fetchRuns = useCallback(async () => {
    setRunsLoading(true);
    setRunsError(null);
    try {
      const res = await apiFetch("/v1/retailer/settings/auto-order/runs");
      if (!res.ok) {
        throw new Error(`runs_failed_${res.status}`);
      }
      const data = (await res.json()) as { items?: AutoOrderRun[] };
      setRuns(Array.isArray(data.items) ? data.items : []);
    } catch (err) {
      setRunsError(err instanceof Error ? err.message : t("retailer_desktop.residual.text.could_not_load_runs"));
      setRuns([]);
    } finally {
      setRunsLoading(false);
    }
  }, []);

  const fetchReorderSuggestions = useCallback(async () => {
    try {
      const res = await apiFetch("/v1/retailer/reorder-suggestions");
      if (!res.ok) {
        setReorderSuggestions([]);
        return;
      }
      const data = (await res.json()) as { items?: RetailerReorderSuggestion[] };
      setReorderSuggestions(Array.isArray(data.items) ? data.items : []);
    } catch {
      setReorderSuggestions([]);
    }
  }, []);

  const fetchShadowProposals = useCallback(async () => {
    try {
      const res = await apiFetch(
        "/v1/retailer/settings/auto-order/shadow-proposals",
      );
      if (!res.ok) {
        setShadowProposals([]);
        return;
      }
      const data = (await res.json()) as { items?: AutoOrderShadowProposal[] };
      setShadowProposals(Array.isArray(data.items) ? data.items : []);
    } catch {
      setShadowProposals([]);
    }
  }, []);

  useEffect(() => {
    void fetchPredictions();
  }, [fetchPredictions]);

  useEffect(() => {
    void fetchRuns();
  }, [fetchRuns]);

  useEffect(() => {
    void fetchReorderSuggestions();
  }, [fetchReorderSuggestions]);

  useEffect(() => {
    void fetchShadowProposals();
  }, [fetchShadowProposals]);

  const refreshAll = useCallback(() => {
    setSyncMessage(null);
    void mutateSettings();
    void fetchPredictions();
    void fetchRuns();
    void fetchReorderSuggestions();
    void fetchShadowProposals();
  }, [
    mutateSettings,
    fetchPredictions,
    fetchRuns,
    fetchReorderSuggestions,
    fetchShadowProposals,
  ]);

  const runAutoOrder = async (mode: "shadow" | "draft" | "place") => {
    setRunning(true);
    setRunningMode(mode);
    setSyncMessage(null);
    setLastRun(null);
    setPlaceConfirmOpen(false);
    try {
      const res = await apiFetch(
        `/v1/retailer/settings/auto-order/run?mode=${mode}`,
        { method: "POST" },
      );
      const json = (await res.json().catch(() => ({}))) as AutoOrderRun & {
        error?: string;
        permission?: string;
      };
      if (!res.ok) {
        const detail =
          json.error === "forbidden" && json.permission
            ? `Missing permission: ${json.permission}`
            : json.error === "place_requires_manager"
              ? "Place requires OWNER, ADMIN, or MANAGER"
              : json.error || `run_failed_${res.status}`;
        throw new Error(detail);
      }
      setLastRun(json);
      const placed = json.placed_lines ?? 0;
      const orders = json.placed_orders?.length ?? 0;
      if (mode === "place") {
        setSyncMessage(
          placed > 0
            ? `Place run: ${placed} line(s) in ${orders} order(s)` +
                (json.message ? ` — ${json.message}` : "")
            : `Place run ${json.status}${json.message ? `: ${json.message}` : ""}`,
        );
      } else if (mode === "shadow") {
        setSyncMessage(
          json.status === "OK" || json.status === "PARTIAL"
            ? `Shadow run: ${json.draft_lines} proposal(s) recorded (no orders)` +
                (json.message ? ` — ${json.message}` : "")
            : `Shadow ${json.status}${json.message ? `: ${json.message}` : ""}`,
        );
      } else {
        setSyncMessage(
          json.status === "OK" || json.status === "PARTIAL"
            ? `Draft run complete: ${json.draft_lines} line(s)` +
                (json.message ? ` — ${json.message}` : "")
            : `Run ${json.status}${json.message ? `: ${json.message}` : ""}`,
        );
      }
      await fetchRuns();
      await fetchReorderSuggestions();
      await fetchShadowProposals();
    } catch (err) {
      setSyncMessage(err instanceof Error ? err.message : t("retailer_desktop.residual.text.auto_order_run_failed"));
    } finally {
      setRunning(false);
      setRunningMode(null);
    }
  };

  const setExecutionMode = async (mode: AutoOrderExecutionMode) => {
    try {
      const res = await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          execution_mode: mode,
          global_enabled: mode !== "off",
        }),
      });
      if (!res.ok) {
        const json = (await res.json().catch(() => ({}))) as {
          error?: string;
        };
        throw new Error(json.error || "execution_mode_update_failed");
      }
      await mutateSettings();
      const labels: Record<AutoOrderExecutionMode, string> = {
        off: "Auto-order off.",
        shadow: "Mode: Shadow (proposals only — recommended).",
        draft: "Mode: Draft cart lines.",
        place: "Mode: Place (still requires Place now + server flag).",
      };
      setSyncMessage(labels[mode]);
    } catch (err) {
      setSyncMessage(
        err instanceof Error ? err.message : t("retailer_desktop.residual.text.failed_to_update_execution_mode"),
      );
    }
  };

  const toggleGlobal = async (enabled: boolean, useHistory?: boolean) => {
    try {
      const res = await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ global_enabled: enabled, use_history: useHistory }),
      });
      if (!res.ok) throw new Error("Update failed");
      await mutateSettings();
    } catch {
      setSyncMessage("Failed to update global auto-order.");
    }
  };

  const toggleScoped = async (
    level: "supplier" | "category" | "product" | "variant",
    id: string,
    enabled: boolean,
    useHistory?: boolean
  ) => {
    try {
      const res = await apiFetch(`/v1/retailer/settings/auto-order/${level}/${id}`, {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ enabled, use_history: useHistory }),
      });
      if (!res.ok) throw new Error("Update failed");
      await mutateSettings();
    } catch (err) {
      setSyncMessage(`Failed to update ${level} override.`);
    }
  };

  const [pendingAction, setPendingAction] = useState<{
    type: "global" | "supplier" | "category" | "product" | "variant";
    id?: string;
  } | null>(null);

  const handleToggle = (
    type: "global" | "supplier" | "category" | "product" | "variant",
    enabled: boolean,
    hasHistory: boolean,
    id?: string
  ) => {
    if (enabled && hasHistory) {
      setPendingAction({ type, id });
    } else {
      if (type === "global") {
        void toggleGlobal(enabled, false);
      } else if (id) {
        void toggleScoped(type, id, enabled, false);
      }
    }
  };

  const confirmAction = (useHistory: boolean) => {
    if (!pendingAction) return;
    const { type, id } = pendingAction;
    if (type === "global") {
      void toggleGlobal(true, useHistory);
    } else if (id) {
      void toggleScoped(type, id, true, useHistory);
    }
    setPendingAction(null);
  };

  const isLoading = settingsLoading || predictionsLoading;

  return (
    <div
      className="min-h-full p-6 md:p-8"
      style={{ background: "var(--desk-canvas)" }}
    >
      <PageChrome
        icon="wand.and.stars"
        title={t("retailer_desktop.auto_order.text.auto_order_engine")}
        description={t("retailer_desktop.residual.text.empathy_engine_intelligence_with_5_level_granular_control")}
        loading={isLoading}
        skeletonVariant="form"
        actions={
          <div className="flex flex-wrap items-center gap-3">
            <button
              type="button"
              disabled={running || isLoading || executionMode === "off"}
              onClick={() => void runAutoOrder("shadow")}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light inline-flex items-center gap-2 disabled:opacity-60"
            >
              {running && runningMode === "shadow" ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                <Play size={16} />
              )}
              {running && runningMode === "shadow" ? "Shadowing…" : "Shadow now"}
            </button>
            <button
              type="button"
              disabled={running || isLoading || executionMode === "off"}
              onClick={() => void runAutoOrder("draft")}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light inline-flex items-center gap-2 disabled:opacity-60"
            >
              {running && runningMode === "draft" ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                <Play size={16} />
              )}
              {running && runningMode === "draft" ? "Drafting…" : "Draft now"}
            </button>
            <button
              type="button"
              disabled={running || isLoading || executionMode === "off"}
              onClick={() => setPlaceConfirmOpen(true)}
              className="portal-btn portal-btn--primary h-11 px-5 rounded-xl font-light inline-flex items-center gap-2 disabled:opacity-60"
            >
              {running && runningMode === "place" ? (
                <Loader2 size={16} className="animate-spin" />
              ) : (
                <ShoppingCart size={16} />
              )}
              {running && runningMode === "place" ? "Placing…" : "Place now"}
            </button>
            <button
              type="button"
              disabled={isLoading || isSettingsRefreshing || running}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isSettingsRefreshing ? "animate-spin" : ""}`}
              />
              {isSettingsRefreshing ? "Syncing" : "Sync"}
            </button>
          </div>
        }
      >
        {placeConfirmOpen && (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center p-4"
            style={{ background: "rgba(0,0,0,0.45)" }}
            role="dialog"
            aria-modal="true"
            aria-labelledby="place-confirm-title"
          >
            <div className="max-w-md w-full rounded-2xl border border-[var(--desk-border)] bg-[var(--desk-surface)] p-6 shadow-xl">
              <h2
                id="place-confirm-title"
                className="text-lg font-medium text-[var(--desk-text-primary)] mb-2"
              >
                Create real supplier orders?
              </h2>
              <p className="text-sm text-[var(--desk-text-secondary)] mb-4">
                Place mode creates real procurement orders via the order
                aggregate (AUTO_ORDER). Requires primary location geo, place
                permission, and server flag{" "}
                <code className="text-xs">AUTO_ORDER_PLACE_ENABLED</code>. This
                cannot be undone as a draft.
              </p>
              <div className="flex justify-end gap-2">
                <button
                  type="button"
                  className="portal-btn portal-btn--ghost h-10 px-4 rounded-xl"
                  onClick={() => setPlaceConfirmOpen(false)}
                  disabled={running}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="portal-btn portal-btn--primary h-10 px-4 rounded-xl"
                  disabled={running}
                  onClick={() => void runAutoOrder("place")}
                >
                  Confirm place
                </button>
              </div>
            </div>
          </div>
        )}

        {syncMessage && (
          <div
            className={`mb-6 flex items-center gap-2 p-3 rounded-xl border ${
              lastRun?.status === "OK" || lastRun?.status === "PARTIAL"
                ? "bg-[var(--desk-accent)]/10 text-[var(--desk-accent)] border-[var(--desk-accent)]/30"
                : "bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border-[var(--desk-warning)]/30"
            }`}
          >
            <AlertTriangle size={16} />
            <span className="md-typescale-body-small">{syncMessage}</span>
          </div>
        )}

        {settingsError && (
          <div className="mb-6 flex items-center gap-2 p-3 rounded-xl bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border border-[var(--desk-warning)]/30">
            <AlertTriangle size={16} />
            <span className="md-typescale-body-small">
              Settings unavailable: {settingsError.message}
            </span>
          </div>
        )}

        {predictionsError && (
          <div className="mb-6 flex items-center gap-2 p-3 rounded-xl bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border border-[var(--desk-warning)]/30">
            <AlertTriangle size={16} />
            <span className="md-typescale-body-small">
              Predictions unavailable: {predictionsError.message}
            </span>
          </div>
        )}

        <div className="mb-8 p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)] mb-1">
            Execution mode
          </h3>
          <p className="text-xs text-[var(--desk-text-tertiary)] mb-4">
            How aggressive globally. Scopes below choose which SKUs participate. Off disables
            worker action. Shadow is recommended until acceptance looks good.
          </p>
          <div className="flex flex-wrap items-center gap-2">
            {(
              [
                ["off", "Off"],
                ["shadow", "Shadow"],
                ["draft", "Draft cart"],
                ["place", "Place orders"],
              ] as const
            ).map(([mode, label]) => (
              <button
                key={mode}
                type="button"
                className={`text-xs px-3 py-1.5 rounded-lg border ${
                  executionMode === mode
                    ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/10 text-[var(--desk-accent)]"
                    : "border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
                }`}
                onClick={() => void setExecutionMode(mode)}
              >
                {label}
                {mode === "place" ? " *" : mode === "shadow" ? " ✓" : ""}
              </button>
            ))}
          </div>
          <p className="mt-3 text-xs text-[var(--desk-text-tertiary)]">
            * Place still needs AUTO_ORDER_PLACE_ENABLED + manager permission.
            {settings?.shadow_stats
              ? ` · 30d WAPE ${(settings.shadow_stats.wape * 100).toFixed(0)}% · accept ${(settings.shadow_stats.unmodified_accept_rate * 100).toFixed(0)}% (${settings.shadow_stats.proposal_count} proposals)`
              : ""}
          </p>
        </div>

        <div className="mb-8 p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between gap-3 mb-4">
            <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
              Shadow inbox
            </h3>
            <span className="text-xs text-[var(--desk-text-tertiary)]">
              Inventory (R,s,S) proposals — no cart or orders
            </span>
          </div>
          {shadowProposals.length === 0 ? (
            <p className="text-sm text-[var(--desk-text-secondary)]">
              No shadow proposals yet. Set mode to Shadow and run Shadow now.
            </p>
          ) : (
            <div className="space-y-2">
              {shadowProposals.slice(0, 12).map((p) => (
                <div
                  key={p.proposal_id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)] px-4 py-3"
                >
                  <p className="text-sm font-light text-[var(--desk-text-primary)]">
                    {p.sku} · qty {p.proposed_qty}
                    {` · IP ${p.ip.toFixed?.(0) ?? p.ip}`}
                    {` · ROP ${p.reorder_point.toFixed?.(0) ?? p.reorder_point}`}
                    {` · S ${p.order_up_to.toFixed?.(0) ?? p.order_up_to}`}
                  </p>
                  <span className="text-xs text-[var(--desk-text-tertiary)]">
                    {p.bucket_date}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Reorder suggestions (sell-through aware) */}
        <div className="mb-8 p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between gap-3 mb-4">
            <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
              Reorder suggestions
            </h3>
            <span className="text-xs text-[var(--desk-text-tertiary)]">
              Feeds Run auto-order when OPEN (Store POS / Wholesale)
            </span>
          </div>
          {reorderSuggestions.length === 0 ? (
            <p className="text-sm text-[var(--desk-text-secondary)]">
              No OPEN suggestions yet. POS sell-through and demand batch populate this list.
            </p>
          ) : (
            <div className="space-y-2">
              {reorderSuggestions.slice(0, 12).map((row) => (
                <div
                  key={row.sku}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)] px-4 py-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-light text-[var(--desk-text-primary)]">
                      {row.sku} · qty {row.suggested_qty}
                      {row.current_stock != null ? ` · stock ${row.current_stock}` : ""}
                    </p>
                    <div className="mt-1">
                      <DemandSourceChips sources={row.sources} />
                    </div>
                  </div>
                  {row.sell_through_velocity != null && row.sell_through_velocity > 0 && (
                    <span className="text-xs text-[var(--desk-text-tertiary)]">
                      POS vel {row.sell_through_velocity.toFixed(1)}/d
                    </span>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Worker: run audit */}
        <div className="mb-8 p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
          <div className="flex items-center justify-between gap-3 mb-4">
            <div className="flex items-center gap-2">
              <History size={16} className="text-[var(--desk-accent)]" />
              <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">
                Last runs
              </h3>
            </div>
            <span className="text-xs text-[var(--desk-text-tertiary)]">
              Draft stages cart · Place creates real orders (flag + geo required)
            </span>
          </div>

          {lastRun && (
            <div className="mb-4 rounded-xl border border-[var(--desk-accent)]/30 bg-[var(--desk-accent)]/5 px-4 py-3">
              <p className="text-sm font-light text-[var(--desk-text-primary)]">
                Latest: {lastRun.mode} · draft {lastRun.draft_lines}
                {(lastRun.placed_lines ?? 0) > 0
                  ? ` · placed ${lastRun.placed_lines}`
                  : ""}{" "}
                · {lastRun.status}
                {lastRun.candidate_source
                  ? ` · via ${lastRun.candidate_source}`
                  : ""}
              </p>
              {lastRun.message && (
                <p className="text-xs text-[var(--desk-text-secondary)] mt-1">
                  {lastRun.message}
                </p>
              )}
              {(lastRun.placed_orders?.length ?? 0) > 0 && (
                <ul className="mt-2 space-y-1">
                  {lastRun.placed_orders!.map((po) => (
                    <li key={po.order_id} className="text-xs">
                      <Link
                        href={`/orders`}
                        className="text-[var(--desk-accent)] underline-offset-2 hover:underline"
                      >
                        {po.order_id}
                      </Link>
                      {po.supplier_id ? ` · ${po.supplier_id}` : ""} ·{" "}
                      {po.line_count} line(s)
                      {po.skus?.length ? ` · ${po.skus.join(", ")}` : ""}
                    </li>
                  ))}
                </ul>
              )}
              {(lastRun.skipped?.length ?? 0) > 0 && (
                <p className="text-xs text-[var(--desk-text-tertiary)] mt-1">
                  Skipped:{" "}
                  {lastRun.skipped!
                    .slice(0, 5)
                    .map((s) => `${s.sku || "—"} (${s.reason})`)
                    .join(", ")}
                  {(lastRun.skipped?.length ?? 0) > 5 ? "…" : ""}
                </p>
              )}
            </div>
          )}

          {runsLoading ? (
            <div className="flex justify-center py-8">
              <Loader2 size={20} className="animate-spin text-[var(--desk-text-tertiary)]" />
            </div>
          ) : runsError ? (
            <p className="text-sm text-[var(--desk-warning)]">{runsError}</p>
          ) : runs.length === 0 ? (
            <p className="text-sm text-[var(--desk-text-secondary)]">
              No runs yet. Enable auto-order and use{" "}
              <span className="font-medium">{t("retailer_desktop.auto_order.text.draft_now")}</span> or{" "}
              <span className="font-medium">{t("retailer_desktop.auto_order.text.place_now")}</span>.
            </p>
          ) : (
            <div className="space-y-2">
              {runs.map((run) => (
                <div
                  key={run.run_id}
                  className="flex flex-wrap items-center justify-between gap-2 rounded-xl border border-[var(--desk-border)] bg-[var(--desk-canvas)] px-4 py-3"
                >
                  <div className="min-w-0">
                    <p className="text-sm font-light text-[var(--desk-text-primary)]">
                      {run.schedule_bucket || run.started_at?.slice(0, 10)} ·{" "}
                      {run.mode} · draft {run.draft_lines}
                      {(run.placed_lines ?? 0) > 0
                        ? ` · placed ${run.placed_lines}`
                        : ""}
                    </p>
                    <p className="text-xs text-[var(--desk-text-tertiary)]">
                      {run.started_at}
                      {run.message ? ` · ${run.message}` : ""}
                    </p>
                  </div>
                  <span
                    className={`text-xs font-medium px-2 py-1 rounded-lg ${
                      run.status === "OK" || run.status === "PARTIAL"
                        ? "bg-emerald-500/10 text-emerald-700"
                        : "bg-amber-500/10 text-amber-700"
                    }`}
                  >
                    {run.status}
                  </span>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="space-y-6">
          <AutoOrderRules settings={settings ?? undefined} handleToggle={handleToggle} />
          <AutoOrderList predictions={predictions} />

          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)] mt-6">
            <div className="flex items-center gap-2 mb-4">
              <Info size={16} className="text-[var(--desk-accent)]" />
              <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">{t("retailer_desktop.auto_order.text.how_it_works")}</h3>
            </div>
            <ol className="list-decimal list-inside space-y-2 text-[var(--desk-text-secondary)] md-typescale-body-small">
              <li>{t("retailer_desktop.auto_order.text.the_ai_analyzes_your_purchase_patterns_even_when_auto_order_is_o")}</li>
              <li>{t("retailer_desktop.auto_order.text.when_you_enable_choose_to_use_your_history_or_start_fresh")}</li>
              <li>{t("retailer_desktop.auto_order.text.starting_fresh_requires_at_least_2_orders_per_product")}</li>
              <li>{t("retailer_desktop.auto_order.text.overrides_size_variant_and_gt_product_and_gt_category_and_gt_sup")}</li>
              <li>
                Modes: Shadow records proposals only; Draft stages cart; Place creates
                AUTO_ORDER when the server flag and manager permission allow.
              </li>
            </ol>
          </div>
        </div>

        {pendingAction && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
            <div className="bg-[var(--desk-surface)] w-full max-w-sm rounded-2xl p-6 shadow-xl border border-[var(--desk-border)]">
              <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)] mb-2">{t("retailer_desktop.auto_order.text.use_previous_analytics")}</h3>
              <p className="md-typescale-body-medium text-[var(--desk-text-secondary)] mb-6">
                Enable this auto-order using your existing order history, or start fresh? Starting fresh requires at least 2 orders before predictions begin.
              </p>
              <div className="flex items-center gap-3">
                <button
                  type="button"
                  className="flex-1 h-10 rounded-lg border border-[var(--desk-border)] md-typescale-label-large font-light hover:bg-[var(--desk-surface-subtle)]"
                  onClick={() => setPendingAction(null)}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  className="flex-1 h-10 rounded-lg bg-[var(--desk-danger)] text-white md-typescale-label-large font-light hover:bg-[var(--desk-danger)]/90"
                  onClick={() => confirmAction(false)}
                >
                  Start Fresh
                </button>
                <button
                  type="button"
                  className="flex-1 h-10 rounded-lg bg-[var(--desk-accent)] text-white md-typescale-label-large font-light hover:bg-[var(--desk-accent)]/90"
                  onClick={() => confirmAction(true)}
                >
                  Use History
                </button>
              </div>
            </div>
          </div>
        )}
      </PageChrome>
    </div>
  );
}
