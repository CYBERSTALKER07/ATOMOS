/**
 * Desktop distribution + OTA for Tauri 2 shells.
 *
 * Two channels (do not mix in one binary):
 * - **enterprise** (default): website / GCS CDN, signed Tauri updater
 * - **store**: Microsoft Store / Mac App Store — `channel=production`,
 *   no CDN self-update; open the store listing when policy says outdated
 *
 * Set at build time:
 *   NEXT_PUBLIC_DESKTOP_DISTRIBUTION=enterprise|store
 * Optional store listing:
 *   NEXT_PUBLIC_DESKTOP_STORE_URL=ms-windows-store://… or https://apps.apple.com/…
 */

import { isTauri } from "./tauri-runtime";

/** Client-policy channel for website-installed desktop shells. */
export const ENTERPRISE_DESKTOP_CHANNEL = "enterprise" as const;

/** Client-policy channel for Microsoft Store / Mac App Store builds. */
export const STORE_DESKTOP_CHANNEL = "production" as const;

/** Platform string for GET /v1/platform/client-policy from Tauri shells. */
export const ENTERPRISE_DESKTOP_PLATFORM = "desktop" as const;

export type DesktopDistribution = "enterprise" | "store";

/**
 * Build-time distribution mode.
 * Store binaries MUST set NEXT_PUBLIC_DESKTOP_DISTRIBUTION=store (or production).
 */
export function desktopDistribution(): DesktopDistribution {
  if (typeof process !== "undefined" && process.env) {
    const raw = (
      process.env.NEXT_PUBLIC_DESKTOP_DISTRIBUTION ||
      process.env.DESKTOP_DISTRIBUTION ||
      ""
    )
      .trim()
      .toLowerCase();
    if (
      raw === "store" ||
      raw === "production" ||
      raw === "ms-store" ||
      raw === "microsoft-store" ||
      raw === "mac-app-store" ||
      raw === "appstore"
    ) {
      return "store";
    }
  }
  if (typeof window !== "undefined") {
    const w = window as Window & { __PEGASUSX_DESKTOP_DISTRIBUTION__?: string };
    const raw = (w.__PEGASUSX_DESKTOP_DISTRIBUTION__ || "").trim().toLowerCase();
    if (raw === "store" || raw === "production") return "store";
  }
  return "enterprise";
}

export function isDesktopStoreBuild(): boolean {
  return desktopDistribution() === "store";
}

/** CDN / Tauri plugin OTA only for website enterprise builds. */
export function isDesktopCdnOtaEnabled(): boolean {
  return isTauri() && desktopDistribution() === "enterprise";
}

/**
 * Policy tuple for client-policy calls.
 * - Browser (no Tauri) → web / production
 * - Tauri enterprise → desktop / enterprise
 * - Tauri store → desktop / production
 */
export function desktopClientPolicyContext(): {
  platform: typeof ENTERPRISE_DESKTOP_PLATFORM | "web";
  channel: typeof ENTERPRISE_DESKTOP_CHANNEL | typeof STORE_DESKTOP_CHANNEL;
} {
  if (!isTauri()) {
    return { platform: "web", channel: STORE_DESKTOP_CHANNEL };
  }
  if (isDesktopStoreBuild()) {
    return {
      platform: ENTERPRISE_DESKTOP_PLATFORM,
      channel: STORE_DESKTOP_CHANNEL,
    };
  }
  return {
    platform: ENTERPRISE_DESKTOP_PLATFORM,
    channel: ENTERPRISE_DESKTOP_CHANNEL,
  };
}

/** Optional listing URL for Microsoft Store / Mac App Store (store builds). */
export function desktopStoreListingUrl(): string | null {
  if (typeof process !== "undefined" && process.env) {
    const u = (
      process.env.NEXT_PUBLIC_DESKTOP_STORE_URL ||
      process.env.DESKTOP_STORE_URL ||
      ""
    ).trim();
    if (u) return u;
  }
  if (typeof window !== "undefined") {
    const w = window as Window & { __PEGASUSX_DESKTOP_STORE_URL__?: string };
    const u = (w.__PEGASUSX_DESKTOP_STORE_URL__ || "").trim();
    if (u) return u;
  }
  return null;
}

/**
 * Open Microsoft Store / Mac App Store listing (store builds).
 * Uses Tauri shell plugin when available, else window.open.
 */
