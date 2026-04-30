'use client';

/**
 * Delta-Sync Patch Engine for the Admin Portal.
 *
 * Maintains a local entity cache and applies incoming DeltaEvent patches
 * via shallow merge. Reduces WebSocket bandwidth by ~90% — only changed
 * fields are transmitted, clients reconstruct full state locally.
 *
 * Wire format (from backend ws/events.go):
 *   { t: "ORD_UP", i: "uuid", d: { s: "LOADED" }, ts: 1713456000 }
 *
 * V.O.I.D. Short-Key Dictionary:
 *   s  → status          l  → location [lat,lng]
 *   v  → volumetric_units   i  → id
 *   o  → order_id         d  → driver_id
 *   w  → warehouse_id     at → updated_at
 */

// ── Delta Event Types ───────────────────────────────────────────────────────
export const DeltaType = {
  ORDER_UPDATE: 'ORD_UP',
  DRIVER_UPDATE: 'DRV_UP',
  FLEET_GPS: 'FLT_GPS',
  WAREHOUSE_LOAD: 'WH_LOAD',
  PAYMENT_UPDATE: 'PAY_UP',
  ROUTE_UPDATE: 'RTE_UP',
  NEGOTIATION: 'NEG_UP',
  CREDIT_UPDATE: 'CRD_UP',
} as const;

export type DeltaEventType = (typeof DeltaType)[keyof typeof DeltaType];

// ── Wire Format ─────────────────────────────────────────────────────────────
export interface DeltaEvent {
  t: DeltaEventType; // Event type
  i: string;         // Entity ID
  d: Record<string, unknown>; // Changed fields (short-keyed)
  ts: number;        // Unix timestamp
}

// ── V.O.I.D. Short-Key Expansion ────────────────────────────────────────────
// Reverse of backend ShortKeyMap — expand short wire keys to readable names.

const SHORT_KEY_MAP: Record<string, string> = {
  s: 'status',
  l: 'location',
  v: 'volumetric_units',
  i: 'id',
  o: 'order_id',
  d: 'driver_id',
  w: 'warehouse_id',
  at: 'updated_at',
};

/**
 * Expand short-keyed delta into human-readable field names.
 * Unknown keys are passed through unchanged.
 */
export function expandDelta(compressed: Record<string, unknown>): Record<string, unknown> {
  const expanded: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(compressed)) {
    const long = SHORT_KEY_MAP[k] ?? k;
    expanded[long] = v;
  }
  return expanded;
}

// ── Entity Cache ────────────────────────────────────────────────────────────

type EntityMap = Map<string, Record<string, unknown>>;

const caches = new Map<string, EntityMap>();

function getCache(entityType: string): EntityMap {
  let cache = caches.get(entityType);
  if (!cache) {
    cache = new Map();
    caches.set(entityType, cache);
  }
  return cache;
}

/**
 * Apply a DeltaEvent to the local entity cache.
 * Returns the full merged entity state after patching.
 */
export function applyDelta(event: DeltaEvent): Record<string, unknown> {
  const entityType = event.t;
  const cache = getCache(entityType);

  const existing = cache.get(event.i) ?? { id: event.i };
  const expanded = expandDelta(event.d);

  // Shallow merge — new fields overwrite old, existing fields preserved
  const merged = { ...existing, ...expanded, _lastDelta: event.ts };
  cache.set(event.i, merged);

  return merged;
}

/**
 * Get the current cached state for an entity.
 */
export function getEntity(entityType: string, entityId: string): Record<string, unknown> | undefined {
  return getCache(entityType).get(entityId);
}

/**
 * Get all cached entities for a given type.
 */
export function getAllEntities(entityType: string): Record<string, unknown>[] {
  return Array.from(getCache(entityType).values());
}

/**
 * Seed the cache with a full entity snapshot (e.g., from initial REST fetch
 * or catch-up endpoint). Call this before starting delta ingestion.
 */
export function seedCache(entityType: string, entities: Record<string, unknown>[]): void {
  const cache = getCache(entityType);
  for (const entity of entities) {
    const id = (entity.id ?? entity.order_id ?? entity.driver_id) as string;
    if (id) cache.set(id, entity);
  }
}

/**
 * Clear the cache for a specific entity type or all types.
 */
export function clearCache(entityType?: string): void {
  if (entityType) {
    caches.delete(entityType);
  } else {
    caches.clear();
  }
}

/**
 * Check if a WebSocket message is a DeltaEvent.
 */
export function isDeltaEvent(msg: unknown): msg is DeltaEvent {
  if (typeof msg !== 'object' || msg === null) return false;
  const obj = msg as Record<string, unknown>;
  return typeof obj.t === 'string' && typeof obj.i === 'string' && typeof obj.d === 'object' && typeof obj.ts === 'number';
}

/**
 * Process an incoming WebSocket message. If it's a DeltaEvent, apply it
 * and return the merged state. Otherwise return null.
 */
export function processDeltaMessage(msg: unknown): { type: DeltaEventType; id: string; state: Record<string, unknown> } | null {
  if (!isDeltaEvent(msg)) return null;
  const state = applyDelta(msg);
  return { type: msg.t, id: msg.i, state };
}
