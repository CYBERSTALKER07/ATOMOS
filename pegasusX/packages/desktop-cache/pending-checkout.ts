import { withDatabase } from "./db";

const LEGACY_STORAGE_KEY = "retailer_pending_checkouts_v1";

export interface PendingCheckout {
  id: string;
  payloadJson: string;
  idempotencyKey: string;
  createdAt: number;
  retryCount: number;
  lastError?: string;
}

type PendingRow = {
  id: string;
  payload_json: string;
  idempotency_key: string;
  created_at: number;
  retry_count: number;
  last_error: string | null;
};

function rowToEntry(row: PendingRow): PendingCheckout {
  return {
    id: row.id,
    payloadJson: row.payload_json,
    idempotencyKey: row.idempotency_key,
    createdAt: row.created_at,
    retryCount: row.retry_count,
    lastError: row.last_error ?? undefined,
  };
}

function readLegacyQueue(): PendingCheckout[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(LEGACY_STORAGE_KEY);
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

function writeLegacyQueue(queue: PendingCheckout[]) {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(LEGACY_STORAGE_KEY, JSON.stringify(queue));
  } catch {
    // ignore quota
  }
}

async function readQueueFromDb(): Promise<PendingCheckout[] | null> {
  return withDatabase(async (db) => {
    const rows = await db.select<PendingRow[]>(
      "SELECT id, payload_json, idempotency_key, created_at, retry_count, last_error FROM pending_checkouts ORDER BY created_at ASC",
    );
    return rows.map(rowToEntry);
  });
}

async function writeQueueToDb(queue: PendingCheckout[]): Promise<boolean> {
  const result = await withDatabase(async (db) => {
    await db.execute("DELETE FROM pending_checkouts");
    for (const entry of queue) {
      await db.execute(
        `INSERT INTO pending_checkouts (id, payload_json, idempotency_key, created_at, retry_count, last_error)
         VALUES (?1, ?2, ?3, ?4, ?5, ?6)`,
        [
          entry.id,
          entry.payloadJson,
          entry.idempotencyKey,
          entry.createdAt,
          entry.retryCount,
          entry.lastError ?? null,
        ],
      );
    }
    return true;
  });
  return result === true;
}

/** One-time migration from localStorage queue into SQLite when on Tauri. */
export async function migrateLegacyPendingCheckouts(): Promise<void> {
  const legacy = readLegacyQueue();
  if (legacy.length === 0) return;
  const migrated = await writeQueueToDb(legacy);
  if (migrated && typeof localStorage !== "undefined") {
    localStorage.removeItem(LEGACY_STORAGE_KEY);
  }
}

export async function listPendingCheckouts(): Promise<PendingCheckout[]> {
  await migrateLegacyPendingCheckouts();
  const fromDb = await readQueueFromDb();
  if (fromDb !== null) return fromDb;
  return readLegacyQueue().sort((a, b) => a.createdAt - b.createdAt);
}

async function persistQueue(queue: PendingCheckout[]): Promise<void> {
  const wrote = await writeQueueToDb(queue);
  if (!wrote) {
    writeLegacyQueue(queue);
  }
}

export async function enqueuePendingCheckout(
  payload: unknown,
  idempotencyKey: string,
): Promise<PendingCheckout> {
  const entry: PendingCheckout = {
    id: crypto.randomUUID(),
    payloadJson: JSON.stringify(payload),
    idempotencyKey,
    createdAt: Date.now(),
    retryCount: 0,
  };
  const queue = await listPendingCheckouts();
  await persistQueue([...queue, entry]);
  return entry;
}

export async function removePendingCheckout(id: string): Promise<void> {
  const queue = await listPendingCheckouts();
  await persistQueue(queue.filter((item) => item.id !== id));
}

export async function updatePendingCheckout(
  id: string,
  patch: Partial<Pick<PendingCheckout, "retryCount" | "lastError">>,
): Promise<void> {
  const queue = await listPendingCheckouts();
  const next = queue.map((item) =>
    item.id === id ? { ...item, ...patch } : item,
  );
  await persistQueue(next);
}

export function isRetryableCheckoutError(error: unknown): boolean {
  if (error instanceof TypeError) return true;
  if (error instanceof Error) {
    return /failed to fetch|network|load failed|offline/i.test(error.message);
  }
  return false;
}

export function pendingCheckoutQueuedMessage(error: unknown): string {
  if (isRetryableCheckoutError(error)) {
    return "Network issue during checkout. Your cart was saved and will retry automatically when you reconnect.";
  }
  return error instanceof Error ? error.message : "Checkout failed";
}
