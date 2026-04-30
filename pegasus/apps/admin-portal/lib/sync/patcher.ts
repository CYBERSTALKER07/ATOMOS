/**
 * PatchEngine: Merges compressed deltas into local state.
 * Uses the V.O.I.D. Short-Key Dictionary to map compressed keys back
 * to camelCase UI properties for direct React state consumption.
 *
 * Wire format: { t: "ORD_UP", i: "uuid", d: { s: "LOADED" }, ts: 1713456000 }
 *
 * Unlike delta-sync.ts (which expands to snake_case for cache storage),
 * the patcher produces camelCase output ready for React component state.
 */

/** V.O.I.D. Short-Key → camelCase UI property mapping */
export const keyMap: Record<string, string> = {
  s: 'status',
  l: 'location',
  v: 'volumetricUnits',
  i: 'id',
  o: 'orderId',
  d: 'driverId',
  w: 'warehouseId',
  at: 'updatedAt',
};

/**
 * Apply a compressed delta to an existing state object.
 * Short keys are expanded to camelCase via the V.O.I.D. dictionary.
 * Unknown keys pass through unchanged for forward compatibility.
 *
 * @param currentState The current entity state
 * @param delta The compressed delta (short-keyed map from DeltaEvent.d)
 * @returns A new state object with delta fields merged
 */
export function applyDelta<T extends Record<string, unknown>>(
  currentState: T,
  delta: Record<string, unknown>,
): T {
  const patch: Record<string, unknown> = {};

  for (const [key, value] of Object.entries(delta)) {
    const fullKey = keyMap[key] ?? key;
    patch[fullKey] = value;
  }

  return { ...currentState, ...patch };
}
