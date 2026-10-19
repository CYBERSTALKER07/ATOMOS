"use client";

import type { PlanBrainTab } from "@pegasusx/types";

export type PlanBrainTabsProps = {
  value: PlanBrainTab;
  onChange: (tab: PlanBrainTab) => void;
  className?: string;
};

/** Planning | Digital Brain. Query-string owner stays on the page. */
export function PlanBrainTabs({ value, onChange, className }: PlanBrainTabsProps) {
  return (
    <div className={className} data-testid="gs-u-plan-brain-tabs" role="tablist" aria-label="Plan and Brain">
      {(["planning", "brain"] as const).map((tab) => (
        <button
          key={tab}
          type="button"
          role="tab"
          data-tab={tab}
          aria-selected={value === tab}
          onClick={() => onChange(tab)}
          style={{
            marginRight: 8,
            padding: "6px 12px",
            borderRadius: 8,
            border: "1px solid var(--desk-border, #e5e7eb)",
            background: value === tab ? "var(--desk-surface-raised, #fff)" : "transparent",
            color: "inherit",
            cursor: "pointer",
            fontSize: "0.8125rem",
          }}
        >
          {tab === "planning" ? "Planning" : "Digital Brain"}
        </button>
      ))}
    </div>
  );
}
