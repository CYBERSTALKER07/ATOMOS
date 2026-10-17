"use client";

import { usePortalT } from "@/lib/i18n";
import React, { useState } from "react";
import Link from "next/link";
import { PageSection } from "../PageSection";
import EmptyState from "../EmptyState";
import { confirmAiOrder, rejectAiOrder } from "../../lib/api";
import type { RetailerAIPrediction } from "../../lib/types";
import {
  aiPredictionQty,
  aiPredictionTitle,
  formatMinorAmount,
} from "../../lib/types";

interface AiPredictionSectionProps {
  predictionList: RetailerAIPrediction[];
  onRefresh: () => void;
}

export function AiPredictionSection({
  predictionList,
  onRefresh,
}: AiPredictionSectionProps) {
  const t = usePortalT();
  const [actingId, setActingId] = useState<string | null>(null);

  const runAction = async (orderId: string, action: "confirm" | "reject") => {
    setActingId(orderId);
    try {
      const res =
        action === "confirm"
          ? await confirmAiOrder(orderId)
          : await rejectAiOrder(orderId, "Retailer rejected");
      if (!res.ok) {
        throw new Error(`ai_${action}_failed_${res.status}`);
      }
      onRefresh();
    } finally {
      setActingId(null);
    }
  };

  return (
    <PageSection
      title={t("retailer_desktop.dashboard.ai_prediction_section.text.ai_restock")}
      description={t("retailer_desktop.residual.text.high_confidence_replenishment_signals_for_this_cycle")}
      actions={
        <Link
          href="/insights"
          className="text-[var(--desk-accent)] md-typescale-label-small font-light uppercase tracking-widest hover:underline"
        >
          View All
        </Link>
      }
    >
      <div className="flex flex-col gap-3 !mt-0">
        {predictionList.length === 0 ? (
          <EmptyState
            headline={t("retailer_desktop.residual.text.no_ai_restock_signals")}
            body={t("retailer_desktop.residual.text.prediction_feed_is_currently_empty_for_this_cycle")}
            variant="no-predictions"
            action="Sync"
            onAction={onRefresh}
          />
        ) : (
          predictionList.slice(0, 5).map((item) => (
            <div
              key={item.order_id}
              className="p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl flex items-center gap-4"
            >
              <div className="w-10 h-10 rounded-full border-2 flex items-center justify-center text-[9px] font-light uppercase tracking-tighter border-[var(--desk-warning)] text-[var(--desk-warning)]">
                {(item.confirmation_status || "PENDING").replace(/_/g, " ").slice(0, 7)}
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 min-w-0">
                  <p className="md-typescale-title-small font-light truncate text-[var(--desk-text-primary)]">
                    {aiPredictionTitle(item)}
                  </p>
                </div>
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                  {item.requested_delivery_date
                    ? item.requested_delivery_date.slice(0, 10)
                    : item.order_id}
                </p>
              </div>
              <div className="text-right shrink-0">
                <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                  {formatMinorAmount(item.total_minor, item.currency)}
                </p>
                <p className="text-[10px] text-[var(--desk-text-tertiary)] uppercase font-light tracking-tighter">
                  {aiPredictionQty(item)} units
                </p>
                <div className="mt-2 flex items-center justify-end gap-2">
                  <button
                    type="button"
                    disabled={actingId === item.order_id}
                    onClick={() => void runAction(item.order_id, "confirm")}
                    className="text-[10px] font-light uppercase tracking-wide text-[var(--desk-accent)] hover:underline disabled:opacity-50"
                  >
                    Confirm
                  </button>
                  <button
                    type="button"
                    disabled={actingId === item.order_id}
                    onClick={() => void runAction(item.order_id, "reject")}
                    className="text-[10px] font-light uppercase tracking-wide text-[var(--desk-text-tertiary)] hover:text-red-600 disabled:opacity-50"
                  >
                    Reject
                  </button>
                </div>
              </div>
            </div>
          ))
        )}
      </div>
    </PageSection>
  );
}
