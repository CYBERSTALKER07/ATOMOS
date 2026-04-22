'use client';

import { useState, useEffect, useMemo } from 'react';

export type SupplierRole = 'GLOBAL_ADMIN' | 'NODE_ADMIN' | 'FACTORY_ADMIN' | 'FACTORY_PAYLOADER' | '';

interface AuthClaims {
  userId: string;
  role: string;
  supplierRole: SupplierRole;
  warehouseId: string;
  factoryId: string;
  exp: number;
}

const EMPTY_CLAIMS: AuthClaims = {
  userId: '',
  role: '',
  supplierRole: '',
  warehouseId: '',
  factoryId: '',
  exp: 0,
};

/** Decode JWT payload without verification (client-safe). */
function decodeJwt(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function readToken(): string {
  if (typeof document === 'undefined') return '';
  const m = document.cookie.match(/(?:^|; )admin_jwt=([^;]*)/) ||
            document.cookie.match(/(?:^|; )supplier_jwt=([^;]*)/);
  return m ? decodeURIComponent(m[1]) : '';
}

function extractClaims(token: string): AuthClaims {
  if (!token) return EMPTY_CLAIMS;
  const p = decodeJwt(token);
  if (!p) return EMPTY_CLAIMS;
  return {
    userId: (p.user_id as string) || (p.sub as string) || '',
    role: (p.role as string) || '',
    supplierRole: ((p.supplier_role as string) || '') as SupplierRole,
    warehouseId: (p.warehouse_id as string) || '',
    factoryId: (p.factory_id as string) || '',
    exp: (p.exp as number) || 0,
  };
}

/**
 * Returns decoded auth claims from the JWT cookie.
 * Hydration-safe: returns empty claims on server and first render.
 */
export function useAuth() {
  const [claims, setClaims] = useState<AuthClaims>(EMPTY_CLAIMS);

  useEffect(() => {
    setClaims(extractClaims(readToken()));
  }, []);

  return useMemo(() => {
    const sr = claims.supplierRole;
    return {
      ...claims,
      // Empty supplier_role = backward-compat GLOBAL_ADMIN (root supplier login)
      isGlobalAdmin: sr === '' || sr === 'GLOBAL_ADMIN',
      isNodeAdmin: sr === 'NODE_ADMIN',
      isFactoryAdmin: sr === 'FACTORY_ADMIN',
      isFactoryPayloader: sr === 'FACTORY_PAYLOADER',
      isFactoryStaff: sr === 'FACTORY_ADMIN' || sr === 'FACTORY_PAYLOADER',
      /** Convenience: the scoped node ID (warehouse or factory) */
      scopeId: claims.warehouseId || claims.factoryId,
    };
  }, [claims]);
}
