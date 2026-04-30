import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

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
  const token = request.cookies.get('warehouse_jwt')?.value;
  const hasValidToken = !!token && !isTokenExpired(token);

  if (pathname.startsWith('/auth/')) {
    if (hasValidToken) {
      return NextResponse.redirect(new URL('/', request.url));
    }
    return NextResponse.next();
  }

  if (!hasValidToken) {
    const res = NextResponse.redirect(new URL('/auth/login', request.url));
    if (token && isTokenExpired(token)) {
      res.cookies.delete('warehouse_jwt');
    }
    return res;
  }
  return NextResponse.next();
}

export const config = {
  matcher: [
    '/auth/:path*',
    '/',
    '/supply-requests/:path*',
    '/demand-forecast/:path*',
    '/dispatch-locks/:path*',
    '/staff/:path*',
    '/drivers/:path*',
    '/vehicles/:path*',
    '/orders/:path*',
    '/inventory/:path*',
    '/products/:path*',
    '/manifests/:path*',
    '/analytics/:path*',
    '/crm/:path*',
    '/returns/:path*',
    '/treasury/:path*',
    '/dispatch/:path*',
    '/payment-config/:path*',
  ],
};
