import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { ApiClient } from "@pegasusx/api-core";
import { createSupplierApi } from "../api";
import { readTokenFromCookie, supplierSessionFetch } from "../auth";

vi.mock("@/lib/bridge", () => ({
  isTauri: () => false,
  getStoredToken: vi.fn().mockResolvedValue(null),
  storeToken: vi.fn(),
  clearStoredToken: vi.fn(),
}));

describe("readTokenFromCookie", () => {
  beforeEach(() => {
    Object.defineProperty(document, "cookie", { writable: true, value: "" });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns empty when cookie unset", () => {
    expect(readTokenFromCookie()).toBe("");
  });

  it("reads supplier_jwt from cookie", () => {
    Object.defineProperty(document, "cookie", {
      writable: true,
      value: "supplier_jwt=tok-123",
    });
    expect(readTokenFromCookie()).toBe("tok-123");
  });

  it("decodes URI-encoded cookie value", () => {
    Object.defineProperty(document, "cookie", {
      writable: true,
      value: `supplier_jwt=${encodeURIComponent("a/b+c")}`,
    });
    expect(readTokenFromCookie()).toBe("a/b+c");
  });

  it("resolves relative /api baseUrl without invalid URL error and preserves Headers in supplierSessionFetch", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ status: "FOUND", normalized_inn: "301234567", legal_name: "Demo LLC" }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    const api = createSupplierApi();
    const res = await api.lookupPlatformINN({ inn: "301234567", tenant_type: "SUPPLIER" });
    expect(res.status).toBe("FOUND");
    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(fetchMock.mock.calls[0][0]).toBe("/api/v1/platform/inn/lookup");

    const client = new ApiClient({
      baseUrl: "/api",
      getAuthToken: () => "tok-abc",
      fetchImpl: supplierSessionFetch,
    });
    await client.request("/v1/supplier/profile", "POST", {
      body: { ping: true },
      idempotencyKey: "idem-key-1",
    });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    const [, init] = fetchMock.mock.calls[1];
    const hdrs = new Headers(init?.headers);
    expect(hdrs.get("authorization")).toBe("Bearer tok-abc");
    expect(hdrs.get("idempotency-key")).toBe("idem-key-1");
  });

  it("gives migration endpoints a 300s timeout and honors caller AbortSignal while keeping 30s default", async () => {
    const setTimeoutSpy = vi.spyOn(window, "setTimeout");
    const fetchMock = vi.fn().mockImplementation((_url: string, reqInit?: RequestInit) => {
      return new Promise<Response>((_resolve, reject) => {
        const sig = reqInit?.signal;
        if (sig?.aborted) {
          reject(new DOMException("Aborted", "AbortError"));
          return;
        }
        sig?.addEventListener("abort", () => reject(new DOMException("Aborted", "AbortError")), {
          once: true,
        });
      });
    });
    vi.stubGlobal("fetch", fetchMock);

    const migAbort = new AbortController();
    const p1 = supplierSessionFetch("/v1/supplier/migrations/sessions/sess-1/apply", {
      method: "POST",
      signal: migAbort.signal,
    });
    await vi.waitFor(() => {
      expect(setTimeoutSpy).toHaveBeenCalledWith(expect.any(Function), 300_000);
    });
    const passedSignal = fetchMock.mock.calls[0][1]?.signal as AbortSignal;
    expect(passedSignal.aborted).toBe(false);
    migAbort.abort();
    expect(passedSignal.aborted).toBe(true);
    await expect(p1).rejects.toThrow(/timed out/i);

    const stdAbort = new AbortController();
    stdAbort.abort();
    await expect(
      supplierSessionFetch("/v1/supplier/profile", { signal: stdAbort.signal }),
    ).rejects.toThrow(/timed out/i);
    expect(setTimeoutSpy).toHaveBeenLastCalledWith(expect.any(Function), 30_000);
    expect((fetchMock.mock.calls[1][1]?.signal as AbortSignal).aborted).toBe(true);
  });
});
