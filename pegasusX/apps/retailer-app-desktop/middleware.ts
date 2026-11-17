import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

const RETAILER_JWT_COOKIE = 'pegasus_retailer_jwt';
const RETAILER_REFRESH_COOKIE = 'pegasus_retailer_refresh';

const API_BASE = (
  process.env.NEXT_PUBLIC_API_URL ||
  'http://localhost:8180'
).replace(/\/$/, '');

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

async function refreshRetailerToken(refreshToken: string): Promise<{ token?: string; refresh_token?: string } | null> {
  try {
    const res = await fetch(`${API_BASE}/v1/auth/retailer/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { token?: string; refresh_token?: string };
    if (!data.token) return null;
    return data;
  } catch {
    return null;
  }
}

function applyCookies(
  res: NextResponse,
  refreshed?: { token?: string; refresh_token?: string } | null,
  isHttps = false,
): NextResponse {
  if (refreshed?.token) {
    res.cookies.set(RETAILER_JWT_COOKIE, refreshed.token, {
      path: '/',
      maxAge: 86400,
      sameSite: 'strict',
      secure: isHttps,
    });
  }
  if (refreshed?.refresh_token) {
    res.cookies.set(RETAILER_REFRESH_COOKIE, refreshed.refresh_token, {
      path: '/',
      maxAge: 604800,
      sameSite: 'strict',
      secure: isHttps,
    });
  }
  return res;
}

export async function middleware(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (pathname.startsWith('/_next') || pathname === '/favicon.ico') {
    return NextResponse.next();
  }

  // Allow API routes
  if (pathname.startsWith('/api/')) {
    return NextResponse.next();
  }

  let token = request.cookies.get(RETAILER_JWT_COOKIE)?.value;
  const refreshToken = request.cookies.get(RETAILER_REFRESH_COOKIE)?.value;
  const isHttps = request.nextUrl.protocol === 'https:';

  let refreshedSession: { token?: string; refresh_token?: string } | null = null;

  // Refresh-before-delete: when jwt is missing or expired, attempt refresh
  if ((!token || isTokenExpired(token)) && refreshToken) {
    refreshedSession = await refreshRetailerToken(refreshToken);
    if (refreshedSession?.token) {
      token = refreshedSession.token;
    }
  }

  const hasValidToken = !!token && !isTokenExpired(token);
  const payload = hasValidToken ? decodeJwtPayload(token!) : null;
  const isConfigured = payload?.is_configured === true;

  // Redirect root and auth pages based on token
  if (pathname === '/' || pathname.startsWith('/auth')) {
    if (hasValidToken) {
      if (!isConfigured) {
        return applyCookies(NextResponse.redirect(new URL('/setup/tax', request.url)), refreshedSession, isHttps);
      }
      return applyCookies(NextResponse.redirect(new URL('/dashboard', request.url)), refreshedSession, isHttps);
    }
    if (pathname === '/') {
      return NextResponse.redirect(new URL('/auth/register', request.url));
    }
    return NextResponse.next();
  }

  // Allow setup path if valid token but not configured
  if (pathname.startsWith('/setup')) {
    if (!hasValidToken) {
      const authPath = token ? '/auth/login' : '/auth/register';
      const res = NextResponse.redirect(new URL(authPath, request.url));
      // Only delete when no refresh token exists or refresh fails
      if (token && isTokenExpired(token)) {
        res.cookies.delete(RETAILER_JWT_COOKIE);
      }
      return res;
    }
    if (isConfigured) {
      return applyCookies(NextResponse.redirect(new URL('/dashboard', request.url)), refreshedSession, isHttps);
    }
    return applyCookies(NextResponse.next(), refreshedSession, isHttps);
  }

  if (!hasValidToken) {
    const authPath = token ? '/auth/login' : '/auth/register';
    const res = NextResponse.redirect(new URL(authPath, request.url));
    // Only delete when no refresh token exists or refresh fails
    if (token && isTokenExpired(token)) {
      res.cookies.delete(RETAILER_JWT_COOKIE);
    }
    return res;
  }

  // Enforce configuration
  if (!isConfigured) {
    return applyCookies(NextResponse.redirect(new URL('/setup/tax', request.url)), refreshedSession, isHttps);
  }

  return applyCookies(NextResponse.next(), refreshedSession, isHttps);
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
