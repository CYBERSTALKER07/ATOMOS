export const supportedLocales = ["en", "ru", "uz"] as const;

export type Locale = (typeof supportedLocales)[number];

export const defaultLocale: Locale = "en";
export const localeCookieKey = "pegasus_locale";

export const localeLabels: Record<Locale, string> = {
  en: "English",
  ru: "Russian",
  uz: "Uzbek",
};

export function isSupportedLocale(value: string): value is Locale {
  return supportedLocales.includes(value as Locale);
}

export function resolveLocale(
  requestedLocale?: string | null,
  fallbackLocale: Locale = defaultLocale,
): Locale {
  if (requestedLocale && isSupportedLocale(requestedLocale)) {
    return requestedLocale;
  }

  return fallbackLocale;
}
