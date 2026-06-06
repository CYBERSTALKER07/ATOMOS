import { NextRequest } from "next/server";

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
]);

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
  const pathname = resolved.path.join("/");
  const url = targetURL(pathname, req.nextUrl.search);

  const outboundHeaders = new Headers();
  req.headers.forEach((value, key) => {
    const lower = key.toLowerCase();
    if (HOP_BY_HOP_HEADERS.has(lower)) {
      return;
    }
    outboundHeaders.set(key, value);
  });

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
      if (!HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
        responseHeaders.append(key, value);
      }
    });

    return new Response(upstream.body, {
      status: upstream.status,
      headers: responseHeaders,
    });
  } catch {
    return Response.json({ error: "upstream_unreachable", target: url }, { status: 502 });
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
