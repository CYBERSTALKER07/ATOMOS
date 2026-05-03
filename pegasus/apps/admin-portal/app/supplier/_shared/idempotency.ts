export function buildSupplierOrgInviteIdempotencyKey(
  name: string,
  email: string,
  phone: string,
  supplierRole: string,
  assignedWarehouseId: string,
  assignedFactoryId: string,
): string {
  return [
    'supplier-org-invite',
    name.trim().toUpperCase(),
    email.trim().toLowerCase(),
    phone.trim(),
    supplierRole.trim().toUpperCase(),
    assignedWarehouseId.trim(),
    assignedFactoryId.trim(),
  ].join(':');
}

function stableSerialize(value: unknown): string {
  if (Array.isArray(value)) {
    return `[${value.map((item) => stableSerialize(item)).join(',')}]`;
  }
  if (value && typeof value === 'object') {
    return `{${Object.entries(value as Record<string, unknown>)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, nestedValue]) => `${key}:${stableSerialize(nestedValue)}`)
      .join(',')}}`;
  }
  return JSON.stringify(value);
}

export function buildSupplierProfileUpdateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-profile-update', stableSerialize(payload)].join(':');
}

export function buildSupplierShiftIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-shift-update', stableSerialize(payload)].join(':');
}

export function buildSupplierOrgMemberActionIdempotencyKey(
  userId: string,
  method: 'PUT' | 'DELETE',
  payload?: Record<string, unknown>,
): string {
  return ['supplier-org-member', method, userId.trim(), stableSerialize(payload ?? {})].join(':');
}

export function buildSupplierWarehouseCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-warehouse-create', stableSerialize(payload)].join(':');
}

export function buildSupplierWarehouseActionIdempotencyKey(
  warehouseId: string,
  method: 'PUT' | 'DELETE',
  payload?: Record<string, unknown>,
): string {
  return ['supplier-warehouse', method, warehouseId.trim(), stableSerialize(payload ?? {})].join(':');
}

export function buildSupplierWarehouseCoverageIdempotencyKey(
  warehouseId: string,
  payload: Record<string, unknown>,
): string {
  return ['supplier-warehouse-coverage', warehouseId.trim(), stableSerialize(payload)].join(':');
}

export function buildSupplierFactoryCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-factory-create', stableSerialize(payload)].join(':');
}

export function buildSupplierFactoryWarehouseAssignmentIdempotencyKey(
  factoryId: string,
  warehouseIds: string[],
): string {
  const normalizedWarehouseIds = warehouseIds.map((id) => id.trim()).filter(Boolean).sort();
  return ['supplier-factory-warehouse-assignment', factoryId.trim(), stableSerialize(normalizedWarehouseIds)].join(':');
}

export function buildSupplierCountryOverrideSaveIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-country-override-save', stableSerialize(payload)].join(':');
}

export function buildSupplierCountryOverrideDeleteIdempotencyKey(countryCode: string): string {
  return ['supplier-country-override-delete', countryCode.trim().toUpperCase()].join(':');
}

export function buildSupplierPricingRuleUpsertIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-pricing-rule-upsert', stableSerialize(payload)].join(':');
}

export function buildSupplierPricingRuleDeleteIdempotencyKey(tierId: string): string {
  return ['supplier-pricing-rule-delete', tierId.trim()].join(':');
}

export function buildSupplierRetailerOverrideCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-retailer-override-create', stableSerialize(payload)].join(':');
}

export function buildSupplierRetailerOverrideDeleteIdempotencyKey(overrideId: string): string {
  return ['supplier-retailer-override-delete', overrideId.trim()].join(':');
}

export function buildSupplierReturnResolveIdempotencyKey(
  lineItemId: string,
  resolution: string,
  notes: string,
): string {
  return ['supplier-return-resolve', lineItemId.trim(), resolution.trim().toUpperCase(), JSON.stringify(notes.trim())].join(':');
}

export function buildSupplierFleetDriverCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-fleet-driver-create', stableSerialize(payload)].join(':');
}

