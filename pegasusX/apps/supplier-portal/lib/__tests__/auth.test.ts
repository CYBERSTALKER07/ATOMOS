import { beforeEach, describe, expect, it, vi } from "vitest";
import { readTokenFromCookie } from "../auth";

vi.mock("@/lib/bridge", () => ({
  isTauri: () => false,
  getStoredToken: vi.fn(),
  storeToken: vi.fn(),
  clearStoredToken: vi.fn(),
}));

describe("readTokenFromCookie", () => {
  beforeEach(() => {
    Object.defineProperty(document, "cookie", { writable: true, value: "" });
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
});
