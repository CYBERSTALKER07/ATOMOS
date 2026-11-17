"use client";

import React, { useMemo } from "react";
import {
  type OrderStatus,
  ORDER_LIFECYCLE_NODES,
  CANONICAL_ORDER_GRAPH,
  type LifecycleNodeMeta,
} from "@pegasusx/types";

interface OrderStateMachineGraphProps {
  currentStatus: OrderStatus;
  onSelectTargetStatus?: (status: OrderStatus) => void;
  className?: string;
}

export function OrderStateMachineGraph({
  currentStatus,
  onSelectTargetStatus,
  className = "",
}: OrderStateMachineGraphProps) {
  const allowedNext = useMemo(
    () => new Set(CANONICAL_ORDER_GRAPH[currentStatus] || []),
    [currentStatus]
  );

  const phaseColors: Record<LifecycleNodeMeta["phase"], string> = {
    intake: "border-sky-500/30 bg-sky-950/20 text-sky-300",
    fulfillment: "border-indigo-500/30 bg-indigo-950/20 text-indigo-300",
    exception: "border-amber-500/30 bg-amber-950/20 text-amber-300",
    fiscal: "border-rose-500/30 bg-rose-950/20 text-rose-300",
    terminal: "border-emerald-500/30 bg-emerald-950/20 text-emerald-300",
  };

  return (
    <div className={`rounded-xl border border-border/40 bg-card p-6 shadow-2xl ${className}`}>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-4">
        <div>
          <h3 className="text-lg font-semibold tracking-tight text-foreground">
            Canonical Order State Machine
          </h3>
          <p className="text-xs text-muted-foreground mt-0.5">
            Current Active State:{" "}
            <span className="font-mono font-bold text-primary">{currentStatus}</span>
          </p>
        </div>
        <div className="flex items-center gap-2">
          {allowedNext.size > 0 ? (
            <span className="inline-flex items-center rounded-full bg-emerald-500/10 px-3 py-1 text-xs font-medium text-emerald-400 border border-emerald-500/20">
              {allowedNext.size} Allowed Transitions
            </span>
          ) : (
            <span className="inline-flex items-center rounded-full bg-neutral-500/10 px-3 py-1 text-xs font-medium text-neutral-400 border border-neutral-500/20">
              Terminal State
            </span>
          )}
        </div>
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {Object.values(ORDER_LIFECYCLE_NODES).map((node) => {
          const isCurrent = node.status === currentStatus;
          const isAllowed = allowedNext.has(node.status);

          return (
            <div
              key={node.status}
              onClick={() => isAllowed && onSelectTargetStatus?.(node.status)}
              className={`relative flex flex-col justify-between rounded-lg border p-4 transition-all duration-200 ${
                isCurrent
                  ? "border-primary bg-primary/10 shadow-lg shadow-primary/10 ring-2 ring-primary"
                  : isAllowed
                  ? "border-emerald-500/60 bg-emerald-950/20 cursor-pointer hover:border-emerald-400 hover:scale-[1.02]"
                  : `${phaseColors[node.phase]} opacity-40`
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="text-[10px] font-mono uppercase tracking-wider text-muted-foreground">
                  {node.phase}
                </span>
                {isCurrent && (
                  <span className="flex h-2 w-2 rounded-full bg-primary animate-pulse" />
                )}
                {isAllowed && (
                  <span className="rounded bg-emerald-500/20 px-1.5 py-0.5 text-[10px] font-semibold text-emerald-300">
                    PERMITTED
                  </span>
                )}
              </div>
              <div className="mt-2">
                <div className="font-mono text-sm font-bold text-foreground">
                  {node.status}
                </div>
                <div className="text-xs text-muted-foreground mt-0.5 line-clamp-1">
                  {node.label}
                </div>
                {node.description && (
                  <div className="mt-1 text-[11px] text-muted-foreground/80 line-clamp-2">
                    {node.description}
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

export default OrderStateMachineGraph;
