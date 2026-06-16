import { apiFetch } from "./auth";

const STORAGE_KEY = "retailer_pending_checkouts_v1";

export interface PendingCheckout {
  id: string;
  payloadJson: string;
  idempotencyKey: string;
  createdAt: number;
  retryCount: number;
  lastError?: string;
}

function readQueue(): PendingCheckout[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is PendingCheckout => {
      if (!item || typeof item !== "object") return false;
      const row = item as PendingCheckout;
      return (
        typeof row.id === "string" &&
        typeof row.payloadJson === "string" &&
        typeof row.idempotencyKey === "string"
      );
    });
  } catch {
    return [];
  }
}

function writeQueue(queue: PendingCheckout[]) {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(queue));
  } catch {
    // Ignore quota failures.
  }
}

export function listPendingCheckouts(): PendingCheckout[] {
  return readQueue().sort((a, b) => a.createdAt - b.createdAt);
}

export function enqueuePendingCheckout(payload: unknown, idempotencyKey: string): PendingCheckout {
  const entry: PendingCheckout = {
    id: crypto.randomUUID(),
    payloadJson: JSON.stringify(payload),
    idempotencyKey,
    createdAt: Date.now(),
    retryCount: 0,
  };
  writeQueue([...readQueue(), entry]);
  return entry;
}

export function removePendingCheckout(id: string) {
  writeQueue(readQueue().filter((item) => item.id !== id));
}

function isRetryableCheckoutError(error: unknown): boolean {
  if (error instanceof TypeError) return true;
  if (error instanceof Error) {
    return /failed to fetch|network|load failed|offline/i.test(error.message);
  }
  return false;
}

export async function flushPendingCheckouts(): Promise<{ flushed: number; failed: number }> {
  const queue = listPendingCheckouts();
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
        if (res.status === 409 && (code === "ALL_ITEMS_OUT_OF_STOCK" || code === "PARTIAL_OUT_OF_STOCK_REJECTED")) {
          removePendingCheckout(entry.id);
          failed += 1;
          continue;
        }
        throw new Error(typeof body?.error === "string" ? body.error : `Checkout replay failed (${res.status})`);
      }
      removePendingCheckout(entry.id);
      flushed += 1;
    } catch (error) {
      failed += 1;
      const next = readQueue().map((item) =>
        item.id === entry.id
          ? {
              ...item,
              retryCount: item.retryCount + 1,
              lastError: error instanceof Error ? error.message : "Replay failed",
            }
          : item,
      );
      writeQueue(next);
      if (!isRetryableCheckoutError(error)) {
        // Keep non-network failures in queue for operator visibility.
      }
    }
  }

  return { flushed, failed };
}

export function pendingCheckoutQueuedMessage(error: unknown): string {
  if (isRetryableCheckoutError(error)) {
    return "Network issue during checkout. Your cart was saved and will retry automatically when you reconnect.";
  }
  return error instanceof Error ? error.message : "Checkout failed";
}
