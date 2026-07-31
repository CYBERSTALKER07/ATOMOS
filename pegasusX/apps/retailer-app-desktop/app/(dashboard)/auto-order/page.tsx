"use client";

import { useCallback, useEffect, useState, useMemo } from "react";
import {
  AlertTriangle,
  RefreshCw,
  Info,
} from "lucide-react";
import { AutoOrderRules } from "@/components/auto-order/AutoOrderRules";
import { AutoOrderList } from "@/components/auto-order/AutoOrderList";
import { PageChrome } from "@/components/PageChrome";
import { BentoGrid, BentoCard } from "@/components/BentoGrid";
import { PageSection } from "@/components/PageSection";
import { Skeleton } from "@/components/Skeleton";
import { useLiveData } from "@/lib/hooks";
import { apiFetch } from "@/lib/auth";
import { getRetailerId } from "@/lib/retailer-profile";
import type { AutoOrderSettings, Prediction } from "@/lib/types";

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

  const retailerId = getRetailerId();

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

  useEffect(() => {
    void fetchPredictions();
  }, [fetchPredictions]);

  const refreshAll = useCallback(() => {
    setSyncMessage(null);
    void mutateSettings();
    void fetchPredictions();
  }, [mutateSettings, fetchPredictions]);

  const toggleGlobal = async (enabled: boolean, useHistory?: boolean) => {
    try {
      const res = await apiFetch("/v1/retailer/settings/auto-order/global", {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ global_enabled: enabled, use_history: useHistory }),
      });
      if (!res.ok) throw new Error("Update failed");
      await mutateSettings();
    } catch (err) {
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
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={isLoading || isSettingsRefreshing}
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
        {syncMessage && (
          <div className="mb-6 flex items-center gap-2 p-3 rounded-xl bg-[var(--desk-warning)]/10 text-[var(--desk-warning)] border border-[var(--desk-warning)]/30">
            <AlertTriangle size={16} />
            <span className="md-typescale-body-small">{syncMessage}</span>
          </div>
        )}

        <BentoGrid className="mb-8">
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
                Categories
              </span>
              <span className="md-typescale-metric text-[var(--desk-text-primary)]">
                {settings?.category_overrides?.length || 0}
              </span>
            </div>
          </BentoCard>
          <BentoCard interactive={false}>
            <div className="flex flex-col gap-1">
              <span className="md-typescale-label-small uppercase tracking-widest text-[var(--desk-text-tertiary)] mb-2">
                Products
              </span>
              <span className="md-typescale-metric text-[var(--desk-text-primary)]">
                {settings?.product_overrides?.length || 0}
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
