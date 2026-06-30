"use client";

import { useRef } from "react";
import { useGSAP } from "@gsap/react";
import gsap from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { useReducedMotion } from "./ReducedMotionProvider";
import { getMobileViewport } from "@/lib/reduced-motion";

gsap.registerPlugin(ScrollTrigger, useGSAP);

type PinSectionOptions = {
  end?: string;
  pinSpacing?: boolean;
  onProgress?: (progress: number) => void;
};

export function usePinSection(
  sectionRef: React.RefObject<HTMLElement | null>,
  options: PinSectionOptions = {},
  deps: unknown[] = [],
) {
  const reducedMotion = useReducedMotion();
  const onProgressRef = useRef(options.onProgress);
  onProgressRef.current = options.onProgress;

  useGSAP(
    () => {
      const el = sectionRef.current;
      if (!el || reducedMotion) return;

      const end = options.end ?? (getMobileViewport() ? "+=100%" : "+=200%");

      ScrollTrigger.create({
        trigger: el,
        start: "top top",
        end,
        pin: true,
        pinSpacing: options.pinSpacing ?? true,
        anticipatePin: 1,
        onUpdate: (self) => {
          onProgressRef.current?.(self.progress);
        },
      });
    },
    { scope: sectionRef, dependencies: [reducedMotion, ...deps], revertOnUpdate: true },
  );
}

export function mapScrollProgress(
  progress: number,
  inMin: number,
  inMax: number,
  outMin: number,
  outMax: number,
): number {
  return gsap.utils.mapRange(inMin, inMax, outMin, outMax, progress);
}
