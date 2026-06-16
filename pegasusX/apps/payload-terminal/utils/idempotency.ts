/** Deterministic idempotency keys — aligned with payload-app-android PayloadIdempotencyKeys. */
export function buildPayloadIdempotencyKey(action: string, entityId: string): string {
  return `payload-${action}-${entityId}`;
}

export function payloadInboundScanKey(barcode: string, sessionId: string): string {
  return buildPayloadIdempotencyKey('inbound-scan', `${barcode}-${sessionId}`);
}

export function payloadApplyReassignKey(orderId: string, toDriverId: string): string {
  return buildPayloadIdempotencyKey('reassign-order', `${orderId}-${toDriverId}`);
}
