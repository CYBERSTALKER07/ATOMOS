import { useEffect, useRef, useCallback } from 'react';

/**
 * Adaptive polling hook that:
 * - Cancels in-flight requests on unmount
 * - Automatically refetches when tab regains focus
 * - Throttles or halts when backend sends X-Backpressure-Interval via 'backpressure' event
 * - Does not poll when tab is hidden or offline
 */
export function usePolling(
  fn: (signal: AbortSignal) => Promise<void>,
  intervalMs: number,
  deps: unknown[] = [],
) {
  const fnRef = useRef(fn);
  fnRef.current = fn;

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const controllerRef = useRef<AbortController | null>(null);
  const currentIntervalRef = useRef(intervalMs);
  const isFetchingRef = useRef(false);
  const lastFetchedRef = useRef<number>(0);

  const doFetch = useCallback(async () => {
    if (isFetchingRef.current) return;
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      // Re-schedule when offline instead of failing
      if (timerRef.current) clearTimeout(timerRef.current);
      timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      return;
    }

    controllerRef.current?.abort();

    const controller = new AbortController();
    controllerRef.current = controller;

    isFetchingRef.current = true;
    lastFetchedRef.current = Date.now();

    try {
      if (!controller.signal.aborted) {
        await fnRef.current(controller.signal);
      }
    } catch (e: unknown) {
      if ((e as Error).name !== 'AbortError') {
        // Suppress other immediate fetch errors to keep polling alive
      }
    } finally {
      isFetchingRef.current = false;

      // Schedule next tick if tab is visible, otherwise wait for visibilitychange
      if (document.visibilityState === 'visible' && !controller.signal.aborted) {
        timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      }
    }
  }, []);

  useEffect(() => {
    currentIntervalRef.current = intervalMs;
    doFetch();

    const onVisibilityChange = () => {
      if (document.visibilityState === 'visible' && (typeof navigator === 'undefined' || navigator.onLine)) {
        currentIntervalRef.current = intervalMs;
        if (timerRef.current) clearTimeout(timerRef.current);
        doFetch();
      } else {
        if (timerRef.current) clearTimeout(timerRef.current);
      }
    };
    
    const onOnline = () => {
      if (document.visibilityState === 'visible') {
        currentIntervalRef.current = intervalMs;
        if (timerRef.current) clearTimeout(timerRef.current);
        doFetch();
      }
    };

    const onBackpressure = (e: Event) => {
      const waitMs = (e as CustomEvent<number>).detail;
      currentIntervalRef.current = Math.max(currentIntervalRef.current, waitMs);

      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = setTimeout(doFetch, currentIntervalRef.current);
      }
    };

    document.addEventListener('visibilitychange', onVisibilityChange);
    window.addEventListener('online', onOnline);
    window.addEventListener('backpressure', onBackpressure);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      controllerRef.current?.abort();
      document.removeEventListener('visibilitychange', onVisibilityChange);
      window.removeEventListener('online', onOnline);
      window.removeEventListener('backpressure', onBackpressure);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [intervalMs, ...deps]);

  return {
    refetch: () => {
      if (timerRef.current) clearTimeout(timerRef.current);
      doFetch();
    },
    lastFetchedRef,
  };
}
