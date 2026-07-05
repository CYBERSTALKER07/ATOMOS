import { isTauri } from "./tauri-runtime";

const SCHEME_PREFIX = /^[a-z0-9-]+:\/\//i;

/** Map `pegasusx-retailer://orders` → `/orders` for Next.js router. */
export function desktopDeepLinkToPath(url: string): string | null {
  const trimmed = url.trim();
  if (!SCHEME_PREFIX.test(trimmed)) {
    return trimmed.startsWith("/") ? trimmed : null;
  }
  const withoutScheme = trimmed.replace(/^[^:]+:\/\//, "");
  if (!withoutScheme) {
    return "/";
  }
  return withoutScheme.startsWith("/") ? withoutScheme : `/${withoutScheme}`;
}

/** Subscribe to OS deep links (Tauri single-instance + deep-link plugin). */
export async function subscribeDesktopDeepLinks(
  onNavigate: (path: string) => void,
): Promise<() => void> {
  if (!isTauri()) {
    return () => {};
  }

  const cleanups: Array<() => void> = [];

  const handleUrl = (url: string) => {
    const path = desktopDeepLinkToPath(url);
    if (path) {
      onNavigate(path);
    }
  };

  try {
    const { listen } = await import("@tauri-apps/api/event") as {
      listen: (event: string, handler: (event: { payload: string }) => void) => Promise<() => void>;
    };
    const unlisten = await listen("pegasusx-deep-link", (event) => {
      handleUrl(event.payload);
    });
    cleanups.push(unlisten);
  } catch {
    // ignore — plugin may emit only via deep-link JS API
  }

  try {
    const { onOpenUrl } = await import("@tauri-apps/plugin-deep-link") as {
      onOpenUrl: (handler: (urls: string[]) => void) => Promise<() => void>;
    };
    const unlisten = await onOpenUrl((urls) => {
      for (const url of urls) {
        handleUrl(url);
      }
    });
    cleanups.push(unlisten);
  } catch {
    // deep-link plugin optional on web dev
  }

  return () => {
    for (const cleanup of cleanups) {
      cleanup();
    }
  };
}
