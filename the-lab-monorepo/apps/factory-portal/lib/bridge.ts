/**
 * Dual-mode bridge for Web ↔ Tauri Desktop.
 * On desktop, uses Tauri IPC. On web, degrades to no-ops.
 */

export function isTauri(): boolean {
  if (typeof window === "undefined") return false;
  return !!(window as unknown as Record<string, unknown>).__TAURI_INTERNALS__;
}

async function getTauriCore() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return await import("@tauri-apps/api/core");
}

async function getTauriEvent() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return await import("@tauri-apps/api/event");
}

export async function tauriInvoke<T>(
  cmd: string,
  args?: Record<string, unknown>
): Promise<T> {
  const { invoke } = await getTauriCore();
  return invoke(cmd, args) as Promise<T>;
}

export async function tauriListen<T>(
  event: string,
  handler: (payload: T) => void
): Promise<() => void> {
  const { listen } = await getTauriEvent();
  const unlisten = await listen(event, (e: any) => handler(e.payload));
  return unlisten;
}

interface TokenResult {
  success: boolean;
  token: string | null;
  error: string | null;
}

export async function storeToken(
  token: string,
  refreshToken?: string
): Promise<boolean> {
  if (!isTauri()) return false;
  const result = await tauriInvoke<TokenResult>("store_token", {
    token,
    refreshToken: refreshToken ?? null,
  });
  return result.success;
}

export async function getStoredToken(): Promise<string | null> {
  if (!isTauri()) return null;
  const result = await tauriInvoke<TokenResult>("get_token");
  return result.token;
}

export async function clearStoredToken(): Promise<boolean> {
  if (!isTauri()) return false;
  const result = await tauriInvoke<TokenResult>("clear_token");
  return result.success;
}
