import { decodeJwtPayload, readTokenFromCookie } from "@/lib/auth";

export function supplierScopeId(): string {
  const token = readTokenFromCookie();
  if (!token) return "supplier";
  const claims = decodeJwtPayload(token);
  return typeof claims?.supplier_id === "string" ? claims.supplier_id : "supplier";
}
