function stableHash(input: string): string {
  let hash = 2166136261;
  for (let i = 0; i < input.length; i++) {
    hash ^= input.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

/** Deterministic idempotency keys — safe to retry after reconnect without double-apply. */
export function driverDeliverKey(orderId: string, driverId: string): string {
  return `driver-deliver:${driverId}:${orderId}`;
}

export function driverOffloadKey(orderId: string, driverId: string): string {
  return `driver-offload:${driverId}:${orderId}`;
}

export function driverCompleteKey(orderId: string, driverId: string): string {
  return `driver-complete:${driverId}:${orderId}`;
}

export function driverCollectCashKey(orderId: string, driverId: string): string {
  return `driver-collect-cash:${driverId}:${orderId}`;
}

export function retailerCheckoutKey(retailerId: string, cartFingerprint: string): string {
  return `retailer-checkout:${retailerId}:${stableHash(cartFingerprint)}`;
}

export function supplierDispatchKey(
  supplierId: string,
  warehouseId: string,
  mode: string,
  routeFingerprint: string,
): string {
  return `supplier-dispatch:${supplierId}:${warehouseId}:${mode}:${stableHash(routeFingerprint)}`;
}

export function warehouseDispatchKey(
  warehouseId: string,
  actorId: string,
  routeFingerprint: string,
): string {
  return `warehouse-dispatch:${warehouseId}:${actorId}:${stableHash(routeFingerprint)}`;
}

export function payloadSealKey(manifestId: string, payloaderId: string): string {
  return `payload-seal:${payloaderId}:${manifestId}`;
}

export function payloadInjectKey(manifestId: string, orderId: string): string {
  return `payload-inject:${manifestId}:${orderId}`;
}

export function supplierManifestSealKey(manifestId: string, supplierId: string): string {
  return `supplier-manifest-seal:${supplierId}:${manifestId}`;
}
