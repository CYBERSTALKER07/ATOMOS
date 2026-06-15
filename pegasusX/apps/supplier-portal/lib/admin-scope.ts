import { decodeJwtPayload, readTokenFromCookie } from '@/lib/auth';

const ASSIGN_ORDER_ROLES = new Set(['ADMIN', 'WAREHOUSE_ADMIN', 'FACTORY_ADMIN']);
const PATCH_ORDER_STATUS_ROLES = new Set(['ADMIN']);

function readJwtRole(): string {
  const token = readTokenFromCookie();
  if (!token) return '';
  const claims = decodeJwtPayload(token);
  return typeof claims?.role === 'string' ? claims.role : '';
}

/** True when JWT may assign a driver to an order. */
export function canAssignOrder(): boolean {
  return ASSIGN_ORDER_ROLES.has(readJwtRole());
}

/** True when JWT may patch canonical order status (ADMIN only on supplier portal). */
export function canPatchOrderStatus(): boolean {
  return PATCH_ORDER_STATUS_ROLES.has(readJwtRole());
}

/** True when any admin order operation is available on the orders page. */
export function canAdminOrderOps(): boolean {
  return canAssignOrder() || canPatchOrderStatus();
}
