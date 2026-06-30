'use client';

import { useEffect, useRef, useState } from 'react';

type UseInViewOptions = {
  rootMargin?: string;
  threshold?: number;
  /** Unmount / deactivate when leaving viewport (frees WebGL, etc.) */
  exit?: boolean;
};

export function useInView<T extends HTMLElement>(options: UseInViewOptions = {}) {
  const { exit = false, rootMargin = '120px', threshold = 0 } = options;
  const ref = useRef<T>(null);
  const [isInView, setIsInView] = useState(false);

  useEffect(() => {
    const node = ref.current;
    if (!node) return;

    const observer = new IntersectionObserver(
      ([entry]) => {
        setIsInView((prev) => {
          if (entry.isIntersecting) return true;
          return exit ? false : prev;
        });
      },
      { rootMargin, threshold }
    );

    observer.observe(node);
    return () => observer.disconnect();
  }, [exit, rootMargin, threshold]);

  return { ref, isInView };
}
