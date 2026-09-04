import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';

/** Sync `?lang=en|ru` into the language cookie for SSR + hreflang targets. */
export function middleware(request: NextRequest) {
  const lang = request.nextUrl.searchParams.get('lang');
  if (lang !== 'en' && lang !== 'ru') {
    return NextResponse.next();
  }

  const response = NextResponse.next();
  response.cookies.set('pegasus_lang', lang, {
    path: '/',
    maxAge: 60 * 60 * 24 * 365,
    sameSite: 'lax',
  });
  return response;
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp|ico)$).*)'],
};
