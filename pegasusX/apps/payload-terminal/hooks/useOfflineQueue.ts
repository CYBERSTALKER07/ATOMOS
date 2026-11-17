import { useRef, useState } from 'react';
import * as SecureStore from 'expo-secure-store';

import { authFetch } from '../authSession';
import type { ShowToast } from './useToast';

// ─── Types ────────────────────────────────────────────────────────────────────

// Offline action queue — persisted in SecureStore, flushed on reconnect
export type QueuedAction = {
  id: string;
  endpoint: string;
  method: string;
  body: string;
  createdAt: number;
  status?: 'pending' | 'conflict' | 'failed';
  error?: string;
};

export type ProcessItemResult =
  | { outcome: 'succeeded' }
  | { outcome: 'conflict'; action: QueuedAction }
  | { outcome: 'retry'; action: QueuedAction }
  | { outcome: 'dropped' };

export function isIdempotent409(status: number, responseText: string): boolean {
  if (status !== 409) return false;
  const lower = responseText.toLowerCase();
  const markers = ['idempotent', 'duplicate', 'already_exists', 'already_processed'];
  return markers.some((k) => lower.includes(k));
}

export async function processQueueItem(
  action: QueuedAction,
  fetchFn: typeof authFetch = authFetch
): Promise<ProcessItemResult> {
  try {
    const res = await fetchFn(action.endpoint, {
      method: action.method,
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': action.id,
      },
      body: action.body,
    });
    if (res.ok) {
      return { outcome: 'succeeded' };
    }
    if (res.status === 409) {
      const text = await res.text().catch(() => '');
      if (isIdempotent409(res.status, text)) {
        return { outcome: 'succeeded' };
      }
      return {
        outcome: 'conflict',
        action: {
          ...action,
          status: 'conflict',
          error: text || 'conflict_409',
        },
      };
    }
    if (
      res.status === 401 ||
      res.status === 403 ||
      res.status === 408 ||
      res.status === 429 ||
      res.status >= 500
    ) {
      return { outcome: 'retry', action };
    }
    // Non-retryable 4xx (like 400 or 404) are dropped so poison-pill entries cannot block queue drain.
    return { outcome: 'dropped' };
  } catch {
    return { outcome: 'retry', action };
  }
}

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useOfflineQueue({
  token,
  tx,
  showToast,
}: {
  token: string | null;
  tx: (key: string) => string;
  showToast: ShowToast;
}) {
  const [offlineQueue, setOfflineQueue] = useState<QueuedAction[]>([]);
  const isFlushingRef = useRef(false);

  // ── Phase C: Offline Queue Flush ──────────────────────────────────────
  const flushOfflineQueue = async () => {
    if (offlineQueue.length === 0 || !token || isFlushingRef.current) return;
    isFlushingRef.current = true;
    try {
      const remaining: QueuedAction[] = [];
      let succeededCount = 0;
      let droppedCount = 0;
      let failedCount = 0;

      for (const action of offlineQueue) {
        const itemResult = await processQueueItem(action, authFetch);
        switch (itemResult.outcome) {
          case 'succeeded':
            succeededCount++;
            break;
          case 'conflict':
            failedCount++;
            remaining.push(itemResult.action);
            break;
          case 'retry':
            remaining.push(itemResult.action);
            break;
          case 'dropped':
            droppedCount++;
            break;
        }
      }

      setOfflineQueue(remaining);
      await SecureStore.setItemAsync('offline_queue', JSON.stringify(remaining));

      if (succeededCount > 0 && failedCount === 0) {
        showToast(tx('payload.alert.sync_complete_title'), `${succeededCount} queued action(s) synced.`, 'success', 3200);
      } else if (failedCount > 0) {
        showToast(tx('payload.alert.sync_failed_title'), `${failedCount} action(s) have business conflicts.`, 'error', 4000);
      } else if (droppedCount > 0 && remaining.length === 0) {
        showToast(tx('payload.alert.sync_failed_title'), `${droppedCount} invalid action(s) could not be processed.`, 'error', 4000);
      }
    } finally {
      isFlushingRef.current = false;
    }
  };

  return { offlineQueue, setOfflineQueue, flushOfflineQueue };
}
