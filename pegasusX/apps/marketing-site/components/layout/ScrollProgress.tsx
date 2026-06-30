"use client";

import { useLenis } from "@/components/motion/LenisProvider";

export function ScrollProgress() {
  const { scrollProgress } = useLenis();
  const width = `${Math.min(100, Math.max(0, scrollProgress * 100))}%`;

  return (
    <div
      className="fixed left-0 top-0 z-[60] h-[2px] w-full bg-transparent"
      aria-hidden="true"
    >
      <div
        className="h-full bg-[var(--mkt-text)] transition-[width] duration-75"
        style={{ width }}
      />
    </div>
  );
}
