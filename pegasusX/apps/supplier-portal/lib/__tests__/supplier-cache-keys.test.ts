import { describe, expect, it, vi } from "vitest";
import { supplierOrdersCacheKey } from "../supplier-cache-keys";

vi.mock("@/lib/supplier-scope", () => ({
  supplierScopeId: () => "sup-test",
}));

describe("supplierOrdersCacheKey", () => {
  it("scopes orders cache key to supplier id and query", () => {
    expect(
      supplierOrdersCacheKey({ limit: 25, offset: 0, filter: "ACTIVE" }),
    ).toBe("/v1/supplier/orders:sup-test?limit=25&offset=0&filter=ACTIVE");
  });
});
