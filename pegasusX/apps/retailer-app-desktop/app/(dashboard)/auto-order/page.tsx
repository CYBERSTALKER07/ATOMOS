"use client";

import { useCallback, useEffect, useState, useMemo } from "react";
import {
  Wand2,
  AlertTriangle,
  RefreshCw,
  Building2,
  Layers,
  Package,
  Box,
  CheckCircle2,
  Info,
} from "lucide-react";
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
    } catch (err: any) {
      setPredictionsError(err);
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
        skeletonVariant="text"
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
          <div className="p-6 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)]">
            <div className="flex items-center justify-between">
              <div>
                <h3 className="md-typescale-title-medium font-light text-[var(--desk-text-primary)]">Global Auto-Order</h3>
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] mt-1">Auto-order everything from all suppliers</p>
              </div>
              <label className="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  className="sr-only peer"
                  checked={settings?.global_enabled || false}
                  onChange={(e) => handleToggle("global", e.target.checked, settings?.has_any_history || false)}
                />
                <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
              </label>
            </div>
            {settings?.global_enabled && (
              <div className="mt-4 flex items-center gap-2 text-[var(--desk-success)]">
                <CheckCircle2 size={16} />
                <span className="md-typescale-body-small">Global auto-order active. Overrides all granular settings.</span>
              </div>
            )}
          </div>

          {(settings?.supplier_overrides?.length ?? 0) > 0 && (
            <PageSection title="Supplier Overrides">
              <div className="space-y-2">
                {settings?.supplier_overrides.map((item) => (
                  <div key={item.supplier_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                    <div className="flex items-center gap-3">
                      <Building2 size={18} className="text-[var(--desk-text-tertiary)]" />
                      <div>
                        <div className="md-typescale-body-medium">{item.supplier_id}</div>
                        <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Supplier-level override</div>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={item.enabled}
                        onChange={(e) => handleToggle("supplier", e.target.checked, item.has_history, item.supplier_id)}
                      />
                      <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                    </label>
                  </div>
                ))}
              </div>
            </PageSection>
          )}

          {(settings?.category_overrides?.length ?? 0) > 0 && (
            <PageSection title="Category Overrides">
              <div className="space-y-2">
                {settings?.category_overrides.map((item) => (
                  <div key={item.category_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                    <div className="flex items-center gap-3">
                      <Layers size={18} className="text-[var(--desk-text-tertiary)]" />
                      <div>
                        <div className="md-typescale-body-medium">{item.category_id}</div>
                        <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Category-level override</div>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={item.enabled}
                        onChange={(e) => handleToggle("category", e.target.checked, item.has_history, item.category_id)}
                      />
                      <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                    </label>
                  </div>
                ))}
              </div>
            </PageSection>
          )}

          {(settings?.product_overrides?.length ?? 0) > 0 && (
            <PageSection title="Product Overrides">
              <div className="space-y-2">
                {settings?.product_overrides.map((item) => (
                  <div key={item.product_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                    <div className="flex items-center gap-3">
                      <Package size={18} className="text-[var(--desk-text-tertiary)]" />
                      <div>
                        <div className="md-typescale-body-medium">{item.product_id}</div>
                        <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Product-level override</div>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={item.enabled}
                        onChange={(e) => handleToggle("product", e.target.checked, false, item.product_id)}
                      />
                      <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                    </label>
                  </div>
                ))}
              </div>
            </PageSection>
          )}

          {(settings?.variant_overrides?.length ?? 0) > 0 && (
            <PageSection title="Variant / SKU Overrides">
              <div className="space-y-2">
                {settings?.variant_overrides.map((item) => (
                  <div key={item.variant_id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                    <div className="flex items-center gap-3">
                      <Box size={18} className="text-[var(--desk-text-tertiary)]" />
                      <div>
                        <div className="md-typescale-body-medium">{item.variant_id}</div>
                        <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Variant / SKU override</div>
                      </div>
                    </div>
                    <label className="relative inline-flex items-center cursor-pointer">
                      <input
                        type="checkbox"
                        className="sr-only peer"
                        checked={item.enabled}
                        onChange={(e) => handleToggle("variant", e.target.checked, false, item.variant_id)}
                      />
                      <div className="w-11 h-6 bg-[var(--desk-surface-subtle)] rounded-full peer peer-checked:after:translate-x-full after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-[var(--desk-accent)]"></div>
                    </label>
                  </div>
                ))}
              </div>
            </PageSection>
          )}

          {predictions.length > 0 && (
            <PageSection title="Active Predictions">
              <div className="space-y-2">
                {predictions.map((pred) => (
                  <div key={pred.id} className="flex items-center justify-between p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-xl">
                    <div className="flex items-center gap-4">
                      <div className="relative flex items-center justify-center w-10 h-10">
                        <svg className="absolute w-full h-full transform -rotate-90">
                          <circle cx="20" cy="20" r="18" stroke="var(--desk-border)" strokeWidth="2" fill="none" />
                          <circle cx="20" cy="20" r="18" stroke={pred.confidence > 0.8 ? "var(--desk-success)" : "var(--desk-warning)"} strokeWidth="2" fill="none" strokeDasharray="113" strokeDashoffset={113 - (113 * pred.confidence)} />
                        </svg>
                        <span className="text-[10px] font-bold" style={{ color: pred.confidence > 0.8 ? "var(--desk-success)" : "var(--desk-warning)" }}>
                          {Math.round(pred.confidence * 100)}%
                        </span>
                      </div>
                      <div>
                        <div className="md-typescale-body-medium">{pred.productName || pred.product_name}</div>
                        <div className="md-typescale-body-small text-[var(--desk-text-tertiary)]">Order by {pred.suggestedOrderDate || pred.suggested_order_date}</div>
                      </div>
                    </div>
                    <div className="text-right">
                      <div className="md-typescale-title-medium font-light text-[var(--desk-accent)]">{pred.predictedQuantity || pred.predicted_quantity}</div>
                      <div className="md-typescale-label-small text-[var(--desk-text-tertiary)]">units</div>
                    </div>
                  </div>
                ))}
              </div>
            </PageSection>
          )}

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
