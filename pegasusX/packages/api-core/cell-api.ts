/**
 * GS-C5 — map session `home_cell` to the cell API URL.
 * Catalog hostnames match backend auth.ListCells / terraform api_hostname.
 * Local / proxy bootstraps are never rewritten to api.pegasusx.app.
 */

export const CELL_API_URLS: Record<string, string> = {
  "cell-uz": "https://api.pegasusx.app",
  "cell-eu": "https://api-eu.pegasusx.app",
  "cell-us": "https://api-us.pegasusx.app",
  "cell-kz": "https://api-kz.pegasusx.app",
};

export type PinApiBaseUrlOptions = {
  bootstrap: string;
  homeCell?: string | null;
  sessionApiUrl?: string | null;
};

export function trimApiBase(url: string): string {
  return url.replace(/\/$/, "");
}

export function isDevApiBootstrap(url: string): boolean {
  const raw = trimApiBase(url).toLowerCase();
  if (!raw) return true;
  if (raw === "/api" || raw.endsWith("/api")) return true;
  try {
    const u = new URL(raw.includes("://") ? raw : `http://${raw}`);
    const host = u.hostname;
    if (host === "localhost" || host === "127.0.0.1" || host === "10.0.2.2") return true;
    if (host.startsWith("192.168.") || host.startsWith("10.")) return true;
    if (host.startsWith("172.")) {
      const second = Number(host.split(".")[1] || "0");
      if (second >= 16 && second <= 31) return true;
    }
  } catch {
    return true;
  }
  return false;
}

export function homeCellFromJwt(token: string): string {
  const parts = token.split(".");
  if (parts.length < 2) return "";
  try {
    const b64 = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const pad = b64.length % 4 === 0 ? "" : "=".repeat(4 - (b64.length % 4));
    const json = JSON.parse(atob(b64 + pad)) as { home_cell?: unknown };
    return String(json.home_cell ?? "").toLowerCase().trim();
  } catch {
    return "";
  }
}

/** After login, pin to session.api_url / JWT home_cell. Dev bootstrap wins. */
export function pinApiBaseUrl(opts: PinApiBaseUrlOptions): string {
  const boot = trimApiBase(opts.bootstrap || "");
  if (isDevApiBootstrap(boot)) return boot || "http://localhost:8180";
  const fromSession = trimApiBase(opts.sessionApiUrl || "");
  if (fromSession) return fromSession;
  const cell = String(opts.homeCell || "").toLowerCase().trim();
  if (cell && CELL_API_URLS[cell]) return CELL_API_URLS[cell];
  return boot;
}

export function wsUrlFromApi(api: string): string {
  const base = trimApiBase(api);
  if (base.startsWith("https://")) return `wss://${base.slice("https://".length)}`;
  if (base.startsWith("http://")) return `ws://${base.slice("http://".length)}`;
  return base;
}
