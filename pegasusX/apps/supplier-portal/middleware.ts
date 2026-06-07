import { NextResponse, type NextRequest } from "next/server";

// Onboarding gate: any supplier whose JWT claim `is_configured` is false is
// redirected to /setup/billing until they finish bank + payment-gateway
// configuration (or explicitly skip). Public routes are excluded.
//
// HARD PRODUCT INVARIANT: this gate must not redirect to /auth/register —
// banking/payments live at /setup/billing, never inside the wizard.

const PUBLIC_PATHS = new Set<string>([
  "/",
  "/auth/register",
  "/auth/login",
  "/setup/billing",
  "/setup/business",
  "/favicon.ico",
]);

const PUBLIC_PREFIXES = ["/_next/", "/api/", "/static/"];

export function middleware(req: NextRequest) {
  const { pathname } = req.nextUrl;

  if (PUBLIC_PATHS.has(pathname)) return NextResponse.next();
  if (PUBLIC_PREFIXES.some((p) => pathname.startsWith(p))) return NextResponse.next();

  const session = req.cookies.get("supplier_jwt")?.value;
  if (!session || isTokenExpired(session)) {
    const url = req.nextUrl.clone();
    url.pathname = session ? "/auth/login" : "/auth/register";
    const res = NextResponse.redirect(url);
    if (session) {
      res.cookies.delete("supplier_jwt");
    }
    return res;
  }

  const isConfigured = readIsConfigured(session);
  if (!isConfigured) {
    const url = req.nextUrl.clone();
    url.pathname = "/setup/business";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

// Decode the JWT payload without verifying the signature. Verification
// happens on the backend; the gate only needs to read the is_configured
// claim to decide where to redirect. Anything malformed → treat as
// unconfigured (safest default — pushes the supplier toward setup).
function isTokenExpired(jwt: string): boolean {
  const parts = jwt.split(".");
  if (parts.length !== 3) return true;
  try {
    const json = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    const claims = JSON.parse(json) as { exp?: number };
    if (typeof claims.exp !== "number") return true;
    return claims.exp * 1000 < Date.now();
  } catch {
    return true;
  }
}

function readIsConfigured(jwt: string): boolean {
  const parts = jwt.split(".");
  if (parts.length !== 3) return false;
  try {
    const json = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    const claims = JSON.parse(json) as { is_configured?: boolean };
    return claims.is_configured === true;
  } catch {
    return false;
  }
}

export const config = {
  matcher: ["/((?!_next/static|_next/image|favicon.ico).*)"],
};
