import { describe, it, expect, vi, beforeEach } from "vitest";
import { GET } from "../route";
import { cookies } from "next/headers";

vi.mock("next/headers", () => ({
  cookies: vi.fn(),
}));

function makeJWT(payload: Record<string, any>): string {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url");
  const body = Buffer.from(JSON.stringify(payload)).toString("base64url");
  const sig = Buffer.from("signature").toString("base64url");
  return `${header}.${body}.${sig}`;
}

describe("GET /api/auth/session — httpOnly token leak remediation (P0 Audit Item 10)", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("returns safe metadata and NEVER returns the raw token when valid cookie present", async () => {
    const rawToken = makeJWT({
      sub: "pa-ops-42",
      role: "PLATFORM_ADMIN",
      exp: Math.floor(Date.now() / 1000) + 3600,
    });

    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockImplementation((name: string) => {
        if (name === "pegasusx_admin_token") {
          return { name, value: rawToken };
        }
        return undefined;
      }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(true);
    expect(data.subject).toBe("pa-ops-42");
    expect(data.role).toBe("PLATFORM_ADMIN");

    // Critical invariant: Raw token must NEVER be returned in response JSON
    expect(data.token).toBeUndefined();
    expect(JSON.stringify(data)).not.toContain(rawToken);
  });

  it("returns authenticated: false when cookie is missing", async () => {
    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockReturnValue(undefined),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
  });

  it("returns authenticated: false when cookie value is empty string", async () => {
    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockReturnValue({ name: "pegasusx_admin_token", value: "" }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
  });

  it("returns authenticated: false when token is expired", async () => {
    const expiredToken = makeJWT({
      sub: "pa-expired",
      role: "PLATFORM_ADMIN",
      exp: Math.floor(Date.now() / 1000) - 300, // expired 5 minutes ago
    });

    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockReturnValue({ name: "pegasusx_admin_token", value: expiredToken }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
  });

  it("returns authenticated: false when token has malformed segments", async () => {
    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockReturnValue({ name: "pegasusx_admin_token", value: "invalid.jwt" }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
  });

  it("returns authenticated: false when token payload is not valid JSON", async () => {
    const header = Buffer.from(JSON.stringify({ alg: "HS256" })).toString("base64url");
    const badBody = Buffer.from("not-valid-json").toString("base64url");
    const sig = Buffer.from("signature").toString("base64url");
    const corruptToken = `${header}.${badBody}.${sig}`;

    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockReturnValue({ name: "pegasusx_admin_token", value: corruptToken }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
  });

  it("rejects supplier ADMIN role because admin-portal requires PLATFORM_ADMIN", async () => {
    const supplierAdminToken = makeJWT({
      sub: "sup-admin-99",
      role: "ADMIN",
      supplier_id: "sup-global-1",
      exp: Math.floor(Date.now() / 1000) + 7200,
    });

    vi.mocked(cookies).mockResolvedValue({
      get: vi.fn().mockImplementation((name: string) => {
        if (name === "pegasusx_admin_token") {
          return { name, value: supplierAdminToken };
        }
        return undefined;
      }),
    } as any);

    const response = await GET();
    const data = await response.json();

    expect(data.authenticated).toBe(false);
    expect(data.token).toBeUndefined();
    expect(JSON.stringify(data)).not.toContain(supplierAdminToken);
  });
});
