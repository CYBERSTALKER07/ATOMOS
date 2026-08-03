import { decodeJwtPayload, getSupplierToken, readTokenFromCookie } from "@/lib/auth";

/** Soft scope for cache keys — may return a placeholder when unauthenticated. */
export function supplierScopeId(): string {
  return sessionSupplierId() ?? "supplier";
}

/**
 * Fail-closed session supplier id from JWT.
 * Returns null when missing — callers must not fall back to demo tenants.
 */
export function sessionSupplierId(): string | null {
  const token = getSupplierToken() || readTokenFromCookie();
  if (!token) return null;
  const claims = decodeJwtPayload(token);
  const sid =
    (typeof claims?.supplier_id === "string" && claims.supplier_id) ||
    (typeof claims?.SupplierID === "string" && claims.SupplierID) ||
    (typeof claims?.sid === "string" && claims.sid) ||
    "";
  const trimmed = sid.trim();
  return trimmed || null;
}
