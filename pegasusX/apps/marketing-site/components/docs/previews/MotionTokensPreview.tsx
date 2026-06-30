"use client";

import { duration, easing } from "@pegasusx/motion-tokens";

export function MotionTokensPreview() {
  return (
    <div className="space-y-6">
      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--mkt-subtle)]">
          Durations (seconds)
        </p>
        <div className="grid grid-cols-2 gap-2 text-sm md:grid-cols-4">
          {Object.entries(duration).map(([key, value]) => (
            <div key={key} className="mkt-card px-3 py-2 font-mono text-xs">
              <span className="text-[var(--mkt-text)]">{key}</span>
              <span className="ml-2 text-[var(--mkt-muted)]">{value}s</span>
            </div>
          ))}
        </div>
      </div>
      <div>
        <p className="mb-2 text-xs font-semibold uppercase tracking-wider text-[var(--mkt-subtle)]">
          Easing curves
        </p>
        <div className="space-y-3">
          {Object.entries(easing).map(([key, value]) => (
            <div key={key} className="flex items-center gap-4">
              <span className="w-40 shrink-0 font-mono text-xs">{key}</span>
              <div className="relative h-8 flex-1 overflow-hidden rounded border border-[var(--mkt-border)]">
                <div
                  className="motion-ease-demo absolute left-0 top-1/2 h-4 w-4 -translate-y-1/2 rounded bg-[var(--mkt-text)]"
                  style={{ animationTimingFunction: `cubic-bezier(${value.join(",")})` }}
                />
              </div>
              <span className="font-mono text-[10px] text-[var(--mkt-subtle)]">
                [{value.join(", ")}]
              </span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
