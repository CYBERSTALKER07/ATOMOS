/**
 * Dual-mode bridge for Web ↔ Tauri Desktop.
 *
 * On desktop (Tauri), this module uses the Tauri IPC layer to invoke Rust
 * commands and listen to native events. On web, all calls gracefully degrade
 * to browser-native APIs or no-ops.
 *
 * Detection: `window.__TAURI_INTERNALS__` is injected by Tauri 2.0 runtime.
 */

/* ── Environment Detection ─────────────────────────────────────────────── */

export function isTauri(): boolean {
  if (typeof window === "undefined") return false;
  return !!(window as unknown as Record<string, unknown>).__TAURI_INTERNALS__;
}

/* ── Lazy Tauri API Import ─────────────────────────────────────────────── */

async function getTauriCore() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return await import("@tauri-apps/api/core");
}

async function getTauriEvent() {
  if (!isTauri()) throw new Error("Not running in Tauri");
  return await import("@tauri-apps/api/event");
}

/* ── Generic IPC ───────────────────────────────────────────────────────── */

/**
 * Invoke a Rust command via Tauri IPC. Throws if not in Tauri context.
 */
export async function tauriInvoke<T>(
  cmd: string,
  args?: Record<string, unknown>
): Promise<T> {
  const { invoke } = await getTauriCore();
  return invoke<T>(cmd, args);
}

/**
 * Listen to a Tauri event. Returns an unlisten function.
 */
export async function tauriListen<T>(
  event: string,
  handler: (payload: T) => void
): Promise<() => void> {
  const { listen } = await getTauriEvent();
  const unlisten = await listen<T>(event, (e) => handler(e.payload));
  return unlisten;
}

/* ── App Info ──────────────────────────────────────────────────────────── */

export interface AppInfo {
  version: string;
  platform: string;
  arch: string;
  debug: boolean;
}

export async function getAppInfo(): Promise<AppInfo | null> {
  if (!isTauri()) return null;
  return tauriInvoke<AppInfo>("get_app_info");
}

/* ── Security / Token Storage ──────────────────────────────────────────── */

interface TokenResult {
  success: boolean;
  token: string | null;
  error: string | null;
}

/**
 * Store JWT in OS keychain (desktop) or cookie (web).
 */
export async function storeToken(
  token: string,
  refreshToken?: string
): Promise<boolean> {
  if (!isTauri()) {
    // Web: set cookie (existing flow handles this)
    return false;
  }
  const result = await tauriInvoke<TokenResult>("store_token", {
    token,
    refreshToken: refreshToken ?? null,
  });
  return result.success;
}

/**
 * Retrieve JWT from OS keychain (desktop) or cookie (web).
 */
export async function getStoredToken(): Promise<string | null> {
  if (!isTauri()) return null;
  const result = await tauriInvoke<TokenResult>("get_token");
  return result.token;
}

/**
 * Clear all stored tokens (desktop keychain + web cookies).
 */
export async function clearStoredToken(): Promise<boolean> {
  if (!isTauri()) return false;
  const result = await tauriInvoke<TokenResult>("clear_token");
  return result.success;
}

/* ── Video Evidence ────────────────────────────────────────────────────── */

export interface EvidenceResult {
  success: boolean;
  output_path: string | null;
  output_size_bytes: number | null;
  upload_url: string | null;
  error: string | null;
}

export interface CompressionProgress {
  percent: number;
  stage: string;
}

/**
 * Process video evidence:
 * - Desktop: Compress via FFmpeg (C++/Rust sidecar) then upload.
 * - Web: Standard multipart upload (no compression).
 */
export async function processVideoEvidence(
  file: File,
  apiUrl: string,
  token: string,
  orderId: string,
  onProgress?: (p: CompressionProgress) => void
): Promise<EvidenceResult> {
  if (isTauri()) {
    // Desktop: full compression pipeline via Rust
    let unlisten: (() => void) | undefined;
    if (onProgress) {
      unlisten = await tauriListen<CompressionProgress>(
        "evidence:progress",
        onProgress
      );
    }

    try {
      // For Tauri, we need the file path from the native file dialog
      const filePath = (file as unknown as { path?: string }).path;
      if (!filePath) {
        throw new Error("File path not available. Use native file picker.");
      }

      return await tauriInvoke<EvidenceResult>("compress_and_upload", {
        filePath,
        uploadConfig: { apiUrl, token, orderId },
      });
    } finally {
      unlisten?.();
    }
  }

  // Web: standard upload
  onProgress?.({ percent: 10, stage: "Uploading…" });
  const form = new FormData();
  form.append("file", file);
  form.append("order_id", orderId);

  const resp = await fetch(`${apiUrl}/v1/evidence/upload`, {
    method: "POST",
    headers: { Authorization: `Bearer ${token}` },
    body: form,
  });

  onProgress?.({ percent: 100, stage: "Done" });

  if (!resp.ok) {
    return {
      success: false,
      output_path: null,
      output_size_bytes: null,
      upload_url: null,
      error: `Upload failed: ${resp.status} ${resp.statusText}`,
    };
  }

  const data = await resp.json();
  return {
    success: true,
    output_path: null,
    output_size_bytes: file.size,
    upload_url: data.url ?? null,
    error: null,
  };
}

