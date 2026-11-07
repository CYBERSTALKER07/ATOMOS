import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

export function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

  // Allow static files, api routes, and root
  if (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") ||
    pathname === "/"
  ) {
    return NextResponse.next();
  }

  const token = request.cookies.get("pegasusx_admin_token")?.value;

  // If visiting protected route without cookie, redirect to /?return_to=...
  if (!token) {
    const returnTo = encodeURIComponent(`${pathname}${search}`);
    const loginUrl = new URL(`/?return_to=${returnTo}`, request.url);
    return NextResponse.redirect(loginUrl);
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
