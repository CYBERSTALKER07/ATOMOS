import { apiFetch } from "./auth";
import {
  clearParkedPosCart,
  countActivePendingPosSales,
  countPendingForSession,
  enqueuePendingPosSale as cacheEnqueue,
  isRetryablePosSyncError,
  listPendingPosSales as cacheList,
  loadParkedPosCart,
  removePendingPosSale as cacheRemove,
  saveParkedPosCart,
  updatePendingPosSale as cacheUpdate,
  type ParkedPosCart,
  type PendingPosSale,
} from "@pegasusx/desktop-cache";
import { retailerPosSaleKey } from "@pegasusx/api-client";

export type { ParkedPosCart, PendingPosSale };
export {
  clearParkedPosCart,
  countActivePendingPosSales,
  countPendingForSession,
  loadParkedPosCart,
  saveParkedPosCart,
};

export async function listPendingPosSales(): Promise<PendingPosSale[]> {
  return cacheList();
}

export async function enqueueOfflinePosSale(input: {
  clientSaleId: string;
  clientReceipt: string;
  sessionId: string;
  payload: unknown;
}): Promise<PendingPosSale> {
  return cacheEnqueue({
    ...input,
    idempotencyKey: retailerPosSaleKey(input.clientSaleId),
  });
}

export async function removePendingPosSale(id: string): Promise<void> {
  await cacheRemove(id);
}

export async function flushPendingPosSales(): Promise<{
  flushed: number;
  failed: number;
  remaining: number;
}> {
  const queue = await listPendingPosSales();
  const active = queue.filter(
    (q) => q.status === "PENDING" || q.status === "FAILED" || q.status === "SYNCING",
  );
  if (active.length === 0) {
    return { flushed: 0, failed: 0, remaining: 0 };
  }

  let flushed = 0;
  let failed = 0;

  for (const entry of active) {
    await cacheUpdate(entry.id, { status: "SYNCING", lastError: undefined });
    try {
      const res = await apiFetch("/v1/retailer/pos/sales", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Idempotency-Key": entry.idempotencyKey,
        },
        body: entry.payloadJson,
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) {
        const code =
          typeof (body as { error?: string }).error === "string"
            ? (body as { error: string }).error
            : `sale_failed_${res.status}`;
        // Permanent session closed — keep FAILED, don't spin forever
        await cacheUpdate(entry.id, {
          status: "FAILED",
          retryCount: entry.retryCount + 1,
          lastError: code,
        });
        failed += 1;
        continue;
      }
      const sale = body as { sale_id?: string; receipt_number?: string };
      await cacheUpdate(entry.id, {
        status: "SYNCED",
        serverSaleId: sale.sale_id,
        serverReceiptNumber: sale.receipt_number,
        lastError: undefined,
      });
      // Drop synced rows to keep queue lean
      await cacheRemove(entry.id);
      flushed += 1;
    } catch (error) {
      failed += 1;
      await cacheUpdate(entry.id, {
        status: "FAILED",
        retryCount: entry.retryCount + 1,
        lastError: error instanceof Error ? error.message : "Replay failed",
      });
      if (!isRetryablePosSyncError(error)) {
        // keep for operator visibility
      }
    }
  }

  const remaining = countActivePendingPosSales(await listPendingPosSales());
  return { flushed, failed, remaining };
}

export function newClientSaleId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) {
    return crypto.randomUUID();
  }
  return `pos-${Date.now()}-${Math.random().toString(36).slice(2, 10)}`;
}

export function provisionalReceipt(seq: number): string {
  const short = Date.now().toString(36).toUpperCase().slice(-6);
  return `OFF-${short}-${seq}`;
}
