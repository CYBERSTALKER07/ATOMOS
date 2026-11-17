import { NextRequest, NextResponse } from "next/server";

export const dynamic = "force-dynamic";

const MFA_TOKEN_REFRESH_ROUTES = new Set([
  "v1/platform-admin/mfa/confirm",
  "v1/platform-admin/mfa/verify",
]);

const DEFAULT_BACKEND_BASE = "http://localhost:8080";
const HOP_BY_HOP_AND_SPOOFED = new Set([
  "connection",
  "keep-alive",
  "proxy-authenticate",
  "proxy-authorization",
  "te",
  "trailer",
  "transfer-encoding",
  "upgrade",
  "host",
  "content-length",
  "cookie",
  "set-cookie",
  "x-forwarded-for",
  "x-real-ip",
  "forwarded",
  "x-forwarded-host",
  "x-forwarded-proto",
  "x-client-ip",
]);

const ALLOWED_ADMIN_PREFIXES = [
  "v1/platform-admin/",
  "v1/admin/",
  "v1/finance/",
  "v1/audit/",
  "v1/iam/",
  "v1/foundry/",
  "v1/auth/",
];

function backendBaseURL(): string {
  return (
    process.env.ADMIN_BACKEND_BASE_URL ||
    process.env.BACKEND_BASE_URL ||
    process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
    DEFAULT_BACKEND_BASE
  ).replace(/\/$/, "");
}

function validateAdminProxyPath(segments: string[]): string | null {
  if (!Array.isArray(segments) || segments.length === 0) return null;
  for (const seg of segments) {
    if (!seg || seg === "." || seg === ".." || seg.includes("\\") || /[\x00-\x1f]/.test(seg)) {
      return null;
    }
  }
  const normalized = segments.join("/").replace(/^\/+/, "");
  const ok = ALLOWED_ADMIN_PREFIXES.some(
    (prefix) => normalized === prefix.replace(/\/$/, "") || normalized.startsWith(prefix),
  );
  return ok ? normalized : null;
}

type RouteContext = { params: Promise<{ path: string[] }> };

async function proxy(req: NextRequest, context: RouteContext) {
  const resolved = await context.params;
  const pathname = validateAdminProxyPath(resolved.path);
  if (!pathname) {
    return Response.json({ error: "route_not_allowed" }, { status: 403 });
  }

  const outboundHeaders = new Headers();
  req.headers.forEach((value, key) => {
    if (!HOP_BY_HOP_AND_SPOOFED.has(key.toLowerCase())) {
      outboundHeaders.set(key, value);
    }
  });

  const usedCookieAuth = !outboundHeaders.has("Authorization");
  if (usedCookieAuth) {
    const cookieToken = req.cookies.get("pegasusx_admin_token")?.value;
    if (!cookieToken) {
      return Response.json({ error: "unauthorized" }, { status: 401 });
    }
    outboundHeaders.set("Authorization", `Bearer ${cookieToken}`);
  }

  const init: RequestInit = {
    method: req.method,
    headers: outboundHeaders,
    redirect: "manual",
    cache: "no-store",
  };
  if (!["GET", "HEAD"].includes(req.method.toUpperCase())) {
    init.body = await req.arrayBuffer();
  }

  try {
    const upstream = await fetch(`${backendBaseURL()}/${pathname}${req.nextUrl.search}`, init);
    const responseHeaders = new Headers();
    upstream.headers.forEach((value, key) => {
      if (!HOP_BY_HOP_AND_SPOOFED.has(key.toLowerCase())) {
        responseHeaders.append(key, value);
      }
    });

    if (usedCookieAuth && upstream.ok && MFA_TOKEN_REFRESH_ROUTES.has(pathname)) {
      const payload = (await upstream.json()) as Record<string, unknown>;
      const refreshedToken = typeof payload.token === "string" ? payload.token.trim() : "";
      const { token: _stripped, ...safePayload } = payload;
      const res = NextResponse.json(safePayload, {
        status: upstream.status,
        headers: responseHeaders,
      });
      if (refreshedToken) {
        res.cookies.set("pegasusx_admin_token", refreshedToken, {
          httpOnly: true,
          secure: process.env.NODE_ENV === "production",
          sameSite: "strict",
          path: "/",
          maxAge: 60 * 60 * 24,
        });
      }
      return res;
    }

    return new Response(upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return Response.json({ error: "upstream_unreachable" }, { status: 502 });
  }
}

export const GET = proxy;
export const POST = proxy;
export const PUT = proxy;
export const PATCH = proxy;
export const DELETE = proxy;
