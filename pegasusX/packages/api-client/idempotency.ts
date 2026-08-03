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

export function driverFiscalRetryKey(orderId: string, driverId: string): string {
  return `driver-fiscal-retry:${driverId}:${orderId}:${Math.floor(Date.now() / 60_000)}`;
}

export function adminForceCompleteKey(orderId: string, reasonCode: string): string {
  return `admin-force-complete:${orderId}:${stableHash(reasonCode)}`;
}

export function driverAmendOrderKey(
  orderId: string,
  driverId: string,
  itemFingerprint: string,
): string {
  return `driver-amend:${driverId}:${orderId}:${stableHash(itemFingerprint)}`;
}

export function driverTransitionStateKey(
  orderId: string,
  driverId: string,
  newState: string,
): string {
  return `driver-transition-state:${driverId}:${orderId}:${newState.trim().toUpperCase()}`;
}

export function driverAvailabilityKey(
  driverId: string,
  onShift: boolean,
  reason = '',
  note = '',
): string {
  const fingerprint = stableHash(`${onShift}:${reason.trim()}:${note.trim()}`);
  return `driver-availability:${driverId}:${fingerprint}`;
}

export function retailerCheckoutKey(retailerId: string, cartFingerprint: string): string {
  return `retailer-checkout:${retailerId}:${stableHash(cartFingerprint)}`;
}

/** Unified checkout cart session — gateway + sorted line-item fingerprint (mobile + desktop). */
export function retailerUnifiedCheckoutKey(gateway: string, cartFingerprint: string): string {
  return `retailer-checkout:${gateway}:${cartFingerprint}`;
}

export function retailerCardCheckoutKey(orderId: string, gateway: string): string {
  return `retailer-card-checkout:${orderId}:${gateway}`;
}

export function retailerCashCheckoutKey(orderId: string): string {
  return `retailer-cash-checkout:${orderId}`;
}

export function retailerSupplierAddKey(supplierId: string): string {
  return `retailer-supplier-add:${supplierId}`;
}

/** Family → Team migrate (owner staff.manage). */
export function retailerFamilyMigrateKey(retailerId: string, dayBucket: string): string {
  return `retailer-family-migrate:${retailerId}:${dayBucket}`;
}

/** Auto-order worker manual run (order.place). */
export function retailerAutoOrderRunKey(retailerId: string, dayBucket: string, mode: string): string {
  return `retailer-auto-order-run:${retailerId}:${dayBucket}:${mode}`;
}

/** Offline/online POS sale — stable per client_sale_id (never regenerate on retry). */
export function retailerPosSaleKey(clientSaleId: string): string {
  return `pos-sale:${clientSaleId.trim()}`;
}