/* ── Telemetry WebSocket ───────────────────────────────────────────────── */

export interface WsHealth {
  connected: boolean;
  reconnect_count: number;
  last_message_epoch: number;
  uptime_seconds: number;
}

/**
 * Connect the persistent Rust-backed telemetry WebSocket (desktop only).
 * On web, returns false — the page should use its own browser WebSocket.
 */
export async function connectNativeTelemetry(
  apiUrl: string,
  token: string
): Promise<boolean> {
  if (!isTauri()) return false;
  await tauriInvoke("connect_telemetry", { apiUrl, token });
  return true;
}

/**
 * Disconnect the Rust-backed telemetry WebSocket.
 */
export async function disconnectNativeTelemetry(): Promise<void> {
  if (!isTauri()) return;
  await tauriInvoke("disconnect_telemetry");
}

/**
 * Get telemetry WebSocket health metrics (desktop only).
 */
export async function getTelemetryHealth(): Promise<WsHealth | null> {
  if (!isTauri()) return null;
  return tauriInvoke<WsHealth>("get_ws_health");
}

/**
 * Listen for GPS pings from the Rust telemetry WebSocket.
 * Returns unlisten function. On web, returns a no-op.
 */
export async function onTelemetryPing(
  handler: (ping: { driver_id: string; latitude: number; longitude: number; timestamp?: number }) => void
): Promise<() => void> {
  if (!isTauri()) return () => {};
  return tauriListen("telemetry:ping", handler);
}

/**
 * Listen for telemetry connection status changes.
 * Status values: "CONNECTING" | "LIVE" | "OFFLINE"
 */
export async function onTelemetryStatus(
  handler: (status: string) => void
): Promise<() => void> {
  if (!isTauri()) return () => {};
  return tauriListen("telemetry:status", handler);
}

/* ── File Picker ───────────────────────────────────────────────────────── */

/**
 * Open a native file dialog (desktop) or trigger input[type=file] (web).
 * Returns the selected file or null if cancelled.
 */
export async function pickFile(
  filters?: { name: string; extensions: string[] }[]
): Promise<File | null> {
  if (isTauri()) {
    const { open } = await import("@tauri-apps/plugin-dialog");
    const filePath = await open({
      multiple: false,
      filters: filters?.map((f) => ({ name: f.name, extensions: f.extensions })),
    });
    if (!filePath) return null;

    // Read file through Tauri FS
    const { readFile } = await import("@tauri-apps/plugin-fs");
    const contents = await readFile(filePath as string);
    const name = (filePath as string).split("/").pop() ?? "file";
    return new File([contents], name);
  }

  // Web: use hidden input element
  return new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    if (filters?.length) {
      input.accept = filters
        .flatMap((f) => f.extensions.map((e) => `.${e}`))
        .join(",");
    }
    input.onchange = () => {
      resolve(input.files?.[0] ?? null);
    };
    input.click();
  });
}

/* ── Gateway Connect ───────────────────────────────────────────────────── */

export interface GatewayConnectEvent {
  session_id: string;
  gateway: string;
  status: "opened" | "closed" | "failed";
  error?: string;
}

/**
 * Open a gateway connect window (desktop) or popup (web).
 * Returns the window label (desktop) or null (web).
 */
export async function openGatewayConnect(
  sessionId: string,
  gateway: string,
  redirectUrl: string
): Promise<string | null> {
  if (isTauri()) {
    return tauriInvoke<string>("open_gateway_connect", {
      request: { session_id: sessionId, gateway, redirect_url: redirectUrl },
    });
  }

  // Web: open a controlled popup
  const popup = window.open(
    redirectUrl,
    `gateway-connect-${sessionId}`,
    "width=600,height=700,scrollbars=yes,resizable=no"
  );
  if (!popup) {
    return null; // Popup blocked
  }
  return `gateway-connect-${sessionId}`;
}

/**
 * Close a gateway connect window by label.
 */
export async function closeGatewayConnect(label: string): Promise<void> {
  if (isTauri()) {
    await tauriInvoke("close_gateway_connect", { label });
    return;
  }
  // Web: no reliable way to close cross-origin popup
}

/**
 * Listen for gateway connect lifecycle events.
 * On web, polls the popup window for closure.
 */
export async function onGatewayConnect(
  handler: (event: GatewayConnectEvent) => void,
  popupRef?: { label: string; popup: Window | null }
): Promise<() => void> {
  if (isTauri()) {
    return tauriListen<GatewayConnectEvent>("gateway:connect", handler);
  }

  // Web: poll popup for closure
  if (!popupRef?.popup) return () => {};
  const interval = setInterval(() => {
    if (popupRef.popup?.closed) {
      clearInterval(interval);
      handler({
        session_id: popupRef.label.replace("gateway-connect-", ""),
        gateway: "",
        status: "closed",
      });
    }
  }, 500);
  return () => clearInterval(interval);
}
