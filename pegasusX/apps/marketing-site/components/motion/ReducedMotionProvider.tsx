"use client";

import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";
import { getSystemReducedMotion } from "@/lib/reduced-motion";

type ReducedMotionContextValue = {
  prefersReducedMotion: boolean;
  overrideReducedMotion: boolean | null;
  effectiveReducedMotion: boolean;
  setOverrideReducedMotion: (value: boolean | null) => void;
};

const ReducedMotionContext = createContext<ReducedMotionContextValue | null>(null);

export function ReducedMotionProvider({ children }: { children: ReactNode }) {
  const [prefersReducedMotion, setPrefersReducedMotion] = useState(false);
  const [overrideReducedMotion, setOverrideReducedMotion] = useState<boolean | null>(null);

  useEffect(() => {
    setPrefersReducedMotion(getSystemReducedMotion());
    const media = window.matchMedia("(prefers-reduced-motion: reduce)");
    const handler = () => setPrefersReducedMotion(media.matches);
    media.addEventListener("change", handler);
    return () => media.removeEventListener("change", handler);
  }, []);

  const effectiveReducedMotion =
    overrideReducedMotion ?? prefersReducedMotion;

  const value = useMemo(
    () => ({
      prefersReducedMotion,
      overrideReducedMotion,
      effectiveReducedMotion,
      setOverrideReducedMotion,
    }),
    [prefersReducedMotion, overrideReducedMotion, effectiveReducedMotion],
  );

  return (
    <ReducedMotionContext.Provider value={value}>
      {children}
    </ReducedMotionContext.Provider>
  );
}

export function useReducedMotion(): boolean {
  const ctx = useContext(ReducedMotionContext);
  if (!ctx) return getSystemReducedMotion();
  return ctx.effectiveReducedMotion;
}

export function useReducedMotionControls() {
  const ctx = useContext(ReducedMotionContext);
  if (!ctx) {
    throw new Error("useReducedMotionControls must be used within ReducedMotionProvider");
  }
  return ctx;
}
