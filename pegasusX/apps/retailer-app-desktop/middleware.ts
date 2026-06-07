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

  // Allow API routes
  if (pathname.startsWith('/api/')) {
    return NextResponse.next();
  }

  const payload = hasValidToken ? decodeJwtPayload(token) : null;
  const isConfigured = payload?.is_configured === true;

  // Redirect root and auth pages based on token
  if (pathname === '/' || pathname.startsWith('/auth')) {
    if (hasValidToken) {
      if (!isConfigured) {
        return NextResponse.redirect(new URL('/setup', request.url));
      }
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    if (pathname === '/') {
      return NextResponse.redirect(new URL('/auth/login', request.url));
    }
    return NextResponse.next();
  }

  // Allow setup path if valid token but not configured
  if (pathname.startsWith('/setup')) {
    if (!hasValidToken) {
      return NextResponse.redirect(new URL('/auth/login', request.url));
    }
    if (isConfigured) {
      return NextResponse.redirect(new URL('/dashboard', request.url));
    }
    return NextResponse.next();
  }

  if (!hasValidToken) {
    const res = NextResponse.redirect(new URL('/auth/login', request.url));
    if (token && isTokenExpired(token)) {
      res.cookies.delete(RETAILER_JWT_COOKIE);
    }
    return res;
  }

  // Enforce configuration
  if (!isConfigured) {
    return NextResponse.redirect(new URL('/setup', request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    '/',
    '/auth/:path*',
    '/dashboard/:path*',
    '/catalog/:path*',
    '/orders/:path*',
    '/tracking/:path*',
    '/procurement/:path*',
    '/notifications/:path*',
    '/insights/:path*',
    '/settings/:path*',
    '/dock/:path*',
    '/setup/:path*',
  ],
};
