"use client";

import { type ReactNode, useRef } from "react";
import { usePinSection } from "./usePinSection";

type PinSectionProps = {
  id?: string;
  children: ReactNode;
  className?: string;
  end?: string;
  onProgress?: (progress: number) => void;
};

export function PinSection({
  id,
  children,
  className = "",
  end,
  onProgress,
}: PinSectionProps) {
  const ref = useRef<HTMLElement>(null);
  usePinSection(ref, { end, onProgress });

  return (
    <section ref={ref} id={id} className={className}>
      {children}
    </section>
  );
}
