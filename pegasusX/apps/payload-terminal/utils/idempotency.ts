/** Deterministic idempotency keys — aligned with @pegasusx/api-client idempotency.ts */

export function buildPayloadIdempotencyKey(action: string, entityId: string): string {
  return `payload-${action}-${entityId}`;
}

export function payloadSupplierStartLoadingKey(manifestId: string): string {
  return buildPayloadIdempotencyKey('supplier-start-loading', manifestId);
}

export function payloadSupplierInjectKey(manifestId: string, orderId: string): string {
  return buildPayloadIdempotencyKey('supplier-inject-order', `${manifestId}-${orderId}`);
}

export function payloadSealCompletedKey(manifestIds: string[]): string {
  const sorted = [...manifestIds].map((id) => id.trim()).filter(Boolean).sort();
  return buildPayloadIdempotencyKey('seal-completed', sorted.join(','));
}

export function payloadSealAllKey(payloaderId: string): string {
  return buildPayloadIdempotencyKey('seal-all', payloaderId || 'payloader');
}

export function payloadOrderSealKey(orderId: string): string {
  return buildPayloadIdempotencyKey('payload-seal', orderId);
}

export function payloadRecommendReassignKey(orderId: string): string {
  return buildPayloadIdempotencyKey('recommend-reassign', orderId);
}

export function payloadManifestExceptionKey(manifestId: string, orderId: string): string {
  return buildPayloadIdempotencyKey('manifest-exception', `${manifestId}-${orderId}`);
}

export function payloadMissingItemsKey(orderId: string): string {
  return buildPayloadIdempotencyKey('missing-items', orderId);
}

export function payloadInboundScanKey(barcode: string, sessionId: string): string {
  return buildPayloadIdempotencyKey('inbound-scan', `${barcode}-${sessionId}`);
}

export function payloadInboundConfirmKey(returnIds: string[], disposition: string): string {
  const sorted = [...returnIds].sort().join(',');
  return buildPayloadIdempotencyKey('inbound-confirm', `${disposition}-${sorted}`);
}

export function payloadApplyReassignKey(orderId: string, toDriverId: string): string {
  return buildPayloadIdempotencyKey('reassign-order', `${orderId}-${toDriverId}`);
}
