import { defaultLocale, type Locale } from "./locales";

export type CatalogValue = string | CatalogTree;

export interface CatalogTree {
  [key: string]: CatalogValue;
}

export type FlatCatalog = Record<string, string>;

export interface TranslationContract {
  defaultLocale: Locale;
  supportedLocales: readonly Locale[];
  uzbekScript: "latin";
  fallbackChain: readonly Locale[];
  placeholderSyntax: "{name}";
  interpolationRules: {
    currency: "Intl.NumberFormat";
    date: "Intl.DateTimeFormat";
    number: "Intl.NumberFormat";
    plural: "Intl.PluralRules";
  };
}

export const translationContract: TranslationContract = {
  defaultLocale,
  supportedLocales: ["en", "ru", "uz"],
  uzbekScript: "latin",
  fallbackChain: ["en"],
  placeholderSyntax: "{name}",
  interpolationRules: {
    currency: "Intl.NumberFormat",
    date: "Intl.DateTimeFormat",
    number: "Intl.NumberFormat",
    plural: "Intl.PluralRules",
  },
};

export function flattenCatalog(
  tree: CatalogTree,
  prefix = "",
  entries: FlatCatalog = {},
): FlatCatalog {
  for (const [key, value] of Object.entries(tree)) {
    const nextKey = prefix ? `${prefix}.${key}` : key;

    if (typeof value === "string") {
      entries[nextKey] = value;
      continue;
    }

    flattenCatalog(value, nextKey, entries);
  }

  return entries;
}

export function getTranslation(
  catalogs: Partial<Record<Locale, FlatCatalog>>,
  locale: Locale,
  key: string,
): string {
  const localized = catalogs[locale]?.[key];
  if (localized) {
    return localized;
  }

  return catalogs[defaultLocale]?.[key] ?? key;
}

export function formatTranslation(
  template: string,
  values: Record<string, string | number>,
): string {
  return template.replace(/\{([a-zA-Z0-9_]+)\}/g, (_, token: string) => {
    const value = values[token];
    return value === undefined ? `{${token}}` : String(value);
  });
}
