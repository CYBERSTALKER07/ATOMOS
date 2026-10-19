"use client";

import { RefreshCw } from "lucide-react";
import { usePortalT } from "@/lib/i18n";

interface OrderFiltersProps {
  activeTab: "ALL" | "ACTIVE" | "COMPLETED";
  setActiveTab: (tab: "ALL" | "ACTIVE" | "COMPLETED") => void;
  isOrdersRefreshing: boolean;
  refreshAll: () => void;
}

export function OrderFilters({
  activeTab,
  setActiveTab,
  isOrdersRefreshing,
  refreshAll,
}: OrderFiltersProps) {
  const t = usePortalT();
  const tabLabels = {
    ALL: t("portal.page.orders.filter.all"),
    ACTIVE: t("portal.page.orders.filter.active_short"),
    COMPLETED: t("portal.page.orders.filter.completed_short"),
  } as const;

  return (
    <div className="flex items-center gap-3 mb-6 border-b border-[var(--desk-border)] pb-3">
      {(["ALL", "ACTIVE", "COMPLETED"] as const).map((tab) => (
        <button
          key={tab}
          onClick={() => setActiveTab(tab)}
          className={`px-5 py-2 rounded-full md-typescale-label-large font-light transition-all ${
            activeTab === tab
              ? "bg-[var(--desk-text-primary)] text-white shadow-[var(--shadow-sm)]"
              : "text-[var(--desk-text-secondary)] hover:bg-[var(--desk-surface-subtle)]"
          }`}
        >
          {tabLabels[tab]}
        </button>
      ))}
      <div className="flex-1" />
      <button
        type="button"
        disabled={isOrdersRefreshing}
        onClick={refreshAll}
        className="portal-btn portal-btn--ghost desk-icon-btn text-[var(--desk-text-tertiary)]"
        aria-label={t("portal.page.orders.action.refresh")}
      >
        <RefreshCw size={16} className={isOrdersRefreshing ? "animate-spin" : ""} />
      </button>
    </div>
  );
}
