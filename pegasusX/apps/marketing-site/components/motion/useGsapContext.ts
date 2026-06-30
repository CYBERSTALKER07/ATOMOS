"use client";

import { useEffect, useRef, type RefObject } from "react";
import gsap from "gsap";

export function useGsapContext(
  scopeRef: RefObject<HTMLElement | null>,
  factory: (ctx: gsap.Context) => void,
  deps: unknown[] = [],
) {
  const factoryRef = useRef(factory);
  factoryRef.current = factory;

  useEffect(() => {
    const el = scopeRef.current;
    if (!el) return;

    const ctx = gsap.context(() => {
      factoryRef.current(ctx);
    }, el);

    return () => ctx.revert();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
}
