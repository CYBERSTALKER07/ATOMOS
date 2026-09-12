"use client";

import { usePortalT } from "@/lib/i18n";
import { PageSection } from "@/components/PageSection";
import type { Prediction } from "@/lib/types";

export function AutoOrderList({ predictions }: { predictions: Prediction[] }) {
  const t = usePortalT();
  if (predictions.length === 0) return null;

  return (
    <PageSection title={t("retailer_desktop.auto_order.auto_order_list.text.active_predictions")}>
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
              <div className="md-typescale-label-small text-[var(--desk-text-tertiary)]">{t("retailer_desktop.auto_order.auto_order_list.text.units")}</div>
            </div>
          </div>
        ))}
      </div>
    </PageSection>
  );
}
