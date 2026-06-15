import { decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';

/** Warehouse operator home node from JWT (used for idempotency keys and scoped queries). */
export function warehouseHomeNodeId(): string {
  const token = readTokenFromCookie();
  if (!token) return '';
  const session = decodeJwtPayload(token);
  return typeof session?.home_node_id === 'string' ? session.home_node_id : '';
}

export function warehouseScopeQuery(): { warehouse_id?: string } {
  const warehouseId = warehouseHomeNodeId();
  return warehouseId ? { warehouse_id: warehouseId } : {};
}
