"use client";

import { useEffect, type RefObject } from "react";
import gsap from "gsap";
import { duration } from "@pegasusx/motion-tokens";
import { useReducedMotion } from "./ReducedMotionProvider";

type TextRevealOptions = {
  split?: "lines" | "chars";
  stagger?: number;
  delay?: number;
};

export function useTextReveal(
  ref: RefObject<HTMLElement | null>,
  options: TextRevealOptions = {},
) {
  const reducedMotion = useReducedMotion();
  const { split = "lines", stagger = 0.08, delay = 0 } = options;

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const targets =
      split === "lines"
        ? el.querySelectorAll("[data-reveal-line]")
        : el.querySelectorAll("[data-reveal-char]");

    if (targets.length === 0) return;

    if (reducedMotion) {
      gsap.set(targets, { opacity: 1, y: 0, clipPath: "inset(0% 0% 0% 0%)" });
      return;
    }

    gsap.set(targets, {
      opacity: 0,
      y: 24,
      clipPath: "inset(100% 0% 0% 0%)",
    });

    gsap.to(targets, {
      opacity: 1,
      y: 0,
      clipPath: "inset(0% 0% 0% 0%)",
      duration: duration.medium4,
      ease: "power3.out",
      stagger,
      delay,
    });
  }, [ref, reducedMotion, split, stagger, delay]);
}

export function splitLines(text: string): string[] {
  return text.split(/\n|<br\s*\/?>/).filter(Boolean);
}
