import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/** Decode JWT payload without verification (Edge-safe). Returns null if malformed. */
function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

function isTokenExpired(token: string): boolean {
  const payload = decodeJwtPayload(token);
  if (!payload || typeof payload.exp !== 'number') return true;
  return payload.exp * 1000 < Date.now();
}

export function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const token = request.cookies.get('admin_jwt')?.value || request.cookies.get('supplier_jwt')?.value;
  const hasValidToken = !!token && !isTokenExpired(token);

  // ── Auth pages: redirect to dashboard only if token is still valid ──
  if (pathname.startsWith('/auth/') || pathname === '/login' || pathname === '/signup') {
    if (hasValidToken) {
      return NextResponse.redirect(new URL('/', request.url));
    }
    return NextResponse.next();
  }

  // ── All other routes: require a non-expired token ──
  if (!hasValidToken) {
    const res = NextResponse.redirect(new URL('/auth/login', request.url));
    // Clear stale cookies so the login page isn't redirect-looped
    if (token && isTokenExpired(token)) {
      res.cookies.delete('admin_jwt');
      res.cookies.delete('supplier_jwt');
    }
    return res;
  }

  // ── Onboarding gate: unconfigured suppliers must set up billing first ──
  const payload = token ? decodeJwtPayload(token) : null;
  const isConfigured = payload?.is_configured as boolean | undefined;
  const role = (payload?.role as string) || '';
  if (
    (role === 'SUPPLIER' || role === 'ADMIN') &&
    isConfigured === false &&
    !pathname.startsWith('/setup/') &&
    !pathname.startsWith('/auth/') &&
    !pathname.startsWith('/api/')
  ) {
    return NextResponse.redirect(new URL('/setup/billing', request.url));
  }
  const supplierRole = (payload?.supplier_role as string) || '';
  if (supplierRole === 'NODE_ADMIN') {
    const globalOnlyPrefixes = [
      '/ledger', '/reconciliation', '/treasury', '/kyc',
      '/configuration', '/dlq', '/dashboard',
      '/admin/', '/supplier/factories', '/supplier/supply-lanes',
      '/supplier/warehouses', '/supplier/payment-config',
      '/supplier/settings', '/supplier/org',
    ];
    if (globalOnlyPrefixes.some(p => pathname === p || pathname.startsWith(p + '/'))) {
      return NextResponse.redirect(new URL('/', request.url));
    }
  }

  // ── Factory staff gate: FACTORY_ADMIN/FACTORY_PAYLOADER cannot access warehouse/logistics routes ──
  if (supplierRole === 'FACTORY_ADMIN' || supplierRole === 'FACTORY_PAYLOADER') {
    const factoryBlockedPrefixes = [
      // Global sovereignty (same as NODE_ADMIN)
      '/ledger', '/reconciliation', '/treasury', '/kyc',
      '/configuration', '/dlq', '/dashboard',
      '/admin/', '/supplier/factories', '/supplier/supply-lanes',
      '/supplier/warehouses', '/supplier/payment-config',
      '/supplier/settings', '/supplier/org',
      // Warehouse/logistics operations
      '/supplier/manifests', '/supplier/dispatch', '/supplier/fleet',
      '/supplier/delivery-zones', '/supplier/returns',
      '/supplier/depot-reconciliation', '/supplier/crm',
      '/supplier/staff', '/supplier/pricing',
      '/supplier/country-overrides',
      '/fleet',
    ];
    if (factoryBlockedPrefixes.some(p => pathname === p || pathname.startsWith(p + '/'))) {
      return NextResponse.redirect(new URL('/', request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    // Auth pages (reverse redirect for already-auth'd users)
    '/auth/:path*',
    // Supplier sub-routes (all protected — including dashboard)
    '/supplier/dashboard/:path*',
    '/supplier/catalog/:path*',
    '/supplier/crm/:path*',
    '/supplier/fleet/:path*',
    '/supplier/inventory/:path*',
    '/supplier/manifests/:path*',
    '/supplier/orders/:path*',
    '/supplier/dispatch/:path*',
    '/supplier/pricing/:path*',
    '/supplier/returns/:path*',
    '/supplier/staff/:path*',
    '/supplier/org/:path*',
    '/supplier/onboarding/:path*',
    '/supplier/delivery-zones/:path*',
    '/supplier/depot-reconciliation/:path*',
    '/supplier/country-overrides/:path*',
    '/supplier/factories/:path*',
    '/supplier/supply-lanes/:path*',
    '/supplier/warehouses/:path*',
    '/supplier/payment-config/:path*',
    '/supplier/settings/:path*',
    '/supplier/analytics/:path*',
    '/supplier/products/:path*',
    // Admin routes
    '/',
    '/dashboard/:path*',
    '/ledger/:path*',
    '/reconciliation/:path*',
    '/treasury/:path*',
    '/fleet/:path*',
    '/kyc/:path*',
    '/configuration/:path*',
    '/dlq/:path*',
    '/admin/:path*',
    // Post-registration onboarding
    '/setup/:path*',
  ],
};
