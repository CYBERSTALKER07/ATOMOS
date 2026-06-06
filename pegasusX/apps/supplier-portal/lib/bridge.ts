export function isTauri(): boolean {
  if (typeof window === "undefined") return false;
  return !!(window as unknown as Record<string, unknown>).__TAURI_INTERNALS__;
}

async function getTauriCore() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return await import("@tauri-apps/api/core");
}

interface TokenResult {
  success: boolean;
  token: string | null;
  error: string | null;
}

export async function storeToken(token: string): Promise<boolean> {
  if (!isTauri()) return false;
  const result = await (await getTauriCore()).invoke<TokenResult>("store_token", { token });
  return result.success;
}

export async function getStoredToken(): Promise<string | null> {
  if (!isTauri()) return null;
  const result = await (await getTauriCore()).invoke<TokenResult>("get_token");
  return result.token;
}

export async function clearStoredToken(): Promise<boolean> {
  if (!isTauri()) return false;
  const result = await (await getTauriCore()).invoke<TokenResult>("clear_token");
  return result.success;
}
