import { defaultLocale, localeCookieKey, resolveLocale, type Locale } from "./locales";

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

export function syncDocumentLocale(locale: Locale, storageKey = localeCookieKey): Locale {
  if (typeof document !== "undefined") {
    document.documentElement.lang = locale;
    document.cookie = `${storageKey}=${encodeURIComponent(locale)}; path=/; max-age=31536000; SameSite=Lax`;
  }

  if (typeof window !== "undefined") {
    try {
      window.localStorage.setItem(storageKey, locale);
    } catch {
      // Ignore storage write failures; lang attribute is already updated.
    }
  }

  return locale;
}

export function bootstrapBrowserLocale(storageKey = localeCookieKey): Locale {
  const storedLocale = readStoredLocale(storageKey);
  if (storedLocale) {
    return syncDocumentLocale(storedLocale, storageKey);
  }

  return syncDocumentLocale(detectBrowserLocale(), storageKey);
}
