/**
 * Shared Tauri desktop bridge for pegasusX role-row portals.
 * OS keyring token storage, IPC helpers, runtime detection.
 */

export interface TokenResult {
  success: boolean;
  token: string | null;
  error: string | null;
}

export interface AppInfo {
  name: string;
  version: string;
  platform: string;
}

import { isTauri } from "./tauri-runtime";

export { isTauri } from "./tauri-runtime";

async function getTauriCore() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return import("@tauri-apps/api/core");
}

async function getTauriEvent() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return import("@tauri-apps/api/event");
}

export async function tauriInvoke<T>(
  cmd: string,
  args?: Record<string, unknown>,
): Promise<T> {
  const { invoke } = await getTauriCore();
  return (await invoke(cmd, args)) as T;
}

export async function tauriListen<T>(
  event: string,
  handler: (payload: T) => void,
): Promise<() => void> {
  const { listen } = await getTauriEvent();
  const unlisten = await listen(event, (e: { payload: T }) => handler(e.payload));
  return unlisten;
}

/** Persist JWT (and optional refresh token) in the OS keyring. No-op on web. */
export async function storeToken(
  token: string,
  refreshToken?: string | null,
): Promise<boolean> {
  if (!isTauri()) return false;
  try {
    const result = await tauriInvoke<TokenResult>("store_token", {
      token,
      refreshToken: refreshToken ?? null,
    });
    return result.success;
  } catch (err) {
    console.warn("desktop-bridge storeToken failed", err);
    return false;
  }
}

/** Read JWT from OS keyring. Returns null on web or when unset. */
export async function getStoredToken(): Promise<string | null> {
  if (!isTauri()) return null;
  try {
    const result = await tauriInvoke<TokenResult>("get_token");
    return result.token;
  } catch (err) {
    console.warn("desktop-bridge getStoredToken failed", err);
    return null;
  }
}

/** Clear keyring credentials. No-op on web. */
export async function clearStoredToken(): Promise<boolean> {
  if (!isTauri()) return false;
  try {
    const result = await tauriInvoke<TokenResult>("clear_token");
    return result.success;
  } catch (err) {
    console.warn("desktop-bridge clearStoredToken failed", err);
    return false;
  }
}

/** App metadata from the Tauri shell (`get_app_info` command). */
export async function getAppInfo(): Promise<AppInfo | null> {
  if (!isTauri()) return null;
  try {
    return await tauriInvoke<AppInfo>("get_app_info");
  } catch {
    return null;
  }
}

export { escapeCsvCell, formatCsv } from "./format-csv";
export {
  downloadCsv,
  exportCsv,
  saveTextFile,
  type SaveTextFileOptions,
  type SaveTextFileResult,
} from "./file-export";
export {
  desktopPrint,
  isDesktopPrintAvailable,
  isNativeFileExportAvailable,
  savePrintableHtml,
  type DesktopPrintOptions,
} from "./print";
