import { defaultLocale, localeCookieKey, resolveLocale, type Locale } from "./locales";

/** Dispatched on `window` after `syncDocumentLocale` so React translators can re-render. */
export const LOCALE_CHANGE_EVENT = "pegasusx:locale";

function normalizeLanguageTag(language?: string | null): string | null {
  if (!language) {
    return null;
  }

  return language.split("-")[0]?.toLowerCase() ?? null;
}

export function detectBrowserLocale(): Locale {
  if (typeof navigator === "undefined") {
    return defaultLocale;
  }

  return resolveLocale(normalizeLanguageTag(navigator.language));
}

export function readStoredLocale(storageKey = localeCookieKey): Locale | null {
  if (typeof window === "undefined") {
    return null;
  }

  try {
    const storedLocale = window.localStorage.getItem(storageKey);
    return storedLocale ? resolveLocale(storedLocale) : null;
  } catch {
    return null;
  }
}

/** Active portal locale: stored preference, else browser language. */
export function resolveActiveLocale(storageKey = localeCookieKey): Locale {
  return readStoredLocale(storageKey) ?? detectBrowserLocale();
}

export function syncDocumentLocale(locale: Locale, storageKey = localeCookieKey): Locale {
  const resolved = resolveLocale(locale, defaultLocale);

  if (typeof document !== "undefined") {
    document.documentElement.lang = resolved;
    document.cookie = `${storageKey}=${encodeURIComponent(resolved)}; path=/; max-age=31536000; SameSite=Lax`;
  }

  if (typeof window !== "undefined") {
    try {
      window.localStorage.setItem(storageKey, resolved);
    } catch {
      // Ignore storage write failures; lang attribute is already updated.
    }
    window.dispatchEvent(new CustomEvent(LOCALE_CHANGE_EVENT, { detail: { locale: resolved } }));
  }

  return resolved;
}

export function bootstrapBrowserLocale(storageKey = localeCookieKey): Locale {
  const storedLocale = readStoredLocale(storageKey);
  if (storedLocale) {
    return syncDocumentLocale(storedLocale, storageKey);
  }

  return syncDocumentLocale(detectBrowserLocale(), storageKey);
}
