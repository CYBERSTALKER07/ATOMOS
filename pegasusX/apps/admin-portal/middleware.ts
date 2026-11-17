import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { verifyAdminJwt } from "@/lib/jwt-verify";

export async function middleware(request: NextRequest) {
  const { pathname, search } = request.nextUrl;

  if (
    pathname.startsWith("/_next") ||
    pathname.startsWith("/api") ||
    pathname.includes(".") ||
    pathname === "/"
  ) {
    return NextResponse.next();
  }

  const token = request.cookies.get("pegasusx_admin_token")?.value;
  const verified = token ? await verifyAdminJwt(token) : null;

  if (!verified) {
    const returnTo = encodeURIComponent(`${pathname}${search}`);
    const loginUrl = new URL(`/?return_to=${returnTo}`, request.url);
    const res = NextResponse.redirect(loginUrl);
    if (token) {
      res.cookies.delete("pegasusx_admin_token");
    }
    return res;
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
