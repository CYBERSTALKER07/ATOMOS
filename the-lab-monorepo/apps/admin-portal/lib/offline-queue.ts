'use client';

import { useState, useEffect, useCallback, useRef } from 'react';

/**
 * Offline mutation queue for the Admin Portal.
 * When the network drops, mutations (POST/PUT/PATCH/DELETE) are queued
 * in memory and flushed sequentially when connectivity returns.
 */

interface QueuedMutation {
  id: string;
  url: string;
  method: string;
  body?: string;
  headers: Record<string, string>;
  timestamp: number;
  retries: number;
}

const MAX_RETRIES = 3;
const RETRY_DELAY_MS = 2000;
const OFFLINE_QUEUE_STORAGE_KEY = 'admin_portal_offline_mutation_queue_v1';

function isStorageAvailable(): boolean {
  return typeof window !== 'undefined' && typeof window.localStorage !== 'undefined';
}

function readPersistedQueue(): QueuedMutation[] {
  if (!isStorageAvailable()) return [];
  try {
    const raw = window.localStorage.getItem(OFFLINE_QUEUE_STORAGE_KEY);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];

    return parsed.filter((item): item is QueuedMutation => {
      return (
        typeof item?.id === 'string' &&
        typeof item?.url === 'string' &&
        typeof item?.method === 'string' &&
        typeof item?.timestamp === 'number' &&
        typeof item?.retries === 'number' &&
        item?.headers &&
        typeof item.headers === 'object'
      );
    });
  } catch {
    return [];
  }
}

function persistQueue() {
  if (!isStorageAvailable()) return;
  try {
    window.localStorage.setItem(OFFLINE_QUEUE_STORAGE_KEY, JSON.stringify(queue));
  } catch {
    // Best-effort persistence only; queue still functions in-memory.
  }
}

let queue: QueuedMutation[] = readPersistedQueue();
let isFlushing = false;

/** Enqueue a failed mutation for later replay */
export function enqueueMutation(
  url: string,
  method: string,
  body?: string,
  headers: Record<string, string> = {}
) {
  const id = `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
  queue.push({ id, url, method, body, headers, timestamp: Date.now(), retries: 0 });
  persistQueue();
  return id;
}

/** Get the current queue length */
export function getQueueLength(): number {
  return queue.length;
}

/** Flush all queued mutations sequentially */
export async function flushQueue(): Promise<{ succeeded: number; failed: number }> {
  if (isFlushing || queue.length === 0) return { succeeded: 0, failed: 0 };
  isFlushing = true;

  let succeeded = 0;
  let failed = 0;
  const remaining: QueuedMutation[] = [];

  for (const mutation of queue) {
    try {
      const res = await fetch(mutation.url, {
        method: mutation.method,
        headers: mutation.headers,
        body: mutation.body,
      });
      if (res.ok || res.status < 500) {
        succeeded++;
      } else {
        mutation.retries++;
        if (mutation.retries < MAX_RETRIES) {
          remaining.push(mutation);
        } else {
          failed++;
        }
      }
    } catch {
      mutation.retries++;
      if (mutation.retries < MAX_RETRIES) {
        remaining.push(mutation);
      } else {
        failed++;
      }
    }
  }

  queue = remaining;
  persistQueue();
  isFlushing = false;
  return { succeeded, failed };
}

/** React hook that tracks network status and auto-flushes the offline queue */
export function useNetworkStatus() {
  const [isOnline, setIsOnline] = useState(
    typeof navigator !== 'undefined' ? navigator.onLine : true
  );
  const [pendingCount, setPendingCount] = useState(0);
  const flushTimeoutRef = useRef<ReturnType<typeof setTimeout>>();

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
    // Re-sync on mount in case another tab updated the persisted queue.
    queue = readPersistedQueue();
    setPendingCount(queue.length);

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      if (flushTimeoutRef.current) clearTimeout(flushTimeoutRef.current);
    };
  }, [handleOnline, handleOffline]);

  // Poll pending count
  useEffect(() => {
    const interval = setInterval(() => setPendingCount(getQueueLength()), 2000);
    return () => clearInterval(interval);
  }, []);

  return { isOnline, pendingCount, flushQueue };
}
