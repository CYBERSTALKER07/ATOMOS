"use client";

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
import { BentoGrid, BentoCard } from "@/components/BentoGrid";
import { useLiveData } from "@/lib/hooks";
import { apiFetch } from "@/lib/auth";
import { getRetailerId } from "@/lib/retailer-profile";
import type {
  AutoOrderRun,
  AutoOrderSettings,
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
  const [runningMode, setRunningMode] = useState<"draft" | "place" | null>(null);
  const [lastRun, setLastRun] = useState<AutoOrderRun | null>(null);
  const [reorderSuggestions, setReorderSuggestions] = useState<
    RetailerReorderSuggestion[]
  >([]);
  const [placeConfirmOpen, setPlaceConfirmOpen] = useState(false);

  const retailerId = getRetailerId();
  const executionMode =
    settings?.execution_mode === "place" ? "place" : "draft";

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
      setRunsError(err instanceof Error ? err.message : "Could not load runs");
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

  useEffect(() => {
    void fetchPredictions();
  }, [fetchPredictions]);

  useEffect(() => {
    void fetchRuns();
  }, [fetchRuns]);

  useEffect(() => {
    void fetchReorderSuggestions();
  }, [fetchReorderSuggestions]);

  const refreshAll = useCallback(() => {
    setSyncMessage(null);
    void mutateSettings();
    void fetchPredictions();
    void fetchRuns();
    void fetchReorderSuggestions();
  }, [mutateSettings, fetchPredictions, fetchRuns, fetchReorderSuggestions]);

  const runAutoOrder = async (mode: "draft" | "place") => {
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
    } catch (err) {
      setSyncMessage(err instanceof Error ? err.message : "Auto-order run failed");
    } finally {
      setRunning(false);
      setRunningMode(null);
    }
  };

  const setExecutionMode = async (mode: "draft" | "place") => {
    try {
      const res = await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ execution_mode: mode }),
      });
      if (!res.ok) {
        const json = (await res.json().catch(() => ({}))) as {
          error?: string;
        };
        throw new Error(json.error || "execution_mode_update_failed");
      }
      await mutateSettings();
      setSyncMessage(
        mode === "place"
          ? "Default execution set to Place (still requires Place now + confirm)."
          : "Default execution set to Draft.",
      );
    } catch (err) {
      setSyncMessage(
        err instanceof Error ? err.message : "Failed to update execution mode",
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
        title="Auto-Order Engine"
        description="Empathy Engine Intelligence with 5-level granular control."
        loading={isLoading}
        skeletonVariant="form"
        actions={
          <div className="flex flex-wrap items-center gap-3">
            <button
              type="button"
              disabled={running || isLoading}
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
              disabled={running || isLoading}
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

        <BentoGrid className="mb-8">
          <BentoCard interactive={false}>
            <div className="flex flex-col gap-1">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                Execution default
              </span>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  className={`text-xs px-3 py-1.5 rounded-lg border ${
                    executionMode === "draft"
                      ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/10 text-[var(--desk-accent)]"
                      : "border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
                  }`}
                  onClick={() => void setExecutionMode("draft")}
                >
                  Draft
                </button>
                <button
                  type="button"
                  className={`text-xs px-3 py-1.5 rounded-lg border ${
                    executionMode === "place"
                      ? "border-[var(--desk-accent)] bg-[var(--desk-accent)]/10 text-[var(--desk-accent)]"
                      : "border-[var(--desk-border)] text-[var(--desk-text-secondary)]"
                  }`}
                  onClick={() => void setExecutionMode("place")}
                >
                  Place
                </button>
              </div>
            </div>
          </BentoCard>
          <BentoCard interactive={false}>
            <div className="flex flex-col gap-1">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                Suppliers
              </span>
              <span className="md-typescale-metric text-[var(--desk-text-primary)]">
                {settings?.supplier_overrides?.length || 0}
              </span>
            </div>
          </BentoCard>
          <BentoCard interactive={false}>
            <div className="flex flex-col gap-1">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                Suggestions
              </span>
              <span className="md-typescale-metric text-[var(--desk-text-primary)]">
                {reorderSuggestions.length}
              </span>
            </div>
          </BentoCard>
          <BentoCard interactive={false}>
            <div className="flex flex-col gap-1">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                Predictions
              </span>
              <span className="md-typescale-metric text-[var(--desk-text-primary)]">
                {predictions.length}
              </span>
            </div>
          </BentoCard>
        </BentoGrid>

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
              <span className="font-medium">Draft now</span> or{" "}
              <span className="font-medium">Place now</span>.
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
              <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">How It Works</h3>
            </div>
            <ol className="list-decimal list-inside space-y-2 text-[var(--desk-text-secondary)] md-typescale-body-small">
              <li>The AI analyzes your purchase patterns even when auto-order is off.</li>
              <li>When you enable, choose to use your history or start fresh.</li>
              <li>Starting fresh requires at least 2 orders per product.</li>
              <li>Overrides: Variant &gt; Product &gt; Category &gt; Supplier &gt; Global.</li>
              <li>
                <strong>Run auto-order now</strong> drafts cart lines for today
                (idempotent per SKU). Place mode still drafts until order.create is wired.
              </li>
            </ol>
          </div>
        </div>

        {pendingAction && (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm p-4">
            <div className="bg-[var(--desk-surface)] w-full max-w-sm rounded-2xl p-6 shadow-xl border border-[var(--desk-border)]">
              <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)] mb-2">Use Previous Analytics?</h3>
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
