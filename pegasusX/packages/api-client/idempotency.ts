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

export function warehouseOrderDelayKey(orderId: string): string {
  return `warehouse-order-delay:${orderId}`;
}

export function warehouseOrderRejectKey(orderId: string, reason: string): string {
  return `warehouse-order-reject:${orderId}:${stableHash(reason)}`;
}

export function warehouseOrderOverflowKey(orderId: string): string {
  return `warehouse-order-overflow:${orderId}`;
}

export function warehouseDispatchLockAcquireKey(
  warehouseId: string,
  entityType: string,
  entityId: string,
): string {
  return `warehouse-dispatch-lock-acquire:${warehouseId}:${entityType}:${entityId}`;
}

export function warehouseDispatchLockReleaseKey(lockId: string): string {
  return `warehouse-dispatch-lock-release:${lockId}`;
}

export function payloadStartLoadingKey(manifestId: string): string {
  return `payload-start-loading-${manifestId}`;
}

export function payloadSupplierStartLoadingKey(manifestId: string): string {
  return `payload-supplier-start-loading-${manifestId}`;
}

export function payloadSealKey(manifestId: string, payloaderId: string): string {
  return `payload-seal-${payloaderId}-${manifestId}`;
}

export function payloadOrderSealKey(orderId: string): string {
  return `payload-payload-seal-${orderId}`;
}

export function payloadInjectKey(manifestId: string, orderId: string): string {
  return `payload-payloader_inject-${manifestId}_${orderId}`;
}

export function payloadSupplierInjectKey(manifestId: string, orderId: string): string {
  return `payload-supplier-inject-order-${manifestId}-${orderId}`;
}

export function payloadSealCompletedKey(manifestIds: string[]): string {
  const sorted = [...manifestIds].map((id) => id.trim()).filter(Boolean).sort();
  return `payload-seal-completed-${sorted.join(",")}`;
}

export function payloadRecommendReassignKey(orderId: string): string {
  return `payload-recommend-reassign-${orderId}`;
}

export function payloadFleetReassignKey(orderIds: string[]): string {
  const sorted = [...orderIds].map((id) => id.trim()).filter(Boolean).sort();
  return `payload-fleet-reassign-${sorted.join(",")}`;
}

export function payloadApplyReassignKey(orderId: string, toDriverId: string): string {
  return `payload-reassign-order-${orderId}-${toDriverId}`;
}

export function payloadSupplierSealManifestKey(manifestId: string): string {
  return `payload-supplier-seal-manifest-${manifestId}`;
}

export function supplierManifestSealKey(manifestId: string, supplierId: string): string {
  return `supplier-manifest-seal:${supplierId}:${manifestId}`;
}

export function supplierManifestStartLoadingKey(manifestId: string): string {
  return `supplier-start-loading:${manifestId}`;
}

export function supplierManifestInjectKey(manifestId: string, orderId: string): string {
  return `supplier-inject-order:${manifestId}:${orderId}`;
}

export function supplierVetOrderKey(orderId: string, decision: string): string {
  return `supplier-vet-order:${orderId}:${decision.toUpperCase()}`;
}

export function supplierImportCreateKey(
  scopeId: string,
  fileName: string,
  fileSizeBytes: number,
): string {
  return `supplier-import-create:${scopeId}:${stableHash(`${fileName}:${fileSizeBytes}`)}`;
}

export function supplierImportIngestKey(sessionId: string, csvBody: string): string {
  return `supplier-import-ingest:${sessionId}:${stableHash(csvBody)}`;
}

export function supplierImportApproveKey(sessionId: string): string {
  return `supplier-import-approve:${sessionId}`;
}

export function supplierImportApplyKey(sessionId: string): string {
  return `supplier-import-apply:${sessionId}`;
}

export function supplierBroadcastKey(
  scopeId: string,
  role: string,
  title: string,
  body: string,
): string {
  return `supplier-broadcast:${scopeId}:${stableHash(`${role}:${title}:${body}`)}`;
}

export function supplierPaymentBypassKey(orderId: string, reason: string): string {
  return `supplier-payment-bypass:${orderId}:${stableHash(reason)}`;
}

export function supplierApproveEarlyCompleteKey(driverId: string): string {
  return `supplier-approve-early-complete:${driverId}`;
}

export function driverConfirmPaymentBypassKey(driverId: string, orderId: string): string {
  return `driver-confirm-payment-bypass:${driverId}:${orderId}`;
}

export function driverBypassOffloadKey(driverId: string, orderId: string): string {
  return `driver-bypass-offload:${driverId}:${orderId}`;
}

export function driverReportShopClosedKey(driverId: string, orderId: string): string {
  return `driver-report-shop-closed:${driverId}:${orderId}`;
}

export function supplierShopClosedResolveKey(attemptId: string, action: string): string {
  return `shop-closed-resolve:${attemptId}:${action}`;
}

export function driverDepartKey(driverId: string, truckId: string): string {
  return `driver-depart:${driverId}:${truckId}`;
}

