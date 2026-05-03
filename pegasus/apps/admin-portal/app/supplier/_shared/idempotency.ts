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
