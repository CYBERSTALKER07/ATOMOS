import { afterEach, describe, expect, it } from "vitest";
import { clearStoredToken, getStoredToken, isTauri, storeToken } from "../index";

describe("isTauri", () => {
  afterEach(() => {
    delete (window as { __TAURI__?: unknown }).__TAURI__;
    delete (window as { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__;
  });

  it("returns false without tauri globals", () => {
    expect(isTauri()).toBe(false);
  });

  it("detects __TAURI__", () => {
    (window as { __TAURI__?: unknown }).__TAURI__ = {};
    expect(isTauri()).toBe(true);
  });

  it("detects __TAURI_INTERNALS__", () => {
    (window as { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ = {};
    expect(isTauri()).toBe(true);
  });
});

describe("token helpers on web", () => {
  afterEach(() => {
    delete (window as { __TAURI__?: unknown }).__TAURI__;
    delete (window as { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__;
  });

  it("storeToken returns false outside tauri", async () => {
    await expect(storeToken("jwt")).resolves.toBe(false);
  });

  it("getStoredToken returns null outside tauri", async () => {
    await expect(getStoredToken()).resolves.toBeNull();
  });

  it("clearStoredToken returns false outside tauri", async () => {
    await expect(clearStoredToken()).resolves.toBe(false);
  });
});