export function buildSupplierFleetDriverAssignIdempotencyKey(driverId: string, vehicleId: string): string {
  return ['supplier-fleet-driver-assign', driverId.trim(), vehicleId.trim()].join(':');
}

export function buildSupplierFleetVehicleCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-fleet-vehicle-create', stableSerialize(payload)].join(':');
}

export function buildSupplierFleetVehicleDeactivateIdempotencyKey(vehicleId: string): string {
  return ['supplier-fleet-vehicle-deactivate', vehicleId.trim()].join(':');
}

export function buildSupplierFleetClearReturnsIdempotencyKey(vehicleId: string): string {
  return ['supplier-fleet-clear-returns', vehicleId.trim()].join(':');
}

export function buildSupplierManifestInjectOrderIdempotencyKey(manifestId: string, orderId: string): string {
  return ['supplier-manifest-inject-order', manifestId.trim(), orderId.trim()].join(':');
}

export function buildSupplierManifestSealIdempotencyKey(manifestId: string, reason: string): string {
  return ['supplier-manifest-seal', manifestId.trim(), JSON.stringify(reason.trim())].join(':');
}

export function buildSupplierAutoDispatchIdempotencyKey(excludedTruckIds: string[]): string {
  return ['supplier-auto-dispatch', stableSerialize([...excludedTruckIds].sort())].join(':');
}

export function buildSupplierManualDispatchIdempotencyKey(driverId: string, orderIds: string[]): string {
  return ['supplier-manual-dispatch', driverId.trim(), stableSerialize([...orderIds].sort())].join(':');
}

export function buildSupplierFleetDispatchIdempotencyKey(routeId: string, orderIds: string[]): string {
  return ['supplier-fleet-dispatch', routeId.trim(), stableSerialize([...orderIds].sort())].join(':');
}

export function buildSupplierFleetReassignIdempotencyKey(newRouteId: string, orderIds: string[]): string {
  return ['supplier-fleet-reassign', newRouteId.trim(), stableSerialize([...orderIds].sort())].join(':');
}

export function buildSupplierDepotReconciliationIdempotencyKey(
  lineItemIds: string[],
  action: 'RESTOCK' | 'WRITE_OFF_DAMAGED',
): string {
  return ['supplier-depot-reconciliation', action, stableSerialize([...lineItemIds].sort())].join(':');
}

export function buildSupplierShopClosedResolveIdempotencyKey(attemptId: string, action: string): string {
  return ['supplier-shop-closed-resolve', attemptId.trim(), action.trim().toUpperCase()].join(':');
}

export function buildSupplierInventoryAdjustIdempotencyKey(
  productId: string,
  adjustment: number,
  reason: string,
): string {
  return ['supplier-inventory-adjust', productId.trim(), String(adjustment), reason.trim().toUpperCase()].join(':');
}

export function buildSupplierProductUpdateIdempotencyKey(
  skuId: string,
  payload: Record<string, unknown>,
): string {
  return ['supplier-product-update', skuId.trim(), stableSerialize(payload)].join(':');
}

export function buildSupplierApproveCancelIdempotencyKey(orderId: string): string {
  return ['supplier-approve-cancel', orderId.trim()].join(':');
}

export function buildSupplierResolveCreditIdempotencyKey(orderId: string, decision: string): string {
  return ['supplier-resolve-credit', orderId.trim(), decision.trim().toUpperCase()].join(':');
}

export function buildSupplierApproveEarlyCompleteIdempotencyKey(driverId: string): string {
  return ['supplier-approve-early-complete', driverId.trim()].join(':');
}

export function buildSupplierNegotiationResolveIdempotencyKey(proposalId: string, decision: string): string {
  return ['supplier-negotiation-resolve', proposalId.trim(), decision.trim().toUpperCase()].join(':');
}

export function buildSupplierWarehouseStaffCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-warehouse-staff-create', stableSerialize(payload)].join(':');
}

export function buildSupplierWarehouseStaffToggleIdempotencyKey(
  workerId: string,
  payload: Record<string, unknown>,
): string {
  return ['supplier-warehouse-staff-toggle', workerId.trim(), stableSerialize(payload)].join(':');
}
