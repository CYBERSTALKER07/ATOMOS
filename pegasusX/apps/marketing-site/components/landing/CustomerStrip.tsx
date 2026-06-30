"use client";

import { customerStrip } from "@/content/landing/enterprise";

export function CustomerStrip() {
  const items = [...customerStrip.labels, ...customerStrip.labels];

  return (
    <section
      aria-label="Operational domains"
      className="border-y border-[var(--mkt-border)] bg-[var(--mkt-surface)] py-8"
    >
      <div className="mx-auto max-w-7xl px-4 md:px-6">
        <p className="text-center font-mono text-[10px] uppercase tracking-[0.18em] text-[var(--mkt-subtle)]">
          {customerStrip.headline}
        </p>
        <p className="mt-2 text-center text-sm text-[var(--mkt-muted)]">
          {customerStrip.subline}
        </p>
      </div>
      <div className="mkt-marquee mt-8" aria-hidden>
        <div className="mkt-marquee__track">
          {items.map((label, i) => (
            <span key={`${label}-${i}`} className="mkt-marquee__item">
              {label}
            </span>
          ))}
        </div>
      </div>
    </section>
  );
}
