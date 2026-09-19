import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  DEFAULT_CACHE_MAX_AGE_MS,
  cacheClearAll,
  cacheClearPrefix,
  cacheDelete,
  cacheGet,
  cacheSet,
  scopedCacheKey,
} from "../kv";

function createLocalStorage() {
  const store = new Map<string, string>();
  return {
    getItem: (key: string) => store.get(key) ?? null,
    setItem: (key: string, value: string) => {
      store.set(key, value);
    },
    removeItem: (key: string) => {
      store.delete(key);
    },
    clear: () => {
      store.clear();
    },
    get length() {
      return store.size;
    },
    key: (index: number) => [...store.keys()][index] ?? null,
  };
}

describe("kv web fallback", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      value: createLocalStorage(),
      configurable: true,
    });
    vi.useRealTimers();
  });

  afterEach(() => {
    localStorage.clear();
    vi.useRealTimers();
  });

  it("round-trips JSON via localStorage when not in tauri", async () => {
    await cacheSet("orders", [{ id: "o1" }]);
    const value = await cacheGet<{ id: string }[]>("orders");
    expect(value).toEqual([{ id: "o1" }]);
  });

  it("deletes cached values", async () => {
    await cacheSet("profile", { id: "r1" });
    await cacheDelete("profile");
    await expect(cacheGet("profile")).resolves.toBeNull();
  });

  it("honors maxAgeMs and deletes stale entries", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    await cacheSet("orders", [{ id: "o1" }]);
    vi.setSystemTime(new Date("2026-01-01T00:20:00Z"));
    await expect(
      cacheGet("orders", { maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS }),
    ).resolves.toBeNull();
    await expect(cacheGet("orders")).resolves.toBeNull();
  });

  it("returns fresh entries within maxAgeMs", async () => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-01-01T00:00:00Z"));
    await cacheSet("orders", [{ id: "o1" }]);
    vi.setSystemTime(new Date("2026-01-01T00:05:00Z"));
    await expect(
      cacheGet<{ id: string }[]>("orders", { maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS }),
    ).resolves.toEqual([{ id: "o1" }]);
  });

  it("clears all keys", async () => {
    await cacheSet("a", 1);
    await cacheSet("b", 2);
    await cacheClearAll();
    await expect(cacheGet("a")).resolves.toBeNull();
    await expect(cacheGet("b")).resolves.toBeNull();
  });

  it("clears by prefix", async () => {
    await cacheSet(scopedCacheKey("r1", "/v1/orders"), []);
    await cacheSet(scopedCacheKey("r2", "/v1/orders"), [{ id: "keep" }]);
    await cacheClearPrefix(scopedCacheKey("r1", ""));
    await expect(cacheGet(scopedCacheKey("r1", "/v1/orders"))).resolves.toBeNull();
    await expect(cacheGet(scopedCacheKey("r2", "/v1/orders"))).resolves.toEqual([
      { id: "keep" },
    ]);
  });

  it("builds scoped keys", () => {
    expect(scopedCacheKey("  ", "/x")).toBe("anon::/x");
    expect(scopedCacheKey("ret-1", "/v1/x")).toBe("ret-1::/v1/x");
  });
});