export async function openDesktopStoreListing(
  url?: string | null,
): Promise<boolean> {
  const target = (url ?? desktopStoreListingUrl())?.trim();
  if (!target) {
    console.warn(
      "desktop-bridge openDesktopStoreListing: no NEXT_PUBLIC_DESKTOP_STORE_URL",
    );
    return false;
  }
  try {
    if (isTauri()) {
      try {
        const shell = await importPeer<{
          open: (path: string) => Promise<void>;
        }>("@tauri-apps/plugin-shell");
        await shell.open(target);
        return true;
      } catch {
        // fall through
      }
    }
    if (typeof window !== "undefined") {
      window.open(target, "_blank", "noopener,noreferrer");
      return true;
    }
    return false;
  } catch (err) {
    console.warn("desktop-bridge openDesktopStoreListing failed", err);
    return false;
  }
}

export type DesktopUpdateInfo = {
  version: string;
  currentVersion: string;
  body?: string | null;
  date?: string | null;
};

export type DesktopUpdateProgress =
  | { event: "Started"; contentLength?: number | null }
  | { event: "Progress"; chunkLength: number }
  | { event: "Finished" };

type PendingUpdate = {
  version: string;
  currentVersion: string;
  body?: string | null;
  date?: string | null;
  downloadAndInstall: (
    onEvent?: (event: {
      event: string;
      data: { contentLength?: number; chunkLength?: number };
    }) => void,
  ) => Promise<void>;
};

// Specifiers as runtime strings so Vitest/Vite does not hard-fail when peers
// are not linked in pure unit-test installs (apps provide them via workspace).
const UPDATER_PKG = "@tauri-apps/plugin-updater";
const PROCESS_PKG = "@tauri-apps/plugin-process";

let pending: PendingUpdate | null = null;

async function importPeer<T>(specifier: string): Promise<T> {
  // eslint-disable-next-line @typescript-eslint/no-implied-eval -- dynamic peer load
  return (await new Function("s", "return import(s)")(specifier)) as T;
}

/** True when running inside a Tauri desktop shell that can self-update via CDN. */
export function isDesktopUpdaterAvailable(): boolean {
  return isDesktopCdnOtaEnabled();
}

/**
 * Poll the configured GCS updater.json.
 * Returns null when up-to-date, not in Tauri, or **store** distribution
 * (Microsoft Store / Mac App Store own the update path).
 */
export async function checkDesktopUpdate(): Promise<DesktopUpdateInfo | null> {
  if (!isDesktopCdnOtaEnabled()) return null;
  try {
    const mod = await importPeer<{
      check: () => Promise<PendingUpdate | null>;
    }>(UPDATER_PKG);
    const update = await mod.check();
    if (!update) {
      pending = null;
      return null;
    }
    pending = update;
    return {
      version: update.version,
      currentVersion: update.currentVersion,
      body: update.body ?? null,
      date: update.date ?? null,
    };
  } catch (err) {
    console.warn("desktop-bridge checkDesktopUpdate failed", err);
    pending = null;
    return null;
  }
}

/**
 * Download + install the last pending CDN update, then relaunch.
 * No-op on store builds (use {@link openDesktopStoreListing}).
 */
export async function installPendingDesktopUpdate(
  onProgress?: (p: DesktopUpdateProgress) => void,
): Promise<boolean> {
  if (!isDesktopCdnOtaEnabled() || !pending) return false;
  try {
    await pending.downloadAndInstall((event) => {
      if (!onProgress) return;
      if (event.event === "Started") {
        onProgress({
          event: "Started",
          contentLength: event.data.contentLength ?? null,
        });
      } else if (event.event === "Progress") {
        onProgress({
          event: "Progress",
          chunkLength: event.data.chunkLength ?? 0,
        });
      } else if (event.event === "Finished") {
        onProgress({ event: "Finished" });
      }
    });
    pending = null;
    const processMod = await importPeer<{ relaunch: () => Promise<void> }>(
      PROCESS_PKG,
    );
    await processMod.relaunch();
    return true;
  } catch (err) {
    console.warn("desktop-bridge installPendingDesktopUpdate failed", err);
    return false;
  }
}

/**
 * Enterprise: check → download → relaunch.
 * Store: opens store listing if url configured (does not install from CDN).
 */
export async function checkAndInstallDesktopUpdate(
  onProgress?: (p: DesktopUpdateProgress) => void,
): Promise<{ updated: boolean; version?: string; openedStore?: boolean }> {
  if (isDesktopStoreBuild()) {
    const opened = await openDesktopStoreListing();
    return { updated: false, openedStore: opened };
  }
  const info = await checkDesktopUpdate();
  if (!info) return { updated: false };
  const ok = await installPendingDesktopUpdate(onProgress);
  return { updated: ok, version: info.version };
}
