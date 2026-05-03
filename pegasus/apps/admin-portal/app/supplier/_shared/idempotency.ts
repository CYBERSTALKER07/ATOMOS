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

export function buildSupplierWarehouseStaffCreateIdempotencyKey(payload: Record<string, unknown>): string {
  return ['supplier-warehouse-staff-create', stableSerialize(payload)].join(':');
}

export function buildSupplierWarehouseStaffToggleIdempotencyKey(
  workerId: string,
  payload: Record<string, unknown>,
): string {
  return ['supplier-warehouse-staff-toggle', workerId.trim(), stableSerialize(payload)].join(':');
}
