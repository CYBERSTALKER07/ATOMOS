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
  const token = request.cookies.get('pegasus_factory_jwt')?.value;
  const hasValidToken = !!token && !isTokenExpired(token);

  const payload = hasValidToken ? decodeJwtPayload(token) : null;
  const isConfigured = payload?.is_configured === true;

  // Auth pages: redirect to dashboard if already authenticated
  if (pathname.startsWith('/auth/')) {
    if (hasValidToken) {
      if (!isConfigured) {
        return NextResponse.redirect(new URL('/setup/factory', request.url));
      }
      return NextResponse.redirect(new URL('/', request.url));
    }
    return NextResponse.next();
  }

  // Setup pages
  if (pathname.startsWith('/setup')) {
    if (!hasValidToken) {
      const authPath = token ? '/auth/login' : '/auth/register';
      return NextResponse.redirect(new URL(authPath, request.url));
    }
    if (isConfigured) {
      return NextResponse.redirect(new URL('/', request.url));
    }
    return NextResponse.next();
  }

  // All other routes: require valid factory token
  if (!hasValidToken) {
    const authPath = token ? '/auth/login' : '/auth/register';
    const res = NextResponse.redirect(new URL(authPath, request.url));
    if (token && isTokenExpired(token)) {
      res.cookies.delete('pegasus_factory_jwt');
    }
    return res;
  }

  if (!isConfigured) {
    return NextResponse.redirect(new URL('/setup/factory', request.url));
  }
  return NextResponse.next();
}

export const config = {
  matcher: [
    '/auth/:path*',
    '/',
    '/loading-bay/:path*',
    '/transfers/:path*',
    '/fleet/:path*',
    '/staff/:path*',
    '/insights/:path*',
    '/analytics/:path*',
    '/manifests/:path*',
    '/manifest-exceptions/:path*',
    '/supply-requests/:path*',
    '/payload/:path*',
    '/payload-override/:path*',
    '/settings/:path*',
    '/setup/:path*',
  ],
};
