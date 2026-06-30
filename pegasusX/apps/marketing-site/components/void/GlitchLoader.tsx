"use client";

import { useEffect, useState } from "react";
import { GlitchText } from "./GlitchText";
import { useReducedMotion } from "@/components/motion/ReducedMotionProvider";

const LOADER_KEY = "pegasus-loader-seen";

export function GlitchLoader() {
  const reducedMotion = useReducedMotion();
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (reducedMotion) return;
    const seen = sessionStorage.getItem(LOADER_KEY);
    if (seen) return;
    setVisible(true);
    const timer = window.setTimeout(() => {
      sessionStorage.setItem(LOADER_KEY, "1");
      setVisible(false);
    }, 1800);
    return () => window.clearTimeout(timer);
  }, [reducedMotion]);

  if (!visible) return null;

  return (
    <div
      className="fixed inset-0 z-[9999] flex items-center justify-center overflow-hidden bg-black"
      aria-live="polite"
      aria-label="Loading"
    >
      <GlitchText text="PEGASUS" className="text-[clamp(2.5rem,12vw,7rem)]" />
    </div>
  );
}
