import { withDatabase } from "./db";

const LEGACY_STORAGE_KEY = "retailer_pending_pos_sales_v1";
const PARKED_CART_KEY = "retailer_pos_parked_cart_v1";

export type PendingPosSaleStatus = "PENDING" | "SYNCING" | "SYNCED" | "FAILED";

export interface PendingPosSale {
  id: string;
  clientSaleId: string;
  clientReceipt: string;
  sessionId: string;
  payloadJson: string;
  idempotencyKey: string;
  createdAt: number;
  retryCount: number;
  status: PendingPosSaleStatus;
  lastError?: string;
  serverSaleId?: string;
  serverReceiptNumber?: string;
}

export interface ParkedPosCart {
  sessionId: string;
  lines: Array<{
    sku: string;
    name: string;
    qty: number;
    unit_price_minor: number;
  }>;
  updatedAt: number;
}

type PendingRow = {
  id: string;
  client_sale_id: string;
  client_receipt: string;
  session_id: string;
  payload_json: string;
  idempotency_key: string;
  created_at: number;
  retry_count: number;
  status: string;
  last_error: string | null;
  server_sale_id: string | null;
  server_receipt_number: string | null;
};

function rowToEntry(row: PendingRow): PendingPosSale {
  return {
    id: row.id,
    clientSaleId: row.client_sale_id,
    clientReceipt: row.client_receipt,
    sessionId: row.session_id,
    payloadJson: row.payload_json,
    idempotencyKey: row.idempotency_key,
    createdAt: row.created_at,
    retryCount: row.retry_count,
    status: (row.status as PendingPosSaleStatus) || "PENDING",
    lastError: row.last_error ?? undefined,
    serverSaleId: row.server_sale_id ?? undefined,
    serverReceiptNumber: row.server_receipt_number ?? undefined,
  };
}

function readLegacyQueue(): PendingPosSale[] {
  if (typeof localStorage === "undefined") return [];
  try {
    const raw = localStorage.getItem(LEGACY_STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item): item is PendingPosSale => {
      if (!item || typeof item !== "object") return false;
      const row = item as PendingPosSale;
      return (
        typeof row.id === "string" &&
        typeof row.payloadJson === "string" &&
        typeof row.clientSaleId === "string"
      );
    });
  } catch {
    return [];
  }
}

function writeLegacyQueue(queue: PendingPosSale[]) {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(LEGACY_STORAGE_KEY, JSON.stringify(queue));
  } catch {
    // ignore quota
  }
}

async function readQueueFromDb(): Promise<PendingPosSale[] | null> {
  return withDatabase(async (db) => {
    const rows = await db.select<PendingRow[]>(
      `SELECT id, client_sale_id, client_receipt, session_id, payload_json, idempotency_key,
              created_at, retry_count, status, last_error, server_sale_id, server_receipt_number
       FROM pending_pos_sales ORDER BY created_at ASC`,
    );
    return rows.map(rowToEntry);
  });
}

async function writeQueueToDb(queue: PendingPosSale[]): Promise<boolean> {
  const result = await withDatabase(async (db) => {
    await db.execute("DELETE FROM pending_pos_sales");
    for (const entry of queue) {
      await db.execute(
        `INSERT INTO pending_pos_sales (
           id, client_sale_id, client_receipt, session_id, payload_json, idempotency_key,
           created_at, retry_count, status, last_error, server_sale_id, server_receipt_number
         ) VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8, ?9, ?10, ?11, ?12)`,
        [
          entry.id,
          entry.clientSaleId,
          entry.clientReceipt,
          entry.sessionId,
          entry.payloadJson,
          entry.idempotencyKey,
          entry.createdAt,
          entry.retryCount,
          entry.status,
          entry.lastError ?? null,
          entry.serverSaleId ?? null,
          entry.serverReceiptNumber ?? null,
        ],
      );
    }
    return true;
  });
  return result === true;
}

export async function listPendingPosSales(): Promise<PendingPosSale[]> {
  const fromDb = await readQueueFromDb();
  if (fromDb !== null) return fromDb;
  return readLegacyQueue().sort((a, b) => a.createdAt - b.createdAt);
}

async function persistQueue(queue: PendingPosSale[]): Promise<void> {
  const wrote = await writeQueueToDb(queue);
  if (!wrote) writeLegacyQueue(queue);
}

export async function enqueuePendingPosSale(input: {
  clientSaleId: string;
  clientReceipt: string;
  sessionId: string;
  payload: unknown;
  idempotencyKey: string;
}): Promise<PendingPosSale> {
  const entry: PendingPosSale = {
    id: input.clientSaleId,
    clientSaleId: input.clientSaleId,
    clientReceipt: input.clientReceipt,
    sessionId: input.sessionId,
    payloadJson: JSON.stringify(input.payload),
    idempotencyKey: input.idempotencyKey,
    createdAt: Date.now(),
    retryCount: 0,
    status: "PENDING",
  };
  const queue = await listPendingPosSales();
  // Replace same client id if re-enqueued
  const next = [...queue.filter((q) => q.clientSaleId !== entry.clientSaleId), entry];
  await persistQueue(next);
  return entry;
}

export async function removePendingPosSale(id: string): Promise<void> {
  const queue = await listPendingPosSales();
  await persistQueue(queue.filter((item) => item.id !== id));
}

export async function updatePendingPosSale(
  id: string,
  patch: Partial<
    Pick<
      PendingPosSale,
      "retryCount" | "lastError" | "status" | "serverSaleId" | "serverReceiptNumber"
    >
  >,
): Promise<void> {
  const queue = await listPendingPosSales();
  const next = queue.map((item) => (item.id === id ? { ...item, ...patch } : item));
  await persistQueue(next);
}

export function countActivePendingPosSales(queue: PendingPosSale[]): number {
  return queue.filter((q) => q.status === "PENDING" || q.status === "SYNCING" || q.status === "FAILED")
    .length;
}

export function countPendingForSession(queue: PendingPosSale[], sessionId: string): number {
  return queue.filter(
    (q) =>
      q.sessionId === sessionId &&
      (q.status === "PENDING" || q.status === "SYNCING" || q.status === "FAILED"),
  ).length;
}

export async function saveParkedPosCart(cart: ParkedPosCart): Promise<void> {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.setItem(PARKED_CART_KEY, JSON.stringify(cart));
  } catch {
    // ignore
  }
}

export async function loadParkedPosCart(sessionId?: string): Promise<ParkedPosCart | null> {
  if (typeof localStorage === "undefined") return null;
  try {
    const raw = localStorage.getItem(PARKED_CART_KEY);
    if (!raw) return null;
    const cart = JSON.parse(raw) as ParkedPosCart;
    if (sessionId && cart.sessionId !== sessionId) return null;
    return cart;
  } catch {
    return null;
  }
}

export async function clearParkedPosCart(): Promise<void> {
  if (typeof localStorage === "undefined") return;
  try {
    localStorage.removeItem(PARKED_CART_KEY);
  } catch {
    // ignore
  }
}

export function isRetryablePosSyncError(error: unknown): boolean {
  if (error instanceof TypeError) return true;
  if (error instanceof Error) {
    return /failed to fetch|network|load failed|offline/i.test(error.message);
  }
  return false;
}
