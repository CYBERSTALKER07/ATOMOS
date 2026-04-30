import { useEffect, useRef } from 'react';

/**
 * Polling hook with AbortController. Fires `fn` immediately, then every `interval` ms.
 * Aborts in-flight requests on unmount or when deps change.
 */
export function usePolling(
  fn: (signal: AbortSignal) => Promise<void>,
  intervalMs: number,
  deps: unknown[] = [],
) {
  const fnRef = useRef(fn);
  fnRef.current = fn;

  useEffect(() => {
    const controller = new AbortController();
    const { signal } = controller;

    const tick = () => {
      if (signal.aborted) return;
      fnRef.current(signal).catch(() => {});
    };

    tick();
    const id = setInterval(tick, intervalMs);

    return () => {
      controller.abort();
      clearInterval(id);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, ...deps]);
}
