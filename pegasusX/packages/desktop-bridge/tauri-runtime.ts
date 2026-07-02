type TauriWindow = Window & {
  __TAURI__?: unknown;
  __TAURI_INTERNALS__?: unknown;
};

/** True when running inside a Tauri webview (v1 and v2 globals). */
export function isTauri(): boolean {
  if (typeof window === "undefined") return false;
  const w = window as TauriWindow;
  return w.__TAURI__ !== undefined || w.__TAURI_INTERNALS__ !== undefined;
}