export function driverReturnCompleteKey(driverId: string, truckId: string): string {
  return `driver-return-complete:${driverId}:${truckId}`;
}

export function driverSyncBatchKey(driverId: string, orderSignatures: string[]): string {
  const sorted = [...orderSignatures].map((s) => s.trim()).filter(Boolean).sort();
  return `driver-sync-batch:${driverId}:${stableHash(sorted.join(","))}`;
}

export function driverMarkArrivedKey(orderId: string): string {
  return `driver-mark-arrived-${orderId}`;
}

export function driverSplitPaymentKey(
  driverId: string,
  orderId: string,
  cashMinor: number,
  cardMinor: number,
): string {
  return `driver-split-payment:${driverId}:${orderId}:${cashMinor}:${cardMinor}`;
}

export function driverCreditDeliveryKey(driverId: string, orderId: string): string {
  return `driver-credit-delivery:${driverId}:${orderId}`;
}

export function driverMissingItemsKey(driverId: string, orderId: string): string {
  return `driver-missing-items:${driverId}:${orderId}`;
}

export function payloadMissingItemsKey(orderId: string): string {
  return `payload-missing-items-${orderId}`;
}

export function driverReportDamageKey(driverId: string, orderId: string): string {
  return `driver-report-damage:${driverId}:${orderId}`;
}

export function driverNegotiateKey(driverId: string, orderId: string): string {
  return `driver-negotiate:${driverId}:${orderId}`;
}

/** Matches retailer Android/iOS/desktop procurement order-create keys. */
export function retailerOrderCreateKey(procurementFingerprint: string): string {
  return `retailer-procurement:${procurementFingerprint}`;
}

export function retailerConfirmCashKey(orderId: string): string {
  return `retailer-confirm-cash:${orderId}`;
}

export function retailerCancelKey(orderId: string): string {
  return `retailer-cancel:${orderId}`;
}

export function retailerRequestCancelKey(orderId: string): string {
  return `retailer-request-cancel:${orderId}`;
}

export function retailerConfirmPreorderKey(orderId: string): string {
  return `retailer-confirm-preorder:${orderId}`;
}

export function retailerConfirmAIKey(orderId: string): string {
  return `retailer-confirm-ai:${orderId}`;
}

export function adminOrderAssignKey(orderId: string, driverId: string): string {
  return `admin-order-assign:${orderId}:${driverId}`;
}

export function adminOrderStatusPatchKey(orderId: string, status: string): string {
  return `admin-order-status:${orderId}:${status}`;
}

export function warehouseEmergencyTransferKey(
  warehouseId: string,
  volumeVu: number,
  notes?: string,
): string {
  return `warehouse-emergency-transfer:${warehouseId}:${volumeVu}:${stableHash(notes ?? "")}`;
}

export function warehouseForceReceiveKey(
  warehouseId: string,
  volumeVu: number,
  notes?: string,
  factoryId?: string,
): string {
  return `warehouse-force-receive:${warehouseId}:${factoryId ?? ""}:${volumeVu}:${stableHash(notes ?? "")}`;
}

export function warehouseReceiveTransferKey(transferId: string): string {
  return `warehouse-receive-transfer:${transferId}`;
}

export function factoryManifestStartLoadingKey(manifestId: string): string {
  return `factory-start-loading:${manifestId}`;
}

export function factoryManifestSealKey(manifestId: string, factoryId: string): string {
  return `factory-manifest-seal:${factoryId}:${manifestId}`;
}

export function factoryManifestDispatchKey(manifestId: string, factoryId: string): string {
  return `factory-manifest-dispatch:${factoryId}:${manifestId}`;
}

export function factoryManifestCompleteKey(manifestId: string, factoryId: string): string {
  return `factory-manifest-complete:${factoryId}:${manifestId}`;
}

export function factoryBatchDispatchKey(factoryId: string, transferIds: string[]): string {
  const sorted = [...transferIds].map((id) => id.trim()).filter(Boolean).sort();
  return `factory-dispatch:${factoryId}:${stableHash(sorted.join(","))}`;
}

export function factoryManifestRebalanceKey(
  manifestId: string,
  transferId: string,
  targetFingerprint: string,
): string {
  return `factory-manifest-rebalance:${manifestId}:${transferId}:${stableHash(targetFingerprint)}`;
}

export function factoryManifestCancelTransferKey(manifestId: string, transferId: string): string {
  return `factory-manifest-cancel-transfer:${manifestId}:${transferId}`;
}

export function factoryManifestCancelKey(manifestId: string, reason?: string): string {
  return `factory-manifest-cancel:${manifestId}:${stableHash(reason ?? "")}`;
}

export function factoryTransferCreateKey(
  factoryId: string,
  orderId: string,
  totalVu: number,
  driverId = "",
  vehicleId = "",
): string {
  return `factory-transfer-create:${factoryId}:${stableHash(`${orderId}:${totalVu}:${driverId}:${vehicleId}`)}`;
}

export function factoryTransferTransitionKey(transferId: string, targetState: string): string {
  return `factory-transfer-transition:${transferId}:${targetState.trim().toUpperCase()}`;
}
