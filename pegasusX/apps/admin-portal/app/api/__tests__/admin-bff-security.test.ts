import { afterEach, describe, expect, it, vi } from "vitest";
import crypto from "node:crypto";
import { NextRequest } from "next/server";

const mockCookieStore = {
  set: vi.fn(),
  get: vi.fn(),
  delete: vi.fn(),
};

vi.mock("next/headers", () => ({
  cookies: vi.fn(async () => mockCookieStore),
}));

import { verifyAdminJwt } from "@/lib/jwt-verify";
import { POST as loginPost } from "@/app/api/auth/login/route";
import { POST as proxyPost } from "@/app/api/proxy/[...path]/route";
import { api } from "@/lib/api";

function signHS256(payload: Record<string, unknown>, secret: string, alg = "HS256"): string {
  const headerB64 = Buffer.from(JSON.stringify({ alg, typ: "JWT" })).toString("base64url");
  const payloadB64 = Buffer.from(JSON.stringify(payload)).toString("base64url");
  const sig = crypto
    .createHmac("sha256", secret)
    .update(`${headerB64}.${payloadB64}`)
    .digest("base64url");
  return `${headerB64}.${payloadB64}.${sig}`;
}

describe("admin-portal BFF cookie, JWT, and proxy hardening", () => {
  afterEach(() => {
    vi.unstubAllEnvs();
    vi.unstubAllGlobals();
    mockCookieStore.set.mockReset();
    mockCookieStore.get.mockReset();
    mockCookieStore.delete.mockReset();
  });

  it("verifies HS256 JWT claims, rejects wrong role/alg/sig, and fails closed in prod without secret", async () => {
    const secret = "test-admin-jwt-secret-32bytes-minimum";
    const exp = Math.floor(Date.now() / 1000) + 3600;
    const valid = signHS256({ sub: "ops@pegasusx.uz", role: "PLATFORM_ADMIN", exp }, secret);

    vi.stubEnv("ADMIN_JWT_SECRET", secret);
    const ok = await verifyAdminJwt(valid);
    expect(ok).toEqual({ subject: "ops@pegasusx.uz", role: "PLATFORM_ADMIN", exp });

    const wrongRole = signHS256({ sub: "sup", role: "SUPPLIER", exp }, secret);
    expect(await verifyAdminJwt(wrongRole)).toBeNull();

    const wrongAlg = signHS256({ sub: "ops", role: "PLATFORM_ADMIN", exp }, secret, "none");
    expect(await verifyAdminJwt(wrongAlg)).toBeNull();

    const wrongSig = signHS256({ sub: "ops", role: "PLATFORM_ADMIN", exp }, "other-secret");
    expect(await verifyAdminJwt(wrongSig)).toBeNull();

    const expired = signHS256(
      { sub: "ops", role: "PLATFORM_ADMIN", exp: Math.floor(Date.now() / 1000) - 10 },
      secret,
    );
    expect(await verifyAdminJwt(expired)).toBeNull();

    vi.stubEnv("ADMIN_JWT_SECRET", "");
    vi.stubEnv("JWT_SECRET", "");
    vi.stubEnv("NODE_ENV", "production");
    expect(await verifyAdminJwt(valid)).toBeNull();
  });

  it("blocks directToken in production and omits raw JWT from login JSON", async () => {
    const secret = "test-admin-jwt-secret-32bytes-minimum";
    const exp = Math.floor(Date.now() / 1000) + 3600;
    const adminJwt = signHS256({ sub: "root@pegasusx.uz", role: "PLATFORM_ADMIN", exp }, secret);

    vi.stubEnv("NODE_ENV", "production");
    vi.stubEnv("ADMIN_JWT_SECRET", secret);

    const directReq = new Request("http://localhost:3004/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ directToken: adminJwt }),
    });
    const directRes = await loginPost(directReq);
    expect(directRes.status).toBe(403);

    vi.stubGlobal(
      "fetch",
      vi.fn(async () =>
        new Response(
          JSON.stringify({ token: adminJwt, role: "PLATFORM_ADMIN", subject: "root@pegasusx.uz" }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        ),
      ),
    );

    const passReq = new Request("http://localhost:3004/api/auth/login", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ subject: "root@pegasusx.uz", password: "correct-horse-battery" }),
    });
    const passRes = await loginPost(passReq);
    expect(passRes.status).toBe(200);
    const body = (await passRes.json()) as Record<string, unknown>;
    expect(body.ok).toBe(true);
    expect("token" in body).toBe(false);
    expect(mockCookieStore.set).toHaveBeenCalledWith(
      "pegasusx_admin_token",
      adminJwt,
      expect.objectContaining({ httpOnly: true, secure: true, sameSite: "strict" }),
    );
  });

  it("blocks disallowed proxy paths, strips spoofed headers, and rotates HttpOnly cookie on MFA verify", async () => {
    const traversalReq = new NextRequest("http://localhost:3004/api/proxy/v1/platform-admin/../supplier/me", {
      method: "POST",
    });
    const traversalRes = await proxyPost(traversalReq, {
      params: Promise.resolve({ path: ["v1", "platform-admin", "..", "supplier", "me"] }),
    });
    expect(traversalRes.status).toBe(403);

    const forbiddenReq = new NextRequest("http://localhost:3004/api/proxy/v1/supplier/orders", {
      method: "POST",
    });
    const forbiddenRes = await proxyPost(forbiddenReq, {
      params: Promise.resolve({ path: ["v1", "supplier", "orders"] }),
    });
    expect(forbiddenRes.status).toBe(403);

    const unauthReq = new NextRequest("http://localhost:3004/api/proxy/v1/platform-admin/mfa/verify", {
      method: "POST",
      body: JSON.stringify({ code: "123456" }),
    });
    const unauthRes = await proxyPost(unauthReq, {
      params: Promise.resolve({ path: ["v1", "platform-admin", "mfa", "verify"] }),
    });
    expect(unauthRes.status).toBe(401);

    let forwardedHeaders: Headers | undefined;
    vi.stubGlobal(
      "fetch",
      vi.fn(async (_url: string, init?: RequestInit) => {
        forwardedHeaders = new Headers(init?.headers);
        return new Response(
          JSON.stringify({ ok: true, verified: true, token: "new-mfa-verified-jwt" }),
          { status: 200, headers: { "Content-Type": "application/json" } },
        );
      }),
    );

    const mfaReq = new NextRequest("http://localhost:3004/api/proxy/v1/platform-admin/mfa/verify", {
      method: "POST",
      headers: {
        cookie: "pegasusx_admin_token=initial-jwt",
        "x-forwarded-for": "1.2.3.4",
        "Content-Type": "application/json",
      },
      body: JSON.stringify({ code: "123456" }),
    });
    const mfaRes = await proxyPost(mfaReq, {
      params: Promise.resolve({ path: ["v1", "platform-admin", "mfa", "verify"] }),
    });
    expect(mfaRes.status).toBe(200);
    expect(forwardedHeaders?.get("Authorization")).toBe("Bearer initial-jwt");
    expect(forwardedHeaders?.has("cookie")).toBe(false);
    expect(forwardedHeaders?.has("x-forwarded-for")).toBe(false);
    const mfaBody = (await mfaRes.json()) as Record<string, unknown>;
    expect(mfaBody.verified).toBe(true);
    expect("token" in mfaBody).toBe(false);
    expect(mfaRes.headers.get("set-cookie")).toContain("pegasusx_admin_token=new-mfa-verified-jwt");
  });

  it("routes cookie-session API calls through /api/proxy with credentials=include and no Bearer header", async () => {
    const fetchCalls: Array<{ url: string; init?: RequestInit }> = [];
    vi.stubGlobal(
      "fetch",
      vi.fn(async (url: string, init?: RequestInit) => {
        fetchCalls.push({ url, init });
        return new Response(JSON.stringify({ enrolled: true, required: true, verified: true }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        });
      }),
    );

    await api.mfaStatus("__cookie_session__");
    expect(fetchCalls).toHaveLength(1);
    expect(fetchCalls[0].url).toBe("/api/proxy/v1/platform-admin/mfa/status");
    expect(fetchCalls[0].init?.credentials).toBe("include");
    const hdrs = fetchCalls[0].init?.headers as Record<string, string>;
    expect(hdrs.Authorization).toBeUndefined();
  });
});
