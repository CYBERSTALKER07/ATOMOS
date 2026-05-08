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
    const onOnline = () => {
      if (document.visibilityState === 'visible') doFetch();
    };
    const onFocus = () => {
      if (document.visibilityState === 'visible' && (typeof navigator === 'undefined' || navigator.onLine)) {
        doFetch();
      }
    };
    document.addEventListener('visibilitychange', onVisible);
    window.addEventListener('online', onOnline);
    window.addEventListener('focus', onFocus);
    window.addEventListener('pageshow', onFocus);

    return () => {
      controllerRef.current?.abort();
      document.removeEventListener('visibilitychange', onVisible);
      window.removeEventListener('online', onOnline);
      window.removeEventListener('focus', onFocus);
      window.removeEventListener('pageshow', onFocus);
    };
  }, [doFetch]);

  return { refetch: doFetch, lastFetchedRef };
}
