"use client";

import { usePortalT } from "@/lib/i18n";
import { type ReactNode } from "react";
import { Info } from "lucide-react";

interface MetricCardProps {
  title: string;
  value: string | number;
  icon: ReactNode;
  trend?: {
    direction: "up" | "down" | "neutral";
    value: string;
  };
  tooltipText?: string;
  isAlert?: boolean;
}

export function MetricCard({
  title,
  value,
  icon,
  trend,
  tooltipText,
  isAlert = false,
}: MetricCardProps) {
  const t = usePortalT();
  return (
    <div
      className={`relative p-6 rounded-2xl border ${
        isAlert
          ? "border-[var(--desk-danger)]/30 bg-[var(--desk-danger)]/5"
          : "border-[var(--desk-border)] bg-[var(--desk-surface)]"
      } shadow-[var(--shadow-sm)] flex flex-col justify-between transition-colors`}
    >
      <div className="flex justify-between items-start mb-4">
        <div className="flex items-center gap-2">
          <div
            className={`p-2 rounded-xl ${
              isAlert
                ? "bg-[var(--desk-danger)]/10 text-[var(--desk-danger)]"
                : "bg-[var(--desk-surface-subtle)] text-[var(--desk-text-secondary)]"
            }`}
          >
            {icon}
          </div>
          <span className="md-typescale-body-medium font-medium text-[var(--desk-text-primary)]">
            {title}
          </span>
          {tooltipText && (
            <div title={tooltipText} className="inline-block cursor-help">
              <Info size={16} className="text-[var(--desk-text-tertiary)] cursor-help" />
            </div>
          )}
        </div>
      </div>
      <div className="mt-2">
        <span
          className={`text-4xl font-semibold tracking-tight ${
            isAlert ? "text-[var(--desk-danger)]" : "text-[var(--desk-text-primary)]"
          }`}
        >
          {value}
        </span>
      </div>
      {trend && (
        <div className="mt-4 flex items-center gap-1.5 md-typescale-body-small">
          <span
            className={`${
              trend.direction === "up"
                ? "text-[var(--desk-danger)]" // Often in compliance, up is bad (more mismatches)
                : trend.direction === "down"
                ? "text-[var(--desk-success)]"
                : "text-[var(--desk-text-secondary)]"
            } font-medium`}
          >
            {trend.value}
          </span>
          <span className="text-[var(--desk-text-tertiary)]">{t("supplier_portal.compliance.metric_card.text.vs_last_week")}</span>
        </div>
      )}
    </div>
  );
}
