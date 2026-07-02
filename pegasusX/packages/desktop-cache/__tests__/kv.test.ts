import { afterEach, beforeEach, describe, expect, it } from "vitest";
import { cacheDelete, cacheGet, cacheSet } from "../kv";

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
  };
}

describe("kv web fallback", () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, "localStorage", {
      value: createLocalStorage(),
      configurable: true,
    });
  });

  afterEach(() => {
    localStorage.clear();
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
});
