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

function backendBaseURL() {
  return (process.env.SUPPLIER_BACKEND_BASE_URL || process.env.BACKEND_BASE_URL || DEFAULT_BACKEND_BASE_URL).replace(/\/$/, "");
}

function backendWebSocketURL() {
  const base = backendBaseURL();
  if (base.startsWith("https://")) {
    return `${base.replace(/^https:\/\//, "wss://")}/v1/ws`;
  }
  if (base.startsWith("http://")) {
    return `${base.replace(/^http:\/\//, "ws://")}/v1/ws`;
  }
  return `${base}/v1/ws`;
}

export async function GET(req: NextRequest) {
  const outboundHeaders = new Headers();
  req.headers.forEach((value: string, key: string) => {
    if (HOP_BY_HOP_HEADERS.has(key.toLowerCase())) {
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

  try {
    const upstream = await fetch(`${backendBaseURL()}/v1/supplier/ws-session`, {
      method: "GET",
      headers: outboundHeaders,
      redirect: "manual",
      cache: "no-store",
    });

    const payload = (await upstream.json().catch(() => null)) as
      | { token?: string; expires_at?: string; error?: string }
      | null;

    if (!upstream.ok) {
      return Response.json(payload ?? { error: "upstream_request_failed" }, { status: upstream.status });
    }

    if (!payload || typeof payload.token !== "string" || typeof payload.expires_at !== "string") {
      return Response.json({ error: "invalid_ws_session_response" }, { status: 502 });
    }

    return Response.json(
      {
        token: payload.token,
        expires_at: payload.expires_at,
        websocket_url: backendWebSocketURL(),
      },
      {
        status: 200,
        headers: {
          "Cache-Control": "no-store",
        },
      },
    );
  } catch {
    return Response.json({ error: "upstream_unreachable" }, { status: 502 });
  }
}