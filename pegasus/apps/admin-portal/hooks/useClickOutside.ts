'use client';

import { useEffect, useRef } from 'react';

/**
 * Hook that alerts clicks outside of the passed ref.
 * Useful for closing modals, dropdowns, and side drawers.
 */
export function useClickOutside<T extends HTMLElement>(
  handler: () => void,
  listenCapturing = true
) {
  const ref = useRef<T>(null);

  useEffect(() => {
    const handleClick = (e: MouseEvent | TouchEvent) => {
      if (ref.current && !ref.current.contains(e.target as Node)) {
        handler();
      }
    };

    document.addEventListener('mousedown', handleClick, listenCapturing);
    document.addEventListener('touchstart', handleClick, listenCapturing);

    return () => {
      document.removeEventListener('mousedown', handleClick, listenCapturing);
      document.removeEventListener('touchstart', handleClick, listenCapturing);
    };
  }, [handler, listenCapturing]);

  return ref;
}