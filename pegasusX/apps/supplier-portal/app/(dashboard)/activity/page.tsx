"use client";

import { useCallback } from "react";
import { PageChrome } from "@/components/PageChrome";
import { PageSection } from "@/components/PageSection";
import { useLiveData } from "@/lib/hooks";
import { RefreshCw, Activity } from "lucide-react";

type SupplierActivityEvent = {
  id: string;
  type: string;
  description: string;
  timestamp: string;
};

type SupplierActivityResponse = {
  events: SupplierActivityEvent[];
};

export default function ActivityPage() {
  const {
    data,
    loading,
    error,
    isRefreshing,
    mutate,
  } = useLiveData<SupplierActivityResponse>("/v1/supplier/activity");

  const rows = data?.events || [];

  const refreshAll = useCallback(() => {
    void mutate();
  }, [mutate]);

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: "var(--desk-canvas)" }}>
      <PageChrome
        icon="overview"
        title="Activity"
        description="Stream of operational events and recent actions."
        loading={loading}
        skeletonVariant="table"
        actions={
          <div className="flex items-center gap-3">
            <button
              type="button"
              disabled={loading || isRefreshing}
              onClick={refreshAll}
              className="portal-btn portal-btn--ghost h-11 px-5 rounded-xl font-light"
            >
              <RefreshCw
                size={16}
                className={`mr-2 ${isRefreshing ? "animate-spin" : ""}`}
              />
              {isRefreshing ? "Syncing" : "Sync"}
            </button>
          </div>
        }
      >
        {error && (
          <div className="mb-6 p-4 rounded-xl border bg-[var(--desk-danger)]/10 text-[var(--desk-danger)] border-[var(--desk-danger)]/30">
            {error.message || "Failed to load activity."}
          </div>
        )}

        <PageSection title="Recent Activity" description="Latest events across your network.">
          <div className="bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl shadow-[var(--shadow-sm)] overflow-hidden">
            {rows.length === 0 ? (
              <div className="p-12 text-center text-[var(--desk-text-tertiary)] flex flex-col items-center">
                <Activity size={48} className="opacity-20 mb-4" />
                <p className="md-typescale-body-large">No recent activity.</p>
                <p className="md-typescale-body-small mt-1">Operational events will stream here.</p>
              </div>
            ) : (
              <div className="divide-y divide-[var(--desk-border)]">
                {rows.map((row: SupplierActivityEvent) => (
                  <div key={row.id || row.timestamp} className="p-4 hover:bg-[var(--desk-surface-hover)] transition-colors flex gap-4">
                    <div className="w-10 h-10 rounded-full bg-[var(--desk-surface-subtle)] flex items-center justify-center shrink-0">
                      <Activity size={18} className="text-[var(--desk-text-tertiary)]" />
                    </div>
                    <div>
                      <div className="md-typescale-body-medium font-medium text-[var(--desk-text-primary)]">
                        {row.type}
                      </div>
                      <div className="md-typescale-body-small text-[var(--desk-text-secondary)] mt-0.5">
                        {row.description}
                      </div>
                      <div className="md-typescale-label-small text-[var(--desk-text-tertiary)] mt-1.5 uppercase tracking-wide">
                        {new Date(row.timestamp).toLocaleString()}
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        </PageSection>
      </PageChrome>
    </div>
  );
}
