import { apiFetch } from "./auth";
import {
  enqueuePendingCheckout as cacheEnqueue,
  isRetryableCheckoutError,
  listPendingCheckouts as cacheList,
  pendingCheckoutQueuedMessage as cacheQueuedMessage,
  removePendingCheckout as cacheRemove,
  updatePendingCheckout,
  type PendingCheckout,
} from "@pegasusx/desktop-cache";

export type { PendingCheckout };

export async function listPendingCheckouts(): Promise<PendingCheckout[]> {
  return cacheList();
}

export async function enqueuePendingCheckout(
  payload: unknown,
  idempotencyKey: string,
): Promise<PendingCheckout> {
  return cacheEnqueue(payload, idempotencyKey);
}

export async function removePendingCheckout(id: string): Promise<void> {
  await cacheRemove(id);
}

export async function flushPendingCheckouts(): Promise<{ flushed: number; failed: number }> {
  const queue = await listPendingCheckouts();
  if (queue.length === 0) return { flushed: 0, failed: 0 };

  let flushed = 0;
  let failed = 0;

  for (const entry of queue) {
    try {
      const res = await apiFetch("/v1/checkout/unified", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": entry.idempotencyKey,
        },
        body: entry.payloadJson,
      });
      if (!res.ok) {
        const body = await res.json().catch(() => null);
        const code = typeof body?.code === "string" ? body.code : "";
        if (
          res.status === 409 &&
          (code === "ALL_ITEMS_OUT_OF_STOCK" || code === "PARTIAL_OUT_OF_STOCK_REJECTED")
        ) {
          await removePendingCheckout(entry.id);
          failed += 1;
          continue;
        }
        throw new Error(
          typeof body?.error === "string" ? body.error : `Checkout replay failed (${res.status})`,
        );
      }
      await removePendingCheckout(entry.id);
      flushed += 1;
    } catch (error) {
      failed += 1;
      await updatePendingCheckout(entry.id, {
        retryCount: entry.retryCount + 1,
        lastError: error instanceof Error ? error.message : "Replay failed",
      });
      if (!isRetryableCheckoutError(error)) {
        // Keep non-network failures in queue for operator visibility.
      }
    }
  }

  return { flushed, failed };
}

export function pendingCheckoutQueuedMessage(error: unknown): string {
  return cacheQueuedMessage(error);
}

export { isRetryableCheckoutError };
