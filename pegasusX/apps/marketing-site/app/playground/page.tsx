"use client";

import { Nav } from "@/components/layout/Nav";
import { useReducedMotionControls } from "@/components/motion/ReducedMotionProvider";

export default function PlaygroundPage() {
  const { effectiveReducedMotion, setOverrideReducedMotion, prefersReducedMotion } =
    useReducedMotionControls();

  return (
    <>
      <Nav />
      <main className="mx-auto max-w-3xl px-4 py-16 md:px-6">
        <p className="mkt-section-label">Playground</p>
        <h1 className="mkt-display mt-3 text-4xl">Motion debug</h1>
        <p className="mt-4 text-[var(--mkt-text-secondary)]">
          Toggle reduced motion override for scroll scene debugging. System preference:{" "}
          {prefersReducedMotion ? "reduce" : "no-preference"}.
        </p>

        <div className="mt-10 flex flex-wrap gap-3">
          <button
            type="button"
            className="mkt-btn mkt-btn-primary"
            onClick={() => setOverrideReducedMotion(true)}
          >
            Force reduced motion
          </button>
          <button
            type="button"
            className="mkt-btn mkt-btn-outline"
            onClick={() => setOverrideReducedMotion(false)}
          >
            Force full motion
          </button>
          <button
            type="button"
            className="mkt-btn mkt-btn-ghost"
            onClick={() => setOverrideReducedMotion(null)}
          >
            Use system
          </button>
        </div>

        <p className="mt-8 mkt-card p-4 text-sm">
          Effective reduced motion:{" "}
          <strong>{effectiveReducedMotion ? "on" : "off"}</strong>
        </p>
      </main>
    </>
  );
}
