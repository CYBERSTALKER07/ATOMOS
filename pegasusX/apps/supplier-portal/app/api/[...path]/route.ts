import { NextRequest } from "next/server";

// Dynamic server-side BFF proxy for web mode; stashed by scripts/next_tauri_static_export.sh during Tauri static export.
export const dynamic = "force-dynamic";

const DEFAULT_BACKEND_BASE_URL = "http://localhost:8180";
const HOP_BY_HOP_HEADERS = new Set([
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

const ALLOWED_SUPPLIER_PREFIXES = [
  "healthz",
  "v1/supplier/",
  "v1/warehouse/",
  "v1/auth/",
  "v1/orders/",
  "v1/catalog/",
  "v1/payment/",
  "v1/compliance/",
  "v1/admin/credit/",
  "v1/admin/tax-regimes",
  "v1/as2/",
  "v1/user/",
  "v1/platform/",
  "v1/approvals/",
  "v1/proposals/",
  "v1/finance/",
  "v1/fiscal/",
  "v1/foundry/",
  "v1/credit/",
  "v1/planning/",
  "v1/replenishment/",
  "v1/roi/",
  "v1/oidc/",
  "v1/ontology/",
  "v1/connectors/",
  "v1/org/",
  "v1/p2p/",
  "v1/processes/",
  "v1/anomalies/",
  "v1/iam/",
  "v1/whatif/",
  "v1/export-packs/",
  "v1/accounts/",
  "v1/audit/",
  "v1/control-tower/",
  "v1/twin/",
  "scim/v2/",
];

function isAllowedUpstreamPath(segments: string[]): string | null {
  if (!Array.isArray(segments) || segments.length === 0) return null;
  for (const seg of segments) {
    if (!seg || seg === "." || seg === ".." || seg.includes("\\") || /[\x00-\x1f]/.test(seg)) {
      return null;
    }
  }
  const normalized = segments.join("/").replace(/^\/+/, "");
  if (normalized.startsWith("v1/platform-admin/")) return null;
  const ok = ALLOWED_SUPPLIER_PREFIXES.some(
    (prefix) => normalized === prefix.replace(/\/$/, "") || normalized.startsWith(prefix),
  );
  return ok ? normalized : null;
}

function backendBaseURL() {
  return (process.env.SUPPLIER_BACKEND_BASE_URL || process.env.BACKEND_BASE_URL || DEFAULT_BACKEND_BASE_URL).replace(/\/$/, "");
}

function targetURL(pathname: string, search: string) {
  return `${backendBaseURL()}/${pathname}${search}`;
}

type RouteParams = { path: string[] };
type RouteContext = { params: Promise<RouteParams> };

async function proxy(req: NextRequest, context: RouteContext) {
  const resolved = await context.params;
  const pathname = isAllowedUpstreamPath(resolved.path);
  if (!pathname) {
    return Response.json({ error: "route_not_allowed" }, { status: 403 });
  }
  const url = targetURL(pathname, req.nextUrl.search);

  const outboundHeaders = new Headers();
  req.headers.forEach((value: string, key: string) => {
    const lower = key.toLowerCase();
    if (HOP_BY_HOP_HEADERS.has(lower)) {
      return;
    }
    outboundHeaders.set(key, value);
  });

  if (!outboundHeaders.has("Authorization")) {
    const cookieJwt = req.cookies.get("supplier_jwt")?.value;
    if (cookieJwt) {
      outboundHeaders.set("Authorization", `Bearer ${cookieJwt}`);
    }
  }

  if (!outboundHeaders.has("Idempotency-Key") && req.headers.has("X-Idempotency-Key")) {
    outboundHeaders.set("Idempotency-Key", req.headers.get("X-Idempotency-Key") || "");
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
    const upstream = await fetch(url, init);
    const responseHeaders = new Headers();
    upstream.headers.forEach((value, key) => {
      const lower = key.toLowerCase();
      if (!HOP_BY_HOP_HEADERS.has(lower)) {
        responseHeaders.append(key, value);
      }
    });

    return new Response(upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return Response.json({ error: "upstream_unreachable" }, { status: 502 });
  }
}

export async function GET(req: NextRequest, context: RouteContext) {
  return proxy(req, context);
}

export async function POST(req: NextRequest, context: RouteContext) {
  return proxy(req, context);
}

export async function PUT(req: NextRequest, context: RouteContext) {
  return proxy(req, context);
}

export async function PATCH(req: NextRequest, context: RouteContext) {
  return proxy(req, context);
}

export async function DELETE(req: NextRequest, context: RouteContext) {
  return proxy(req, context);
}
