"use client";

import { useState } from "react";
import { financialSpecs } from "@/content/landing";
import { PinSection } from "@/components/motion/PinSection";
import { SectionHeader, SpecTable } from "@/components/docs/SpecTable";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

const LEDGER_ROWS = [
  { label: "Order receivable", debit: 1, credit: 0 },
  { label: "Retailer payable", debit: 0, credit: 1 },
  { label: "Delivery fee", debit: 0.8, credit: 0 },
  { label: "Driver earnings accrual", debit: 0, credit: 0.8 },
];

export function FinancialIntegritySection() {
  const reducedMotion = useReducedMotion();
  const [progress, setProgress] = useState(0);

  const inner = (
    <div className="mx-auto flex min-h-screen max-w-7xl flex-col justify-center px-4 py-8 md:px-6">
      <div className="grid gap-10 lg:grid-cols-2">
        <div>
          <SectionHeader
            platformFrame
            label="Financial integrity"
            title="Idempotent webhooks. Version gates. Double-entry semantics."
            description="Payment mutations commit with the same transactional discipline as fulfillment. Replay-safe consumers preserve treasury truth."
            titleId="financial-integrity-title"
          />
          <div className="mt-8">
            <SpecTable rows={financialSpecs} />
          </div>
        </div>

        <div>
          <div className="grid grid-cols-2 gap-x-8 gap-y-4 text-xs font-mono uppercase tracking-wider text-[var(--mkt-subtle)]">
            <span>Entry</span>
            <span className="grid grid-cols-2 gap-4">
              <span>Debit</span>
              <span>Credit</span>
            </span>
          </div>
          <div className="mt-4 space-y-4 font-mono text-sm">
            {LEDGER_ROWS.map((row, i) => {
              const rowProgress = Math.min(1, Math.max(0, progress * LEDGER_ROWS.length - i));
              return (
                <div key={row.label} className="grid grid-cols-2 items-center gap-8">
                  <span className="text-[var(--mkt-muted)]">{row.label}</span>
                  <div className="grid grid-cols-2 gap-4">
                    <div className="h-1.5 overflow-hidden rounded bg-[var(--mkt-elevated)]">
                      <div
                        className="h-full origin-left bg-[var(--mkt-text)]"
                        style={{ transform: `scaleX(${row.debit * rowProgress})` }}
                      />
                    </div>
                    <div className="h-1.5 overflow-hidden rounded bg-[var(--mkt-elevated)]">
                      <div
                        className="h-full origin-left bg-[var(--mkt-muted)]"
                        style={{ transform: `scaleX(${row.credit * rowProgress})` }}
                      />
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );

  if (reducedMotion) {
    return (
      <section id="financial-integrity" aria-labelledby="financial-integrity-title" className="py-24">
        {inner}
      </section>
    );
  }

  return (
    <PinSection id="financial-integrity" end="+=150%" onProgress={setProgress}>
      {inner}
    </PinSection>
  );
}
