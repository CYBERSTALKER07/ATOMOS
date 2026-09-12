import { useState } from 'react';
import * as SecureStore from 'expo-secure-store';

import { authFetch } from '../authSession';
import type { ShowToast } from './useToast';

// ─── Types ────────────────────────────────────────────────────────────────────

// Offline action queue — persisted in SecureStore, flushed on reconnect
export type QueuedAction = { id: string; endpoint: string; method: string; body: string; createdAt: number };

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

  // ── Phase C: Offline Queue Flush ──────────────────────────────────────
  const flushOfflineQueue = async () => {
    if (offlineQueue.length === 0 || !token) return;
    const remaining: QueuedAction[] = [];
    for (const action of offlineQueue) {
      try {
        const res = await authFetch(action.endpoint, {
          method: action.method,
          headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': action.id,
          },
          body: action.body,
        });
        if (res.ok || res.status === 409) continue;
        if (res.status === 401 || res.status === 403 || res.status === 408 || res.status === 429 || res.status >= 500) {
          remaining.push(action); // retryable
        }
        // Non-retryable 4xx (like 400 or 404) are dropped so poison-pill entries cannot block queue drain.
      } catch {
        remaining.push(action);
      }
    }
    setOfflineQueue(remaining);
    await SecureStore.setItemAsync('offline_queue', JSON.stringify(remaining));
    if (remaining.length === 0 && offlineQueue.length > 0) {
      showToast(tx('payload.alert.sync_complete_title'), `${offlineQueue.length} queued actions synced.`, 'success', 3200);
    }
  };

  return { offlineQueue, setOfflineQueue, flushOfflineQueue };
}
