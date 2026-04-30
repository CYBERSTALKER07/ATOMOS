/**
 * OfflineManager: localStorage-backed mutation queue.
 * When the network drops, mutations (POST/PUT/PATCH/DELETE) are persisted
 * in localStorage and replayed sequentially when connectivity returns.
 *
 * Dispatches a 'sync-pending' CustomEvent so UI components can react
 * to queue length changes without polling.
 */

interface QueuedRequest {
  id: string;
  url: string;
  method: string;
  body: string | null;
  headers: Record<string, string>;
  timestamp: number;
}

export class OfflineManager {
  private static QUEUE_KEY = 'void_offline_queue';

  /** Enqueue a failed mutation for later replay */
  static enqueue(request: Omit<QueuedRequest, 'id' | 'timestamp'>) {
    const queue = this.getQueue();
    queue.push({
      ...request,
      id: crypto.randomUUID(),
      timestamp: Date.now(),
    });
    localStorage.setItem(this.QUEUE_KEY, JSON.stringify(queue));

    // Trigger "Sync Pending" UI notification
    window.dispatchEvent(
      new CustomEvent('sync-pending', { detail: queue.length }),
    );
  }

  /**
   * Drain the queue sequentially. Stops on first failure to preserve
   * ordering guarantees — partial drain is safer than reordered replay.
   */
  static async drainQueue(
    executor: (url: string, method: string, body: string | null, headers: Record<string, string>) => Promise<Response>,
  ) {
    const queue = this.getQueue();
    if (queue.length === 0) return;

    for (const req of queue) {
      try {
        const res = await executor(req.url, req.method, req.body, req.headers);
        if (!res.ok && res.status >= 500) {
          // Server error — stop draining to avoid spamming a broken link
          console.error(`[OFFLINE_QUEUE] Drain halted: server ${res.status} for ${req.id}`);
          break;
        }
        this.removeFromQueue(req.id);
      } catch {
        console.error(`[OFFLINE_QUEUE] Drain failed for ${req.id}, keeping in queue.`);
        break;
      }
    }

    // Notify UI of updated queue length
    window.dispatchEvent(
      new CustomEvent('sync-pending', { detail: this.getQueue().length }),
    );
  }

  /** Get the current queue length */
  static getLength(): number {
    return this.getQueue().length;
  }

  private static getQueue(): QueuedRequest[] {
    try {
      return JSON.parse(localStorage.getItem(this.QUEUE_KEY) || '[]');
    } catch {
      return [];
    }
  }

  private static removeFromQueue(id: string) {
    const queue = this.getQueue().filter((r) => r.id !== id);
    localStorage.setItem(this.QUEUE_KEY, JSON.stringify(queue));
  }
}
