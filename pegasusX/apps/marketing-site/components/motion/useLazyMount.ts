"use client";

import { useEffect, useState, type RefObject } from "react";

export function useLazyMount(ref: RefObject<HTMLElement | null>, rootMargin = "200px") {
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry?.isIntersecting) {
          setMounted(true);
          observer.disconnect();
        }
      },
      { rootMargin },
    );

    observer.observe(el);
    return () => observer.disconnect();
  }, [ref, rootMargin]);

  return mounted;
}
