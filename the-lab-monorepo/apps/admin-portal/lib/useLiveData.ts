import { useEffect, useRef, useCallback } from 'react';

/**
 * Hook that provides:
 * - AbortController signal for cancelling in-flight requests on unmount
 * - Automatic refetch when the browser tab regains focus
 * - "Last fetched" timestamp for stale data indicators
 */
export function useLiveData(fetchFn: (signal: AbortSignal) => Promise<void>) {
  const controllerRef = useRef<AbortController | null>(null);
  const lastFetchedRef = useRef<number>(0);

  const doFetch = useCallback(() => {
    controllerRef.current?.abort();
    const controller = new AbortController();
    controllerRef.current = controller;
    lastFetchedRef.current = Date.now();
    fetchFn(controller.signal).catch(() => {});
  }, [fetchFn]);

  useEffect(() => {
    doFetch();

    const onVisible = () => {
      if (document.visibilityState === 'visible') doFetch();
    };
    document.addEventListener('visibilitychange', onVisible);

    return () => {
      controllerRef.current?.abort();
      document.removeEventListener('visibilitychange', onVisible);
    };
  }, [doFetch]);

  return { refetch: doFetch, lastFetchedRef };
}
