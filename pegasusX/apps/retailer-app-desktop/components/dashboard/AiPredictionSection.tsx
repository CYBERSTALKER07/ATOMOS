import React from "react";
import Link from "next/link";
import { PageSection } from "../PageSection";
import EmptyState from "../EmptyState";
import type { Prediction } from "../../lib/types";
import { isPredictionBlocked } from "../../lib/types";

interface AiPredictionSectionProps {
  predictionList: Prediction[];
  onRefresh: () => void;
}

export function AiPredictionSection({
  predictionList,
  onRefresh,
}: AiPredictionSectionProps) {
  return (
    <PageSection
      title="AI Restock"
      description="High-confidence replenishment signals for this cycle."
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
            headline="No AI restock signals"
            body="Prediction feed is currently empty for this cycle."
            variant="no-predictions"
            action="Sync"
            onAction={onRefresh}
          />
        ) : (
          predictionList.slice(0, 5).map((forecast) => (
            <div
              key={forecast.id}
              className="p-4 bg-[var(--desk-surface)] border border-[var(--desk-border)] rounded-2xl flex items-center gap-4"
            >
              <div
                className={`w-10 h-10 rounded-full border-2 flex items-center justify-center text-xs font-light ${forecast.confidence > 0.8 ? "border-[var(--desk-success)] text-[var(--desk-success)]" : "border-[var(--desk-warning)] text-[var(--desk-warning)]"}`}
              >
                {Math.round(forecast.confidence * 100)}%
              </div>
              <div className="flex-1 min-w-0">
                <div className="flex items-center gap-2 min-w-0">
                  <p className="md-typescale-title-small font-light truncate text-[var(--desk-text-primary)]">
                    {forecast.productName}
                  </p>
                  {isPredictionBlocked(forecast) ? (
                    <span className="shrink-0 text-[9px] font-black tracking-tighter px-2 py-0.5 rounded bg-amber-100 border border-amber-200 text-amber-800">
                      INSUFFICIENT HISTORY
                    </span>
                  ) : null}
                </div>
                <p className="md-typescale-body-small text-[var(--desk-text-tertiary)] line-clamp-1">
                  {forecast.reasoning}
                </p>
              </div>
              <div className="text-right">
                <p className="md-typescale-title-small font-light text-[var(--desk-text-primary)]">
                  {forecast.predictedQuantity}
                </p>
                <p className="text-[10px] text-[var(--desk-text-tertiary)] uppercase font-light tracking-tighter">
                  Units
                </p>
              </div>
            </div>
          ))
        )}
      </div>
    </PageSection>
  );
}
