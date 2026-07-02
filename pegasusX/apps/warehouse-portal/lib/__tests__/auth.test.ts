import { beforeEach, describe, expect, it, vi } from "vitest";
import { decodeJwtPayload, readTokenFromCookie } from "../auth";

vi.mock("@/lib/bridge", () => ({
  isTauri: () => false,
  getStoredToken: vi.fn(),
  storeToken: vi.fn(),
  clearStoredToken: vi.fn(),
}));

describe("warehouse auth utilities", () => {
  describe("readTokenFromCookie", () => {
    beforeEach(() => {
      Object.defineProperty(document, "cookie", { writable: true, value: "" });
    });

    it("reads pegasus_warehouse_jwt from cookie", () => {
      Object.defineProperty(document, "cookie", {
        writable: true,
        value: "pegasus_warehouse_jwt=wh-tok",
      });
      expect(readTokenFromCookie()).toBe("wh-tok");
    });
  });

  describe("decodeJwtPayload", () => {
    it("decodes a valid JWT payload", () => {
      const payload = { sub: "user123", role: "warehouse_admin" };
      const encodedPayload = btoa(JSON.stringify(payload));
      const fakeToken = `header.${encodedPayload}.signature`;
      expect(decodeJwtPayload(fakeToken)).toEqual(payload);
    });

    it("returns null for invalid JWT formats", () => {
      expect(decodeJwtPayload("invalid-token")).toBeNull();
    });
  });
});
