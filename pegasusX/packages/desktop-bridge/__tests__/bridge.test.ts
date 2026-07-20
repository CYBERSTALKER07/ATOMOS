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

describe("desktop updater helpers on web", () => {
  afterEach(() => {
    delete (window as { __TAURI__?: unknown }).__TAURI__;
    delete (window as { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__;
  });

  it("isDesktopUpdaterAvailable is false outside tauri", async () => {
    const { isDesktopUpdaterAvailable } = await import("../updater");
    expect(isDesktopUpdaterAvailable()).toBe(false);
  });

  it("checkDesktopUpdate returns null outside tauri", async () => {
    const { checkDesktopUpdate } = await import("../updater");
    await expect(checkDesktopUpdate()).resolves.toBeNull();
  });

  it("installPendingDesktopUpdate returns false without pending", async () => {
    const { installPendingDesktopUpdate } = await import("../updater");
    await expect(installPendingDesktopUpdate()).resolves.toBe(false);
  });

  it("desktopClientPolicyContext is web/production outside Tauri", async () => {
    const { desktopClientPolicyContext, ENTERPRISE_DESKTOP_CHANNEL } =
      await import("../updater");
    expect(desktopClientPolicyContext()).toEqual({
      platform: "web",
      channel: "production",
    });
    expect(ENTERPRISE_DESKTOP_CHANNEL).toBe("enterprise");
  });

  it("desktopClientPolicyContext is desktop/enterprise inside Tauri (default)", async () => {
    (window as { __TAURI_INTERNALS__?: unknown }).__TAURI_INTERNALS__ = {};
    const { desktopClientPolicyContext, isDesktopCdnOtaEnabled } =
      await import("../updater");
    expect(desktopClientPolicyContext()).toEqual({
      platform: "desktop",
      channel: "enterprise",
    });
    expect(isDesktopCdnOtaEnabled()).toBe(true);
  });

  it("store distribution disables CDN OTA and uses production channel", async () => {
    (window as { __TAURI_INTERNALS__?: unknown; __PEGASUSX_DESKTOP_DISTRIBUTION__?: string }).__TAURI_INTERNALS__ =
      {};
    (
      window as { __PEGASUSX_DESKTOP_DISTRIBUTION__?: string }
    ).__PEGASUSX_DESKTOP_DISTRIBUTION__ = "store";
    const {
      desktopClientPolicyContext,
      isDesktopCdnOtaEnabled,
      isDesktopStoreBuild,
      checkDesktopUpdate,
    } = await import("../updater");
    expect(isDesktopStoreBuild()).toBe(true);
    expect(isDesktopCdnOtaEnabled()).toBe(false);
    expect(desktopClientPolicyContext()).toEqual({
      platform: "desktop",
      channel: "production",
    });
    await expect(checkDesktopUpdate()).resolves.toBeNull();
    delete (window as { __PEGASUSX_DESKTOP_DISTRIBUTION__?: string })
      .__PEGASUSX_DESKTOP_DISTRIBUTION__;
  });
});
