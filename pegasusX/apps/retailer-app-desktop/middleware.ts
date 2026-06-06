import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const RETAILER_JWT_COOKIE = 'pegasus_retailer_jwt';

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
  const token = request.cookies.get(RETAILER_JWT_COOKIE)?.value;
  const hasValidToken = !!token && !isTokenExpired(token);

  if (pathname.startsWith('/_next') || pathname === '/favicon.ico') {
    return NextResponse.next();
  }

  if (pathname === '/' || pathname.startsWith('/api/')) {
    if (hasValidToken && pathname === '/') {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    return NextResponse.next();
  }

  if (!hasValidToken) {
    const res = NextResponse.redirect(new URL('/', request.url));
    if (token && isTokenExpired(token)) {
      res.cookies.delete(RETAILER_JWT_COOKIE);
    }
    return res;
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/',
    '/dashboard/:path*',
    '/catalog/:path*',
    '/orders/:path*',
    '/tracking/:path*',
    '/procurement/:path*',
    '/notifications/:path*',
    '/insights/:path*',
    '/settings/:path*',
    '/dock/:path*',
  ],
};
