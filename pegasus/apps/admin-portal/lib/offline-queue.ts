'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { OfflineManager } from './api/offlineQueue';

/**
 * Offline mutation queue for the Admin Portal.
 * This module now delegates to the persisted OfflineManager used by apiFetch
 * so network status UI and offline replay share one canonical queue.
 */
const RETRY_DELAY_MS = 2000;

/** Enqueue a failed mutation for later replay */
export function enqueueMutation(
  url: string,
  method: string,
  body?: string,
  headers: Record<string, string> = {}
) {
  OfflineManager.enqueue({
    url,
    method,
    body: body ?? null,
    headers,
  });
  return '';
}

/** Get the current queue length */
export function getQueueLength(): number {
  return OfflineManager.getLength();
}

/** Flush all queued mutations sequentially */
export async function flushQueue(): Promise<{ succeeded: number; failed: number }> {
  const before = OfflineManager.getLength();
  if (before === 0) return { succeeded: 0, failed: 0 };

  await OfflineManager.drainQueue(async (url, method, body, headers) => {
    return fetch(
      `${process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080'}${url}`,
      { method, body: body ?? undefined, headers },
    );
  });

  const after = OfflineManager.getLength();
  return {
    succeeded: Math.max(before - after, 0),
    failed: 0,
  };
}

/** React hook that tracks network status and auto-flushes the offline queue */
export function useNetworkStatus() {
  const [isOnline, setIsOnline] = useState(
    typeof navigator !== 'undefined' ? navigator.onLine : true
  );
  const [pendingCount, setPendingCount] = useState(() => getQueueLength());
  const flushTimeoutRef = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  const handleOnline = useCallback(async () => {
    setIsOnline(true);
    // Auto-flush after a short delay to let connections stabilize
    flushTimeoutRef.current = setTimeout(async () => {
      const result = await flushQueue();
      setPendingCount(getQueueLength());
      if (result.succeeded > 0) {
        console.log(`[OFFLINE_QUEUE] Flushed ${result.succeeded} mutations`);
      }
      if (result.failed > 0) {
        console.warn(`[OFFLINE_QUEUE] ${result.failed} mutations failed permanently`);
      }
    }, RETRY_DELAY_MS);
  }, []);

  const handleOffline = useCallback(() => {
    setIsOnline(false);
    if (flushTimeoutRef.current) clearTimeout(flushTimeoutRef.current);
  }, []);

  useEffect(() => {
    const syncPending = (event: Event) => {
      setPendingCount((event as CustomEvent<number>).detail);
    };

    setPendingCount(getQueueLength());
    window.addEventListener('sync-pending', syncPending);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('sync-pending', syncPending);
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      if (flushTimeoutRef.current) clearTimeout(flushTimeoutRef.current);
    };
  }, [handleOnline, handleOffline]);

  return { isOnline, pendingCount, flushQueue };
}
