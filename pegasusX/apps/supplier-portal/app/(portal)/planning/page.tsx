"use client";

import PlanningBrainPanel from "@/components/PlanningBrainPanel";

/** Dedicated ops surface for S&OP / scenario sandbox (P2-26). */
export default function PlanningPage() {
  return (
    <div className="mx-auto max-w-5xl p-4 md:p-6">
      <header className="mb-4">
        <h1 className="md-typescale-headline-small" style={{ color: "var(--desk-text-primary)" }}>
          Planning
        </h1>
        <p className="md-typescale-body-small mt-1" style={{ color: "var(--desk-text-secondary)" }}>
          Network S&amp;OP snapshot and what-if scenarios — same Class A planning APIs as mobile.
        </p>
      </header>
      <PlanningBrainPanel />
    </div>
  );
}
