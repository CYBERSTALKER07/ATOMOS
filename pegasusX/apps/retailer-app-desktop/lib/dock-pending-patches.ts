import type { TrackingOrder } from "./types";

const STORAGE_KEY = "retailer_dock_pending_patches";

export type DockOrderPatch = {
  orderId: string;
  patch: Partial<TrackingOrder>;
  updatedAt: string;
};

function readAll(): DockOrderPatch[] {
  if (typeof window === "undefined") return [];
  try {
    const raw = window.sessionStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const parsed: unknown = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (row): row is DockOrderPatch =>
        typeof row === "object" &&
        row !== null &&
        typeof (row as DockOrderPatch).orderId === "string" &&
        typeof (row as DockOrderPatch).updatedAt === "string" &&
        typeof (row as DockOrderPatch).patch === "object",
    );
  } catch {
    return [];
  }
}

function writeAll(rows: DockOrderPatch[]) {
  if (typeof window === "undefined") return;
  try {
    if (rows.length === 0) {
      window.sessionStorage.removeItem(STORAGE_KEY);
      return;
    }
    window.sessionStorage.setItem(STORAGE_KEY, JSON.stringify(rows));
  } catch {
    // Ignore quota errors — optimistic overlay is best-effort.
  }
}

export function queueDockOrderPatch(orderId: string, patch: Partial<TrackingOrder>) {
  const rows = readAll().filter((row) => row.orderId !== orderId);
  rows.push({ orderId, patch, updatedAt: new Date().toISOString() });
  writeAll(rows);
}

export function clearDockPendingPatches() {
  writeAll([]);
}

export function applyDockPendingPatches(orders: TrackingOrder[]): TrackingOrder[] {
  const patches = readAll();
  if (patches.length === 0) return orders;

  const byId = new Map(patches.map((row) => [row.orderId, row.patch]));
  return orders.map((order) => {
    const patch = byId.get(order.order_id);
    return patch ? { ...order, ...patch } : order;
  });
}