export function retailerSupplierRemoveKey(supplierId: string): string {
  return `retailer-supplier-remove:${supplierId}`;
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

export function warehouseOrderProposeDeliveryKey(orderId: string, proposedDate: string, reason: string): string {
  return `warehouse-order-propose-delivery:${orderId}:${stableHash(proposedDate)}:${stableHash(reason)}`;
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

export function supplierManifestSealAllKey(splitGroupId: string, supplierId: string): string {
  return `supplier-manifest-seal-all:${supplierId}:${splitGroupId}`;
}

export function payloadSealAllKey(splitGroupId: string, payloaderId: string): string {
  return `payload-seal-all:${payloaderId}:${splitGroupId}`;
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

export function supplierOrgMemberCreateKey(supplierId: string, phone: string): string {
  return `supplier-org-member-create:${supplierId}:${stableHash(phone)}`;
}

export function supplierOrgMemberUpdateKey(supplierId: string, userId: string, revision: string): string {
  return `supplier-org-member-update:${supplierId}:${userId}:${stableHash(revision)}`;
}

export function supplierOrgMemberDeactivateKey(supplierId: string, userId: string): string {
  return `supplier-org-member-deactivate:${supplierId}:${userId}`;
}

export function supplierFleetDriverCreateKey(supplierId: string, phone: string): string {
  return `supplier-fleet-driver-create:${supplierId}:${stableHash(phone)}`;
}

export function supplierFleetVehicleCreateKey(supplierId: string, licensePlate: string): string {
  return `supplier-fleet-vehicle-create:${supplierId}:${stableHash(licensePlate)}`;
}

export function supplierChargebackKey(orderId: string, reason: string): string {
  return `supplier-chargeback:${orderId}:${stableHash(reason)}`;
}

export function supplierChargebackReversalKey(chargebackId: string, reason: string): string {
  return `supplier-chargeback-reversal:${chargebackId}:${stableHash(reason)}`;
}

export function supplierTopologyPutKey(supplierId: string, topologyFingerprint: string): string {
  return `supplier-topology-put:${supplierId}:${stableHash(topologyFingerprint)}`;
}

export function supplierReplenishmentTriggerKey(supplierId: string): string {
  return `supplier-replenishment-trigger:${supplierId}`;
}

export function supplierControlTowerZoneOverrideKey(supplierId: string, action: string, polygonFingerprint: string): string {
  return `supplier-control-tower-override:${supplierId}:${stableHash(`${action}:${polygonFingerprint}`)}`;
}

export function supplierPlanningScenarioKey(supplierId: string, factoryDowntimeHours: number, demandDeltaPct: number): string {
  return `supplier-planning-scenario:${supplierId}:${factoryDowntimeHours}:${demandDeltaPct}`;
}

export function supplierSeasonalOverrideCreateKey(supplierId: string, startDate: string, endDate: string): string {
  return `supplier-seasonal-override:${supplierId}:${stableHash(`${startDate}:${endDate}`)}`;
}

export function supplierGovernedAgentKey(supplierId: string, action: string, idempotencyKey: string): string {
  return `supplier-planning-agent:${supplierId}:${action}:${stableHash(idempotencyKey)}`;
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

export function supplierNegotiationResolveKey(proposalId: string, action: string): string {
  return `supplier-negotiate-resolve:${proposalId}:${action}`;
}

export function driverRequestEarlyCompleteKey(driverId: string, reason: string): string {
  return `driver-request-early-complete:${driverId}:${stableHash(reason)}`;
}

export function driverRouteReorderKey(driverId: string, routeId: string, orderSequence: string[]): string {
  const sorted = orderSequence.map((id) => id.trim()).filter(Boolean);
  return `driver-route-reorder:${driverId}:${routeId}:${stableHash(sorted.join(","))}`;
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

export function driverProximityUnlockKey(driverId: string, orderId: string): string {
  return `driver-proximity-unlock:${driverId}:${orderId}`;
}

export function driverPartialOffloadKey(driverId: string, orderId: string, fingerprint: string): string {
  return `driver-partial-offload:${driverId}:${orderId}:${stableHash(fingerprint)}`;
}

export function supplierShopClosedResolveKey(attemptId: string, action: string): string {
  return `shop-closed-resolve:${attemptId}:${action}`;
}

export function supplierResolveReturnKey(returnId: string, resolution: string): string {
  return `supplier-resolve-return:${returnId}:${resolution.trim().toUpperCase()}`;
}

export function supplierProfileUpdateKey(supplierId: string, payloadFingerprint: string): string {
  return `supplier-profile-update:${supplierId}:${stableHash(payloadFingerprint)}`;
}

export function supplierConfigureKey(supplierId: string, payloadFingerprint: string): string {
  return `supplier-configure:${supplierId}:${stableHash(payloadFingerprint)}`;
}

export function supplierBusinessSetupKey(supplierId: string, payloadFingerprint: string): string {
  return `supplier-business-setup:${supplierId}:${stableHash(payloadFingerprint)}`;
}

export function supplierPricingRulePatchKey(supplierId: string, payloadFingerprint: string): string {
  return `supplier-pricing-rule-patch:${supplierId}:${stableHash(payloadFingerprint)}`;
}

export function supplierInventoryAdjustKey(
  supplierId: string,
  skuId: string,
  quantityDelta: number,
  version: number,
): string {
  return `supplier-inventory-adjust:${supplierId}:${skuId}:${quantityDelta}:${version}`;
}

export function supplierRetailerPriceOverrideCreateKey(
  supplierId: string,
  retailerId: string,
  productId: string,
  priceMinor: number,
): string {
  return `supplier-retailer-price-create:${supplierId}:${retailerId}:${productId}:${priceMinor}`;
}

export function supplierRetailerPriceOverrideDeleteKey(supplierId: string, overrideId: string): string {
  return `supplier-retailer-price-delete:${supplierId}:${overrideId}`;
}

export function supplierPromotionCreateKey(supplierId: string, payloadFingerprint: string): string {
  return `supplier-promotion-create:${supplierId}:${stableHash(payloadFingerprint)}`;
}

export function supplierPromotionUpdateKey(
  supplierId: string,
  promotionId: string,
  payloadFingerprint: string,
): string {
  return `supplier-promotion-update:${supplierId}:${promotionId}:${stableHash(payloadFingerprint)}`;
}

export function supplierPromotionDeactivateKey(supplierId: string, promotionId: string): string {
  return `supplier-promotion-deactivate:${supplierId}:${promotionId}`;
}

export function driverSupplyTransferArriveKey(driverId: string, transferId: string): string {
  return `driver-supply-arrive:${driverId}:${transferId}`;
}

export function warehouseCreateSupplyRequestKey(
  warehouseId: string,
  factoryId: string,
  priority: string,
  notes: string,
): string {
  return `warehouse-create-supply-request:${warehouseId}:${factoryId}:${priority}:${stableHash(notes)}`;
}

export function warehouseSupplyRequestTransitionKey(requestId: string, action: string): string {
  return `warehouse-supply-transition:${requestId}:${action.trim().toUpperCase()}`;
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

export function retailerShopClosedResponseKey(orderId: string, response: string): string {
  return `shop-closed-response:${orderId}:${response}`;
}

export function retailerConfirmPreorderKey(orderId: string): string {
  return `retailer-confirm-preorder:${orderId}`;
}

export function retailerAcceptDeliveryProposalKey(orderId: string): string {
  return `retailer-accept-delivery-proposal:${orderId}`;
}

export function retailerRejectDeliveryProposalKey(orderId: string, reason?: string): string {
  return `retailer-reject-delivery-proposal:${orderId}:${stableHash(reason ?? "")}`;
}

export function retailerRejectPreorderKey(orderId: string, reason?: string): string {
  return `retailer-reject-preorder:${orderId}:${stableHash(reason ?? "")}`;
}

export function retailerConfirmAIKey(orderId: string): string {
  return `retailer-confirm-ai:${orderId}`;
}

export function retailerRejectAIKey(orderId: string, reason?: string): string {
  return `retailer-reject-ai:${orderId}:${stableHash(reason ?? '')}`;
}

export function retailerEditPreorderKey(orderId: string): string {
  return `retailer-edit-preorder:${orderId}`;
}

/** Matches mobile onboarding key `retailer-setup:{retailerId}`. */
export function retailerSetupKey(retailerId: string): string {
  return `retailer-setup:${retailerId}`;
}

export function retailerProfileUpdateKey(retailerId: string, payloadFingerprint: string): string {
  return `retailer-profile-update:${retailerId}:${stableHash(payloadFingerprint)}`;
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

export function warehouseCreateDriverKey(warehouseId: string, phone: string): string {
  return `warehouse-create-driver:${warehouseId}:${stableHash(phone)}`;
}

export function warehouseCreateStaffKey(warehouseId: string, phone: string): string {
  return `warehouse-create-staff:${warehouseId}:${stableHash(phone)}`;
}

export function warehouseCreateVehicleKey(warehouseId: string, licensePlate: string): string {
  return `warehouse-create-vehicle:${warehouseId}:${stableHash(licensePlate)}`;
}

export function warehouseAdjustInventoryKey(warehouseId: string, productId: string, quantity: number): string {
  return `warehouse-adjust-inventory:${warehouseId}:${productId}:${quantity}`;
}

export function warehouseAssignDriverVehicleKey(
  warehouseId: string,
  driverId: string,
  vehicleId: string,
): string {
  return `warehouse-assign-driver-vehicle:${warehouseId}:${driverId}:${vehicleId || "none"}`;
}

export function warehouseUpdateVehicleKey(
  warehouseId: string,
  vehicleId: string,
  isActive: boolean,
  unavailableReason?: string,
): string {
  const reason = isActive ? "active" : (unavailableReason ?? "MANUAL_HOLD").trim().toUpperCase();
  return `warehouse-update-vehicle:${warehouseId}:${vehicleId}:${isActive}:${reason}`;
}

export function warehouseInventoryPolicyKey(
  warehouseId: string,
  productId: string,
  policy: string,
): string {
  return `warehouse-inventory-policy:${warehouseId}:${productId}:${policy.trim().toUpperCase()}`;
}

export function warehouseReplenishmentInsightActionKey(insightId: string, action: string): string {
  return `warehouse-replenishment-action:${insightId}:${action.trim().toLowerCase()}`;
}

export function warehouseDispatchSettingsKey(warehouseId: string, autoDispatchEnabled: boolean): string {
  return `warehouse-dispatch-settings:${warehouseId}:${autoDispatchEnabled}`;
}

export function warehouseOpsSettingsKey(warehouseId: string, revision = ''): string {
  const base = `warehouse-ops-settings:${warehouseId}`;
  return revision ? `${base}:${revision}` : base;
}

export function warehouseOpsLocationKey(
  warehouseId: string,
  lat: number,
  lng: number,
  placeId?: string,
): string {
  const fingerprint = stableHash(`${lat.toFixed(6)}:${lng.toFixed(6)}:${placeId ?? ''}`);
  return `warehouse-ops-location:${warehouseId}:${fingerprint}`;
}

export function warehouseBroadcastKey(
  warehouseId: string,
  role: string,
  title: string,
  body: string,
): string {
  return `warehouse-broadcast:${warehouseId}:${role.trim().toUpperCase()}:${stableHash(`${title}:${body}`)}`;
}

export function warehouseBroadcastTemplateCreateKey(
  warehouseId: string,
  title: string,
  body: string,
): string {
  return `warehouse-broadcast-template-create:${warehouseId}:${stableHash(`${title}:${body}`)}`;
}

export function warehouseBroadcastTemplateDeleteKey(warehouseId: string, templateId: string): string {
  return `warehouse-broadcast-template-delete:${warehouseId}:${templateId}`;
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

export function factorySupplyRequestTransitionKey(requestId: string, action: string): string {
  return `factory-supply-transition:${requestId}:${action.trim().toUpperCase()}`;
}

export function factorySupplyRequestAcceptKey(requestId: string): string {
  return `factory-supply-accept:${requestId}`;
}

export function factoryOpsLocationKey(
  factoryId: string,
  lat: number,
  lng: number,
  placeId?: string,
): string {
  const fingerprint = stableHash(`${lat.toFixed(6)}:${lng.toFixed(6)}:${placeId ?? ''}`);
  return `factory-ops-location:${factoryId}:${fingerprint}`;
}

export function warehouseInboundScanKey(warehouseId: string, barcode: string, sessionId: string): string {
  return `warehouse-inbound-scan:${warehouseId}:${stableHash(barcode)}:${sessionId}`;
}

export function warehouseInboundConfirmKey(
  warehouseId: string,
  returnIds: string[],
  disposition: string,
): string {
  const sorted = [...returnIds].map((id) => id.trim()).filter(Boolean).sort().join(",");
  return `warehouse-inbound-confirm:${warehouseId}:${disposition}:${stableHash(sorted)}`;
}

export function payloadInboundScanKey(barcode: string, sessionId: string): string {
  return `payload-inbound-scan-${barcode}-${sessionId}`;
}

export function payloadInboundConfirmKey(returnIds: string[], disposition: string): string {
  const sorted = [...returnIds].map((id) => id.trim()).filter(Boolean).sort().join(",");
  return `payload-inbound-confirm-${disposition}-${sorted}`;
}

export function payloadManifestExceptionKey(manifestId: string, orderId: string): string {
  return `payload-manifest-exception-${manifestId}-${orderId}`;
}
