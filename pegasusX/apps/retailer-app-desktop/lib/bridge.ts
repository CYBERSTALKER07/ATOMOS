// lib/bridge.ts

declare global {
  interface Window { __TAURI__?: unknown; }
}

// Fallback empty bridge for SSR
export function isTauri(): boolean {
  return typeof window !== 'undefined' && window.__TAURI__ !== undefined;
}

export async function storeToken(token: string, refreshToken: string): Promise<void> {
  if (isTauri()) {
    try {
      const { invoke } = await import('@tauri-apps/api/core');
      await invoke('store_token', { token, refreshToken });
    } catch (err) {
      console.warn('Tauri storeToken failed', err);
    }
  } else {
    // Retailer Desktop is Tauri-only. No web/localStorage fallback.
    // Auth tokens must use OS keyring (macOS: Keychain, Windows: Credential Manager).
    console.warn('storeToken called outside Tauri environment. Token NOT stored.');
  }
}

export async function getStoredToken(): Promise<string | null> {
  if (isTauri()) {
    try {
      const { invoke } = await import('@tauri-apps/api/core');
      const result = (await invoke('get_token')) as { token?: string | null } | null;
      return result?.token ?? null;
    } catch (err) {
      console.warn('Tauri getStoredToken failed', err);
      return null;
    }
  } else {
    // Retailer Desktop is Tauri-only. No web/localStorage fallback.
    console.warn('getStoredToken called outside Tauri environment.');
    return null;
  }
}

export async function clearStoredToken(): Promise<void> {
  if (isTauri()) {
    try {
      const { invoke } = await import('@tauri-apps/api/core');
      await invoke('clear_token');
    } catch (err) {
      console.warn('Tauri clearStoredToken failed', err);
    }
  } else {
    // Retailer Desktop is Tauri-only. No web/localStorage fallback.
    console.warn('clearStoredToken called outside Tauri environment.');
  }
}